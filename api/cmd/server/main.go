package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	fiberlogger "github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/seoxpert/api/internal/cache"
	"github.com/seoxpert/api/internal/db"
	"github.com/seoxpert/api/internal/handlers"
	apimw "github.com/seoxpert/api/internal/middleware"
	"github.com/seoxpert/api/internal/producer"
	"github.com/seoxpert/shared/config"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(log)

	cfg := config.Load()
	slog.Info("starting seoxpert api", "env", cfg.Env, "port", cfg.Port)

	// ── Connections ────────────────────────────────────────────────────────────

	pool, err := db.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("postgres connection failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("postgres connected")

	rdb, err := cache.New(cfg)
	if err != nil {
		slog.Error("redis connection failed", "err", err)
		os.Exit(1)
	}
	defer rdb.Close()
	slog.Info("redis connected")

	prod, err := producer.New(cfg.RedpandaBrokers)
	if err != nil {
		slog.Error("redpanda connection failed", "err", err)
		os.Exit(1)
	}
	defer prod.Close()
	slog.Info("redpanda producer connected")

	// ── Fiber ──────────────────────────────────────────────────────────────────

	app := fiber.New(fiber.Config{
		AppName:      "SeoXpert API",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
		BodyLimit:    1 * 1024 * 1024, // 1 MB
	})

	app.Use(recover.New())
	app.Use(fiberlogger.New(fiberlogger.Config{
		Format: "${time} | ${status} | ${latency} | ${ip} | ${method} ${path}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{cfg.AllowOrigins},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	}))

	// ── Routes ─────────────────────────────────────────────────────────────────

	h := handlers.New(pool, rdb, prod, cfg)

	app.Get("/health", h.Health)

	v1 := app.Group("/api/v1")

	// Public — IP rate limited
	v1.Post("/waitlist", apimw.IPRateLimit(rdb, 5, time.Minute), h.AddToWaitlist)

	// Protected — JWT required
	protected := v1.Group("", apimw.RequireAuth(cfg.SupabaseJWTSecret))
	protected.Get("/projects", h.ListProjects)
	protected.Post("/projects", h.CreateProject)
	protected.Get("/projects/:id", h.GetProject)
	protected.Delete("/projects/:id", h.DeleteProject)

	// Audits — JWT + plan rate limit
	protected.Post("/audits", apimw.PlanRateLimit(rdb), h.TriggerAudit)
	protected.Get("/audits/:id", h.GetAudit)

	// ── Graceful Shutdown ──────────────────────────────────────────────────────

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := app.Listen(":" + cfg.Port); err != nil {
			slog.Error("server error", "err", err)
		}
	}()

	slog.Info("api server ready", "port", cfg.Port)
	<-quit

	slog.Info("shutdown signal received")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
	slog.Info("api server stopped")
}
