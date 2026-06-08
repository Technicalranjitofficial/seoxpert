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

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg := config.Load()
	slog.Info("crawler worker starting")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ── Database ───────────────────────────────────────────────────────────────
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	// ── SEO Engine (headless Chrome) ───────────────────────────────────────────
	eng, err := engine.New(ctx)
	if err != nil {
		slog.Error("engine init failed", "err", err)
		os.Exit(1)
	}

	// ── Writer (persists results to Supabase) ──────────────────────────────────
	w := writer.New(pool)

	// ── Audit Job Handler ──────────────────────────────────────────────────────
	handleAudit := func(ctx context.Context, job events.AuditJob) error {
		slog.Info("audit started", "audit_id", job.AuditID, "domain", job.Domain)

		// Phase 1: crawl homepage only.
		// Phase 2: full site crawl with link following.
		pageURL := "https://" + job.Domain
		result, err := eng.CrawlPage(ctx, job.AuditID, pageURL)
		if err != nil {
			return w.FailAudit(ctx, job.AuditID, err.Error())
		}

		if err := w.UpdateAuditProgress(ctx, job.AuditID, 1, 1); err != nil {
			slog.Error("update progress", "err", err)
		}

		if err := w.SavePageResult(ctx, result); err != nil {
			return w.FailAudit(ctx, job.AuditID, err.Error())
		}

		return w.CompleteAudit(ctx, job.AuditID)
	}

	// ── Consumer ───────────────────────────────────────────────────────────────
	c, err := consumer.New(cfg.RedpandaBrokers, "crawler-workers", handleAudit)
	if err != nil {
		slog.Error("consumer init failed", "err", err)
		os.Exit(1)
	}

	slog.Info("crawler worker ready, consuming from", "topic", events.TopicAuditRequested)
	c.Run(ctx)
	slog.Info("crawler worker stopped")
}
