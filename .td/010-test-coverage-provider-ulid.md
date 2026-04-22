# TD: MEDIUM — Add test coverage for provider/status.go and ulid/ulid.go

**Priority:** Medium
**Category:** Test Coverage
**Scope:** `internal/provider/status.go`, `internal/ulid/ulid.go`

## Problem

- `provider/status.go` — No tests for the session status probing logic. When real
  provider status is implemented (see TD-003), tests will be essential to verify
  parsing and error handling.

- `ulid/ulid.go` — No test file exists. The ULID generation is used as the primary key
  for nights and runs. While the implementation is likely a thin wrapper, tests should
  verify:
  - Output format (prefix + ULID body)
  - Uniqueness across rapid successive calls
  - Monotonicity (lexical sort = temporal sort)

## Suggested Approach

Create `internal/provider/status_test.go` and `internal/ulid/ulid_test.go` with basic
property tests.
