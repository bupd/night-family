# Decisions

Lightweight ADRs. Status: **proposed** (P), **accepted** (A),
**superseded** (S), **rejected** (R).

---

## ADR-0001 — Go for backend, HTMX for UI

**Status**: A

**Context.** We need a single-binary daemon that runs reliably on Linux &
macOS, interacts with git and the Claude CLI, and serves both an API and a
small UI. We want zero JS build step.

**Decision.** Go for the daemon and CLI. `html/template` + HTMX for the UI.
Pure-Go SQLite (`modernc.org/sqlite`) so the binary has no cgo.

**Consequences.**
- Single `go build` produces both `nf` and `nfd`.
- No JS toolchain in CI.
- SQLite perf is slightly below cgo driver; acceptable for our write load.

---

## ADR-0002 — OpenAPI 3.1 as the API source of truth

**Status**: A

**Context.** We want a first-class, documented API. We want a CLI that uses
the same types. We want the spec to never go stale.

**Decision.** Hand-write `api/openapi.yaml` (OAS 3.1). Validate requests at
runtime with `kin-openapi`. Generate Go types with `oapi-codegen` into
`internal/api/gen`. CI fails if generated code is stale.

**Consequences.**
- Schema changes require spec edit + regen. Slower but safer.
- Clients in any language can be generated from the spec.

---

## ADR-0003 — SQLite for all state

**Status**: A

**Context.** We need durable state: runs, PRs, budget snapshots, family
members. We don't need multi-node.

**Decision.** One SQLite DB at `~/.local/share/night-family/nf.db`. WAL
mode. Migrations in `internal/storage/migrations/*.sql`, applied at daemon
start.

**Consequences.**
- Trivial backup (copy the file).
- No external dependencies.
- If we ever need multi-node we'll revisit — likely Postgres.

---

## ADR-0004 — Custom duties are prompts first, plugins later

**Status**: A

**Context.** Users will want custom duties. We don't want to force them to
compile a Go plugin on day one.

**Decision.** v1 ships a **generic prompt-runner duty**: custom duties are
YAML files with a `prompt` field; night-family wires up the worktree, the
output parsing, and the PR/issue plumbing. Native Go duties exist for the
built-in catalog only.

**Consequences.**
- Low barrier to custom duties.
- Prompt-only duties cannot do deterministic pre/post work (e.g. run a
  coverage tool). That's fine for v1.
- Plugin system deferred; TBD in ADR-0008+.

---

## ADR-0005 — Never commit to `main`

**Status**: A

**Context.** Agents are fallible. A single bad commit to `main` can break
`main` for everyone. We also want a human review gate.

**Decision.** night-family refuses to commit to `main`/`master`. Every duty
outputs a branch named `night-family/<member>/<duty>-<shortid>`. At daemon
start we probe branch protection; if it's off we warn loudly and (by default)
refuse to run.

**Consequences.**
- Slightly more friction for one-person repos.
- Much safer default.
- Config flag `i_know_what_im_doing: true` can disable the protection check
  (not the "no commit to main" rule — that stays).

---

## ADR-0006 — Reviewers are configurable, default to `@coderabbitai` + `@cubic-dev-ai`

**Status**: A

**Context.** The user asked for `@cubic` and `@coderabbit` to be tagged on
every PR.

**Decision.** Global config `reviewers: [coderabbitai, cubic-dev-ai]`.
Per-member overrides allowed. Tagging is done via a review-request comment
(not `gh pr create --reviewer`, since those bots don't accept formal review
requests from non-org accounts — a mention in the body or a comment is more
reliable).

**Consequences.**
- Easy to swap for org-specific reviewers.
- If a bot is unavailable, PR still opens; tag is best-effort.

---

## ADR-0007 — Window scheduling, local time

**Status**: A

**Context.** User wants nightly runs 22:00 → 05:00. Timezones are painful.

**Decision.** Window is expressed in the **local system timezone** of the
machine running `nfd`. Stored as `"22:00"` + `"05:00"` strings + IANA zone
(default = system zone). The scheduler recomputes window boundaries every
night from wall-clock time to handle DST correctly.

**Consequences.**
- No surprises across DST transitions.
- A laptop that travels timezones runs at local 22:00 wherever it is.

---

## Open questions (ADR-0000)

- **Session-token introspection.** How do we cleanly read Claude session
  remaining? Options:
  - parse `~/.claude/*` files (brittle)
  - call `claude --print /status` (slow, uses tokens)
  - track spend ourselves (approximate, drifts)
  - First cut: option 3, with periodic reconciliation via option 2.

- **Worktree reuse.** Fresh worktree per run (clean but slow) vs. one
  worktree per night (fast but risk of cross-contamination). Leaning fresh
  per run with worktree pool for warm reuse.

- **Concurrent members.** Do two members ever run simultaneously? v1 likely
  no — serial is simpler and the budget is the bottleneck anyway.

- **What counts as a "run"?** A single PR? A single `claude --print`
  invocation? Probably the former; a duty may need multiple provider calls.
