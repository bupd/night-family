# TD: Fragile SQL query construction in storage/runs.go

**Priority:** High
**Category:** Bug / Security
**Scope:** `internal/storage/runs.go`

## Problem

`ListRuns` in `storage/runs.go` (around lines 82-108) builds SQL queries using string
concatenation with `fmt.Sprintf` or manual `WHERE` clause assembly. While the current
callers pass controlled values (from URL query params that are typed), this pattern is:

1. **Fragile** — adding a new filter requires careful string manipulation and it's easy
   to introduce syntax errors (e.g., mismatched `AND`/`WHERE`).
2. **Injection-adjacent** — if any caller passes unsanitized input, the concatenated SQL
   becomes a vector. Currently safe because `RunStatus` is a typed string, but this is
   a latent risk.

## Suggested Fix

Refactor to use a query builder pattern:
```go
var clauses []string
var args []any
if f.Member != "" {
    clauses = append(clauses, "member = ?")
    args = append(args, f.Member)
}
// ...
where := ""
if len(clauses) > 0 {
    where = " WHERE " + strings.Join(clauses, " AND ")
}
query := "SELECT ... FROM runs" + where + " ORDER BY started_at DESC"
```
This is still plain SQL (no ORM), but eliminates the fragile string splicing.
