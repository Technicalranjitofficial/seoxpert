package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/seoxpert/shared/config"
	"github.com/seoxpert/shared/events"
	"github.com/seoxpert/workers/rank-tracker/internal/consumer"
	"github.com/seoxpert/workers/rank-tracker/internal/engine"
	"github.com/seoxpert/workers/rank-tracker/internal/writer"
)

// Throttle: Google rate-limits aggressively — wait between checks.
const delayBetweenChecks = 5 * time.Second

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg := config.Load()
	slog.Info("rank tracker worker starting")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	w := writer.New(pool)

	// Ensure rank_history table exists
	if err := w.EnsureRankHistoryTable(ctx); err != nil {
		slog.Error("ensure rank_history table", "err", err)
		os.Exit(1)
	}

	eng, err := engine.New(ctx)
	if err != nil {
		slog.Error("engine init failed", "err", err)
		os.Exit(1)
	}

	handleRank := func(ctx context.Context, job events.RankJob) error {
		for _, keyword := range job.Keywords {
			if keyword == "" {
				continue
			}
			slog.Info("checking rank", "keyword", keyword, "domain", job.Domain)

			result, err := eng.CheckRank(ctx, keyword, job.Domain)
			if err != nil {
				slog.Error("rank check failed", "keyword", keyword, "err", err)
				continue // don't fail the whole job for one keyword
			}

			slog.Info("rank result",
				"keyword", keyword,
				"domain", job.Domain,
				"position", result.Position,
				"url", result.URL,
			)

			if err := w.SaveRankResult(ctx, job.ProjectID, result); err != nil {
				slog.Error("save rank result", "err", err)
			}

			// Throttle to avoid triggering CAPTCHA
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delayBetweenChecks):
			}
		}
		return nil
	}

	c, err := consumer.New(
		cfg.RedpandaBrokers,
		cfg.RedpandaSASLUser,
		cfg.RedpandaSASLPassword,
		handleRank,
	)
	if err != nil {
		slog.Error("consumer init failed", "err", err)
		os.Exit(1)
	}

	slog.Info("rank tracker ready, consuming from", "topic", events.TopicRankRequested)
	c.Run(ctx)
	slog.Info("rank tracker stopped")
}
