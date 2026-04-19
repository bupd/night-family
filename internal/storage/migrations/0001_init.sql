-- night-family v1 schema.
--
-- Conventions:
--   * IDs are prefixed ULIDs, stored as TEXT.
--   * Timestamps are RFC 3339 UTC, stored as TEXT.
--   * JSON blobs sit in TEXT columns prefixed *_json.
--   * Foreign keys are declarative; we run with PRAGMA foreign_keys=ON.

CREATE TABLE IF NOT EXISTS nights (
    id           TEXT    PRIMARY KEY CHECK (id LIKE 'night_%'),
    started_at   TEXT    NOT NULL,
    finished_at  TEXT,
    plan_json    TEXT    NOT NULL DEFAULT '{}',
    summary      TEXT
);

CREATE INDEX IF NOT EXISTS idx_nights_started_at ON nights(started_at DESC);

CREATE TABLE IF NOT EXISTS runs (
    id           TEXT    PRIMARY KEY CHECK (id LIKE 'run_%'),
    night_id     TEXT    REFERENCES nights(id) ON DELETE SET NULL,
    member       TEXT    NOT NULL,
    duty         TEXT    NOT NULL,
    status       TEXT    NOT NULL CHECK (status IN ('queued','running','succeeded','failed','cancelled')),
    started_at   TEXT    NOT NULL,
    finished_at  TEXT,
    tokens_in    INTEGER,
    tokens_out   INTEGER,
    branch       TEXT,
    pr_url       TEXT,
    summary      TEXT,
    error        TEXT
);

CREATE INDEX IF NOT EXISTS idx_runs_started_at  ON runs(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_runs_status      ON runs(status);
CREATE INDEX IF NOT EXISTS idx_runs_night       ON runs(night_id);
CREATE INDEX IF NOT EXISTS idx_runs_member_duty ON runs(member, duty);

CREATE TABLE IF NOT EXISTS run_logs (
    run_id       TEXT    NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    ts           TEXT    NOT NULL,
    stream       TEXT    NOT NULL CHECK (stream IN ('stdout','stderr','event')),
    line         TEXT    NOT NULL,
    PRIMARY KEY (run_id, ts, stream, line)
);

CREATE INDEX IF NOT EXISTS idx_run_logs_run_ts ON run_logs(run_id, ts);

CREATE TABLE IF NOT EXISTS prs (
    id           TEXT    PRIMARY KEY CHECK (id LIKE 'pr_%'),
    run_id       TEXT    REFERENCES runs(id) ON DELETE SET NULL,
    url          TEXT    NOT NULL UNIQUE,
    title        TEXT,
    member       TEXT    NOT NULL,
    duty         TEXT    NOT NULL,
    opened_at    TEXT    NOT NULL,
    merged_at    TEXT,
    state        TEXT    NOT NULL CHECK (state IN ('open','closed','merged'))
);

CREATE INDEX IF NOT EXISTS idx_prs_opened_at ON prs(opened_at DESC);
CREATE INDEX IF NOT EXISTS idx_prs_state     ON prs(state);

CREATE TABLE IF NOT EXISTS budget_snapshots (
    id                       INTEGER PRIMARY KEY AUTOINCREMENT,
    taken_at                 TEXT    NOT NULL,
    provider                 TEXT    NOT NULL,
    remaining_tokens_estimate INTEGER NOT NULL,
    window_ends_at           TEXT,
    reserved_for_tonight     INTEGER NOT NULL DEFAULT 0,
    confidence               TEXT    NOT NULL DEFAULT 'medium'
);

CREATE INDEX IF NOT EXISTS idx_budget_snapshots_taken_at ON budget_snapshots(taken_at DESC);

CREATE TABLE IF NOT EXISTS schema_meta (
    key         TEXT PRIMARY KEY,
    value       TEXT NOT NULL
);

INSERT OR IGNORE INTO schema_meta (key, value) VALUES ('version', '0001');
