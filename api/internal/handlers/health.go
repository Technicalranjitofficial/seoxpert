package handlers

import (
	"context"
	"runtime"
	"time"

	"github.com/gofiber/fiber/v3"
)

// Health returns a lightweight status response used by load balancers and monitoring.
func (h *Handler) Health(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	dbOK := "ok"
	if err := h.pool.Ping(ctx); err != nil {
		dbOK = "error: " + err.Error()
	}

	redisOK := "ok"
	if _, err := h.rdb.Get(ctx, "__health__"); err != nil {
		redisOK = "error: " + err.Error()
	}

	return c.JSON(fiber.Map{
		"status":  "ok",
		"service": "seoxpert-api",
		"time":    time.Now().UTC(),
		"go":      runtime.Version(),
		"deps": fiber.Map{
			"postgres": dbOK,
			"redis":    redisOK,
		},
	})
}
