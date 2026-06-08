package engine

import (
	"context"
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
}

// pageSignals holds all DOM data extracted via chromedp.
type pageSignals struct {
	title        string
	metaDesc     string
	h1s          []string
	h2s          []string
	canonical    string
	robotsMeta   string
	langAttr     string
	bodyText     string
	hasViewport  bool
	hasOGTitle   bool
	hasOGDesc    bool
	hasOGImage   bool
	hasTwitter   bool
	imgsMissing  int // images with no alt attribute
	imgsTotal    int
	linksExternal int
	linksBroken  int // placeholder, checked via HEAD requests in future
	pageLoadMs   int64
	hasSchema    bool
	h1Text       string
	titleLen     int
	metaDescLen  int
	wordCount    int
	hasHTTPS     bool
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

// DiscoverLinks extracts all internal links on a page (for multi-page crawl).
func (e *Engine) DiscoverLinks(ctx context.Context, pageURL, domain string) ([]string, error) {
	tabCtx, cancel := chromedp.NewContext(e.allocCtx)
	defer cancel()
	tabCtx, tCancel := context.WithTimeout(tabCtx, 25*time.Second)
	defer tCancel()

	var rawLinks []string
	err := chromedp.Run(tabCtx,
		chromedp.Navigate(pageURL),
		chromedp.WaitReady("body"),
		chromedp.Evaluate(`
			Array.from(new Set(
				Array.from(document.querySelectorAll('a[href]'))
					.map(a => a.href)
					.filter(h => h.startsWith('http'))
			))
		`, &rawLinks),
	)
	if err != nil {
		return nil, err
	}

	var internal []string
	for _, link := range rawLinks {
		u, err := url.Parse(link)
		if err != nil {
			continue
		}
		// Same domain only, drop fragments and query strings
		if strings.TrimPrefix(u.Hostname(), "www.") == strings.TrimPrefix(domain, "www.") {
			u.RawQuery = ""
			u.Fragment = ""
			clean := u.String()
			// Skip common non-content paths
			ext := strings.ToLower(u.Path)
			if strings.HasSuffix(ext, ".pdf") || strings.HasSuffix(ext, ".jpg") ||
				strings.HasSuffix(ext, ".png") || strings.HasSuffix(ext, ".css") ||
				strings.HasSuffix(ext, ".js") || strings.HasSuffix(ext, ".xml") {
				continue
			}
			internal = append(internal, clean)
		}
	}
	return internal, nil
}

// CrawlPage runs all SEO checks on a single URL and returns issues.
func (e *Engine) CrawlPage(ctx context.Context, auditID, pageURL string) (*PageResult, error) {
	start := time.Now()

	tabCtx, cancel := chromedp.NewContext(e.allocCtx)
	defer cancel()
	tabCtx, tCancel := context.WithTimeout(tabCtx, 35*time.Second)
	defer tCancel()

	var sig pageSignals
	sig.hasHTTPS = strings.HasPrefix(pageURL, "https://")

	var navStart time.Time
	err := chromedp.Run(tabCtx,
		chromedp.ActionFunc(func(_ context.Context) error { navStart = time.Now(); return nil }),
		chromedp.Navigate(pageURL),
		chromedp.WaitReady("body"),
		chromedp.ActionFunc(func(_ context.Context) error {
			sig.pageLoadMs = time.Since(navStart).Milliseconds()
			return nil
		}),
		chromedp.Title(&sig.title),
		chromedp.AttributeValue(`meta[name="description"]`, "content", &sig.metaDesc, nil),
		chromedp.AttributeValue(`link[rel="canonical"]`, "href", &sig.canonical, nil),
		chromedp.AttributeValue(`meta[name="robots"]`, "content", &sig.robotsMeta, nil),
		chromedp.AttributeValue(`html`, "lang", &sig.langAttr, nil),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('h1')).map(el => el.innerText.trim())`, &sig.h1s),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('h2')).map(el => el.innerText.trim().slice(0,200))`, &sig.h2s),
		chromedp.Evaluate(`!!document.querySelector('meta[name="viewport"]')`, &sig.hasViewport),
		chromedp.Evaluate(`!!document.querySelector('meta[property="og:title"]')`, &sig.hasOGTitle),
		chromedp.Evaluate(`!!document.querySelector('meta[property="og:description"]')`, &sig.hasOGDesc),
		chromedp.Evaluate(`!!document.querySelector('meta[property="og:image"]')`, &sig.hasOGImage),
		chromedp.Evaluate(`!!document.querySelector('meta[name="twitter:card"]')`, &sig.hasTwitter),
		chromedp.Evaluate(`!!document.querySelector('script[type="application/ld+json"]')`, &sig.hasSchema),
		chromedp.Evaluate(`document.querySelectorAll('img:not([alt])').length`, &sig.imgsMissing),
		chromedp.Evaluate(`document.querySelectorAll('img').length`, &sig.imgsTotal),
		chromedp.Evaluate(`document.body.innerText.slice(0, 8000)`, &sig.bodyText),
	)
	if err != nil {
		return nil, fmt.Errorf("chromedp navigate %s: %w", pageURL, err)
	}

	sig.titleLen = len([]rune(strings.TrimSpace(sig.title)))
	sig.metaDescLen = len([]rune(strings.TrimSpace(sig.metaDesc)))
	sig.wordCount = len(strings.Fields(sig.bodyText))
	if len(sig.h1s) > 0 {
		sig.h1Text = sig.h1s[0]
	}

	result := &PageResult{
		URL:     pageURL,
		CrawlMs: time.Since(start).Milliseconds(),
	}

	// Run all checks
	checks := []func() *models.AuditIssue{
		// ── Title checks ───────────────────────────────────────────────────
		func() *models.AuditIssue { return checkTitle(auditID, pageURL, sig.title, sig.titleLen) },
		func() *models.AuditIssue { return checkTitleKeyword(auditID, pageURL, sig.title) },
		// ── Meta description ───────────────────────────────────────────────
		func() *models.AuditIssue { return checkMetaDesc(auditID, pageURL, sig.metaDesc, sig.metaDescLen) },
		// ── Headings ───────────────────────────────────────────────────────
		func() *models.AuditIssue { return checkH1(auditID, pageURL, sig.h1s) },
		func() *models.AuditIssue { return checkH1Length(auditID, pageURL, sig.h1Text) },
		func() *models.AuditIssue { return checkHeadingStructure(auditID, pageURL, sig.h2s) },
		// ── Mobile / UX ────────────────────────────────────────────────────
		func() *models.AuditIssue { return checkViewport(auditID, pageURL, sig.hasViewport) },
		func() *models.AuditIssue { return checkPageSpeed(auditID, pageURL, sig.pageLoadMs) },
		// ── Internationalisation ────────────────────────────────────────────
		func() *models.AuditIssue { return checkLang(auditID, pageURL, sig.langAttr) },
		// ── Canonical / indexing ───────────────────────────────────────────
		func() *models.AuditIssue { return checkCanonical(auditID, pageURL, sig.canonical) },
		func() *models.AuditIssue { return checkRobotsMeta(auditID, pageURL, sig.robotsMeta) },
		// ── Content quality ────────────────────────────────────────────────
		func() *models.AuditIssue { return checkContentLength(auditID, pageURL, sig.wordCount) },
		func() *models.AuditIssue { return checkDuplicateContent(auditID, pageURL, sig.title, sig.h1Text) },
		// ── Images ─────────────────────────────────────────────────────────
		func() *models.AuditIssue {
			return checkImageAlt(auditID, pageURL, sig.imgsMissing, sig.imgsTotal)
		},
		// ── Social / structured data ───────────────────────────────────────
		func() *models.AuditIssue { return checkOGTags(auditID, pageURL, sig.hasOGTitle, sig.hasOGDesc, sig.hasOGImage) },
		func() *models.AuditIssue { return checkTwitterCard(auditID, pageURL, sig.hasTwitter) },
		func() *models.AuditIssue { return checkSchema(auditID, pageURL, sig.hasSchema) },
		// ── Security ───────────────────────────────────────────────────────
		func() *models.AuditIssue { return checkHTTPS(auditID, pageURL, sig.hasHTTPS) },
	}

	deductions := 0
	for _, check := range checks {
		if iss := check(); iss != nil {
			result.Issues = append(result.Issues, *iss)
			switch iss.Severity {
			case models.SeverityCritical:
				deductions += 20
			case models.SeverityWarning:
				deductions += 8
			case models.SeverityInfo:
				deductions += 2
			}
		}
	}

	score := 100 - deductions
	if score < 0 {
		score = 0
	}
	result.Score = score

	slog.Info("page crawled",
		"url", pageURL,
		"issues", len(result.Issues),
		"score", result.Score,
		"ms", result.CrawlMs,
	)
	return result, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Check functions
// ────────────────────────────────────────────────────────────────────────────

func checkTitle(auditID, pageURL, title string, titleLen int) *models.AuditIssue {
	title = strings.TrimSpace(title)
	if title == "" {
		return issue(auditID, pageURL, "missing_title", models.SeverityCritical,
			"Missing page title",
			"The page has no <title> tag.",
			"Add a descriptive title tag between 50–60 characters.")
	}
	if titleLen < 10 {
		return issue(auditID, pageURL, "title_too_short", models.SeverityWarning,
			"Title too short",
			fmt.Sprintf("Title is only %d characters: %q", titleLen, title),
			"Expand the title to 50–60 characters with the primary keyword.")
	}
	if titleLen > 70 {
		return issue(auditID, pageURL, "title_too_long", models.SeverityWarning,
			"Title too long",
			fmt.Sprintf("Title is %d characters — Google truncates at ~60.", titleLen),
			"Shorten the title to under 60 characters.")
	}
	return nil
}

func checkTitleKeyword(auditID, pageURL, title string) *models.AuditIssue {
	// Heuristic: title should not be all caps or all lowercase (formatting issue)
	t := strings.TrimSpace(title)
	if t == "" {
		return nil
	}
	if t == strings.ToUpper(t) && len(t) > 5 {
		return issue(auditID, pageURL, "title_all_caps", models.SeverityInfo,
			"Title is all caps",
			"Title tags written in ALL CAPS can appear spammy in search results.",
			"Use standard title case or sentence case for the title tag.")
	}
	return nil
}

func checkMetaDesc(auditID, pageURL, desc string, descLen int) *models.AuditIssue {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return issue(auditID, pageURL, "missing_meta_desc", models.SeverityWarning,
			"Missing meta description",
			"No meta description tag found.",
			"Add a meta description of 120–160 characters summarising the page.")
	}
	if descLen < 50 {
		return issue(auditID, pageURL, "meta_desc_too_short", models.SeverityInfo,
			"Meta description too short",
			fmt.Sprintf("Description is only %d characters.", descLen),
			"Expand the meta description to 120–160 characters.")
	}
	if descLen > 165 {
		return issue(auditID, pageURL, "meta_desc_too_long", models.SeverityInfo,
			"Meta description too long",
			fmt.Sprintf("Description is %d characters.", descLen),
			"Keep meta descriptions under 160 characters to avoid SERP truncation.")
	}
	return nil
}

func checkH1(auditID, pageURL string, h1s []string) *models.AuditIssue {
	if len(h1s) == 0 {
		return issue(auditID, pageURL, "missing_h1", models.SeverityCritical,
			"Missing H1 heading",
			"No <h1> tag found on this page.",
			"Add one H1 tag containing the primary keyword for this page.")
	}
	if len(h1s) > 1 {
		return issue(auditID, pageURL, "multiple_h1", models.SeverityWarning,
			"Multiple H1 headings",
			fmt.Sprintf("Found %d H1 tags — pages should have exactly one.", len(h1s)),
			"Keep a single H1 and use H2–H6 for sub-headings.")
	}
	return nil
}

func checkH1Length(auditID, pageURL, h1Text string) *models.AuditIssue {
	if h1Text == "" {
		return nil
	}
	words := len(strings.Fields(h1Text))
	if words > 10 {
		return issue(auditID, pageURL, "h1_too_long", models.SeverityInfo,
			"H1 heading is too long",
			fmt.Sprintf("H1 has %d words: %q", words, h1Text),
			"Keep H1 headings concise — ideally under 8 words with the main keyword.")
	}
	return nil
}

func checkHeadingStructure(auditID, pageURL string, h2s []string) *models.AuditIssue {
	if len(h2s) == 0 {
		return issue(auditID, pageURL, "no_h2_headings", models.SeverityInfo,
			"No H2 subheadings",
			"Page has no H2 headings to break up content.",
			"Add H2 subheadings to improve readability and keyword coverage.")
	}
	return nil
}

func checkViewport(auditID, pageURL string, hasViewport bool) *models.AuditIssue {
	if !hasViewport {
		return issue(auditID, pageURL, "missing_viewport", models.SeverityCritical,
			"Missing viewport meta tag",
			"No <meta name=\"viewport\"> — page is not mobile-friendly.",
			"Add: <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">")
	}
	return nil
}

func checkPageSpeed(auditID, pageURL string, loadMs int64) *models.AuditIssue {
	if loadMs > 5000 {
		return issue(auditID, pageURL, "slow_page_load", models.SeverityCritical,
			"Very slow page load",
			fmt.Sprintf("Page took %dms to load in headless Chrome.", loadMs),
			"Optimise server response time, eliminate render-blocking resources, and enable compression.")
	}
	if loadMs > 3000 {
		return issue(auditID, pageURL, "slow_page_load", models.SeverityWarning,
			"Slow page load",
			fmt.Sprintf("Page took %dms to load. Google recommends under 2.5s (LCP).", loadMs),
			"Reduce JavaScript bundle size, lazy-load images, and use a CDN.")
	}
	return nil
}

func checkLang(auditID, pageURL, lang string) *models.AuditIssue {
	if strings.TrimSpace(lang) == "" {
		return issue(auditID, pageURL, "missing_lang", models.SeverityWarning,
			"Missing lang attribute on <html>",
			"The <html> element has no lang attribute.",
			"Add lang=\"en\" (or appropriate language code) to the <html> tag.")
	}
	return nil
}

func checkCanonical(auditID, pageURL, canonical string) *models.AuditIssue {
	if canonical == "" {
		return issue(auditID, pageURL, "missing_canonical", models.SeverityInfo,
			"Missing canonical link",
			"No <link rel=\"canonical\"> tag found.",
			"Add a canonical URL to prevent duplicate content issues.")
	}
	u, err := url.Parse(canonical)
	if err != nil || !u.IsAbs() {
		return issue(auditID, pageURL, "invalid_canonical", models.SeverityWarning,
			"Canonical URL is relative or invalid",
			fmt.Sprintf("canonical=%q should be an absolute URL.", canonical),
			"Use a full absolute URL for the canonical tag.")
	}
	return nil
}

func checkRobotsMeta(auditID, pageURL, robotsMeta string) *models.AuditIssue {
	lower := strings.ToLower(robotsMeta)
	if strings.Contains(lower, "noindex") {
		return issue(auditID, pageURL, "noindex_page", models.SeverityCritical,
			"Page is blocked from indexing",
			fmt.Sprintf("meta robots contains 'noindex': %q", robotsMeta),
			"Remove 'noindex' from the robots meta tag unless you intentionally want this page excluded from search.")
	}
	return nil
}

func checkContentLength(auditID, pageURL string, wordCount int) *models.AuditIssue {
	if wordCount < 100 {
		return issue(auditID, pageURL, "thin_content", models.SeverityWarning,
			"Thin content",
			fmt.Sprintf("Page has only ~%d words of visible text.", wordCount),
			"Expand the page content to at least 300 words for better topical coverage.")
	}
	return nil
}

func checkDuplicateContent(auditID, pageURL, title, h1 string) *models.AuditIssue {
	t := strings.ToLower(strings.TrimSpace(title))
	h := strings.ToLower(strings.TrimSpace(h1))
	if t != "" && h != "" && t == h {
		return issue(auditID, pageURL, "title_h1_duplicate", models.SeverityInfo,
			"Title and H1 are identical",
			"Page title and H1 contain exactly the same text.",
			"Differentiate the title (for SERPs) and H1 (for users) to target keyword variations.")
	}
	return nil
}

func checkImageAlt(auditID, pageURL string, missing, total int) *models.AuditIssue {
	if total == 0 || missing == 0 {
		return nil
	}
	pct := (missing * 100) / total
	if pct >= 50 {
		return issue(auditID, pageURL, "images_missing_alt", models.SeverityWarning,
			"Images missing alt text",
			fmt.Sprintf("%d of %d images are missing alt attributes (%d%%).", missing, total, pct),
			"Add descriptive alt text to all images for accessibility and image SEO.")
	}
	if missing > 0 {
		return issue(auditID, pageURL, "images_missing_alt", models.SeverityInfo,
			"Some images missing alt text",
			fmt.Sprintf("%d of %d images are missing alt attributes.", missing, total),
			"Add descriptive alt text to all content images.")
	}
	return nil
}

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
			"Missing Open Graph tags",
			"No Open Graph meta tags found (og:title, og:description, og:image).",
			"Add Open Graph tags so your page looks great when shared on social media.")
	}
	if len(missing) > 0 {
		return issue(auditID, pageURL, "incomplete_og_tags", models.SeverityInfo,
			"Incomplete Open Graph tags",
			fmt.Sprintf("Missing: %s", strings.Join(missing, ", ")),
			"Add all three Open Graph tags (og:title, og:description, og:image) for optimal social sharing.")
	}
	return nil
}

func checkTwitterCard(auditID, pageURL string, hasTwitter bool) *models.AuditIssue {
	if !hasTwitter {
		return issue(auditID, pageURL, "missing_twitter_card", models.SeverityInfo,
			"Missing Twitter/X Card meta tag",
			"No <meta name=\"twitter:card\"> found.",
			"Add twitter:card, twitter:title, and twitter:description tags for better Twitter/X sharing.")
	}
	return nil
}

func checkSchema(auditID, pageURL string, hasSchema bool) *models.AuditIssue {
	if !hasSchema {
		return issue(auditID, pageURL, "no_structured_data", models.SeverityInfo,
			"No structured data (JSON-LD)",
			"No <script type=\"application/ld+json\"> found.",
			"Add Schema.org structured data to enable rich results in Google Search.")
	}
	return nil
}

func checkHTTPS(auditID, pageURL string, hasHTTPS bool) *models.AuditIssue {
	if !hasHTTPS {
		return issue(auditID, pageURL, "no_https", models.SeverityCritical,
			"Page not served over HTTPS",
			"The page URL uses HTTP, not HTTPS.",
			"Install an SSL certificate and redirect all HTTP traffic to HTTPS.")
	}
	return nil
}

// issue is a helper to create an AuditIssue.
func issue(auditID, pageURL, checkType string, severity models.AuditSeverity, title, desc, suggestion string) *models.AuditIssue {
	return &models.AuditIssue{
		AuditID:     auditID,
		URL:         pageURL,
		CheckType:   checkType,
		Severity:    severity,
		Title:       title,
		Description: desc,
		Suggestion:  suggestion,
	}
}


// New creates a chromedp browser allocator.
// headless=true, no-sandbox for container environments.
