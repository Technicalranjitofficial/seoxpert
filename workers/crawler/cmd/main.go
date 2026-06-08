package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/seoxpert/shared/config"
	"github.com/seoxpert/shared/events"
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

		// ── Multi-page BFS crawl ──────────────────────────────────────────
		startURL := "https://" + job.Domain
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
					if !visited[link] && len(visited) < maxPages {
						visited[link] = true
						queue = append(queue, link)
					}
				}
			}
		}

		slog.Info("audit complete", "audit_id", job.AuditID, "pages", crawled)
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
