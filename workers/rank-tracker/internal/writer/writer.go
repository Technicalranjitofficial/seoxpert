package writer

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/seoxpert/workers/rank-tracker/internal/engine"
)

// Writer persists rank results to PostgreSQL (Supabase).
type Writer struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Writer {
	return &Writer{pool: pool}
}

// SaveRankResult upserts a rank check result for a keyword+project.
func (w *Writer) SaveRankResult(ctx context.Context, projectID string, r *engine.RankResult) error {
	_, err := w.pool.Exec(ctx, `
		INSERT INTO rank_history (project_id, keyword, domain, position, ranking_url, engine, checked_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, projectID, r.Keyword, r.Domain, r.Position, r.URL, r.Engine, r.CheckedAt)
	return err
}

// EnsureRankHistoryTable creates the rank_history table if it doesn't exist.
// (ClickHouse has its own schema; this is for Supabase Postgres.)
func (w *Writer) EnsureRankHistoryTable(ctx context.Context) error {
	_, err := w.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS rank_history (
			id          UUID DEFAULT gen_random_uuid() PRIMARY KEY,
			project_id  UUID NOT NULL,
			keyword     TEXT NOT NULL,
			domain      TEXT NOT NULL,
			position    INTEGER NOT NULL DEFAULT 0,
			ranking_url TEXT,
			engine      TEXT NOT NULL DEFAULT 'google',
			checked_at  TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	return err
}

// GetLatestPositions returns the most recent rank for each keyword in a project.
func (w *Writer) GetLatestPositions(ctx context.Context, projectID string) ([]RankSummary, error) {
	rows, err := w.pool.Query(ctx, `
		SELECT DISTINCT ON (keyword)
			keyword, position, ranking_url, engine, checked_at
		FROM rank_history
		WHERE project_id = $1
		ORDER BY keyword, checked_at DESC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []RankSummary
	for rows.Next() {
		var rs RankSummary
		if err := rows.Scan(&rs.Keyword, &rs.Position, &rs.URL, &rs.Engine, &rs.CheckedAt); err != nil {
			continue
		}
		results = append(results, rs)
	}
	return results, rows.Err()
}

type RankSummary struct {
	Keyword   string
	Position  int
	URL       string
	Engine    string
	CheckedAt time.Time
}
