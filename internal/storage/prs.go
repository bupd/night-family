package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// PRState is the lifecycle flag on a stored PR record.
type PRState string

const (
	PROpen   PRState = "open"
	PRClosed PRState = "closed"
	PRMerged PRState = "merged"
)

// PR is the persisted shape of a PR night-family opened.
type PR struct {
	ID       string     `json:"id"`
	RunID    *string    `json:"run_id,omitempty"`
	URL      string     `json:"url"`
	Title    *string    `json:"title,omitempty"`
	Member   string     `json:"member"`
	Duty     string     `json:"duty"`
	OpenedAt time.Time  `json:"opened_at"`
	MergedAt *time.Time `json:"merged_at,omitempty"`
	State    PRState    `json:"state"`
}

// InsertPR persists a newly-opened PR record.
func (db *DB) InsertPR(ctx context.Context, p PR) error {
	if p.ID == "" || p.URL == "" || p.Member == "" || p.Duty == "" {
		return errors.New("storage: InsertPR: id/url/member/duty required")
	}
	if p.OpenedAt.IsZero() {
		return errors.New("storage: InsertPR: opened_at required")
	}
	if p.State == "" {
		p.State = PROpen
	}
	_, err := db.raw.ExecContext(ctx, `
		INSERT INTO prs(id, run_id, url, title, member, duty, opened_at, merged_at, state)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, nullableStr(p.RunID), p.URL, nullableStr(p.Title),
		p.Member, p.Duty,
		p.OpenedAt.UTC().Format(time.RFC3339Nano),
		nullableTime(p.MergedAt),
		string(p.State),
	)
	return err
}

// ListPRs returns PRs ordered by opened_at desc.
func (db *DB) ListPRs(ctx context.Context, limit int) ([]PR, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := db.raw.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, run_id, url, title, member, duty, opened_at, merged_at, state
		FROM prs ORDER BY opened_at DESC LIMIT %d`, limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PR
	for rows.Next() {
		p, err := scanPR(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetPR returns a single PR by id, or ErrNotFound.
func (db *DB) GetPR(ctx context.Context, id string) (PR, error) {
	row := db.raw.QueryRowContext(ctx, `
		SELECT id, run_id, url, title, member, duty, opened_at, merged_at, state
		FROM prs WHERE id = ?`, id)
	return scanPR(row)
}

func scanPR(row rowScanner) (PR, error) {
	var p PR
	var (
		runID, title, mergedAt sql.NullString
		openedAt, state        string
	)
	err := row.Scan(&p.ID, &runID, &p.URL, &title, &p.Member, &p.Duty, &openedAt, &mergedAt, &state)
	if errors.Is(err, sql.ErrNoRows) {
		return PR{}, ErrNotFound
	}
	if err != nil {
		return PR{}, err
	}
	if runID.Valid {
		s := runID.String
		p.RunID = &s
	}
	if title.Valid {
		s := title.String
		p.Title = &s
	}
	if t, perr := time.Parse(time.RFC3339Nano, openedAt); perr == nil {
		p.OpenedAt = t
	}
	if mergedAt.Valid {
		if t, perr := time.Parse(time.RFC3339Nano, mergedAt.String); perr == nil {
			p.MergedAt = &t
		}
	}
	p.State = PRState(state)
	return p, nil
}
