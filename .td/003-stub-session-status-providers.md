# TD: Stub SessionStatus in provider/status.go for Claude and Codex

**Priority:** Low
**Category:** Feature Gap
**Scope:** `internal/provider/claude.go`, `internal/provider/codex.go`, `internal/provider/status.go`

## Problem

The `SessionStatus` function in `provider/status.go` is a stub or returns hardcoded
placeholder values for the Claude and Codex providers. This means:

- The budget dashboard shows estimated/fake remaining token counts.
- The planner cannot make informed decisions about budget allocation.
- The `BudgetSnapshot.Confidence` field is always "low" because there's no real probe.

## Suggested Fix

Implement actual session status probing for each provider:
- **Claude:** Use the Anthropic API's usage/billing endpoint (or infer from response
  headers) to report real remaining tokens.
- **Codex:** Use the OpenAI usage API to report remaining quota.

If real probing is not feasible short-term, at minimum log a warning when the stub is
called so operators know the data is synthetic.
