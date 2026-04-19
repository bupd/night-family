# Architecture

Status: **design**. This document describes the intended shape of
night-family. It will be updated as the code catches up.

## Goals

1. **Prompt-configurable agents.** A family member is a YAML file: role,
   system prompt, duties, risk tolerance. No code change to add one.
2. **Small, reviewable output.** Every run produces at most one PR, scoped to
   a single duty, with tagged reviewers. Never commits to `main`.
3. **Budget-aware.** Knows the remaining tokens in the current Claude session
   window and plans accordingly.
4. **Deterministic window.** Runs inside a configured time window (default
   22:00–05:00). Never wakes outside it unless manually triggered.
5. **Observable.** Every run leaves a trail: logs, token spend, PR URL, exit
   state. Queryable via CLI, API, or Web UI.

## Non-goals (for v1)

- Multi-tenant SaaS. night-family is a single-user tool.
- Kubernetes operator. Runs as a user-level daemon.
- Cross-provider orchestration. v1 targets Claude Code; Codex/Copilot are
  stretch goals.
- Fine-grained ACLs. If you can run `nf`, you can edit family members.

## Components

### `nfd` — the daemon

Long-running process. Owns:

- The nightly **scheduler**: waits for window-start, wakes, plans, dispatches,
  sleeps at window-end.
- The **HTTP API** (`/api/v1/*`, OpenAPI 3.1).
- The **Web UI** (HTMX + `html/template`, served by the same server).
- The **SQLite** store.
- A **job queue** (in-process) that funnels duties to agent runners.

Runs under systemd (Linux) or launchd (macOS). Installs via
`nf daemon install`.

### `nf` — the CLI

Thin client that talks to `nfd` over its local socket (unix socket by default,
TCP optional). Subcommands:

```
nf daemon       install | start | stop | status
nf family       list | show | edit | add | remove
nf duty         list | show | run <name>
nf run          list | show <id> | logs <id>
nf budget       show | set
nf schedule     show | set-window
nf config       get | set | path
nf version
```

Everything the CLI does is also exposed on the API.

### `nf-api` (inside `nfd`)

REST, JSON, OpenAPI 3.1 spec at `api/openapi.yaml`. Endpoints in
[`API.md`](API.md). Spec is the source of truth: handlers are scaffolded from
it (`oapi-codegen` or hand-written against `kin-openapi` validation).

### Agent runner

Abstracts the act of "run a prompt in a codebase and return structured
results." v1 ships a **Claude Code adapter**:

- Spawns `claude` in `--print` mode (headless), with
  `--dangerously-skip-permissions` gated behind an explicit config flag.
- Streams output; parses the final JSON/text for summary + file changes.
- Runs inside a fresh git **worktree** so the main checkout is untouched.

Adapter interface (sketch):

```go
type Provider interface {
    Name() string
    Run(ctx context.Context, req RunRequest) (*RunResult, error)
    SessionStatus(ctx context.Context) (*SessionStatus, error)
}
```

### Session / budget tracker

Two concerns, one package:

- **Session status** — how many tokens remain in the current Claude 5-hour
  window? Pulled from (a) local Claude CLI state if readable, (b) a wrapper
  around `claude /status`, or (c) an estimate from our own logged spend.
- **Budget plan** — given remaining tokens and a planned set of duties, emit
  an ordered schedule, pruning what won't fit.

Budget is tracked per-night and per-agent. Stored in SQLite.

### Scheduler

Single-threaded planner, multi-threaded (configurable) dispatcher.

```
every 1m:
  if now < window.start: sleep
  if now > window.end:   finalize_night(); exit
  if no job running and queue empty:
    plan = build_plan(remaining_budget, due_duties)
    enqueue(plan)
  if worker_slot free and queue not empty:
    dispatch(next)
```

### Git orchestrator

Given a completed `RunResult`:

1. Pick branch name: `night-family/<member>/<duty>-<shortid>`
2. In the worktree, stage only the files the agent touched
3. Commit with a conventional message (`docs:`, `test:`, `fix:`, etc.)
4. Push branch to origin
5. Open PR via `gh` or GitHub API, body templated with run metadata
6. Post review-request comment tagging configured reviewers
7. Store PR URL + sha in SQLite

Every step is idempotent and resumable.

### Storage

SQLite (`modernc.org/sqlite` — pure Go, no cgo). Migrations in
`internal/storage/migrations/`. Schema overview in
[`docs/DECISIONS.md`](DECISIONS.md) (ADR-0003).

### Web UI

Server-side rendering with `html/template` + HTMX for interactivity. No JS
build step. Minimal CSS (single stylesheet). Pages:

- `/` Dashboard: tonight's plan, live run status, last night's PRs
- `/family` Roster & edit
- `/duties` Catalog
- `/runs` History + filter
- `/runs/{id}` Detail + streamed logs
- `/settings` Window, budget, reviewers

HTMX endpoints mirror the JSON API but return HTML fragments.

## Data model (short)

```
family_members(id, name, role, system_prompt, risk_tolerance, config_json,
               created_at, updated_at)
duties(id, member_id, type, interval, priority, cost_tier, enabled,
       last_run_at, config_json)
runs(id, duty_id, member_id, status, started_at, finished_at,
     tokens_in, tokens_out, branch, pr_url, summary, error)
run_logs(run_id, ts, stream, line)
budget_snapshots(id, taken_at, remaining_tokens, window_end)
nights(id, started_at, finished_at, plan_json, summary)
```

## Security & safety

- Agents run in a **worktree**, not the main checkout.
- Default max blast radius per PR: configurable (files changed, lines
  changed). Runs that exceed it are flagged, not merged.
- `--dangerously-skip-permissions` is opt-in per family member.
- Secrets: loaded from env / OS keychain; never persisted to SQLite.
- `main` is always protected via PR requirement. night-family refuses to run
  if the daemon can't verify branch protection is on.
- Network: the daemon binds to `127.0.0.1` by default; LAN access requires
  explicit config + token auth.

## Open questions

Tracked in [`docs/DECISIONS.md`](DECISIONS.md) as ADR-0000 (open) until
resolved.

- How do we cleanly read Claude session remaining tokens?
- Should `nfd` manage its own provider processes or spawn per-run?
- Worktree reuse vs. fresh-per-run?
- How to scope test-coverage duty safely (running untrusted test code)?
