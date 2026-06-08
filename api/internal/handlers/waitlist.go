package handlers

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
)

type waitlistRequest struct {
	Email  string `json:"email"`
	Source string `json:"source"` // optional: "landing_page", "referral", etc.
}

// AddToWaitlist validates an email and persists it to the waitlist table.
// Public endpoint — protected only by IP rate limit (5/min).
func (h *Handler) AddToWaitlist(c fiber.Ctx) error {
	var req waitlistRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !isValidEmail(email) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "invalid email address",
		})
	}

	source := req.Source
	if source == "" {
		source = "landing_page"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// INSERT ... ON CONFLICT DO NOTHING — idempotent, no double-counting.
	_, err := h.pool.Exec(ctx, `
		INSERT INTO waitlist (email, source)
		VALUES ($1, $2)
		ON CONFLICT (email) DO NOTHING
	`, email, source)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to save email",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "you're on the list",
	})
}

// isValidEmail performs a minimal structural check — no regex complexity.
func isValidEmail(email string) bool {
	at := strings.LastIndex(email, "@")
	if at < 1 || at >= len(email)-2 {
		return false
	}
	dot := strings.LastIndex(email[at:], ".")
	return dot > 1 && dot < len(email[at:])-1 && len(email) <= 254
}
