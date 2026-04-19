package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Night is the persisted representation of one scheduled (or manually
// triggered) window.
type Night struct {
	ID         string     `json:"id"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	PlanJSON   string     `json:"plan_json,omitempty"`
	Summary    *string    `json:"summary,omitempty"`
}

// InsertNight persists a new Night. The caller owns ID generation.
func (db *DB) InsertNight(ctx context.Context, n Night) error {
	if n.ID == "" {
		return errors.New("storage: InsertNight: id required")
	}
	if n.StartedAt.IsZero() {
		return errors.New("storage: InsertNight: started_at required")
	}
	if n.PlanJSON == "" {
		n.PlanJSON = "{}"
	}
	_, err := db.raw.ExecContext(ctx, `
		INSERT INTO nights(id, started_at, finished_at, plan_json, summary)
		VALUES (?, ?, ?, ?, ?)`,
		n.ID,
		n.StartedAt.UTC().Format(time.RFC3339Nano),
		nullableTime(n.FinishedAt),
		n.PlanJSON,
		nullableStr(n.Summary),
	)
	return err
}

// FinishNight stamps finished_at and a summary on a Night.
func (db *DB) FinishNight(ctx context.Context, id string, finishedAt time.Time, summary string) error {
	_, err := db.raw.ExecContext(ctx, `
		UPDATE nights SET
			finished_at = ?,
			summary     = ?
		WHERE id = ?`,
		finishedAt.UTC().Format(time.RFC3339Nano),
		summary,
		id,
	)
	return err
}

// GetNight returns one Night or ErrNotFound.
func (db *DB) GetNight(ctx context.Context, id string) (Night, error) {
	row := db.raw.QueryRowContext(ctx, `
		SELECT id, started_at, finished_at, plan_json, summary
		FROM nights WHERE id = ?`, id)
	return scanNight(row)
}

// ListNights returns nights ordered by started_at desc.
func (db *DB) ListNights(ctx context.Context, limit int) ([]Night, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := db.raw.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, started_at, finished_at, plan_json, summary
		FROM nights ORDER BY started_at DESC LIMIT %d`, limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Night
	for rows.Next() {
		n, err := scanNight(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func scanNight(row rowScanner) (Night, error) {
	var n Night
	var (
		finishedAt, summary sql.NullString
		startedAt           string
		planJSON            string
	)
	err := row.Scan(&n.ID, &startedAt, &finishedAt, &planJSON, &summary)
	if errors.Is(err, sql.ErrNoRows) {
		return Night{}, ErrNotFound
	}
	if err != nil {
		return Night{}, err
	}
	if t, perr := time.Parse(time.RFC3339Nano, startedAt); perr == nil {
		n.StartedAt = t
	}
	if finishedAt.Valid {
		if t, perr := time.Parse(time.RFC3339Nano, finishedAt.String); perr == nil {
			n.FinishedAt = &t
		}
	}
	n.PlanJSON = planJSON
	if summary.Valid {
		s := summary.String
		n.Summary = &s
	}
	return n, nil
}
