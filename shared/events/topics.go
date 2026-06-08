package events

import "time"

// Redpanda topic names — one constant per topic, never inline strings.
const (
	TopicAuditRequested  = "audit.requested"
	TopicAuditProgress   = "audit.progress"
	TopicAuditCompleted  = "audit.completed"
	TopicRankRequested   = "rank.requested"
	TopicRankResults     = "rank.results"
	TopicKeywordJob      = "keyword.job"
	TopicReportRequested = "report.requested"
)

// AuditJob is the message published when a user triggers a site audit.
type AuditJob struct {
	AuditID   string    `json:"audit_id"`
	ProjectID string    `json:"project_id"`
	UserID    string    `json:"user_id"`
	Domain    string    `json:"domain"`
	MaxPages  int       `json:"max_pages"`
	Plan      string    `json:"plan"` // determines worker priority
	CreatedAt time.Time `json:"created_at"`
}

// AuditProgress is published by the crawler worker for real-time dashboard updates.
type AuditProgress struct {
	AuditID      string  `json:"audit_id"`
	ProjectID    string  `json:"project_id"`
	CrawledPages int     `json:"crawled_pages"`
	TotalPages   int     `json:"total_pages"`
	Progress     float64 `json:"progress"` // 0.0–1.0
	CurrentURL   string  `json:"current_url"`
}

// RankJob is published when rank checking is triggered (daily cron or manual).
type RankJob struct {
	ProjectID string   `json:"project_id"`
	Domain    string   `json:"domain"`
	Keywords  []string `json:"keywords"`
	Engine    string   `json:"engine"`   // "google" | "bing"
	Location  string   `json:"location"` // "us" | "gb" | etc.
	Device    string   `json:"device"`   // "desktop" | "mobile"
}
