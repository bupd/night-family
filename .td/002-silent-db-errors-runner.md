# TD: Silent DB errors in runner package non-fatal operations

**Priority:** Medium
**Category:** Bug / Observability
**Scope:** `internal/runner/night.go`, `internal/runner/runner.go`

## Problem

Several storage operations in the runner package silently discard errors with `_ =`:

1. `runner/night.go:104` — `_ = r.deps.Storage.FinishNight(...)`: If this fails, the night
   record is left in a permanently "running" state in the DB. The dashboard and API will
   show stale in-progress nights.

2. `runner/night.go:71-77` — `_, _ = r.deps.Storage.InsertBudgetSnapshot(...)`: Budget
   tracking silently breaks if this fails, leading to inaccurate budget displays.

3. `runner/runner.go` — Similar patterns where non-fatal DB ops discard errors without
   logging.

## Suggested Fix

Replace `_ =` with logging at `Warn` level (consistent with the existing pattern for
non-fatal errors in the same file, e.g., line 115). Do not return errors from these —
they are intentionally non-fatal — but ensure they're observable in logs so operators
can diagnose issues.
