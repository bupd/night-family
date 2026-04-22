# TD: Missing token counting in provider implementations

**Priority:** Medium
**Category:** Feature Gap / Budget Accuracy
**Scope:** `internal/provider/claude.go`, `internal/provider/codex.go`

## Problem

Neither the Claude nor Codex provider implementation tracks actual token usage from API
responses. The `provider.Result` struct has fields for token counts, but they are left at
zero after a Run completes. This means:

- Budget enforcement is based entirely on planner estimates, not actual consumption.
- The runs table and dashboard show 0 tokens for every completed run.
- Budget snapshots accumulate estimation drift over multiple nights.

## Suggested Fix

After each provider API call, parse the response's usage metadata:
- **Claude:** Extract `input_tokens` and `output_tokens` from the API response.
- **Codex:** Extract `usage.prompt_tokens` and `usage.completion_tokens`.

Populate `provider.Result.InputTokens` and `provider.Result.OutputTokens` so the runner
can record accurate usage and the planner can adjust future estimates.
