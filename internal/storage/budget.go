package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// BudgetSnapshot mirrors the openapi schema: a point-in-time estimate
// of how many tokens we have left in the current provider window.
type BudgetSnapshot struct {
	ID                      int64      `json:"id"`
	TakenAt                 time.Time  `json:"taken_at"`
	Provider                string     `json:"provider"`
	RemainingTokensEstimate int        `json:"remaining_tokens_estimate"`
	WindowEndsAt            *time.Time `json:"window_ends_at,omitempty"`
	ReservedForTonight      int        `json:"reserved_for_tonight"`
	Confidence              string     `json:"confidence"`
}

// InsertBudgetSnapshot persists one sample.
func (db *DB) InsertBudgetSnapshot(ctx context.Context, b BudgetSnapshot) (int64, error) {
	if b.Provider == "" {
		return 0, errors.New("storage: InsertBudgetSnapshot: provider required")
	}
	if b.TakenAt.IsZero() {
		b.TakenAt = time.Now().UTC()
	}
	if b.Confidence == "" {
		b.Confidence = "medium"
	}
	res, err := db.raw.ExecContext(ctx, `
		INSERT INTO budget_snapshots(
			taken_at, provider, remaining_tokens_estimate,
			window_ends_at, reserved_for_tonight, confidence
		) VALUES (?, ?, ?, ?, ?, ?)`,
		b.TakenAt.UTC().Format(time.RFC3339Nano),
		b.Provider, b.RemainingTokensEstimate,
		nullableTime(b.WindowEndsAt), b.ReservedForTonight, b.Confidence,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// LatestBudgetSnapshot returns the most recent sample or ErrNotFound.
func (db *DB) LatestBudgetSnapshot(ctx context.Context) (BudgetSnapshot, error) {
	row := db.raw.QueryRowContext(ctx, `
		SELECT id, taken_at, provider, remaining_tokens_estimate,
			   window_ends_at, reserved_for_tonight, confidence
		FROM budget_snapshots
		ORDER BY taken_at DESC, id DESC
		LIMIT 1`)
	var b BudgetSnapshot
	var (
		taken     string
		windowEnd sql.NullString
	)
	err := row.Scan(&b.ID, &taken, &b.Provider,
		&b.RemainingTokensEstimate, &windowEnd,
		&b.ReservedForTonight, &b.Confidence)
	if errors.Is(err, sql.ErrNoRows) {
		return BudgetSnapshot{}, ErrNotFound
	}
	if err != nil {
		return BudgetSnapshot{}, err
	}
	if t, perr := time.Parse(time.RFC3339Nano, taken); perr == nil {
		b.TakenAt = t
	}
	if windowEnd.Valid {
		if t, perr := time.Parse(time.RFC3339Nano, windowEnd.String); perr == nil {
			b.WindowEndsAt = &t
		}
	}
	return b, nil
}
