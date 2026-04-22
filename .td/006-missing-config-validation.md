# TD: Missing config validation in config/config.go

**Priority:** Medium
**Category:** Bug / UX
**Scope:** `internal/config/config.go`

## Problem

`config.Load()` reads the YAML/env config but performs minimal validation. Invalid or
incomplete configurations are discovered late — often at runtime when a provider call
fails or a nil pointer is hit. Examples:

- No validation that `provider` is a recognized value ("claude", "codex", "mock").
- No validation that `schedule.window_start` / `window_end` are valid time strings.
- No validation that `family_dir` exists or is readable.
- No validation that `db_path` parent directory is writable.
- API keys are not checked for non-empty when a real provider is selected.

## Suggested Fix

Add a `Config.Validate() error` method that checks all invariants at load time and
returns a clear, actionable error message. Call it from `Load()` before returning.
This gives operators fast feedback on misconfiguration instead of cryptic runtime panics.
