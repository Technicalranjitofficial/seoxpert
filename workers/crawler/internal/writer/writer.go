package writer

import (
	"context"
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
