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
// Each audit gets a chromedp context — tabs are reused across page crawls.
type Engine struct {
	allocCtx context.Context // shared browser allocator
	mu       sync.Mutex
}

// Result holds the crawl output for a single page.
type PageResult struct {
	URL      string
	Issues   []models.AuditIssue
	Score    int // 0–100 for this page
	CrawlMs  int64
}

// New creates a chromedp browser allocator.
// headless=true, no-sandbox for container environments.
func New(ctx context.Context) (*Engine, error) {
	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("blink-settings", "imagesEnabled=false"), // save RAM, we don't need images
		chromedp.WindowSize(1280, 720),
	)

	allocCtx, _ := chromedp.NewExecAllocator(ctx, opts...)

	return &Engine{allocCtx: allocCtx}, nil
}

// CrawlPage runs all SEO checks on a single URL and returns issues.
func (e *Engine) CrawlPage(ctx context.Context, auditID, pageURL string) (*PageResult, error) {
	start := time.Now()

	tabCtx, cancel := chromedp.NewContext(e.allocCtx)
	defer cancel()

	tabCtx, tabCancel := context.WithTimeout(tabCtx, 30*time.Second)
	defer tabCancel()

	// ── Extract page signals via JavaScript ───────────────────────────────────
	var (
		title       string
		metaDesc    string
		h1s         []string
		h2s         []string
		canonical   string
		robotsMeta  string
		hasViewport bool
		langAttr    string
		bodyText    string
		// ok flags — chromedp.AttributeValue requires them; we ignore the values
		_ bool
	)

	err := chromedp.Run(tabCtx,
		chromedp.Navigate(pageURL),
		chromedp.WaitReady("body"),
		chromedp.Title(&title),
		chromedp.AttributeValue(`meta[name="description"]`, "content", &metaDesc, nil),
		chromedp.AttributeValue(`link[rel="canonical"]`, "href", &canonical, nil),
		chromedp.AttributeValue(`meta[name="robots"]`, "content", &robotsMeta, nil),
		chromedp.AttributeValue(`html`, "lang", &langAttr, nil),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('h1')).map(el => el.innerText.trim())`, &h1s),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('h2')).map(el => el.innerText.trim())`, &h2s),
		chromedp.Evaluate(`!!document.querySelector('meta[name="viewport"]')`, &hasViewport),
		chromedp.Evaluate(`document.body.innerText.slice(0, 5000)`, &bodyText),
	)
	if err != nil {
		return nil, fmt.Errorf("chromedp navigate %s: %w", pageURL, err)
	}

	// _ is robotsMeta — reserved for future robots-check rule
	_ = robotsMeta

	crawlMs := time.Since(start).Milliseconds()

	// ── Run all checks ────────────────────────────────────────────────────────
	result := &PageResult{
		URL:     pageURL,
		CrawlMs: crawlMs,
	}

	checks := []func() *models.AuditIssue{
		func() *models.AuditIssue { return checkTitle(auditID, pageURL, title) },
		func() *models.AuditIssue { return checkMetaDesc(auditID, pageURL, metaDesc) },
		func() *models.AuditIssue { return checkH1(auditID, pageURL, h1s) },
		func() *models.AuditIssue { return checkViewport(auditID, pageURL, hasViewport) },
		func() *models.AuditIssue { return checkLang(auditID, pageURL, langAttr) },
		func() *models.AuditIssue { return checkCanonical(auditID, pageURL, canonical) },
		func() *models.AuditIssue { return checkContentLength(auditID, pageURL, bodyText) },
	}

	deductions := 0
	for _, check := range checks {
		if issue := check(); issue != nil {
			result.Issues = append(result.Issues, *issue)
			switch issue.Severity {
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
		"ms", crawlMs,
	)

	return result, nil
}

// ── Individual check functions ─────────────────────────────────────────────

func checkTitle(auditID, pageURL, title string) *models.AuditIssue {
	title = strings.TrimSpace(title)
	if title == "" {
		return issue(auditID, pageURL, "missing_title", models.SeverityCritical,
			"Missing page title",
			"The page has no <title> tag.",
			"Add a descriptive title tag between 50–60 characters.")
	}
	if len(title) < 10 {
		return issue(auditID, pageURL, "title_too_short", models.SeverityWarning,
			"Title too short",
			fmt.Sprintf("Title is only %d characters: %q", len(title), title),
			"Expand the title to 50–60 characters with the primary keyword.")
	}
	if len(title) > 70 {
		return issue(auditID, pageURL, "title_too_long", models.SeverityWarning,
			"Title too long",
			fmt.Sprintf("Title is %d characters — Google truncates at ~60.", len(title)),
			"Shorten the title to under 60 characters.")
	}
	return nil
}

func checkMetaDesc(auditID, pageURL, desc string) *models.AuditIssue {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return issue(auditID, pageURL, "missing_meta_desc", models.SeverityWarning,
			"Missing meta description",
			"No meta description tag found.",
			"Add a meta description of 120–160 characters summarising the page.")
	}
	if len(desc) > 165 {
		return issue(auditID, pageURL, "meta_desc_too_long", models.SeverityInfo,
			"Meta description too long",
			fmt.Sprintf("Description is %d characters.", len(desc)),
			"Keep meta descriptions under 160 characters to prevent truncation in SERPs.")
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

func checkViewport(auditID, pageURL string, hasViewport bool) *models.AuditIssue {
	if !hasViewport {
		return issue(auditID, pageURL, "missing_viewport", models.SeverityCritical,
			"Missing viewport meta tag",
			"No <meta name=\"viewport\"> found — page is not mobile-friendly.",
			"Add: <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">")
	}
	return nil
}

func checkLang(auditID, pageURL, lang string) *models.AuditIssue {
	if strings.TrimSpace(lang) == "" {
		return issue(auditID, pageURL, "missing_lang", models.SeverityWarning,
			"Missing lang attribute",
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

func checkContentLength(auditID, pageURL, bodyText string) *models.AuditIssue {
	words := len(strings.Fields(bodyText))
	if words < 100 {
		return issue(auditID, pageURL, "thin_content", models.SeverityWarning,
			"Thin content",
			fmt.Sprintf("Page has only ~%d words of visible text.", words),
			"Expand the page content to at least 300 words for better topical coverage.")
	}
	return nil
}

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
