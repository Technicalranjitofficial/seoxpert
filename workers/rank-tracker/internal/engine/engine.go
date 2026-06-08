package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// RankResult holds position data for a keyword on a search engine.
type RankResult struct {
	Keyword  string
	Domain   string
	Position int    // 1-based; 0 means not found in top 100
	URL      string // the ranking page URL
	Engine   string // "google" or "bing"
	CheckedAt time.Time
}

// Engine uses headless Chrome to scrape SERPs.
type Engine struct {
	allocCtx context.Context
}

// New creates a chromedp browser allocator.
func New(ctx context.Context) (*Engine, error) {
	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		// Rotate user-agent to reduce bot detection
		chromedp.UserAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"),
	)
	allocCtx, _ := chromedp.NewExecAllocator(ctx, opts...)
	return &Engine{allocCtx: allocCtx}, nil
}

// CheckRank checks the position of `domain` for `keyword` on Google.
// Returns position 0 if not found in top 100 results.
func (e *Engine) CheckRank(ctx context.Context, keyword, domain string) (*RankResult, error) {
	tabCtx, cancel := chromedp.NewContext(e.allocCtx)
	defer cancel()
	tabCtx, tCancel := context.WithTimeout(tabCtx, 30*time.Second)
	defer tCancel()

	searchURL := fmt.Sprintf(
		"https://www.google.com/search?q=%s&num=100&hl=en&gl=us",
		strings.ReplaceAll(keyword, " ", "+"),
	)

	// Extract all result links
	var resultLinks []string
	err := chromedp.Run(tabCtx,
		chromedp.Navigate(searchURL),
		chromedp.WaitReady("body"),
		// Accept consent / cookie prompt if present (EU)
		chromedp.Evaluate(`
			const btn = document.querySelector('button[jsname="b3VHJd"]') ||
				document.querySelector('[aria-label*="Accept"]') ||
				document.querySelector('[aria-label*="agree"]');
			if (btn) btn.click();
		`, nil),
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('a[href]'))
				.map(a => a.href)
				.filter(h => h.startsWith('http') && !h.includes('google.com'))
		`, &resultLinks),
	)
	if err != nil {
		return nil, fmt.Errorf("serp fetch failed for %q: %w", keyword, err)
	}

	// Find domain in results
	cleanDomain := strings.TrimPrefix(strings.TrimPrefix(domain, "https://"), "http://")
	cleanDomain = strings.TrimPrefix(cleanDomain, "www.")

	result := &RankResult{
		Keyword:   keyword,
		Domain:    domain,
		Position:  0,
		Engine:    "google",
		CheckedAt: time.Now().UTC(),
	}

	seen := 0
	for _, link := range resultLinks {
		// Skip Google's own URLs and tracking links
		if strings.Contains(link, "google.com") || strings.Contains(link, "googleusercontent") {
			continue
		}
		seen++
		linkDomain := strings.TrimPrefix(strings.Split(link, "/")[2], "www.")
		if strings.HasSuffix(linkDomain, cleanDomain) || linkDomain == cleanDomain {
			result.Position = seen
			result.URL = link
			break
		}
		if seen >= 100 {
			break
		}
	}

	return result, nil
}
