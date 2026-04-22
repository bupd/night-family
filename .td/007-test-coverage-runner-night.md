# TD: CRITICAL — Add test coverage for internal/runner/night.go

**Priority:** Critical
**Category:** Test Coverage
**Scope:** `internal/runner/night.go`

## Problem

`runner/night.go` contains the core `TriggerNight` orchestration logic — the most
important code path in the entire project — but has no dedicated unit tests. The only
coverage comes indirectly from `scheduler/loop_test.go` which exercises a narrow path.

Untested paths include:
- `TriggerNight` with various `NightOptions` combinations (OnlyMembers, OnlyDuties, Budget, DryRun)
- Context cancellation mid-dispatch (the `goto finish` path)
- `renderDigest` with DB errors, empty runs, nil PRs
- `writeDigestBody` directory creation and file writing
- `filterSlots` edge cases (empty allow-lists, partial matches)
- Notification path (Notifier.Notify success and failure)

## Suggested Approach

Create `internal/runner/night_test.go` with table-driven tests using the mock provider
and an in-memory SQLite DB (same pattern as `scheduler/loop_test.go`). Key scenarios:
1. Full night with multiple slots → verify all runs recorded
2. DryRun=true → verify zero dispatches
3. OnlyMembers/OnlyDuties filtering
4. Context cancellation after N dispatches
5. Digest rendering with and without PRs
6. Notifier called when configured
