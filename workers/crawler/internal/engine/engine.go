package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/seoxpert/shared/models"
)

// Engine runs SEO checks on a domain using a pool of headless Chrome tabs.
type Engine struct {
	allocCtx context.Context
	mu       sync.Mutex
}

// PageResult holds the crawl output for a single page.
type PageResult struct {
	URL     string
	Issues  []models.AuditIssue
	Score   int
	CrawlMs int64
	Links   []string // discovered internal links for BFS
}

// ─── Signal types for JSON evaluations ───────────────────────────────────────

type linkSignals struct {
	Total        int `json:"total"`
	Internal     int `json:"internal"`
	External     int `json:"external"`
	Empty        int `json:"empty"`
	Generic      int `json:"generic"`
	NofollowInt  int `json:"nofollowInt"`
	HTTPLinks    int `json:"httpLinks"`
}

type imageSignals struct {
	Total       int `json:"total"`
	MissingAlt  int `json:"missingAlt"`
	AltTooLong  int `json:"altTooLong"`
	AltFilename int `json:"altFilename"`
	NoDimension int `json:"noDimension"`
	NoLazy      int `json:"noLazy"`
	NonWebP     int `json:"nonWebP"`
	EmptyAlt    int `json:"emptyAlt"`
}

type headingSignals struct {
	H2Count         int  `json:"h2Count"`
	H3Count         int  `json:"h3Count"`
	EmptyCount      int  `json:"emptyCount"`
	H1Short         bool `json:"h1Short"`
	HierarchyBroken bool `json:"hierarchyBroken"`
}

type techSignals struct {
	HasFavicon      bool   `json:"hasFavicon"`
	Charset         string `json:"charset"`
	InlineStyles    int    `json:"inlineStyles"`
	DOMNodes        int    `json:"domNodes"`
	ScriptCount     int    `json:"scriptCount"`
	StylesheetCount int    `json:"stylesheetCount"`
	IframeCount     int    `json:"iframeCount"`
	IframeNoTitle   int    `json:"iframeNoTitle"`
	DeprecatedTags  int    `json:"deprecatedTags"`
	TableLayout     int    `json:"tableLayout"`
	MixedContent    int    `json:"mixedContent"`
	LoremIpsum      bool   `json:"loremIpsum"`
	MetaRefresh     bool   `json:"metaRefresh"`
	MetaKeywords    string `json:"metaKeywords"`
	FormNoLabel     int    `json:"formNoLabel"`
	FormHTTPAction  int    `json:"formHttpAction"`
	HasSitemapLink  bool   `json:"hasSitemapLink"`
	OGType          string `json:"ogType"`
	OGURL           string `json:"ogUrl"`
	OGSiteName      string `json:"ogSiteName"`
	HreflangCount   int    `json:"hreflangCount"`
	HasSkipNav      bool   `json:"hasSkipNav"`
	RenderBlocking  int    `json:"renderBlocking"`
	ViewportContent string `json:"viewportContent"`
}


// advancedSignals holds URL, CWV, schema, pagination, content-quality signals.
type advancedSignals struct {
	// URL quality
	URLLength         int    `json:"urlLength"`
	URLHasUppercase   bool   `json:"urlHasUppercase"`
	URLHasUnderscore  bool   `json:"urlHasUnderscore"`
	URLHasSpecialChars bool  `json:"urlHasSpecialChars"`
	URLDepth          int    `json:"urlDepth"`

	// Core Web Vitals (measured via Performance API after load)
	FCPMs             float64 `json:"fcpMs"`
	LCPMs             float64 `json:"lcpMs"`

	// Schema types present
	SchemaTypes       []string `json:"schemaTypes"`

	// Pagination
	HasPrevRel        bool   `json:"hasPrevRel"`
	HasNextRel        bool   `json:"hasNextRel"`
	IsPaginated       bool   `json:"isPaginated"`

	// Hreflang
	HasXDefault       bool   `json:"hasXDefault"`
	HreflangLangs     []string `json:"hreflangLangs"`

	// Content quality
	PublishedDate     string `json:"publishedDate"`
	ModifiedDate      string `json:"modifiedDate"`
	AvgSentenceWords  float64 `json:"avgSentenceWords"`
	KeywordDensityPct float64 `json:"keywordDensityPct"`
	BodyWordsFull     int    `json:"bodyWordsFull"`
}

// pageSignals holds all DOM data extracted via chromedp.
type pageSignals struct {
	title       string
	titleLen    int
	metaDesc    string
	metaDescLen int
	canonical   string
	robotsMeta  string
	langAttr    string
	bodyText    string
	h1s         []string
	h1Text      string
	h2s         []string
	hasOGTitle  bool
	hasOGDesc   bool
	hasOGImage  bool
	hasTwitter  bool
	hasSchema   bool
	pageLoadMs  int64
	hasHTTPS    bool
	wordCount   int

	// Composite JS signal sets
	links    linkSignals
	images   imageSignals
	headings headingSignals
	tech     techSignals
	advanced advancedSignals
}

// New creates a chromedp browser allocator.
func New(ctx context.Context) (*Engine, error) {
	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("blink-settings", "imagesEnabled=false"),
		chromedp.WindowSize(1280, 720),
	)
	allocCtx, _ := chromedp.NewExecAllocator(ctx, opts...)
	return &Engine{allocCtx: allocCtx}, nil
}

// NormaliseURL strips trailing slashes and query strings for deduplication.
func NormaliseURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.RawQuery = ""
	u.Fragment = ""
	if len(u.Path) > 1 {
		u.Path = strings.TrimRight(u.Path, "/")
	}
	return u.String()
}

// CrawlPage runs all SEO checks on a single URL and returns issues.
func (e *Engine) CrawlPage(ctx context.Context, auditID, pageURL string) (*PageResult, error) {
	start := time.Now()

	tabCtx, cancel := chromedp.NewContext(e.allocCtx)
	defer cancel()
	tabCtx, tCancel := context.WithTimeout(tabCtx, 20*time.Second)
	defer tCancel()

	var sig pageSignals
	sig.hasHTTPS = strings.HasPrefix(pageURL, "https://")

	// JSON result holders for composite evaluations
	var linkJSON, imageJSON, headingJSON, techJSON, advancedJSON string

	var navStart time.Time
	err := chromedp.Run(tabCtx,
		chromedp.ActionFunc(func(_ context.Context) error { navStart = time.Now(); return nil }),
		chromedp.Navigate(pageURL),
		chromedp.WaitReady("body"),
		chromedp.ActionFunc(func(_ context.Context) error {
			sig.pageLoadMs = time.Since(navStart).Milliseconds()
			return nil
		}),
		// Basic signals
		chromedp.Title(&sig.title),
		chromedp.AttributeValue(`meta[name="description"]`, "content", &sig.metaDesc, nil),
		chromedp.AttributeValue(`link[rel="canonical"]`, "href", &sig.canonical, nil),
		chromedp.AttributeValue(`meta[name="robots"]`, "content", &sig.robotsMeta, nil),
		chromedp.AttributeValue(`html`, "lang", &sig.langAttr, nil),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('h1')).map(el => el.innerText.trim())`, &sig.h1s),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('h2')).map(el => el.innerText.trim().slice(0,200))`, &sig.h2s),
		chromedp.Evaluate(`!!document.querySelector('meta[property="og:title"]')`, &sig.hasOGTitle),
		chromedp.Evaluate(`!!document.querySelector('meta[property="og:description"]')`, &sig.hasOGDesc),
		chromedp.Evaluate(`!!document.querySelector('meta[property="og:image"]')`, &sig.hasOGImage),
		chromedp.Evaluate(`!!document.querySelector('meta[name="twitter:card"]')`, &sig.hasTwitter),
		chromedp.Evaluate(`!!document.querySelector('script[type="application/ld+json"]')`, &sig.hasSchema),
		chromedp.Evaluate(`document.body.innerText.slice(0, 10000)`, &sig.bodyText),

		// ── Composite: Link analysis ──────────────────────────────────────
		chromedp.Evaluate(`JSON.stringify((() => {
			const host = location.hostname;
			const links = [...document.querySelectorAll('a[href]')];
			return {
				total: links.length,
				internal: links.filter(a => a.hostname === host).length,
				external: links.filter(a => a.hostname && a.hostname !== host).length,
				empty: links.filter(a => !a.textContent.trim() && !a.querySelector('img')).length,
				generic: links.filter(a => /^(click here|read more|here|link|more|this|learn more|go|continue|details|download|visit|see more)$/i.test(a.textContent.trim())).length,
				nofollowInt: links.filter(a => a.hostname === host && (a.rel||'').includes('nofollow')).length,
				httpLinks: links.filter(a => a.href.startsWith('http:')).length
			};
		})())`, &linkJSON),

		// ── Composite: Image analysis ─────────────────────────────────────
		chromedp.Evaluate(`JSON.stringify((() => {
			const imgs = [...document.querySelectorAll('img')];
			return {
				total: imgs.length,
				missingAlt: imgs.filter(i => !i.hasAttribute('alt')).length,
				altTooLong: imgs.filter(i => i.alt && i.alt.length > 125).length,
				altFilename: imgs.filter(i => i.alt && /\.[a-z]{2,5}(\?|$)/i.test(i.alt)).length,
				noDimension: imgs.filter(i => !i.getAttribute('width') && !i.getAttribute('height')).length,
				noLazy: imgs.filter(i => !i.getAttribute('loading')).length,
				nonWebP: imgs.filter(i => i.src && /\.(jpe?g|png|gif|bmp)(\?.*)?$/i.test(i.src)).length,
				emptyAlt: imgs.filter(i => i.alt === '' && !i.getAttribute('role')).length
			};
		})())`, &imageJSON),

		// ── Composite: Heading analysis ───────────────────────────────────
		chromedp.Evaluate(`JSON.stringify((() => {
			const hs = [...document.querySelectorAll('h1,h2,h3,h4,h5,h6')];
			const tags = hs.map(h => parseInt(h.tagName[1]));
			let broken = false;
			for (let i = 1; i < tags.length; i++) {
				if (tags[i] > tags[i-1] + 1) { broken = true; break; }
			}
			const h1 = document.querySelector('h1');
			return {
				h2Count: document.querySelectorAll('h2').length,
				h3Count: document.querySelectorAll('h3').length,
				emptyCount: hs.filter(h => !h.textContent.trim()).length,
				h1Short: h1 ? h1.textContent.trim().split(/\s+/).length < 3 : false,
				hierarchyBroken: broken
			};
		})())`, &headingJSON),

		// ── Composite: Technical analysis ────────────────────────────────
		chromedp.Evaluate(`JSON.stringify((() => {
			const scripts = [...document.querySelectorAll('script[src]')];
			return {
				hasFavicon: !!document.querySelector('link[rel*="icon"]'),
				charset: document.characterSet || '',
				inlineStyles: document.querySelectorAll('[style]').length,
				domNodes: document.querySelectorAll('*').length,
				scriptCount: scripts.length,
				stylesheetCount: document.querySelectorAll('link[rel="stylesheet"]').length,
				iframeCount: document.querySelectorAll('iframe').length,
				iframeNoTitle: document.querySelectorAll('iframe:not([title])').length,
				deprecatedTags: document.querySelectorAll('font,center,strike,blink,marquee,tt,big').length,
				tableLayout: [...document.querySelectorAll('table')].filter(t =>
					!t.querySelector('thead') && !t.querySelector('[scope]') && t.rows.length > 2
				).length,
				mixedContent: [...document.querySelectorAll('img[src],script[src]')].filter(e =>
					(e.src||'').startsWith('http:')
				).length,
				loremIpsum: document.body.innerText.toLowerCase().includes('lorem ipsum'),
				metaRefresh: !!document.querySelector('meta[http-equiv="refresh"]'),
				metaKeywords: document.querySelector('meta[name="keywords"]')?.content || '',
				formNoLabel: [...document.querySelectorAll('input:not([type="hidden"]):not([type="submit"]):not([type="button"]):not([type="reset"]):not([type="image"])')].filter(i =>
					!i.labels?.length && !i.getAttribute('aria-label') && !i.getAttribute('aria-labelledby') &&
					!(i.id && document.querySelector('label[for="'+i.id+'"]'))
				).length,
				formHttpAction: [...document.querySelectorAll('form[action]')].filter(f => f.action.startsWith('http:')).length,
				hasSitemapLink: !!document.querySelector('link[rel="sitemap"]'),
				ogType: document.querySelector('meta[property="og:type"]')?.content || '',
				ogUrl: document.querySelector('meta[property="og:url"]')?.content || '',
				ogSiteName: document.querySelector('meta[property="og:site_name"]')?.content || '',
				hreflangCount: document.querySelectorAll('link[hreflang]').length,
				hasSkipNav: !![...document.querySelectorAll('a[href]')].find(a =>
					/^#(main|content|maincontent|primary|skip)/i.test(a.getAttribute('href')||'')
				),
				renderBlocking: [...document.querySelectorAll('script[src]')].filter(s =>
					!s.async && !s.defer && document.head.contains(s)
				).length,
				viewportContent: document.querySelector('meta[name="viewport"]')?.content || ''
			};
		})())`, &techJSON),

		// ── Composite: Advanced signals (URL, CWV, schema, content quality) ──
		chromedp.Evaluate(`JSON.stringify((() => {
			const path = location.pathname;
			const href = location.href;

			// URL signals
			const urlDepth = path.split('/').filter(Boolean).length;
			const urlHasUppercase = /[A-Z]/.test(path);
			const urlHasUnderscore = path.includes('_');
			const urlHasSpecialChars = /[^a-z0-9\-\/\._~%]/.test(path.toLowerCase());

			// Core Web Vitals via Performance API
			let fcpMs = 0, lcpMs = 0;
			try {
				const paintEntries = performance.getEntriesByType('paint');
				const fcp = paintEntries.find(e => e.name === 'first-contentful-paint');
				if (fcp) fcpMs = fcp.startTime;
				const lcpEntries = performance.getEntriesByType('largest-contentful-paint');
				if (lcpEntries.length > 0) lcpMs = lcpEntries[lcpEntries.length-1].startTime;
			} catch(e) {}

			// Schema types
			const schemaTypes = [...document.querySelectorAll('script[type="application/ld+json"]')]
				.map(s => { try { const d = JSON.parse(s.textContent); return Array.isArray(d) ? d.map(x=>x['@type']||'').join(',') : (d['@type']||''); } catch { return ''; } })
				.filter(Boolean);

			// Pagination
			const hasPrevRel = !!document.querySelector('link[rel="prev"]');
			const hasNextRel = !!document.querySelector('link[rel="next"]');
			const isPaginated = /(?:page[=\/]\d+|\/p\/\d+|\?.*page=\d+)/i.test(href);

			// Hreflang x-default
			const hreflangLinks = [...document.querySelectorAll('link[hreflang]')];
			const hasXDefault = hreflangLinks.some(l => (l.getAttribute('hreflang')||'').toLowerCase() === 'x-default');
			const hreflangLangs = hreflangLinks.map(l => l.getAttribute('hreflang')||'');

			// Content freshness
			const publishedDate = document.querySelector('meta[property="article:published_time"]')?.content
				|| document.querySelector('time[itemprop="datePublished"]')?.getAttribute('datetime')
				|| document.querySelector('[class*="publish"],[class*="date"] time')?.getAttribute('datetime')
				|| '';
			const modifiedDate = document.querySelector('meta[property="article:modified_time"]')?.content
				|| document.querySelector('time[itemprop="dateModified"]')?.getAttribute('datetime')
				|| '';

			// Readability: avg words per sentence
			const paras = [...document.querySelectorAll('p')].map(p => p.textContent.trim()).filter(t => t.length > 30);
			let totalSentences = 0, totalWords = 0;
			paras.forEach(p => {
				const sentences = p.split(/[.!?]+/).filter(s => s.trim().length > 5);
				totalSentences += sentences.length;
				sentences.forEach(s => { totalWords += s.trim().split(/\s+/).length; });
			});
			const avgSentenceWords = totalSentences > 0 ? totalWords / totalSentences : 0;

			// Keyword density: use H1 as proxy for primary keyword
			const h1 = (document.querySelector('h1')?.innerText || '').trim().toLowerCase();
			const bodyText = document.body.innerText.toLowerCase();
			const bodyWords = bodyText.split(/\s+/).filter(Boolean);
			let keywordDensityPct = 0;
			if (h1 && h1.split(/\s+/).length >= 2 && bodyWords.length > 50) {
				const kw = h1.split(/\s+/).slice(0,2).join(' ');
				const kwRegex = new RegExp(kw.replace(/[.*+?^${}()|[\]\\]/g,'\\$&'),'gi');
				const matches = (bodyText.match(kwRegex)||[]).length;
				keywordDensityPct = (matches / bodyWords.length) * 100;
			}

			return {
				urlLength: href.length,
				urlHasUppercase,
				urlHasUnderscore,
				urlHasSpecialChars,
				urlDepth,
				fcpMs,
				lcpMs,
				schemaTypes,
				hasPrevRel,
				hasNextRel,
				isPaginated,
				hasXDefault,
				hreflangLangs,
				publishedDate,
				modifiedDate,
				avgSentenceWords,
				keywordDensityPct,
				bodyWordsFull: bodyWords.length
			};
		})())`, &advancedJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("chromedp navigate %s: %w", pageURL, err)
	}

	// Extract internal links in the SAME tab (no second navigation needed)
	var rawInternalLinks []string
	_ = chromedp.Run(tabCtx, chromedp.Evaluate(`
		(function(){
			var host = location.hostname.replace(/^www\./,'');
			var seen = {};
			var out = [];
			document.querySelectorAll('a[href]').forEach(function(a){
				var h = a.href;
				if (!h || !h.startsWith('http')) return;
				var ah = a.hostname.replace(/^www\./,'');
				if (ah !== host) return;
				var ext = h.split('?')[0].split('#')[0];
				if (/\.(pdf|jpg|jpeg|png|gif|css|js|xml|svg|ico|woff|woff2|ttf)$/i.test(ext)) return;
				ext = ext.replace(/\/$/,'');
				if (!seen[ext]) { seen[ext]=1; out.push(ext); }
			});
			return out;
		})()
	`, &rawInternalLinks))

	// ── Parse derived fields ──────────────────────────────────────────────────
	sig.titleLen    = len([]rune(strings.TrimSpace(sig.title)))
	sig.metaDescLen = len([]rune(strings.TrimSpace(sig.metaDesc)))
	sig.wordCount   = len(strings.Fields(sig.bodyText))
	if len(sig.h1s) > 0 {
		sig.h1Text = sig.h1s[0]
	}

	// Unmarshal composite signals (ignore errors — partial data is fine)
	_ = json.Unmarshal([]byte(linkJSON), &sig.links)
	_ = json.Unmarshal([]byte(imageJSON), &sig.images)
	_ = json.Unmarshal([]byte(headingJSON), &sig.headings)
	_ = json.Unmarshal([]byte(techJSON), &sig.tech)
	_ = json.Unmarshal([]byte(advancedJSON), &sig.advanced)

	// ── Run all checks ────────────────────────────────────────────────────────
	result := &PageResult{URL: pageURL, CrawlMs: time.Since(start).Milliseconds(), Links: rawInternalLinks}

	checks := []func() *models.AuditIssue{
		// ── Title (6 checks) ──────────────────────────────────────────────
		func() *models.AuditIssue { return checkTitle(auditID, pageURL, sig.title, sig.titleLen) },
		func() *models.AuditIssue { return checkTitleTooShort(auditID, pageURL, sig.title, sig.titleLen) },
		func() *models.AuditIssue { return checkTitleTooLong(auditID, pageURL, sig.title, sig.titleLen) },
		func() *models.AuditIssue { return checkTitleAllCaps(auditID, pageURL, sig.title) },
		func() *models.AuditIssue { return checkTitleStopWord(auditID, pageURL, sig.title) },
		func() *models.AuditIssue { return checkTitleSameAsMetaDesc(auditID, pageURL, sig.title, sig.metaDesc) },
		// ── Meta Description (4 checks) ───────────────────────────────────
		func() *models.AuditIssue { return checkMetaDescMissing(auditID, pageURL, sig.metaDesc) },
		func() *models.AuditIssue { return checkMetaDescShort(auditID, pageURL, sig.metaDesc, sig.metaDescLen) },
		func() *models.AuditIssue { return checkMetaDescLong(auditID, pageURL, sig.metaDesc, sig.metaDescLen) },
		func() *models.AuditIssue { return checkMetaKeywords(auditID, pageURL, sig.tech.MetaKeywords) },
		// ── Headings (8 checks) ───────────────────────────────────────────
		func() *models.AuditIssue { return checkH1Missing(auditID, pageURL, sig.h1s) },
		func() *models.AuditIssue { return checkH1Multiple(auditID, pageURL, sig.h1s) },
		func() *models.AuditIssue { return checkH1TooLong(auditID, pageURL, sig.h1Text) },
		func() *models.AuditIssue { return checkH1TooShort(auditID, pageURL, sig.h1Text, sig.headings.H1Short) },
		func() *models.AuditIssue { return checkH2Missing(auditID, pageURL, sig.h2s) },
		func() *models.AuditIssue { return checkEmptyHeadings(auditID, pageURL, sig.headings.EmptyCount) },
		func() *models.AuditIssue { return checkHeadingHierarchy(auditID, pageURL, sig.headings.HierarchyBroken) },
		func() *models.AuditIssue { return checkTooManyH2(auditID, pageURL, sig.headings.H2Count) },
		// ── Content (7 checks) ────────────────────────────────────────────
		func() *models.AuditIssue { return checkThinContent(auditID, pageURL, sig.wordCount) },
		func() *models.AuditIssue { return checkTitleH1Duplicate(auditID, pageURL, sig.title, sig.h1Text) },
		func() *models.AuditIssue { return checkNoInternalLinks(auditID, pageURL, sig.links.Internal) },
		func() *models.AuditIssue { return checkTooManyExternalLinks(auditID, pageURL, sig.links.External) },
		func() *models.AuditIssue { return checkGenericAnchorText(auditID, pageURL, sig.links.Generic) },
		func() *models.AuditIssue { return checkEmptyLinks(auditID, pageURL, sig.links.Empty) },
		func() *models.AuditIssue { return checkLoremIpsum(auditID, pageURL, sig.tech.LoremIpsum) },
		// ── Images (7 checks) ─────────────────────────────────────────────
		func() *models.AuditIssue { return checkImageAlt(auditID, pageURL, sig.images.MissingAlt, sig.images.Total) },
		func() *models.AuditIssue { return checkImageAltTooLong(auditID, pageURL, sig.images.AltTooLong) },
		func() *models.AuditIssue { return checkImageAltFilename(auditID, pageURL, sig.images.AltFilename) },
		func() *models.AuditIssue { return checkImageNoDimensions(auditID, pageURL, sig.images.NoDimension, sig.images.Total) },
		func() *models.AuditIssue { return checkImageNoLazy(auditID, pageURL, sig.images.NoLazy, sig.images.Total) },
		func() *models.AuditIssue { return checkImageNotWebP(auditID, pageURL, sig.images.NonWebP, sig.images.Total) },
		// ── Technical (18 checks) ─────────────────────────────────────────
		func() *models.AuditIssue { return checkViewport(auditID, pageURL, sig.tech.ViewportContent) },
		func() *models.AuditIssue { return checkViewportContent(auditID, pageURL, sig.tech.ViewportContent) },
		func() *models.AuditIssue { return checkPageSpeed(auditID, pageURL, sig.pageLoadMs) },
		func() *models.AuditIssue { return checkLang(auditID, pageURL, sig.langAttr) },
		func() *models.AuditIssue { return checkCanonical(auditID, pageURL, sig.canonical) },
		func() *models.AuditIssue { return checkRobotsMeta(auditID, pageURL, sig.robotsMeta) },
		func() *models.AuditIssue { return checkHTTPS(auditID, pageURL, sig.hasHTTPS) },
		func() *models.AuditIssue { return checkFavicon(auditID, pageURL, sig.tech.HasFavicon) },
		func() *models.AuditIssue { return checkCharset(auditID, pageURL, sig.tech.Charset) },
		func() *models.AuditIssue { return checkDeprecatedHTML(auditID, pageURL, sig.tech.DeprecatedTags) },
		func() *models.AuditIssue { return checkLargeDOM(auditID, pageURL, sig.tech.DOMNodes) },
		func() *models.AuditIssue { return checkTooManyScripts(auditID, pageURL, sig.tech.ScriptCount) },
		func() *models.AuditIssue { return checkRenderBlockingScripts(auditID, pageURL, sig.tech.RenderBlocking) },
		func() *models.AuditIssue { return checkTooManyStylesheets(auditID, pageURL, sig.tech.StylesheetCount) },
		func() *models.AuditIssue { return checkIframeNoTitle(auditID, pageURL, sig.tech.IframeNoTitle) },
		func() *models.AuditIssue { return checkMixedContent(auditID, pageURL, sig.tech.MixedContent) },
		func() *models.AuditIssue { return checkTableLayout(auditID, pageURL, sig.tech.TableLayout) },
		func() *models.AuditIssue { return checkMetaRefresh(auditID, pageURL, sig.tech.MetaRefresh) },
		// ── Forms & Accessibility (3 checks) ─────────────────────────────
		func() *models.AuditIssue { return checkFormLabels(auditID, pageURL, sig.tech.FormNoLabel) },
		func() *models.AuditIssue { return checkFormHTTPAction(auditID, pageURL, sig.tech.FormHTTPAction) },
		func() *models.AuditIssue { return checkInlineStyles(auditID, pageURL, sig.tech.InlineStyles) },
		// ── Links technical (2 checks) ────────────────────────────────────
		func() *models.AuditIssue { return checkNofollowInternal(auditID, pageURL, sig.links.NofollowInt) },
		func() *models.AuditIssue { return checkHTTPLinksOnHTTPS(auditID, pageURL, sig.links.HTTPLinks, sig.hasHTTPS) },
		// ── Social / OG (7 checks) ────────────────────────────────────────
		func() *models.AuditIssue { return checkOGTags(auditID, pageURL, sig.hasOGTitle, sig.hasOGDesc, sig.hasOGImage) },
		func() *models.AuditIssue { return checkTwitterCard(auditID, pageURL, sig.hasTwitter) },
		func() *models.AuditIssue { return checkOGType(auditID, pageURL, sig.tech.OGType) },
		func() *models.AuditIssue { return checkOGSiteName(auditID, pageURL, sig.tech.OGSiteName) },
		func() *models.AuditIssue { return checkOGURLMismatch(auditID, pageURL, sig.tech.OGURL, sig.canonical) },
		// ── Structured data (2 checks) ────────────────────────────────────
		func() *models.AuditIssue { return checkSchema(auditID, pageURL, sig.hasSchema) },
		func() *models.AuditIssue { return checkSitemapLink(auditID, pageURL, sig.tech.HasSitemapLink) },
		// ── Advanced schema (4 checks) ───────────────────────────────────────
		func() *models.AuditIssue { return checkBreadcrumbSchema(auditID, pageURL, sig.advanced.SchemaTypes, sig.advanced.URLDepth) },
		func() *models.AuditIssue { return checkFAQSchema(auditID, pageURL, sig.advanced.SchemaTypes, sig.bodyText) },
		func() *models.AuditIssue { return checkProductSchema(auditID, pageURL, sig.advanced.SchemaTypes, sig.bodyText) },
		func() *models.AuditIssue { return checkRatingSchema(auditID, pageURL, sig.advanced.SchemaTypes, sig.bodyText) },
		// ── URL quality (5 checks) ────────────────────────────────────────────
		func() *models.AuditIssue { return checkURLTooLong(auditID, pageURL, sig.advanced.URLLength) },
		func() *models.AuditIssue { return checkURLUppercase(auditID, pageURL, sig.advanced.URLHasUppercase) },
		func() *models.AuditIssue { return checkURLUnderscores(auditID, pageURL, sig.advanced.URLHasUnderscore) },
		func() *models.AuditIssue { return checkURLSpecialChars(auditID, pageURL, sig.advanced.URLHasSpecialChars) },
		func() *models.AuditIssue { return checkURLDepth(auditID, pageURL, sig.advanced.URLDepth) },
		// ── Core Web Vitals (2 checks) ────────────────────────────────────────
		func() *models.AuditIssue { return checkFCP(auditID, pageURL, sig.advanced.FCPMs) },
		func() *models.AuditIssue { return checkLCP(auditID, pageURL, sig.advanced.LCPMs) },
		// ── Pagination (1 check) ──────────────────────────────────────────────
		func() *models.AuditIssue { return checkPaginationMarkup(auditID, pageURL, sig.advanced.IsPaginated, sig.advanced.HasPrevRel, sig.advanced.HasNextRel) },
		// ── Hreflang (2 checks) ───────────────────────────────────────────────
		func() *models.AuditIssue { return checkHreflangXDefault(auditID, pageURL, sig.tech.HreflangCount, sig.advanced.HasXDefault) },
		func() *models.AuditIssue { return checkHreflangSelfRef(auditID, pageURL, sig.tech.HreflangCount, sig.advanced.HreflangLangs) },
		// ── Content quality (3 checks) ────────────────────────────────────────
		func() *models.AuditIssue { return checkKeywordStuffing(auditID, pageURL, sig.advanced.KeywordDensityPct, sig.h1Text) },
		func() *models.AuditIssue { return checkReadability(auditID, pageURL, sig.advanced.AvgSentenceWords, sig.wordCount) },
		func() *models.AuditIssue { return checkContentFreshness(auditID, pageURL, sig.advanced.PublishedDate, sig.advanced.ModifiedDate) },
	}

	for _, check := range checks {
		if iss := check(); iss != nil {
			result.Issues = append(result.Issues, *iss)
		}
	}

	slog.Info("page crawled",
		"url", pageURL,
		"issues", len(result.Issues),
		"ms", result.CrawlMs,
	)
	return result, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper
// ─────────────────────────────────────────────────────────────────────────────

func issue(auditID, pageURL, checkType string, severity models.AuditSeverity, title, desc, suggestion, value string) *models.AuditIssue {
	return &models.AuditIssue{
		AuditID:     auditID,
		URL:         pageURL,
		CheckType:   checkType,
		Severity:    severity,
		Title:       title,
		Description: desc,
		Suggestion:  suggestion,
		Value:       value,
	}
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

// ─────────────────────────────────────────────────────────────────────────────
// TITLE checks
// ─────────────────────────────────────────────────────────────────────────────

func checkTitle(auditID, pageURL, title string, titleLen int) *models.AuditIssue {
	if strings.TrimSpace(title) == "" {
		return issue(auditID, pageURL, "missing_title", models.SeverityCritical,
			"Missing page title",
			"The page has no <title> tag. Title is one of the strongest on-page SEO signals — Google uses it as the primary SERP headline.",
			"Add a unique, descriptive <title> of 50–60 characters containing the primary keyword.",
			"")
	}
	return nil
}

func checkTitleTooShort(auditID, pageURL, title string, titleLen int) *models.AuditIssue {
	t := strings.TrimSpace(title)
	if t == "" || titleLen >= 10 {
		return nil
	}
	return issue(auditID, pageURL, "title_too_short", models.SeverityWarning,
		"Title too short",
		fmt.Sprintf("Title is only %d characters. Very short titles miss opportunities to include primary keywords and context.", titleLen),
		"Expand the title to 50–60 characters. Include the primary keyword near the beginning.",
		truncate(t, 120))
}

func checkTitleTooLong(auditID, pageURL, title string, titleLen int) *models.AuditIssue {
	t := strings.TrimSpace(title)
	if titleLen <= 60 {
		return nil
	}
	return issue(auditID, pageURL, "title_too_long", models.SeverityWarning,
		"Title too long — truncated in Google",
		fmt.Sprintf("Title is %d characters. Google displays ~60 characters in desktop SERPs; the rest is cut off.", titleLen),
		"Shorten the title to 50–60 characters. Put the most important keyword first.",
		truncate(t, 120))
}

func checkTitleAllCaps(auditID, pageURL, title string) *models.AuditIssue {
	t := strings.TrimSpace(title)
	if t == "" || len(t) <= 5 {
		return nil
	}
	if t == strings.ToUpper(t) {
		return issue(auditID, pageURL, "title_all_caps", models.SeverityInfo,
			"Title is ALL CAPS",
			"Titles in all uppercase look spammy in search results and may reduce click-through rate.",
			"Use standard Title Case or sentence case instead.",
			truncate(t, 120))
	}
	return nil
}

var stopWords = []string{"a ", "an ", "the ", "of ", "in ", "on ", "at ", "to ", "for ", "is ", "are ", "was "}

func checkTitleStopWord(auditID, pageURL, title string) *models.AuditIssue {
	t := strings.ToLower(strings.TrimSpace(title))
	if t == "" {
		return nil
	}
	for _, sw := range stopWords {
		if strings.HasPrefix(t, sw) {
			return issue(auditID, pageURL, "title_starts_stopword", models.SeverityInfo,
				"Title starts with a stop word",
				fmt.Sprintf("Title begins with %q — a filler word. Search engines give less weight to words at the start that aren't the primary keyword.", strings.TrimSpace(sw)),
				"Restructure the title so the primary keyword or brand comes first.",
				truncate(strings.TrimSpace(title), 120))
		}
	}
	return nil
}

func checkTitleSameAsMetaDesc(auditID, pageURL, title, metaDesc string) *models.AuditIssue {
	t := strings.ToLower(strings.TrimSpace(title))
	d := strings.ToLower(strings.TrimSpace(metaDesc))
	if t != "" && d != "" && t == d {
		return issue(auditID, pageURL, "title_same_as_meta_desc", models.SeverityWarning,
			"Title and meta description are identical",
			"Having the same text for both the title and meta description wastes the meta description's opportunity to add context and drive clicks.",
			"Write a unique meta description that expands on the title and includes a call to action.",
			truncate(title, 120))
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// META DESCRIPTION checks
// ─────────────────────────────────────────────────────────────────────────────

func checkMetaDescMissing(auditID, pageURL, desc string) *models.AuditIssue {
	if strings.TrimSpace(desc) == "" {
		return issue(auditID, pageURL, "missing_meta_desc", models.SeverityWarning,
			"Missing meta description",
			"No <meta name=\"description\"> tag found. Google often auto-generates a snippet from page content, which may look poor in search results.",
			"Add a unique meta description of 120–155 characters summarising the page and including the primary keyword.",
			"")
	}
	return nil
}

func checkMetaDescShort(auditID, pageURL, desc string, descLen int) *models.AuditIssue {
	d := strings.TrimSpace(desc)
	if d == "" || descLen >= 70 {
		return nil
	}
	return issue(auditID, pageURL, "meta_desc_too_short", models.SeverityInfo,
		"Meta description too short",
		fmt.Sprintf("Description is only %d characters. Short descriptions don't fully use the available SERP snippet space.", descLen),
		"Expand the meta description to 120–155 characters with a clear value proposition and keyword.",
		truncate(d, 200))
}

func checkMetaDescLong(auditID, pageURL, desc string, descLen int) *models.AuditIssue {
	d := strings.TrimSpace(desc)
	if d == "" || descLen <= 160 {
		return nil
	}
	return issue(auditID, pageURL, "meta_desc_too_long", models.SeverityInfo,
		"Meta description too long — truncated in Google",
		fmt.Sprintf("Description is %d characters. Google truncates meta descriptions at ~155–160 characters in SERPs.", descLen),
		"Trim to 120–155 characters. Front-load the most important information.",
		truncate(d, 200))
}

func checkMetaKeywords(auditID, pageURL, keywords string) *models.AuditIssue {
	if strings.TrimSpace(keywords) == "" {
		return nil
	}
	parts := strings.Split(keywords, ",")
	if len(parts) > 5 {
		return issue(auditID, pageURL, "meta_keywords_spam", models.SeverityInfo,
			"Meta keywords tag with excessive keywords",
			fmt.Sprintf("Found %d keywords in <meta name=\"keywords\">. Google has ignored this tag since 2009, and large keyword lists are a legacy spam signal.", len(parts)),
			"Remove the meta keywords tag entirely. Spend that effort on content quality and semantic relevance instead.",
			truncate(keywords, 200))
	}
	return issue(auditID, pageURL, "meta_keywords_present", models.SeverityInfo,
		"Outdated meta keywords tag present",
		"<meta name=\"keywords\"> is present. Google and Bing have ignored this tag for over a decade and its presence signals outdated SEO practices.",
		"Remove the meta keywords tag. It has no positive ranking effect.",
		truncate(keywords, 200))
}

// ─────────────────────────────────────────────────────────────────────────────
// HEADING checks
// ─────────────────────────────────────────────────────────────────────────────

func checkH1Missing(auditID, pageURL string, h1s []string) *models.AuditIssue {
	if len(h1s) == 0 {
		return issue(auditID, pageURL, "missing_h1", models.SeverityCritical,
			"Missing H1 heading",
			"No <h1> tag found on this page. H1 is the primary on-page heading and a strong topical relevance signal for search engines.",
			"Add exactly one H1 containing the primary keyword for this page.",
			"")
	}
	return nil
}

func checkH1Multiple(auditID, pageURL string, h1s []string) *models.AuditIssue {
	if len(h1s) <= 1 {
		return nil
	}
	return issue(auditID, pageURL, "multiple_h1", models.SeverityWarning,
		"Multiple H1 headings",
		fmt.Sprintf("%d H1 tags found. Multiple H1s dilute topical focus and confuse search engines about the page's primary subject.", len(h1s)),
		"Keep exactly one H1. Demote the others to H2 or H3.",
		strings.Join(h1s, " | "))
}

func checkH1TooLong(auditID, pageURL, h1Text string) *models.AuditIssue {
	if h1Text == "" {
		return nil
	}
	words := strings.Fields(h1Text)
	if len(words) <= 10 {
		return nil
	}
	return issue(auditID, pageURL, "h1_too_long", models.SeverityInfo,
		"H1 heading is too long",
		fmt.Sprintf("H1 has %d words. Long H1s dilute keyword focus and reduce their impact as ranking signals.", len(words)),
		"Keep H1 to 4–8 words, centred on the primary keyword.",
		truncate(h1Text, 200))
}

func checkH1TooShort(auditID, pageURL, h1Text string, h1Short bool) *models.AuditIssue {
	if !h1Short || h1Text == "" {
		return nil
	}
	return issue(auditID, pageURL, "h1_too_short", models.SeverityInfo,
		"H1 heading is very short",
		"H1 has fewer than 3 words. Very short H1s miss the chance to include context and secondary keywords.",
		"Expand the H1 to at least 3–5 words describing the page's main topic.",
		truncate(h1Text, 200))
}

func checkH2Missing(auditID, pageURL string, h2s []string) *models.AuditIssue {
	if len(h2s) > 0 {
		return nil
	}
	return issue(auditID, pageURL, "no_h2_headings", models.SeverityInfo,
		"No H2 subheadings",
		"Page has no H2 tags. Subheadings break up content, improve readability, and signal content structure to search engines.",
		"Add H2 headings for each major section, including relevant secondary keywords.",
		"")
}

func checkEmptyHeadings(auditID, pageURL string, emptyCount int) *models.AuditIssue {
	if emptyCount == 0 {
		return nil
	}
	return issue(auditID, pageURL, "empty_headings", models.SeverityWarning,
		"Empty heading tags found",
		fmt.Sprintf("%d heading tag(s) with no text content detected. Empty headings confuse screen readers and waste keyword opportunities.", emptyCount),
		"Remove empty heading tags or add meaningful text to each.",
		fmt.Sprintf("%d empty heading(s)", emptyCount))
}

func checkHeadingHierarchy(auditID, pageURL string, hierarchyBroken bool) *models.AuditIssue {
	if !hierarchyBroken {
		return nil
	}
	return issue(auditID, pageURL, "heading_hierarchy_broken", models.SeverityWarning,
		"Heading hierarchy is broken",
		"Headings skip levels (e.g. H1 → H3 with no H2). This breaks logical document structure and confuses both users and search engine crawlers.",
		"Ensure headings follow a sequential order: H1 → H2 → H3. Never skip a level.",
		"")
}

func checkTooManyH2(auditID, pageURL string, h2Count int) *models.AuditIssue {
	if h2Count <= 15 {
		return nil
	}
	return issue(auditID, pageURL, "too_many_h2", models.SeverityInfo,
		"Excessive number of H2 headings",
		fmt.Sprintf("%d H2 tags found. An unusually high number of H2s may indicate keyword stuffing or poor content structure.", h2Count),
		"Consolidate or reorganise sections. Aim for a focused structure with H2s only for major topic areas.",
		fmt.Sprintf("%d H2 headings", h2Count))
}

// ─────────────────────────────────────────────────────────────────────────────
// CONTENT checks
// ─────────────────────────────────────────────────────────────────────────────

func checkThinContent(auditID, pageURL string, wordCount int) *models.AuditIssue {
	if wordCount >= 300 {
		return nil
	}
	if wordCount < 100 {
		return issue(auditID, pageURL, "thin_content", models.SeverityCritical,
			"Very thin content",
			fmt.Sprintf("Page has only ~%d words. Google's Panda algorithm targets thin, low-value pages — these rarely rank well.", wordCount),
			"Add substantial, original content (minimum 300 words) that provides real value to users.",
			fmt.Sprintf("~%d words detected", wordCount))
	}
	return issue(auditID, pageURL, "low_word_count", models.SeverityWarning,
		"Low word count",
		fmt.Sprintf("Page has ~%d words. Pages with limited content often struggle to rank competitively.", wordCount),
		"Aim for at least 300–500 words of unique, helpful content. Quality matters more than quantity.",
		fmt.Sprintf("~%d words detected", wordCount))
}

func checkTitleH1Duplicate(auditID, pageURL, title, h1 string) *models.AuditIssue {
	t := strings.ToLower(strings.TrimSpace(title))
	h := strings.ToLower(strings.TrimSpace(h1))
	if t == "" || h == "" || t != h {
		return nil
	}
	return issue(auditID, pageURL, "title_h1_duplicate", models.SeverityInfo,
		"Title and H1 are identical",
		"The page title and H1 use exactly the same text. This misses an opportunity to target keyword variations and provide richer context.",
		"Differentiate them: use the H1 for the user-facing headline and the title for the SERP snippet, each with slightly different keyword angles.",
		truncate(title, 200))
}

func checkNoInternalLinks(auditID, pageURL string, internalCount int) *models.AuditIssue {
	if internalCount > 0 {
		return nil
	}
	return issue(auditID, pageURL, "no_internal_links", models.SeverityWarning,
		"No internal links on this page",
		"Page has no links pointing to other pages on the same domain. Internal links distribute PageRank and help search engines discover content.",
		"Add 3–5 contextual internal links to related pages. Use descriptive anchor text with target keywords.",
		"0 internal links found")
}

func checkTooManyExternalLinks(auditID, pageURL string, externalCount int) *models.AuditIssue {
	if externalCount <= 50 {
		return nil
	}
	return issue(auditID, pageURL, "too_many_external_links", models.SeverityWarning,
		"Excessive external links",
		fmt.Sprintf("%d external links found. Linking out heavily can dilute PageRank passed to other pages and may look spammy.", externalCount),
		"Review outbound links. Keep only the most relevant and authoritative. Consider adding rel=\"nofollow\" to commercial/non-editorial links.",
		fmt.Sprintf("%d external links", externalCount))
}

func checkGenericAnchorText(auditID, pageURL string, genericCount int) *models.AuditIssue {
	if genericCount == 0 {
		return nil
	}
	sev := models.SeverityInfo
	if genericCount >= 5 {
		sev = models.SeverityWarning
	}
	return issue(auditID, pageURL, "generic_anchor_text", sev,
		"Generic anchor text used",
		fmt.Sprintf("%d link(s) use generic text like 'click here', 'read more', or 'here'. Generic anchors provide no keyword context to search engines.", genericCount),
		"Replace generic anchor text with descriptive, keyword-rich phrases that describe the linked page's content.",
		fmt.Sprintf("%d link(s) with generic text", genericCount))
}

func checkEmptyLinks(auditID, pageURL string, emptyCount int) *models.AuditIssue {
	if emptyCount == 0 {
		return nil
	}
	return issue(auditID, pageURL, "empty_anchor_links", models.SeverityWarning,
		"Links with no anchor text",
		fmt.Sprintf("%d link(s) have no visible text or image alt text. These provide zero keyword signal to search engines and are inaccessible to screen readers.", emptyCount),
		"Add descriptive text to all links. For image links, ensure the image has a meaningful alt attribute.",
		fmt.Sprintf("%d empty link(s)", emptyCount))
}

func checkLoremIpsum(auditID, pageURL string, found bool) *models.AuditIssue {
	if !found {
		return nil
	}
	return issue(auditID, pageURL, "lorem_ipsum_content", models.SeverityCritical,
		"Placeholder 'Lorem ipsum' text found",
		"Lorem ipsum placeholder text was detected on this page. Publishing placeholder content signals to Google that the page is unfinished and low-quality.",
		"Replace all lorem ipsum text with real, relevant content before publishing.",
		"Lorem ipsum detected")
}

// ─────────────────────────────────────────────────────────────────────────────
// IMAGE checks
// ─────────────────────────────────────────────────────────────────────────────

func checkImageAlt(auditID, pageURL string, missing, total int) *models.AuditIssue {
	if total == 0 || missing == 0 {
		return nil
	}
	pct := (missing * 100) / total
	sev := models.SeverityInfo
	if pct >= 50 {
		sev = models.SeverityWarning
	}
	if missing == total {
		sev = models.SeverityCritical
	}
	return issue(auditID, pageURL, "images_missing_alt", sev,
		"Images missing alt text",
		fmt.Sprintf("%d of %d images (%d%%) have no alt attribute. Alt text is essential for image SEO and accessibility — screen readers and Google Images rely on it.", missing, total, pct),
		"Add a concise, descriptive alt attribute to every content image. For decorative images, use alt=\"\".",
		fmt.Sprintf("%d/%d images missing alt", missing, total))
}

func checkImageAltTooLong(auditID, pageURL string, count int) *models.AuditIssue {
	if count == 0 {
		return nil
	}
	return issue(auditID, pageURL, "img_alt_too_long", models.SeverityInfo,
		"Image alt text too long",
		fmt.Sprintf("%d image(s) have alt text exceeding 125 characters. Overly long alt text is verbose and may be interpreted as keyword stuffing.", count),
		"Keep alt text under 125 characters. Describe the image succinctly — one clear sentence.",
		fmt.Sprintf("%d image(s) affected", count))
}

func checkImageAltFilename(auditID, pageURL string, count int) *models.AuditIssue {
	if count == 0 {
		return nil
	}
	return issue(auditID, pageURL, "img_alt_is_filename", models.SeverityWarning,
		"Image alt text looks like a filename",
		fmt.Sprintf("%d image(s) have alt text that appears to be a filename (e.g. 'IMG_1234.jpg'). This provides no useful context to users or search engines.", count),
		"Replace filename-style alt text with a natural description of the image's content and context.",
		fmt.Sprintf("%d image(s) affected", count))
}

func checkImageNoDimensions(auditID, pageURL string, count, total int) *models.AuditIssue {
	if total == 0 || count == 0 {
		return nil
	}
	pct := (count * 100) / total
	if pct < 30 {
		return nil
	}
	return issue(auditID, pageURL, "img_no_dimensions", models.SeverityInfo,
		"Images missing width/height attributes",
		fmt.Sprintf("%d of %d images lack explicit width and height attributes. This causes layout shifts (CLS) as the page loads, which hurts Core Web Vitals.", count, total),
		"Set explicit width and height on all <img> tags to match the rendered dimensions. This prevents Cumulative Layout Shift.",
		fmt.Sprintf("%d/%d images missing dimensions", count, total))
}

func checkImageNoLazy(auditID, pageURL string, count, total int) *models.AuditIssue {
	if total < 3 || count == 0 {
		return nil
	}
	pct := (count * 100) / total
	if pct < 50 {
		return nil
	}
	return issue(auditID, pageURL, "img_no_lazy_loading", models.SeverityInfo,
		"Images not using lazy loading",
		fmt.Sprintf("%d of %d images lack loading=\"lazy\". Eager-loading all images increases initial page load time, particularly on image-heavy pages.", count, total),
		"Add loading=\"lazy\" to images below the fold. Keep loading=\"eager\" only for above-the-fold hero images.",
		fmt.Sprintf("%d/%d images without lazy loading", count, total))
}

func checkImageNotWebP(auditID, pageURL string, count, total int) *models.AuditIssue {
	if total < 3 || count == 0 {
		return nil
	}
	pct := (count * 100) / total
	if pct < 50 {
		return nil
	}
	return issue(auditID, pageURL, "img_not_webp", models.SeverityInfo,
		"Images not using next-gen format (WebP/AVIF)",
		fmt.Sprintf("%d of %d images are served in JPEG/PNG instead of WebP or AVIF. Next-gen formats are 25–34%% smaller, reducing load time.", count, total),
		"Convert images to WebP format. Use the <picture> element for browser compatibility. Most modern build tools and CDNs can automate this.",
		fmt.Sprintf("%d/%d images in legacy format", count, total))
}

// ─────────────────────────────────────────────────────────────────────────────
// TECHNICAL checks
// ─────────────────────────────────────────────────────────────────────────────

func checkViewport(auditID, pageURL, viewportContent string) *models.AuditIssue {
	if viewportContent != "" {
		return nil
	}
	return issue(auditID, pageURL, "missing_viewport", models.SeverityCritical,
		"Missing viewport meta tag",
		"No <meta name=\"viewport\"> tag found. Without this, mobile browsers render the page at desktop width and it appears zoomed-out and unreadable on phones.",
		"Add: <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">",
		"viewport meta tag not found")
}

func checkViewportContent(auditID, pageURL, viewportContent string) *models.AuditIssue {
	if viewportContent == "" {
		return nil // Already flagged by checkViewport
	}
	vc := strings.ToLower(viewportContent)
	if strings.Contains(vc, "user-scalable=no") || strings.Contains(vc, "maximum-scale=1") {
		return issue(auditID, pageURL, "viewport_zoom_disabled", models.SeverityWarning,
			"Viewport prevents user zoom",
			"The viewport meta tag disables user zooming (user-scalable=no or maximum-scale=1). This is an accessibility violation and penalised by Google's mobile-friendliness checks.",
			"Remove user-scalable=no and maximum-scale=1 from the viewport tag. Allow users to zoom.",
			viewportContent)
	}
	return nil
}

func checkPageSpeed(auditID, pageURL string, loadMs int64) *models.AuditIssue {
	if loadMs > 5000 {
		return issue(auditID, pageURL, "very_slow_page_load", models.SeverityCritical,
			"Very slow page load (>5s)",
			fmt.Sprintf("Page took %dms to load. This is far above Google's 2.5s LCP threshold and will significantly hurt rankings and user retention.", loadMs),
			"Urgently optimise: enable server-side caching, compress assets, eliminate render-blocking resources, use a CDN.",
			fmt.Sprintf("%dms", loadMs))
	}
	if loadMs > 3000 {
		return issue(auditID, pageURL, "slow_page_load", models.SeverityWarning,
			"Slow page load (>3s)",
			fmt.Sprintf("Page took %dms to load. Google recommends under 2.5s for LCP. Pages above 3s lose ~53%% of mobile visitors.", loadMs),
			"Reduce JavaScript bundle size, lazy-load below-fold images, enable Gzip/Brotli compression, use a CDN.",
			fmt.Sprintf("%dms", loadMs))
	}
	return nil
}

func checkLang(auditID, pageURL, lang string) *models.AuditIssue {
	if strings.TrimSpace(lang) != "" {
		return nil
	}
	return issue(auditID, pageURL, "missing_lang", models.SeverityWarning,
		"Missing lang attribute on <html>",
		"The <html> element has no lang attribute. This prevents screen readers from selecting the correct pronunciation engine and affects Google's language targeting.",
		"Add lang=\"en\" (or appropriate BCP 47 language code) to the <html> tag.",
		"<html> — no lang attribute")
}

func checkCanonical(auditID, pageURL, canonical string) *models.AuditIssue {
	if canonical == "" {
		return issue(auditID, pageURL, "missing_canonical", models.SeverityWarning,
			"Missing canonical link",
			"No <link rel=\"canonical\"> tag found. Without it, search engines may index the same page under multiple URLs (with/without trailing slash, query strings, etc.), splitting PageRank.",
			"Add a self-referencing canonical: <link rel=\"canonical\" href=\"{full-page-url}\">",
			"")
	}
	u, err := url.Parse(canonical)
	if err != nil || !u.IsAbs() {
		return issue(auditID, pageURL, "invalid_canonical", models.SeverityWarning,
			"Canonical URL is relative or invalid",
			"The canonical tag's href must be an absolute URL. A relative canonical may be misinterpreted by search engines.",
			"Change the canonical href to the full absolute URL: https://example.com/page-path",
			truncate(canonical, 200))
	}
	return nil
}

func checkRobotsMeta(auditID, pageURL, robotsMeta string) *models.AuditIssue {
	lower := strings.ToLower(robotsMeta)
	if strings.Contains(lower, "noindex") {
		return issue(auditID, pageURL, "noindex_page", models.SeverityCritical,
			"Page is blocked from search indexing",
			"The robots meta tag contains 'noindex'. This page will not appear in Google search results — it is completely excluded from the index.",
			"Remove 'noindex' from the meta robots tag. Only use it intentionally for pages you don't want in search results.",
			robotsMeta)
	}
	if strings.Contains(lower, "nofollow") {
		return issue(auditID, pageURL, "robots_nofollow", models.SeverityWarning,
			"Page has meta robots nofollow",
			"Meta robots 'nofollow' tells search engines not to follow links on this page. This blocks PageRank flow to linked pages.",
			"Remove 'nofollow' from meta robots unless you specifically intend to block link equity on this page.",
			robotsMeta)
	}
	return nil
}

func checkHTTPS(auditID, pageURL string, hasHTTPS bool) *models.AuditIssue {
	if hasHTTPS {
		return nil
	}
	return issue(auditID, pageURL, "no_https", models.SeverityCritical,
		"Page not served over HTTPS",
		"This page uses HTTP instead of HTTPS. Google has used HTTPS as a ranking signal since 2014. Browsers display 'Not Secure' warnings, reducing trust and conversions.",
		"Install an SSL/TLS certificate and configure server-side redirects from HTTP to HTTPS.",
		pageURL)
}

func checkFavicon(auditID, pageURL string, hasFavicon bool) *models.AuditIssue {
	if hasFavicon {
		return nil
	}
	return issue(auditID, pageURL, "missing_favicon", models.SeverityInfo,
		"Missing favicon",
		"No favicon link tag found in <head>. Favicons appear in browser tabs, bookmarks, and some SERP features — they're a basic trust signal.",
		"Add <link rel=\"icon\" href=\"/favicon.ico\"> to your <head>. Use a 32×32 or 64×64 PNG/ICO file.",
		"")
}

func checkCharset(auditID, pageURL, charset string) *models.AuditIssue {
	if strings.TrimSpace(charset) != "" {
		return nil
	}
	return issue(auditID, pageURL, "missing_charset", models.SeverityWarning,
		"Missing charset declaration",
		"No charset declaration detected. Without an explicit charset, browsers and search engines may misinterpret character encoding, causing garbled text.",
		"Add <meta charset=\"UTF-8\"> as the first element in <head>, before any other meta tags.",
		"")
}

func checkDeprecatedHTML(auditID, pageURL string, count int) *models.AuditIssue {
	if count == 0 {
		return nil
	}
	return issue(auditID, pageURL, "deprecated_html_tags", models.SeverityInfo,
		"Deprecated HTML tags in use",
		fmt.Sprintf("%d deprecated HTML element(s) found (e.g. <font>, <center>, <strike>, <blink>). These are obsolete and signal outdated, unmaintained code.", count),
		"Replace deprecated tags with modern CSS equivalents. E.g. use CSS text-align:center instead of <center>.",
		fmt.Sprintf("%d deprecated element(s)", count))
}

func checkLargeDOM(auditID, pageURL string, domNodes int) *models.AuditIssue {
	if domNodes <= 1500 {
		return nil
	}
	sev := models.SeverityInfo
	if domNodes > 3000 {
		sev = models.SeverityWarning
	}
	return issue(auditID, pageURL, "large_dom_size", sev,
		"Large DOM size",
		fmt.Sprintf("DOM has %d nodes. Google recommends under 1,500 DOM nodes. Large DOMs increase memory usage, cause longer style calculations, and slow rendering.", domNodes),
		"Reduce DOM size by eliminating unnecessary wrapper elements, using CSS for visual effects, and lazy-loading off-screen content.",
		fmt.Sprintf("%d DOM nodes", domNodes))
}

func checkTooManyScripts(auditID, pageURL string, count int) *models.AuditIssue {
	if count <= 20 {
		return nil
	}
	return issue(auditID, pageURL, "too_many_scripts", models.SeverityWarning,
		"Too many JavaScript files",
		fmt.Sprintf("%d external scripts loaded. Each script adds a network round-trip and can block rendering, slowing the page significantly.", count),
		"Bundle and minify JavaScript. Remove unused scripts. Defer non-critical scripts with async/defer.",
		fmt.Sprintf("%d script files", count))
}

func checkRenderBlockingScripts(auditID, pageURL string, count int) *models.AuditIssue {
	if count == 0 {
		return nil
	}
	return issue(auditID, pageURL, "render_blocking_scripts", models.SeverityWarning,
		"Render-blocking scripts in <head>",
		fmt.Sprintf("%d synchronous script(s) in <head> without async or defer. These pause HTML parsing and delay the First Contentful Paint.", count),
		"Add async or defer to all <script src=\"...\"> tags in <head> that don't need to run before page render.",
		fmt.Sprintf("%d blocking script(s) in <head>", count))
}

func checkTooManyStylesheets(auditID, pageURL string, count int) *models.AuditIssue {
	if count <= 5 {
		return nil
	}
	return issue(auditID, pageURL, "too_many_stylesheets", models.SeverityInfo,
		"Too many CSS stylesheets",
		fmt.Sprintf("%d CSS files loaded. Multiple stylesheets require multiple HTTP requests and block rendering until all are downloaded and parsed.", count),
		"Concatenate stylesheets into one file. Remove unused CSS. Use a build tool like Vite or Webpack.",
		fmt.Sprintf("%d stylesheet(s)", count))
}

func checkIframeNoTitle(auditID, pageURL string, count int) *models.AuditIssue {
	if count == 0 {
		return nil
	}
	return issue(auditID, pageURL, "iframe_no_title", models.SeverityWarning,
		"iframes missing title attribute",
		fmt.Sprintf("%d iframe(s) have no title attribute. Screen readers cannot describe their purpose, and search engines have no context for the embedded content.", count),
		"Add a descriptive title attribute to every <iframe>. E.g. <iframe title=\"Google Maps location\">.",
		fmt.Sprintf("%d iframe(s) without title", count))
}

func checkMixedContent(auditID, pageURL string, count int) *models.AuditIssue {
	if count == 0 {
		return nil
	}
	return issue(auditID, pageURL, "mixed_content", models.SeverityCritical,
		"Mixed content — HTTP resources on HTTPS page",
		fmt.Sprintf("%d resource(s) are loaded over HTTP on this HTTPS page. Modern browsers block mixed content, breaking page functionality and triggering security warnings.", count),
		"Update all resource URLs (images, scripts) to use HTTPS. Fix in your CDN/asset config.",
		fmt.Sprintf("%d HTTP resource(s) on HTTPS page", count))
}

func checkTableLayout(auditID, pageURL string, count int) *models.AuditIssue {
	if count == 0 {
		return nil
	}
	return issue(auditID, pageURL, "table_layout_usage", models.SeverityInfo,
		"Tables possibly used for layout",
		fmt.Sprintf("%d table(s) without <thead> or scope attributes detected. Using tables for layout is an outdated practice that harms accessibility and semantic structure.", count),
		"Use CSS Grid or Flexbox for page layouts. Reserve <table> for actual tabular data — always include <thead> and scope attributes.",
		fmt.Sprintf("%d layout table(s) detected", count))
}

func checkMetaRefresh(auditID, pageURL string, hasRefresh bool) *models.AuditIssue {
	if !hasRefresh {
		return nil
	}
	return issue(auditID, pageURL, "meta_refresh_redirect", models.SeverityWarning,
		"Meta refresh redirect tag present",
		"<meta http-equiv=\"refresh\"> was found. This client-side redirect delays user experience and may not pass full link equity. Google recommends server-side 301 redirects.",
		"Replace meta refresh with a proper HTTP 301 redirect on the server.",
		"meta http-equiv=\"refresh\" detected")
}

func checkFormLabels(auditID, pageURL string, count int) *models.AuditIssue {
	if count == 0 {
		return nil
	}
	return issue(auditID, pageURL, "form_inputs_no_label", models.SeverityWarning,
		"Form inputs missing labels",
		fmt.Sprintf("%d form input(s) lack associated <label> elements. Unlabelled inputs are inaccessible to screen readers and violate WCAG accessibility guidelines.", count),
		"Add a <label for=\"inputId\"> for every input, or use aria-label/aria-labelledby attributes.",
		fmt.Sprintf("%d unlabelled input(s)", count))
}

func checkFormHTTPAction(auditID, pageURL string, count int) *models.AuditIssue {
	if count == 0 {
		return nil
	}
	return issue(auditID, pageURL, "form_http_action", models.SeverityCritical,
		"Form submits data over HTTP",
		fmt.Sprintf("%d form(s) have an HTTP action URL on an HTTPS page. Form data is transmitted unencrypted — a serious security risk.", count),
		"Change form action URLs to use HTTPS. Never submit sensitive data over HTTP.",
		fmt.Sprintf("%d form(s) with HTTP action", count))
}

func checkInlineStyles(auditID, pageURL string, count int) *models.AuditIssue {
	if count <= 20 {
		return nil
	}
	if count > 100 {
		return issue(auditID, pageURL, "excessive_inline_styles", models.SeverityWarning,
			"Excessive inline styles",
			fmt.Sprintf("%d elements with inline style attributes. Heavy inline styles bypass CSS caching, bloat HTML payload, and make maintenance very difficult.", count),
			"Move styles to external CSS files. Use CSS classes for reusable styles.",
			fmt.Sprintf("%d elements with inline styles", count))
	}
	return issue(auditID, pageURL, "many_inline_styles", models.SeverityInfo,
		"Many inline style attributes",
		fmt.Sprintf("%d elements with inline styles. Inline styles increase HTML size and override external stylesheets, making theming and maintenance harder.", count),
		"Refactor inline styles into CSS classes in an external stylesheet.",
		fmt.Sprintf("%d inline style attributes", count))
}

// ─────────────────────────────────────────────────────────────────────────────
// LINK technical checks
// ─────────────────────────────────────────────────────────────────────────────

func checkNofollowInternal(auditID, pageURL string, count int) *models.AuditIssue {
	if count == 0 {
		return nil
	}
	return issue(auditID, pageURL, "nofollow_internal_links", models.SeverityWarning,
		"Internal links with rel=\"nofollow\"",
		fmt.Sprintf("%d internal link(s) have rel=\"nofollow\". Nofollowing your own internal links blocks PageRank distribution across your site.", count),
		"Remove rel=\"nofollow\" from internal links. Reserve nofollow for external, user-generated, or paid links.",
		fmt.Sprintf("%d nofollow internal link(s)", count))
}

func checkHTTPLinksOnHTTPS(auditID, pageURL string, count int, isHTTPS bool) *models.AuditIssue {
	if !isHTTPS || count == 0 {
		return nil
	}
	return issue(auditID, pageURL, "http_links_on_https", models.SeverityInfo,
		"HTTP links on HTTPS page",
		fmt.Sprintf("%d link(s) point to HTTP URLs from this HTTPS page. While the links themselves don't cause mixed content errors, they send users and crawlers to potentially insecure destinations.", count),
		"Update all internal links to HTTPS. For external links, check if the destination supports HTTPS and update accordingly.",
		fmt.Sprintf("%d HTTP outbound link(s)", count))
}

// ─────────────────────────────────────────────────────────────────────────────
// SOCIAL / OG checks
// ─────────────────────────────────────────────────────────────────────────────

func checkOGTags(auditID, pageURL string, hasTitle, hasDesc, hasImage bool) *models.AuditIssue {
	missing := []string{}
	if !hasTitle {
		missing = append(missing, "og:title")
	}
	if !hasDesc {
		missing = append(missing, "og:description")
	}
	if !hasImage {
		missing = append(missing, "og:image")
	}
	if len(missing) == 3 {
		return issue(auditID, pageURL, "missing_og_tags", models.SeverityWarning,
			"No Open Graph meta tags",
			"No og:title, og:description, or og:image found. Without Open Graph tags, social media shares will display a generic, unattractive preview that significantly reduces click-through.",
			"Add og:title, og:description, and og:image to the <head>. Use a 1200×630px image for best results.",
			"og:title, og:description, og:image — all missing")
	}
	if len(missing) > 0 {
		return issue(auditID, pageURL, "incomplete_og_tags", models.SeverityInfo,
			"Incomplete Open Graph tags",
			fmt.Sprintf("Missing: %s. Incomplete OG tags result in poor social share previews.", strings.Join(missing, ", ")),
			"Add all missing OG tags for optimal social sharing across Facebook, LinkedIn, and Slack.",
			strings.Join(missing, ", "))
	}
	return nil
}

func checkTwitterCard(auditID, pageURL string, hasTwitter bool) *models.AuditIssue {
	if hasTwitter {
		return nil
	}
	return issue(auditID, pageURL, "missing_twitter_card", models.SeverityInfo,
		"Missing Twitter/X Card meta tag",
		"No twitter:card meta tag found. Without it, links shared on Twitter/X show a plain URL instead of a rich card with image and description.",
		"Add: <meta name=\"twitter:card\" content=\"summary_large_image\"> along with twitter:title and twitter:description.",
		"")
}

func checkOGType(auditID, pageURL, ogType string) *models.AuditIssue {
	if strings.TrimSpace(ogType) != "" {
		return nil
	}
	return issue(auditID, pageURL, "og_type_missing", models.SeverityInfo,
		"Missing og:type tag",
		"No og:type found. This tells social platforms the type of content (website, article, product, etc.) for richer rendering.",
		"Add <meta property=\"og:type\" content=\"website\"> for homepages, or \"article\" for blog posts.",
		"")
}

func checkOGSiteName(auditID, pageURL, ogSiteName string) *models.AuditIssue {
	if strings.TrimSpace(ogSiteName) != "" {
		return nil
	}
	return issue(auditID, pageURL, "og_site_name_missing", models.SeverityInfo,
		"Missing og:site_name tag",
		"No og:site_name found. This tag tells social platforms the name of the website to display alongside the page title in share previews.",
		"Add <meta property=\"og:site_name\" content=\"Your Brand Name\">.",
		"")
}

func checkOGURLMismatch(auditID, pageURL, ogURL, canonical string) *models.AuditIssue {
	ogU := strings.TrimSpace(ogURL)
	can := strings.TrimSpace(canonical)
	if ogU == "" || can == "" {
		return nil
	}
	if strings.TrimRight(ogU, "/") != strings.TrimRight(can, "/") {
		return issue(auditID, pageURL, "og_url_mismatch", models.SeverityInfo,
			"og:url doesn't match canonical URL",
			fmt.Sprintf("og:url (%q) differs from the canonical URL (%q). Mismatches can cause inconsistent crawling and sharing behaviour.", ogU, can),
			"Set og:url to exactly the same value as your canonical URL.",
			fmt.Sprintf("og:url=%s vs canonical=%s", ogU, can))
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// STRUCTURED DATA / MISC checks
// ─────────────────────────────────────────────────────────────────────────────

func checkSchema(auditID, pageURL string, hasSchema bool) *models.AuditIssue {
	if hasSchema {
		return nil
	}
	return issue(auditID, pageURL, "no_structured_data", models.SeverityInfo,
		"No structured data (JSON-LD)",
		"No <script type=\"application/ld+json\"> found. Structured data enables Rich Results in Google Search (star ratings, FAQs, breadcrumbs, sitelinks), which dramatically improve CTR.",
		"Add Schema.org JSON-LD markup appropriate to your content type: Organization, Article, Product, FAQPage, BreadcrumbList, etc.",
		"")
}

func checkSitemapLink(auditID, pageURL string, hasSitemapLink bool) *models.AuditIssue {
	if hasSitemapLink {
		return nil
	}
	return issue(auditID, pageURL, "no_sitemap_link", models.SeverityInfo,
		"No sitemap link in <head>",
		"No <link rel=\"sitemap\"> found in <head>. Sitemap link tags help crawlers discover your XML sitemap directly from HTML pages.",
		"Add <link rel=\"sitemap\" type=\"application/xml\" title=\"Sitemap\" href=\"/sitemap.xml\"> to your <head>.",
		"")
}


// ─────────────────────────────────────────────────────────────────────────────
// ADVANCED SCHEMA checks
// ─────────────────────────────────────────────────────────────────────────────

func containsSchemaType(types []string, want string) bool {
	for _, t := range types {
		if strings.Contains(strings.ToLower(t), strings.ToLower(want)) {
			return true
		}
	}
	return false
}

func checkBreadcrumbSchema(auditID, pageURL string, schemaTypes []string, depth int) *models.AuditIssue {
	if depth <= 1 {
		return nil // homepage — skip
	}
	if containsSchemaType(schemaTypes, "BreadcrumbList") {
		return nil
	}
	return issue(auditID, pageURL, "missing_breadcrumb_schema", models.SeverityInfo,
		"Missing BreadcrumbList structured data",
		"Inner pages without BreadcrumbList JSON-LD miss out on breadcrumb rich results in Google, which improve CTR and help users understand site structure.",
		"Add a BreadcrumbList JSON-LD block listing each parent page. Many CMS plugins (Yoast, RankMath) can generate this automatically.",
		fmt.Sprintf("URL depth: %d levels", depth))
}

func checkFAQSchema(auditID, pageURL string, schemaTypes []string, bodyText string) *models.AuditIssue {
	if containsSchemaType(schemaTypes, "FAQPage") {
		return nil
	}
	lower := strings.ToLower(bodyText)
	// Page looks like an FAQ if it has multiple question-pattern repetitions
	questionCount := strings.Count(lower, "frequently asked") +
		strings.Count(lower, "what is ") + strings.Count(lower, "how do ") +
		strings.Count(lower, "why does") + strings.Count(lower, "can i ")
	if questionCount < 4 {
		return nil
	}
	return issue(auditID, pageURL, "missing_faq_schema", models.SeverityInfo,
		"FAQ content detected but no FAQPage schema",
		"This page appears to contain FAQ-style content (multiple question-answer pairs) but has no FAQPage JSON-LD markup. Google can show FAQ rich results directly in SERPs, dramatically increasing SERP real estate.",
		"Wrap your Q&A pairs in FAQPage JSON-LD schema. Each question needs a 'Question' item with an 'Answer'.",
		fmt.Sprintf("~%d question-like patterns detected", questionCount))
}

func checkProductSchema(auditID, pageURL string, schemaTypes []string, bodyText string) *models.AuditIssue {
	if containsSchemaType(schemaTypes, "Product") || containsSchemaType(schemaTypes, "Offer") {
		return nil
	}
	lower := strings.ToLower(bodyText)
	// Simple heuristic: price pattern + "add to cart" / "buy now"
	hasPrice := strings.Contains(lower, "$") || strings.Contains(lower, "₹") ||
		strings.Contains(lower, "price") || strings.Contains(lower, "buy now") ||
		strings.Contains(lower, "add to cart")
	if !hasPrice {
		return nil
	}
	return issue(auditID, pageURL, "missing_product_schema", models.SeverityWarning,
		"Product page missing Product structured data",
		"This page appears to be a product page (price/buy signals detected) but has no Product JSON-LD. Product schema enables rich results showing price, availability, and ratings directly in Google SERPs.",
		"Add Product JSON-LD with name, description, image, offers (price, availability), and ideally aggregateRating.",
		"Price/buy signals detected without Product schema")
}

func checkRatingSchema(auditID, pageURL string, schemaTypes []string, bodyText string) *models.AuditIssue {
	if containsSchemaType(schemaTypes, "AggregateRating") || containsSchemaType(schemaTypes, "Review") {
		return nil
	}
	lower := strings.ToLower(bodyText)
	hasReviews := (strings.Contains(lower, "review") || strings.Contains(lower, "rating") ||
		strings.Contains(lower, "stars") || strings.Contains(lower, "out of 5")) &&
		(strings.Count(lower, "review") >= 3 || strings.Contains(lower, "verified purchase"))
	if !hasReviews {
		return nil
	}
	return issue(auditID, pageURL, "missing_rating_schema", models.SeverityInfo,
		"Review content detected but no AggregateRating schema",
		"This page appears to have review/rating content but lacks AggregateRating JSON-LD. Star ratings in Google SERPs significantly improve click-through rates.",
		"Add AggregateRating inside your Product or Organization schema with ratingValue, ratingCount, and bestRating.",
		"Review/rating content detected without schema markup")
}

// ─────────────────────────────────────────────────────────────────────────────
// URL QUALITY checks
// ─────────────────────────────────────────────────────────────────────────────

func checkURLTooLong(auditID, pageURL string, urlLen int) *models.AuditIssue {
	if urlLen <= 75 {
		return nil
	}
	sev := models.SeverityInfo
	if urlLen > 115 {
		sev = models.SeverityWarning
	}
	return issue(auditID, pageURL, "url_too_long", sev,
		"URL is too long",
		fmt.Sprintf("URL is %d characters. Long URLs are harder to share, less readable in SERPs, and can reduce click-through rates. Google may also truncate them in search results.", urlLen),
		"Keep URLs under 75 characters. Use short, descriptive slugs. Remove stop words (a, the, of, in) from URL paths.",
		fmt.Sprintf("%d characters", urlLen))
}

func checkURLUppercase(auditID, pageURL string, hasUppercase bool) *models.AuditIssue {
	if !hasUppercase {
		return nil
	}
	u, _ := url.Parse(pageURL)
	path := u.Path
	return issue(auditID, pageURL, "url_has_uppercase", models.SeverityWarning,
		"URL contains uppercase letters",
		"URLs with uppercase letters can cause duplicate content issues — servers may treat /Page and /page as different URLs, splitting PageRank. Uppercase URLs also look inconsistent in links.",
		"Redirect uppercase URL variants to lowercase equivalents at the server level. Ensure all internal links use lowercase URLs.",
		path)
}

func checkURLUnderscores(auditID, pageURL string, hasUnderscore bool) *models.AuditIssue {
	if !hasUnderscore {
		return nil
	}
	u, _ := url.Parse(pageURL)
	return issue(auditID, pageURL, "url_has_underscores", models.SeverityWarning,
		"URL uses underscores instead of hyphens",
		"Google treats underscores as word joiners (my_page = 'mypage'), not word separators like hyphens do. Using hyphens (my-page) ensures each word is treated as a separate keyword.",
		"Replace all underscores with hyphens in URL paths. Set up 301 redirects from old underscore URLs to new hyphen versions.",
		u.Path)
}

func checkURLSpecialChars(auditID, pageURL string, hasSpecial bool) *models.AuditIssue {
	if !hasSpecial {
		return nil
	}
	u, _ := url.Parse(pageURL)
	return issue(auditID, pageURL, "url_special_chars", models.SeverityWarning,
		"URL contains special characters",
		"Special characters (other than hyphens) in URLs require percent-encoding, making URLs ugly and error-prone when shared. They can also cause issues with certain crawlers and analytics tools.",
		"Use only lowercase letters, numbers, and hyphens in URL paths. Avoid spaces, ampersands, commas, or other special characters.",
		u.Path)
}

func checkURLDepth(auditID, pageURL string, depth int) *models.AuditIssue {
	if depth <= 4 {
		return nil
	}
	return issue(auditID, pageURL, "url_too_deep", models.SeverityInfo,
		"URL hierarchy too deep",
		fmt.Sprintf("URL is %d levels deep (%s). Pages buried deep in the site structure are harder for search engines to discover and may be perceived as less important.", depth, pageURL),
		"Flatten your site architecture. Aim for all important pages to be reachable within 3 clicks from the homepage.",
		fmt.Sprintf("%d folder levels deep", depth))
}

// ─────────────────────────────────────────────────────────────────────────────
// CORE WEB VITALS checks
// ─────────────────────────────────────────────────────────────────────────────

func checkFCP(auditID, pageURL string, fcpMs float64) *models.AuditIssue {
	if fcpMs <= 0 || fcpMs <= 1800 {
		return nil
	}
	sev := models.SeverityWarning
	msg := "First Contentful Paint is 1.8–3s — needs improvement"
	if fcpMs > 3000 {
		sev = models.SeverityCritical
		msg = "First Contentful Paint is over 3s — very poor"
	}
	return issue(auditID, pageURL, "slow_fcp", sev,
		msg,
		fmt.Sprintf("FCP was %.0fms. First Contentful Paint measures how quickly users see any content. Google's threshold for 'Good' is under 1.8s. Slow FCP increases bounce rate significantly.", fcpMs),
		"Reduce render-blocking resources, preload critical fonts and CSS, enable server-side caching, and use a CDN to reduce latency.",
		fmt.Sprintf("%.0fms (threshold: 1800ms)", fcpMs))
}

func checkLCP(auditID, pageURL string, lcpMs float64) *models.AuditIssue {
	if lcpMs <= 0 || lcpMs <= 2500 {
		return nil
	}
	sev := models.SeverityWarning
	msg := "Largest Contentful Paint is 2.5–4s"
	if lcpMs > 4000 {
		sev = models.SeverityCritical
		msg = "Largest Contentful Paint is over 4s — poor Core Web Vital"
	}
	return issue(auditID, pageURL, "slow_lcp", sev,
		msg,
		fmt.Sprintf("LCP was %.0fms. Largest Contentful Paint is a primary Core Web Vital used in Google's ranking algorithm. 'Good' threshold is under 2.5s.", lcpMs),
		"Optimise the largest above-fold element (usually hero image or H1): preload it, compress it, serve via CDN, and eliminate any render-blocking resources above it.",
		fmt.Sprintf("%.0fms (threshold: 2500ms)", lcpMs))
}

// ─────────────────────────────────────────────────────────────────────────────
// PAGINATION checks
// ─────────────────────────────────────────────────────────────────────────────

func checkPaginationMarkup(auditID, pageURL string, isPaginated, hasPrev, hasNext bool) *models.AuditIssue {
	if !isPaginated {
		return nil
	}
	if hasPrev || hasNext {
		return nil // at least one is present
	}
	return issue(auditID, pageURL, "missing_pagination_markup", models.SeverityWarning,
		"Paginated page missing rel=prev/next",
		"This URL appears to be a paginated page (page=N pattern detected) but has no rel=prev or rel=next link tags. Without these, Google may not understand the page series relationship.",
		"Add a <link rel=\"prev\"> pointing to the previous page and <link rel=\"next\"> pointing to the next page in each paginated series.",
		pageURL)
}

// ─────────────────────────────────────────────────────────────────────────────
// HREFLANG checks
// ─────────────────────────────────────────────────────────────────────────────

func checkHreflangXDefault(auditID, pageURL string, hreflangCount int, hasXDefault bool) *models.AuditIssue {
	if hreflangCount == 0 || hasXDefault {
		return nil
	}
	return issue(auditID, pageURL, "hreflang_missing_xdefault", models.SeverityWarning,
		"Hreflang set missing x-default",
		fmt.Sprintf("This page has %d hreflang tags but is missing the x-default fallback. Without x-default, Google doesn't know which page to show users whose language/region isn't explicitly targeted.", hreflangCount),
		"Add <link rel=\"alternate\" hreflang=\"x-default\" href=\"{url}\"> pointing to the default/international version of the page.",
		fmt.Sprintf("%d hreflang tags, no x-default", hreflangCount))
}

func checkHreflangSelfRef(auditID, pageURL string, hreflangCount int, langs []string) *models.AuditIssue {
	if hreflangCount == 0 {
		return nil
	}
	// Flag if we have hreflang but fewer than 2 lang tags (likely a mis-implementation)
	uniqueLangs := map[string]bool{}
	for _, l := range langs {
		if l != "" && l != "x-default" {
			uniqueLangs[l] = true
		}
	}
	if len(uniqueLangs) >= 2 {
		return nil
	}
	return issue(auditID, pageURL, "hreflang_single_lang", models.SeverityInfo,
		"Hreflang implemented with only one language",
		"Page has hreflang attributes but only one language variant is specified. Hreflang only makes sense when targeting multiple languages/regions — a single-language hreflang is pointless.",
		"Either remove the lone hreflang tag, or add hreflang tags for all language/region variants of this page including x-default.",
		fmt.Sprintf("Only 1 language in hreflang set: %s", strings.Join(langs, ", ")))
}

// ─────────────────────────────────────────────────────────────────────────────
// CONTENT QUALITY checks
// ─────────────────────────────────────────────────────────────────────────────

func checkKeywordStuffing(auditID, pageURL string, densityPct float64, h1Text string) *models.AuditIssue {
	if h1Text == "" || densityPct == 0 {
		return nil
	}
	if densityPct <= 3.5 {
		return nil
	}
	sev := models.SeverityWarning
	label := "high"
	if densityPct > 6 {
		sev = models.SeverityCritical
		label = "very high"
	}
	return issue(auditID, pageURL, "keyword_stuffing", sev,
		fmt.Sprintf("Keyword density too %s — possible keyword stuffing", label),
		fmt.Sprintf("Primary keyword density is %.1f%% (based on H1 keyword %q). Above 3.5%% risks a Google spam penalty. Natural, readable content typically has keyword density of 1–2%%.", densityPct, h1Text),
		"Write naturally for your readers, not for search engines. Use synonyms and related terms (LSI keywords) instead of repeating the exact keyword.",
		fmt.Sprintf("%.1f%% density for %q", densityPct, truncate(h1Text, 60)))
}

func checkReadability(auditID, pageURL string, avgSentenceWords float64, wordCount int) *models.AuditIssue {
	if wordCount < 150 || avgSentenceWords == 0 {
		return nil
	}
	if avgSentenceWords <= 25 {
		return nil
	}
	sev := models.SeverityInfo
	level := "complex"
	if avgSentenceWords > 35 {
		sev = models.SeverityWarning
		level = "very complex"
	}
	return issue(auditID, pageURL, "content_readability", sev,
		fmt.Sprintf("Content readability is %s", level),
		fmt.Sprintf("Average sentence length is %.0f words. Sentences over 20 words are harder to read — this hurts user engagement and dwell time, which are indirect ranking signals.", avgSentenceWords),
		"Break long sentences into shorter ones. Aim for an average of 15–20 words per sentence. Use bullet points and subheadings to break up dense paragraphs.",
		fmt.Sprintf("Average %.0f words/sentence", avgSentenceWords))
}

func checkContentFreshness(auditID, pageURL, publishedDate, modifiedDate string) *models.AuditIssue {
	if publishedDate == "" {
		return nil
	}
	pub, err := time.Parse(time.RFC3339, publishedDate)
	if err != nil {
		// Try date-only format
		pub, err = time.Parse("2006-01-02", publishedDate[:10])
		if err != nil {
			return nil
		}
	}
	ageYears := time.Since(pub).Hours() / 8760
	if ageYears < 1.5 {
		return nil
	}
	// If modified recently, it's fine
	if modifiedDate != "" {
		mod, err := time.Parse(time.RFC3339, modifiedDate)
		if err == nil {
			modAge := time.Since(mod).Hours() / 8760
			if modAge < 1 {
				return nil
			}
		}
	}
	sev := models.SeverityInfo
	if ageYears > 3 {
		sev = models.SeverityWarning
	}
	return issue(auditID, pageURL, "stale_content", sev,
		fmt.Sprintf("Content may be stale (published %.0f years ago)", ageYears),
		fmt.Sprintf("Content published %.0f years ago without a recent modification date. Google favours fresh, up-to-date content — especially for queries where recency matters.", ageYears),
		"Review and update the content. Add new information, update statistics, refresh examples. Update the lastModified/dateModified metadata to reflect the change.",
		fmt.Sprintf("Published: %s", pub.Format("2006-01-02")))
}
