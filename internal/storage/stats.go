package storage

import (
	"context"
)

// Stats is the aggregated "where are we at" snapshot the dashboard
// and the /api/v1/stats endpoint share.
type Stats struct {
	Nights       int            `json:"nights"`
	Runs         int            `json:"runs"`
	RunsByStatus map[string]int `json:"runs_by_status"`
	PRs          int            `json:"prs"`
	PRsByState   map[string]int `json:"prs_by_state"`
}

// Stats returns a single-query snapshot.
func (db *DB) Stats(ctx context.Context) (Stats, error) {
	s := Stats{
		RunsByStatus: map[string]int{},
		PRsByState:   map[string]int{},
	}
	if err := db.raw.QueryRowContext(ctx, `SELECT COUNT(*) FROM nights`).Scan(&s.Nights); err != nil {
		return s, err
	}
	if err := db.raw.QueryRowContext(ctx, `SELECT COUNT(*) FROM runs`).Scan(&s.Runs); err != nil {
		return s, err
	}
	if err := db.raw.QueryRowContext(ctx, `SELECT COUNT(*) FROM prs`).Scan(&s.PRs); err != nil {
		return s, err
	}
	rows, err := db.raw.QueryContext(ctx, `SELECT status, COUNT(*) FROM runs GROUP BY status`)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var st string
		var n int
		if err := rows.Scan(&st, &n); err != nil {
			return s, err
		}
		s.RunsByStatus[st] = n
	}
	if err := rows.Err(); err != nil {
		return s, err
	}
	rows2, err := db.raw.QueryContext(ctx, `SELECT state, COUNT(*) FROM prs GROUP BY state`)
	if err != nil {
		return s, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var st string
		var n int
		if err := rows2.Scan(&st, &n); err != nil {
			return s, err
		}
		s.PRsByState[st] = n
	}
	if err := rows2.Err(); err != nil {
		return s, err
	}
	return s, nil
}
