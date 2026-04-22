# TD: MEDIUM — Add test coverage for server endpoints

**Priority:** Medium
**Category:** Test Coverage
**Scope:** `internal/server/budget.go`, `internal/server/dashboard.go`, `internal/server/prs.go`, `internal/server/runs.go`, `internal/server/nights.go`

## Problem

Several server endpoints lack test coverage:
- `budget.go` — budget snapshot API has no tests
- `dashboard.go` — HTML dashboard rendering is untested
- `prs.go` — PR listing endpoint has no tests
- `runs.go` — `createRun` (POST) and `runsPage` (HTML) are untested
- `nights.go` — `triggerNight` (POST) is untested

The existing `server/*_test.go` files cover GET endpoints well but skip mutation
endpoints and HTML page rendering.

## Suggested Approach

Follow the existing pattern in `server/runs_test.go` and `server/nights_test.go`:
- Use `httptest.NewRecorder` + `httptest.NewRequest`
- Wire up the server with in-memory storage and mock provider
- Test success paths, validation errors, and 404s
- For HTML pages, assert status 200 and spot-check response body for key strings
