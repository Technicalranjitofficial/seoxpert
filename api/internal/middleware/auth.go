package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/seoxpert/api/internal/auth"
)

// RequireAuth validates the Supabase JWT from the Authorization header.
// On success it sets "user_id", "user_email", "user_role" in locals.
func RequireAuth(jwtSecret string) fiber.Handler {
	return func(c fiber.Ctx) error {
		header := c.Get("Authorization")
		if header == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing authorization header",
			})
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid authorization format, expected: Bearer <token>",
			})
		}

		claims, err := auth.ParseToken(parts[1], jwtSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid or expired token",
			})
		}

		// Expose user context to downstream handlers.
		c.Locals("user_id", claims.Subject)
		c.Locals("user_email", claims.Email)
		c.Locals("user_role", claims.Role)

		return c.Next()
	}
}
