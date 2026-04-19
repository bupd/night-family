# Roadmap

Iterative. Each phase is a set of small, independently-mergeable PRs.

## Phase 0 — Planning (current)

- [x] Repo exists
- [ ] Planning docs (this PR)
- [ ] Decision log seed (ADR-0001 through 0004)
- [ ] License file

## Phase 1 — Skeleton

- [ ] Go module init, `go.mod`, `Taskfile.yml`
- [ ] Directory layout (`cmd/`, `internal/`, `api/`, `web/`)
- [ ] Minimal `nf version` command
- [ ] Minimal `nfd` that starts an HTTP server and serves `/healthz`
- [ ] Basic `html/template` + HTMX hello-world at `/`
- [ ] CI: build + vet + test

## Phase 2 — Storage & config

- [ ] SQLite integration (`modernc.org/sqlite`)
- [ ] Migrations runner
- [ ] Config loader (global YAML + per-repo overlay)
- [ ] `nf config get/set/path`

## Phase 3 — Family members

- [ ] Family YAML schema (JSON Schema)
- [ ] `nf family list/show/validate/add/remove`
- [ ] Seed default roster on first daemon start
- [ ] Web UI: `/family` page

## Phase 4 — OpenAPI & API v1

- [ ] `api/openapi.yaml` draft
- [ ] Handlers for `/family`, `/duties`, `/runs` (read-only first)
- [ ] Request validation via `kin-openapi`
- [ ] `nf` CLI talks to `nfd` for real

## Phase 5 — Provider & runner

- [ ] Claude Code adapter (`claude --print`)
- [ ] Session-status probe
- [ ] Worktree manager (`git worktree add/remove`)
- [ ] First end-to-end: `nf run --member jerry --duty typo-fix`

## Phase 6 — Git orchestration

- [ ] Branch naming
- [ ] Commit synthesis (conventional commits)
- [ ] Push + `gh pr create`
- [ ] Reviewer tagging
- [ ] PR body templating

## Phase 7 — Scheduler & budget

- [ ] Window loop (22:00 → 05:00)
- [ ] Budget snapshotting
- [ ] Planner: pick duties that fit budget
- [ ] First full autonomous night

## Phase 8 — Built-in duties

- [ ] `lint-fix`, `typo-fix` (Jerry)
- [ ] `docs-drift`, `release-notes`, `readme-refresh` (Morty)
- [ ] `test-coverage-gap` (Summer)
- [ ] `vuln-scan`, `arch-review` (Rick)
- [ ] `dead-code`, `refactor-hotspots` (Beth)
- [ ] `todo-triage`

## Phase 9 — Web UI polish

- [ ] Dashboard with live run status (HTMX SSE)
- [ ] Run detail + log viewer
- [ ] Family member editor (textarea + validate)
- [ ] Night history

## Phase 10 — Hardening

- [ ] `main`-protection preflight
- [ ] Blast-radius caps
- [ ] Systemd unit + launchd plist
- [ ] Prometheus metrics
- [ ] Backup/restore of SQLite DB

## Stretch

- Codex / Copilot provider adapters
- Plugin system for custom duties (WASM? Go plugins?)
- Team mode (shared daemon + per-user quotas)
- Cross-repo duties (monorepo-of-monorepos awareness)
- Slack/Discord morning digest
