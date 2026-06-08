package models

import "time"

type Plan string

const (
	PlanFree   Plan = "free"
	PlanPro    Plan = "pro"
	PlanAgency Plan = "agency"
)

// PlanLimits defines per-plan quotas enforced at API layer via Redis.
type PlanLimits struct {
	MaxProjects     int
	MaxKeywords     int
	MaxAuditsPerDay int
	MaxPagesPerAudit int
	RankCheckFreq   string // "daily" | "weekly"
}

var Limits = map[Plan]PlanLimits{
	PlanFree: {
		MaxProjects:      1,
		MaxKeywords:      10,
		MaxAuditsPerDay:  1,
		MaxPagesPerAudit: 50,
		RankCheckFreq:    "weekly",
	},
	PlanPro: {
		MaxProjects:      10,
		MaxKeywords:      500,
		MaxAuditsPerDay:  10,
		MaxPagesPerAudit: 500,
		RankCheckFreq:    "daily",
	},
	PlanAgency: {
		MaxProjects:      100,
		MaxKeywords:      5000,
		MaxAuditsPerDay:  100,
		MaxPagesPerAudit: 2000,
		RankCheckFreq:    "daily",
	},
}

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Plan      Plan      `json:"plan"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
