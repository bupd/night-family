// Package storage persists night-family state in SQLite.
//
// Layout:
//   - One DB file per daemon instance.
//   - WAL journal mode; foreign keys on.
//   - Migrations are plain .sql files under migrations/ applied in
//     lexical order. schema_meta.version stores the latest applied
//     migration id.
//
// Everything here is concurrency-safe (SQLite + database/sql pooling).
// Callers pass context.Context through to every call so the HTTP
// handlers can cancel long queries on request timeout.
package storage

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Common errors exposed to callers.
var (
	ErrNotFound = errors.New("storage: not found")
)

// DB wraps *sql.DB with helpers tuned for night-family's access
// patterns. Use Open to construct; never build one by hand.
type DB struct {
	raw *sql.DB
}

// Open opens (or creates) the SQLite database at dsn and applies any
// pending migrations. Pass "file:nf.db?cache=shared" for the common
// case or ":memory:" in tests.
func Open(ctx context.Context, dsn string) (*DB, error) {
	if !strings.Contains(dsn, ":memory:") {
		// Always enable foreign keys + WAL on disk-backed databases.
		if strings.Contains(dsn, "?") {
			dsn += "&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)"
		} else {
			dsn += "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)"
		}
	} else {
		// In-memory DBs don't support WAL but we still want FKs.
		if strings.Contains(dsn, "?") {
			dsn += "&_pragma=foreign_keys(1)"
		} else {
			dsn += "?_pragma=foreign_keys(1)"
		}
	}
	raw, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("storage: open: %w", err)
	}
	if err := raw.PingContext(ctx); err != nil {
		_ = raw.Close()
		return nil, fmt.Errorf("storage: ping: %w", err)
	}
	db := &DB{raw: raw}
	if err := db.migrate(ctx); err != nil {
		_ = raw.Close()
		return nil, fmt.Errorf("storage: migrate: %w", err)
	}
	return db, nil
}

// Close releases the underlying pool.
func (db *DB) Close() error { return db.raw.Close() }

// Raw exposes the underlying *sql.DB for callers that need it (tests,
// ad-hoc queries). Prefer the typed helpers below.
func (db *DB) Raw() *sql.DB { return db.raw }

// migrate applies pending migrations in lexical order. Each file's
// statements run in a single transaction; a failure rolls back that
// file and returns the error.
func (db *DB) migrate(ctx context.Context) error {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		raw, err := fs.ReadFile(migrationsFS, "migrations/"+name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		if err := db.applyMigration(ctx, name, string(raw)); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
	}
	return nil
}

func (db *DB) applyMigration(ctx context.Context, name, sqlText string) (err error) {
	tx, err := db.raw.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if _, err = tx.ExecContext(ctx, sqlText); err != nil {
		return err
	}
	return tx.Commit()
}

// Version returns the latest applied migration id (e.g. "0001").
// Returns ("", ErrNotFound) when the schema_meta table hasn't been
// populated.
func (db *DB) Version(ctx context.Context) (string, error) {
	var v string
	err := db.raw.QueryRowContext(ctx,
		"SELECT value FROM schema_meta WHERE key='version'").Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return v, err
}
