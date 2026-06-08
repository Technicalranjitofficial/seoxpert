package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/seoxpert/api/internal/cache"
)

// IPRateLimit limits requests per IP using a Redis sliding counter.
// limit = max requests per window duration.
func IPRateLimit(rdb *cache.Client, limit int64, window time.Duration) fiber.Handler {
	return func(c fiber.Ctx) error {
		ip := c.IP()
		key := fmt.Sprintf("rl:ip:%s:%s", c.Path(), ip)

		count, err := rdb.Incr(context.Background(), key, window)
		if err != nil {
			// Redis failure — fail open (don't block legitimate users)
			return c.Next()
		}

		if count > limit {
			ttl, _ := rdb.TTL(context.Background(), key)
			c.Set("Retry-After", fmt.Sprintf("%.0f", ttl.Seconds()))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "rate limit exceeded",
				"retry_after": ttl.Seconds(),
			})
		}

		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", limit-count))
		return c.Next()
	}
}

// PlanRateLimit enforces per-user daily audit limits based on their plan.
// Must be used after RequireAuth (reads "user_id" from locals).
func PlanRateLimit(rdb *cache.Client) fiber.Handler {
	// Daily limits per plan
	limits := map[string]int64{
		"free":   1,
		"pro":    10,
		"agency": 100,
	}

	return func(c fiber.Ctx) error {
		userID, ok := c.Locals("user_id").(string)
		if !ok || userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}

		// Plan is stored in Redis after first auth (set by profile fetch).
		// Falls back to "free" if not cached yet.
		planKey := fmt.Sprintf("plan:%s", userID)
		plan, _ := rdb.Get(context.Background(), planKey)
		if plan == "" {
			plan = "free"
		}

		dailyLimit := limits[plan]
		if dailyLimit == 0 {
			dailyLimit = 1
		}

		// Reset at midnight UTC — use date as part of key.
		today := time.Now().UTC().Format("2006-01-02")
		counterKey := fmt.Sprintf("rl:audit:%s:%s", userID, today)

		count, err := rdb.Incr(context.Background(), counterKey, 24*time.Hour)
		if err != nil {
			return c.Next()
		}

		if count > dailyLimit {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": fmt.Sprintf("daily audit limit reached for %s plan (%d/day)", plan, dailyLimit),
				"plan":  plan,
			})
		}

		return c.Next()
	}
}
