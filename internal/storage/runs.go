package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// RunStatus mirrors the OpenAPI enum.
type RunStatus string

const (
	RunQueued    RunStatus = "queued"
	RunRunning   RunStatus = "running"
	RunSucceeded RunStatus = "succeeded"
	RunFailed    RunStatus = "failed"
	RunCancelled RunStatus = "cancelled"
)

// Run is the persisted shape of a duty execution.
type Run struct {
	ID         string     `json:"id"`
	NightID    *string    `json:"night_id,omitempty"`
	Member     string     `json:"member"`
	Duty       string     `json:"duty"`
	Status     RunStatus  `json:"status"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	TokensIn   *int       `json:"tokens_in,omitempty"`
	TokensOut  *int       `json:"tokens_out,omitempty"`
	Branch     *string    `json:"branch,omitempty"`
	PRURL      *string    `json:"pr_url,omitempty"`
	Summary    *string    `json:"summary,omitempty"`
	Error      *string    `json:"error,omitempty"`
}

// InsertRun persists a new Run. ID must be set and start with "run_";
// StartedAt must be non-zero.
func (db *DB) InsertRun(ctx context.Context, r Run) error {
	if r.ID == "" || r.Member == "" || r.Duty == "" {
		return errors.New("storage: InsertRun: id/member/duty required")
	}
	if r.StartedAt.IsZero() {
		return errors.New("storage: InsertRun: started_at required")
	}
	_, err := db.raw.ExecContext(ctx, `
		INSERT INTO runs(
			id, night_id, member, duty, status,
			started_at, finished_at, tokens_in, tokens_out,
			branch, pr_url, summary, error
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.NightID, r.Member, r.Duty, string(r.Status),
		r.StartedAt.UTC().Format(time.RFC3339Nano),
		nullableTime(r.FinishedAt), nullableInt(r.TokensIn), nullableInt(r.TokensOut),
		nullableStr(r.Branch), nullableStr(r.PRURL), nullableStr(r.Summary), nullableStr(r.Error),
	)
	return err
}

// GetRun returns one Run or ErrNotFound.
func (db *DB) GetRun(ctx context.Context, id string) (Run, error) {
	row := db.raw.QueryRowContext(ctx, `
		SELECT id, night_id, member, duty, status,
			   started_at, finished_at, tokens_in, tokens_out,
			   branch, pr_url, summary, error
		FROM runs WHERE id = ?`, id)
	return scanRun(row)
}

// ListRunsFilter is the optional predicate set for ListRuns.
type ListRunsFilter struct {
	Member string
	Duty   string
	Status RunStatus
	Since  time.Time
	Limit  int
}

// ListRuns returns runs ordered by started_at desc.
func (db *DB) ListRuns(ctx context.Context, f ListRunsFilter) ([]Run, error) {
	q := `SELECT id, night_id, member, duty, status,
			   started_at, finished_at, tokens_in, tokens_out,
			   branch, pr_url, summary, error
		FROM runs WHERE 1=1`
	args := []any{}
	if f.Member != "" {
		q += " AND member = ?"
		args = append(args, f.Member)
	}
	if f.Duty != "" {
		q += " AND duty = ?"
		args = append(args, f.Duty)
	}
	if f.Status != "" {
		q += " AND status = ?"
		args = append(args, string(f.Status))
	}
	if !f.Since.IsZero() {
		q += " AND started_at >= ?"
		args = append(args, f.Since.UTC().Format(time.RFC3339Nano))
	}
	q += " ORDER BY started_at DESC"
	if f.Limit <= 0 || f.Limit > 500 {
		f.Limit = 50
	}
	q += fmt.Sprintf(" LIMIT %d", f.Limit)
	rows, err := db.raw.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Run
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// UpdateRunStatus flips the status / timestamps / token counts on a
// run. Zero-valued fields are left untouched.
func (db *DB) UpdateRunStatus(ctx context.Context, id string, status RunStatus, finishedAt *time.Time, tokensIn, tokensOut *int, summary, errMsg *string) error {
	_, err := db.raw.ExecContext(ctx, `
		UPDATE runs SET
			status = ?,
			finished_at = COALESCE(?, finished_at),
			tokens_in   = COALESCE(?, tokens_in),
			tokens_out  = COALESCE(?, tokens_out),
			summary     = COALESCE(?, summary),
			error       = COALESCE(?, error)
		WHERE id = ?`,
		string(status),
		nullableTime(finishedAt), nullableInt(tokensIn), nullableInt(tokensOut),
		nullableStr(summary), nullableStr(errMsg),
		id,
	)
	return err
}

// rowScanner matches both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanRun(row rowScanner) (Run, error) {
	var r Run
	var (
		nightID, branch, prURL, summary, errMsg, finishedAt sql.NullString
		startedAt                                           string
		tokensIn, tokensOut                                 sql.NullInt64
	)
	err := row.Scan(
		&r.ID, &nightID, &r.Member, &r.Duty, &r.Status,
		&startedAt, &finishedAt, &tokensIn, &tokensOut,
		&branch, &prURL, &summary, &errMsg,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Run{}, ErrNotFound
	}
	if err != nil {
		return Run{}, err
	}
	if nightID.Valid {
		s := nightID.String
		r.NightID = &s
	}
	if t, perr := time.Parse(time.RFC3339Nano, startedAt); perr == nil {
		r.StartedAt = t
	}
	if finishedAt.Valid {
		if t, perr := time.Parse(time.RFC3339Nano, finishedAt.String); perr == nil {
			r.FinishedAt = &t
		}
	}
	if tokensIn.Valid {
		v := int(tokensIn.Int64)
		r.TokensIn = &v
	}
	if tokensOut.Valid {
		v := int(tokensOut.Int64)
		r.TokensOut = &v
	}
	if branch.Valid {
		s := branch.String
		r.Branch = &s
	}
	if prURL.Valid {
		s := prURL.String
		r.PRURL = &s
	}
	if summary.Valid {
		s := summary.String
		r.Summary = &s
	}
	if errMsg.Valid {
		s := errMsg.String
		r.Error = &s
	}
	return r, nil
}

func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func nullableInt(i *int) any {
	if i == nil {
		return nil
	}
	return *i
}

func nullableStr(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}
