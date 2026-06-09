package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/seoxpert/shared/config"
	"github.com/seoxpert/shared/events"
	"github.com/seoxpert/shared/models"
	"github.com/seoxpert/workers/crawler/internal/consumer"
	"github.com/seoxpert/workers/crawler/internal/engine"
	"github.com/seoxpert/workers/crawler/internal/writer"
)

const maxPages = 50 // max pages to crawl per audit

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg := config.Load()
	slog.Info("crawler worker starting")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	eng, err := engine.New(ctx)
	if err != nil {
		slog.Error("engine init failed", "err", err)
		os.Exit(1)
	}

	w := writer.New(pool)

	handleAudit := func(ctx context.Context, job events.AuditJob) error {
		slog.Info("audit started", "audit_id", job.AuditID, "domain", job.Domain)

		// ── Multi-page BFS crawl ──────────────────────────────────────
		startURL := engine.NormaliseURL("https://" + job.Domain)
		visited := map[string]bool{startURL: true}
		queue := []string{startURL}
		crawled := 0

		for len(queue) > 0 && crawled < maxPages {
			// Check for context cancellation between pages
			select {
			case <-ctx.Done():
				return w.FailAudit(ctx, job.AuditID, "worker shutting down")
			default:
			}

			pageURL := queue[0]
			queue = queue[1:]

			result, err := eng.CrawlPage(ctx, job.AuditID, pageURL)
			if err != nil {
				slog.Warn("page crawl failed", "url", pageURL, "err", err)
				continue
			}

			crawled++
			if err := w.UpdateAuditProgress(ctx, job.AuditID, crawled, crawled+len(queue)); err != nil {
				slog.Error("update progress", "err", err)
			}
			if err := w.SavePageResult(ctx, result); err != nil {
				slog.Error("save page result", "url", pageURL, "err", err)
			}

			// Discover links from this page and add unvisited ones to queue
			if crawled < maxPages {
				links, err := eng.DiscoverLinks(ctx, pageURL, job.Domain)
				if err != nil {
					slog.Warn("link discovery failed", "url", pageURL, "err", err)
				}
				for _, link := range links {
					norm := engine.NormaliseURL(link)
					if !visited[norm] && len(visited) < maxPages {
						visited[norm] = true
						queue = append(queue, norm)
					}
				}
			}
		}

		slog.Info("audit complete", "audit_id", job.AuditID, "pages", crawled)
			// ── Post-crawl: robots.txt check ────────────────────────────────
			robotsIssues := checkRobotsTxt(ctx, job.AuditID, "https://"+job.Domain)
			for _, ri := range robotsIssues {
				if err := w.SaveSiteIssue(ctx, &ri); err != nil {
					slog.Warn("save robots issue", "err", err)
				}
			}

			// ── Post-crawl: broken internal links ───────────────────────────
			brokenIssues := checkBrokenLinks(ctx, job.AuditID, "https://"+job.Domain, visited)
			for _, bi := range brokenIssues {
				if err := w.SaveSiteIssue(ctx, &bi); err != nil {
					slog.Warn("save broken link issue", "err", err)
				}
			}

		// ── Post-crawl: duplicate title/meta detection ───────────────────────
		if err := w.SaveDuplicateIssues(ctx, job.AuditID); err != nil {
			slog.Warn("duplicate issue detection", "err", err)
		}
		return w.CompleteAudit(ctx, job.AuditID)
	}

	c, err := consumer.New(cfg.RedpandaBrokers, "crawler-workers", handleAudit)
	if err != nil {
		slog.Error("consumer init failed", "err", err)
		os.Exit(1)
	}

	slog.Info("crawler worker ready, consuming from", "topic", events.TopicAuditRequested)
	c.Run(ctx)
	slog.Info("crawler worker stopped")
}

// ── robots.txt check ─────────────────────────────────────────────────────────

func checkRobotsTxt(ctx context.Context, auditID, baseURL string) []models.AuditIssue {
	robotsURL := strings.TrimRight(baseURL, "/") + "/robots.txt"
	client := &http.Client{Timeout: 8 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, robotsURL, nil)
	resp, err := client.Do(req)
	if err != nil {
		return []models.AuditIssue{{AuditID: auditID, URL: baseURL, CheckType: "robotstxt_missing", Severity: models.SeverityWarning, Title: "robots.txt file not found", Description: "No /robots.txt file found. Robots.txt tells search engine crawlers which pages they may or may not request.", Suggestion: "Create a /robots.txt file at the root of your domain.", Value: robotsURL + " → error"}}
	}
	if resp.StatusCode == 404 {
		return []models.AuditIssue{{
			AuditID:     auditID,
			URL:         baseURL,
			CheckType:   "robotstxt_missing",
			Severity:    models.SeverityWarning,
			Title:       "robots.txt file not found",
			Description: "No /robots.txt file found. Robots.txt tells search engine crawlers which pages they may or may not request. Without it, crawlers may waste crawl budget on unimportant pages.",
			Suggestion:  "Create a /robots.txt file at the root of your domain. At minimum, reference your XML sitemap: Sitemap: https://yourdomain.com/sitemap.xml",
			Value:       robotsURL + " → 404",
		}}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	content := strings.ToLower(string(body))

	var issues []models.AuditIssue

	// Check if robots.txt blocks everything
	if strings.Contains(content, "disallow: /") && strings.Contains(content, "user-agent: *") {
		// Ensure it's not "disallow: /something" - check for bare "disallow: /"
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if line == "disallow: /" {
				issues = append(issues, models.AuditIssue{
					AuditID:     auditID,
					URL:         baseURL,
					CheckType:   "robotstxt_blocks_all",
					Severity:    models.SeverityCritical,
					Title:       "robots.txt is blocking all search engine crawlers",
					Description: "Your robots.txt contains 'Disallow: /' for all user agents, which prevents ALL search engines from crawling any page on your site. This will cause your entire site to disappear from search results.",
					Suggestion:  "Change 'Disallow: /' to 'Allow: /' or remove the Disallow line. Only block directories you specifically don't want indexed (e.g. /admin/, /api/).",
					Value:       "User-agent: * / Disallow: /",
				})
				break
			}
		}
	}

	// Check if sitemap is referenced
	if !strings.Contains(content, "sitemap:") {
		issues = append(issues, models.AuditIssue{
			AuditID:     auditID,
			URL:         baseURL,
			CheckType:   "robotstxt_no_sitemap",
			Severity:    models.SeverityInfo,
			Title:       "robots.txt does not reference XML sitemap",
			Description: "Your robots.txt file exists but doesn't include a Sitemap directive. Adding your sitemap URL helps search engines discover all your pages faster.",
			Suggestion:  fmt.Sprintf("Add this line to /robots.txt: Sitemap: %s/sitemap.xml", strings.TrimRight(baseURL, "/")),
			Value:       robotsURL + " → sitemap directive missing",
		})
	}

	return issues
}

// ── broken internal links check ──────────────────────────────────────────────

func checkBrokenLinks(ctx context.Context, auditID, baseURL string, visited map[string]bool) []models.AuditIssue {
	client := &http.Client{
		Timeout: 6 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	var issues []models.AuditIssue
	checked := 0
	maxCheck := 30 // limit to avoid long audit times

	for pageURL := range visited {
		if checked >= maxCheck {
			break
		}
		if !strings.HasPrefix(pageURL, "http") {
			continue
		}
		checked++

		req, err := http.NewRequestWithContext(ctx, http.MethodHead, pageURL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "SeoXpert-Checker/1.0")

		resp, err := client.Do(req)
		if err != nil {
			// Connection error or timeout — flag as potentially broken
			if strings.Contains(err.Error(), "too many redirects") {
				issues = append(issues, models.AuditIssue{
					AuditID:     auditID,
					URL:         pageURL,
					CheckType:   "redirect_chain",
					Severity:    models.SeverityWarning,
					Title:       "Redirect chain detected",
					Description: "This URL goes through 5 or more redirects. Long redirect chains slow page load, waste crawl budget, and dilute PageRank passed through each redirect.",
					Suggestion:  "Update links to point directly to the final destination URL. Fix the server-side redirect chain to use a single direct 301 redirect.",
					Value:       pageURL + " → chain of 5+ redirects",
				})
			}
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == 404 || resp.StatusCode == 410 {
			issues = append(issues, models.AuditIssue{
				AuditID:     auditID,
				URL:         pageURL,
				CheckType:   "broken_internal_link",
				Severity:    models.SeverityCritical,
				Title:       fmt.Sprintf("Broken internal link (%d)", resp.StatusCode),
				Description: fmt.Sprintf("This internal URL returns HTTP %d. Broken links waste crawl budget, create poor user experience, and tell Google your site is poorly maintained.", resp.StatusCode),
				Suggestion:  "Either fix the URL (update the destination), redirect it with a 301 to the correct page, or remove the link pointing to it.",
				Value:       fmt.Sprintf("%s → HTTP %d", pageURL, resp.StatusCode),
			})
		}
	}

	return issues
}
