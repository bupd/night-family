# TD: HIGH — Add test coverage for internal/storage/runs.go filtering

**Priority:** High
**Category:** Test Coverage
**Scope:** `internal/storage/runs.go`

## Problem

`storage/runs.go` has `ListRuns` with filter logic (by member, duty, status, limit) that
builds dynamic SQL. The existing `storage_test.go` tests basic CRUD but does not exercise
the filter combinations. Regressions in the SQL construction (see TD-005) would go
undetected.

## Suggested Tests

Create or extend `internal/storage/runs_test.go`:
1. Insert 10+ runs with varied member/duty/status values
2. Test each filter individually (member only, duty only, status only, limit only)
3. Test filter combinations (member + duty, status + limit)
4. Test empty results (no matches)
5. Test default behavior (no filters → all runs, ordered by started_at DESC)
