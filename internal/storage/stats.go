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
	TokensIn     int64          `json:"tokens_in"`
	TokensOut    int64          `json:"tokens_out"`
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
	for rows.Next() {
		var st string
		var n int
		if err := rows.Scan(&st, &n); err != nil {
			rows.Close()
			return s, err
		}
		s.RunsByStatus[st] = n
	}
	rows.Close()
	rows, err = db.raw.QueryContext(ctx, `SELECT state, COUNT(*) FROM prs GROUP BY state`)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var st string
		var n int
		if err := rows.Scan(&st, &n); err != nil {
			rows.Close()
			return s, err
		}
		s.PRsByState[st] = n
	}
	rows.Close()

	_ = db.raw.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(tokens_in),0), COALESCE(SUM(tokens_out),0) FROM runs`,
	).Scan(&s.TokensIn, &s.TokensOut)

	return s, nil
}
