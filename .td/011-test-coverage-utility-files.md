# TD: LOW — Add test coverage for gitops/files.go and version/version.go

**Priority:** Low
**Category:** Test Coverage
**Scope:** `internal/gitops/files.go`, `internal/version/version.go`

## Problem

- `gitops/files.go` — File manipulation helpers for git operations. `gitops_test.go`
  exists but may not cover `files.go` specifically. Verify coverage and add tests for
  any uncovered functions.

- `version/version.go` — Build-time version injection. No test file exists. While
  simple, a test ensures the linker flags work correctly and the version string format
  is stable for CLI output and API responses.

## Suggested Approach

Low priority — address when touching these files for other reasons. A single test per
file exercising the happy path is sufficient.
