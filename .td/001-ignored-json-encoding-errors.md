# TD: Systematic ignored JSON encoding errors across CLI

**Priority:** Medium
**Category:** Bug / Error Handling
**Scope:** `cmd/nf/*.go`

## Problem

Across 20+ locations in the CLI files, `json.NewEncoder(os.Stdout).Encode(...)` return
values are discarded with `_ = enc.Encode(...)`. If stdout is a broken pipe or closed fd
(common when piped to `head`, `grep`, etc.), the write error is silently lost and the
process exits 0 — misleading in scripts and CI pipelines.

## Affected Files

- `cmd/nf/night.go` (lines 42, 83, 166)
- `cmd/nf/pr.go` (line 48)
- `cmd/nf/duty.go` (lines 59, 82)
- `cmd/nf/family.go` (lines 59, 140, 167)
- `cmd/nf/run.go` (lines 92, 133, 148)

## Suggested Fix

Create a shared helper like:
```go
func writeJSON(v any) {
    enc := json.NewEncoder(os.Stdout)
    enc.SetIndent("", "  ")
    if err := enc.Encode(v); err != nil {
        fmt.Fprintln(os.Stderr, "nf:", err)
        os.Exit(1)
    }
}
```
Then replace all `_ = enc.Encode(...)` call sites with `writeJSON(v)`.
