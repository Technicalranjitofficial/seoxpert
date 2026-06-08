package models

import "time"

type AuditStatus string

const (
	AuditStatusPending   AuditStatus = "pending"
	AuditStatusRunning   AuditStatus = "running"
	AuditStatusCompleted AuditStatus = "completed"
	AuditStatusFailed    AuditStatus = "failed"
)

type AuditSeverity string

const (
	SeverityCritical AuditSeverity = "critical"
	SeverityWarning  AuditSeverity = "warning"
	SeverityInfo     AuditSeverity = "info"
)

type Audit struct {
	ID           string      `json:"id"`
	ProjectID    string      `json:"project_id"`
	UserID       string      `json:"user_id"`
	Status       AuditStatus `json:"status"`
	TotalPages   int         `json:"total_pages"`
	CrawledPages int         `json:"crawled_pages"`
	Score        int         `json:"score"`   // 0–100
	Issues       int         `json:"issues"`
	CreatedAt    time.Time   `json:"created_at"`
	CompletedAt  *time.Time  `json:"completed_at,omitempty"`
}

// AuditIssue is one SEO problem found during a crawl.
type AuditIssue struct {
	ID          string        `json:"id"`
	AuditID     string        `json:"audit_id"`
	URL         string        `json:"url"`
	CheckType   string        `json:"check_type"` // e.g. "missing_title", "broken_link"
	Severity    AuditSeverity `json:"severity"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Suggestion  string        `json:"suggestion"`
}

// AuditSummary is returned by the API for dashboard cards.
type AuditSummary struct {
	Audit
	CriticalCount int `json:"critical_count"`
	WarningCount  int `json:"warning_count"`
	InfoCount     int `json:"info_count"`
}
