package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/seoxpert/shared/events"
	"github.com/seoxpert/shared/models"
)

// TriggerAudit creates an audit record and publishes a job to Redpanda.
// The actual crawling happens in the crawler worker — this returns immediately.
func (h *Handler) TriggerAudit(c fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	type req struct {
		ProjectID string `json:"project_id"`
	}

	var body req
	if err := c.Bind().JSON(&body); err != nil || body.ProjectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "project_id required"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	// Verify project belongs to user and fetch domain.
	var domain string
	err := h.pool.QueryRow(ctx, `
		SELECT domain FROM projects WHERE id = $1 AND user_id = $2
	`, body.ProjectID, userID).Scan(&domain)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	// Get user plan for page limit.
	plan, _ := h.rdb.Get(ctx, "plan:"+userID)
	if plan == "" {
		plan = "free"
	}
	maxPages := models.Limits[models.Plan(plan)].MaxPagesPerAudit

	// Persist audit row in pending state.
	var audit models.Audit
	err = h.pool.QueryRow(ctx, `
		INSERT INTO audits (project_id, user_id, status)
		VALUES ($1, $2, 'pending')
		RETURNING id, project_id, user_id, status, total_pages, crawled_pages, score, issues, created_at
	`, body.ProjectID, userID).Scan(
		&audit.ID, &audit.ProjectID, &audit.UserID, &audit.Status,
		&audit.TotalPages, &audit.CrawledPages, &audit.Score, &audit.Issues, &audit.CreatedAt,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create audit"})
	}

	// Publish job — crawler worker picks this up asynchronously.
	job := events.AuditJob{
		AuditID:   audit.ID,
		ProjectID: body.ProjectID,
		UserID:    userID,
		Domain:    domain,
		MaxPages:  maxPages,
		Plan:      plan,
		CreatedAt: time.Now().UTC(),
	}
	if err := h.prod.PublishAuditJob(ctx, job); err != nil {
		// Roll back audit row — job didn't reach the queue.
		_, _ = h.pool.Exec(ctx, `DELETE FROM audits WHERE id = $1`, audit.ID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to queue audit"})
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"data":    audit,
		"message": "audit queued — results stream via Supabase Realtime",
	})
}

// GetAudit returns audit details and its issues, scoped to the authenticated user.
func (h *Handler) GetAudit(c fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	auditID := c.Params("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var audit models.Audit
	err := h.pool.QueryRow(ctx, `
		SELECT id, project_id, user_id, status, total_pages, crawled_pages, score, issues, created_at, completed_at
		FROM audits
		WHERE id = $1 AND user_id = $2
	`, auditID, userID).Scan(
		&audit.ID, &audit.ProjectID, &audit.UserID, &audit.Status,
		&audit.TotalPages, &audit.CrawledPages, &audit.Score, &audit.Issues,
		&audit.CreatedAt, &audit.CompletedAt,
	)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "audit not found"})
	}

	// Fetch issues for completed audits.
	var issues []models.AuditIssue
	if audit.Status == models.AuditStatusCompleted {
		rows, err := h.pool.Query(ctx, `
			SELECT id, audit_id, url, check_type, severity, title, description, suggestion, value
			FROM audit_issues
			WHERE audit_id = $1
			ORDER BY
				CASE severity WHEN 'critical' THEN 1 WHEN 'warning' THEN 2 ELSE 3 END,
				url
			LIMIT 1000
		`, auditID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var i models.AuditIssue
				if err := rows.Scan(&i.ID, &i.AuditID, &i.URL, &i.CheckType, &i.Severity, &i.Title, &i.Description, &i.Suggestion, &i.Value); err == nil {
					issues = append(issues, i)
				}
			}
		}
	}

	return c.JSON(fiber.Map{
		"data":   audit,
		"issues": issues,
	})
}

// ListAudits returns recent audits for the authenticated user, optionally filtered by project.
func (h *Handler) ListAudits(c fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	projectID := c.Query("project_id")
	limit := 20

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var (
		pgRows interface {
			Next() bool
			Scan(...any) error
			Close()
		}
		err error
	)
	if projectID != "" {
		pgRows, err = h.pool.Query(ctx, `
			SELECT id, project_id, user_id, status, total_pages, crawled_pages, score, issues, created_at, completed_at
			FROM audits WHERE user_id = $1 AND project_id = $2
			ORDER BY created_at DESC LIMIT $3
		`, userID, projectID, limit)
	} else {
		pgRows, err = h.pool.Query(ctx, `
			SELECT id, project_id, user_id, status, total_pages, crawled_pages, score, issues, created_at, completed_at
			FROM audits WHERE user_id = $1
			ORDER BY created_at DESC LIMIT $2
		`, userID, limit)
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch audits"})
	}
	defer pgRows.Close()

	var audits []models.Audit
	for pgRows.Next() {
		var a models.Audit
		if err := pgRows.Scan(&a.ID, &a.ProjectID, &a.UserID, &a.Status,
			&a.TotalPages, &a.CrawledPages, &a.Score, &a.Issues,
			&a.CreatedAt, &a.CompletedAt); err == nil {
			audits = append(audits, a)
		}
	}
	if audits == nil {
		audits = []models.Audit{}
	}

	return c.JSON(fiber.Map{"data": audits, "count": len(audits)})
}
