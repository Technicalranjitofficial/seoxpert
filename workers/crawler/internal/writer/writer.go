package writer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/seoxpert/shared/models"
	"github.com/seoxpert/workers/crawler/internal/engine"
)

// Writer persists crawl results to PostgreSQL (Supabase).
type Writer struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Writer {
	return &Writer{pool: pool}
}

// SaveSiteIssue writes a single site-level issue (e.g. robots.txt, broken link).
func (w *Writer) SaveSiteIssue(ctx context.Context, issue *models.AuditIssue) error {
	_, err := w.pool.Exec(ctx, `
		INSERT INTO audit_issues (audit_id, url, check_type, severity, title, description, suggestion, value)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, issue.AuditID, issue.URL, issue.CheckType, string(issue.Severity),
		issue.Title, issue.Description, issue.Suggestion, issue.Value)
	return err
}

// SavePageResult writes audit issues for a single crawled page.
// Uses a batch insert for performance.
func (w *Writer) SavePageResult(ctx context.Context, result *engine.PageResult) error {
	if len(result.Issues) == 0 {
		return nil
	}

	batch := &pgxBatch{}
	for _, issue := range result.Issues {
		batch.add(`
			INSERT INTO audit_issues (audit_id, url, check_type, severity, title, description, suggestion, value)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, issue.AuditID, issue.URL, issue.CheckType, string(issue.Severity),
			issue.Title, issue.Description, issue.Suggestion, issue.Value)
	}

	return batch.send(ctx, w.pool)
}

// UpdateAuditProgress updates the crawled_pages count and status.
func (w *Writer) UpdateAuditProgress(ctx context.Context, auditID string, crawled, total int) error {
	_, err := w.pool.Exec(ctx, `
		UPDATE audits
		SET crawled_pages = $2, total_pages = $3, status = 'running'
		WHERE id = $1
	`, auditID, crawled, total)
	return err
}

// CompleteAudit marks an audit finished and calculates the final score.
func (w *Writer) CompleteAudit(ctx context.Context, auditID string) error {
	now := time.Now().UTC()

	// Score = average per-page score (each page starts at 100, loses points per issue).
	// critical=-15, warning=-7, info=-2 per page, minimum 0 per page.
	// Final score = average across all crawled pages, rounded.
	_, err := w.pool.Exec(ctx, `
		UPDATE audits
		SET
			status      = 'completed',
			completed_at = $2,
			issues      = (SELECT COUNT(*) FROM audit_issues WHERE audit_id = $1),
			score       = COALESCE((
				SELECT ROUND(AVG(page_score))::int
				FROM (
					SELECT GREATEST(0,
						100
						- COUNT(*) FILTER (WHERE severity = 'critical') * 15
						- COUNT(*) FILTER (WHERE severity = 'warning')  * 7
						- COUNT(*) FILTER (WHERE severity = 'info')     * 2
					) AS page_score
					FROM audit_issues
					WHERE audit_id = $1
					GROUP BY url
				) s
			), 100)
		WHERE id = $1
	`, auditID, now)

	if err != nil {
		return err
	}

	slog.Info("audit completed", "audit_id", auditID)
	return nil
}

// FailAudit marks an audit as failed with an error message.
func (w *Writer) FailAudit(ctx context.Context, auditID, reason string) error {
	_, err := w.pool.Exec(ctx, `
		UPDATE audits SET status = 'failed', completed_at = NOW() WHERE id = $1
	`, auditID)
	slog.Error("audit failed", "audit_id", auditID, "reason", reason)
	return err
}

// ── pgx batch helper ──────────────────────────────────────────────────────────

type pgxQuery struct {
	sql  string
	args []interface{}
}

type pgxBatch struct {
	queries []pgxQuery
}

func (b *pgxBatch) add(sql string, args ...interface{}) {
	b.queries = append(b.queries, pgxQuery{sql, args})
}

func (b *pgxBatch) send(ctx context.Context, pool *pgxpool.Pool) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint

	for _, q := range b.queries {
		if _, err := tx.Exec(ctx, q.sql, q.args...); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// Ensure models package is used (avoids unused import error while engine is developed).
var _ = models.SeverityCritical

// SaveDuplicateIssues checks all pages in an audit for duplicate titles and meta
// descriptions, then inserts issues for any duplicates found.
func (w *Writer) SaveDuplicateIssues(ctx context.Context, auditID string) error {
	// Find duplicate titles
	titleRows, err := w.pool.Query(ctx, `
		SELECT title_text, COUNT(*) as cnt, ARRAY_AGG(url ORDER BY url) as urls
		FROM (
			SELECT url,
				(SELECT value FROM audit_issues
				 WHERE audit_id = $1 AND check_type = 'title_too_long' AND url = ai.url
				 LIMIT 1) as title_text
			FROM audit_issues ai
			WHERE audit_id = $1
			GROUP BY url
		) t
		WHERE title_text IS NOT NULL AND title_text != ''
		GROUP BY title_text
		HAVING COUNT(*) > 1
	`, auditID)
	if err == nil {
		defer titleRows.Close()
		for titleRows.Next() {
			var titleText string
			var cnt int
			var urls []string
			if err := titleRows.Scan(&titleText, &cnt, &urls); err != nil {
				continue
			}
			for _, u := range urls {
				_, _ = w.pool.Exec(ctx, `
					INSERT INTO audit_issues (audit_id, url, check_type, severity, title, description, suggestion, value)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				`, auditID, u, "duplicate_title", string(models.SeverityWarning),
					"Duplicate page title",
					"This page shares the same title tag with "+fmt.Sprintf("%d", cnt-1)+" other page(s). Duplicate titles confuse search engines about which page to rank and reduce the uniqueness signal for each page.",
					"Write a unique, descriptive title for each page that reflects its specific content. Target 50–60 characters.",
					titleText)
			}
		}
	}

	// Find duplicate meta descriptions using the same approach
	metaRows, err2 := w.pool.Query(ctx, `
		SELECT meta_text, COUNT(*) as cnt, ARRAY_AGG(url ORDER BY url) as urls
		FROM (
			SELECT url,
				(SELECT value FROM audit_issues
				 WHERE audit_id = $1 AND check_type = 'meta_too_long' AND url = ai.url
				 LIMIT 1) as meta_text
			FROM audit_issues ai
			WHERE audit_id = $1
			GROUP BY url
		) t
		WHERE meta_text IS NOT NULL AND meta_text != ''
		GROUP BY meta_text
		HAVING COUNT(*) > 1
	`, auditID)
	if err2 == nil {
		defer metaRows.Close()
		for metaRows.Next() {
			var metaText string
			var cnt int
			var urls []string
			if err := metaRows.Scan(&metaText, &cnt, &urls); err != nil {
				continue
			}
			for _, u := range urls {
				_, _ = w.pool.Exec(ctx, `
					INSERT INTO audit_issues (audit_id, url, check_type, severity, title, description, suggestion, value)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				`, auditID, u, "duplicate_meta", string(models.SeverityWarning),
					"Duplicate meta description",
					"This page shares the same meta description with "+fmt.Sprintf("%d", cnt-1)+" other page(s). Unique meta descriptions improve click-through rates from search results and help Google understand page content.",
					"Write a unique meta description for each page (120–155 characters) summarising its specific content.",
					metaText)
			}
		}
	}

	return nil
}
