# Roadmap

Iterative. Each phase is a set of small, independently-mergeable PRs.

Phases marked ✅ have landed on `main`; the first overnight build session
got the whole v1 loop running end-to-end through the mock provider, and
the Claude adapter shipped as a final pre-dawn PR.

## Phase 0 — Planning ✅

- [x] Repo exists
- [x] Planning docs (PR #2)
- [x] Decision log seed (ADR-0001 through 0007)
- [x] License file

## Phase 1 — Skeleton ✅

- [x] Go module init, `go.mod`, `Taskfile.yml`
- [x] Directory layout (`cmd/`, `internal/`, `api/`, `web/`)
- [x] Minimal `nf version` command
- [x] Minimal `nfd` that starts an HTTP server and serves `/healthz`
- [x] Basic `html/template` + HTMX hello-world at `/`
- [x] CI: build + vet + test

## Phase 2 — Storage & config ✅ *(partial)*

- [x] SQLite integration (`modernc.org/sqlite`)
- [x] Migrations runner
- [x] `--db` / XDG default path
- [ ] Full config loader (global YAML)
- [ ] `nf config get/set/path`

## Phase 3 — Family members ✅

- [x] Family YAML schema (in OpenAPI + standalone JSON Schema)
- [x] `nf family list` / `show`
- [x] Seed default roster on first daemon start
- [x] User YAML overlay via `--family-dir` / XDG
- [x] Web UI: `/family` page

## Phase 4 — OpenAPI & API v1 ✅ *(partial)*

- [x] `api/openapi.yaml` complete (committed under `internal/server/apiassets/`)
- [x] Handlers for `/family`, `/duties`, `/runs`, `/nights`, `/prs`,
      `/schedule`, `/stats` (read + the POST surfaces for runs and nights)
- [x] Swagger UI at `/docs`
- [ ] Runtime request validation via `kin-openapi`
- [ ] Mutation handlers for `/family` (POST/PUT/DELETE — store is
      ready, handlers pending)

## Phase 5 — Provider & runner ✅

- [x] Mock provider adapter
- [x] Claude Code adapter (`claude --print`)
- [x] Runner lifecycle: queued → running → terminal, with persistence
- [x] First end-to-end: `nf run start --member jerry --duty lint-fix`
- [ ] Session-status probe (read Claude 5-hour window remaining)
- [ ] Worktree-per-run isolation

## Phase 6 — Git orchestration ✅

- [x] `internal/gitops` wraps git + gh
- [x] Branch naming (`night-family/<member>/<duty>-<shortid>`)
- [x] Conventional-commit message synthesis
- [x] Push + `gh pr create`
- [x] Reviewer tagging in PR body
- [x] Note-PR fallback when provider produces no file changes
- [ ] Branch-protection preflight
- [ ] Blast-radius caps

## Phase 7 — Scheduler & budget ✅ *(partial)*

- [x] Window loop (22:00 → 05:00, configurable)
- [x] Auto-trigger via `--auto-trigger`
- [x] Planner with priority ordering + budget ceiling + member cap
- [x] First full autonomous night (mock-provider, local smoke)
- [ ] Budget snapshotting to `budget_snapshots` table
- [ ] `GET /api/v1/budget` hooked to real data (currently planner-only)

## Phase 8 — Built-in duties ⚠️ *(metadata only)*

- [x] Duty registry + 18-entry catalogue (`lint-fix`, `docs-drift`,
      `release-notes`, `test-coverage-gap`, `vuln-scan`, …)
- [ ] Concrete per-duty Go implementations (prompt-only is live — the
      Claude adapter consumes it)
- [ ] Custom duty loader from `~/.config/night-family/duties/`

## Phase 9 — Web UI polish ✅ *(live)*

- [x] Dashboard with live counts + schedule indicator (HTMX auto-refresh)
- [x] Family / Duties / Plan / Runs / Nights / PRs pages
- [x] API docs page via Swagger UI
- [ ] Run-detail page with streamed logs (SSE)
- [ ] Family-member editor
- [ ] Per-night breakdown page

## Phase 10 — Hardening ⚠️ *(pending)*

- [ ] `main`-protection preflight
- [ ] Blast-radius caps
- [ ] Systemd unit + launchd plist
- [ ] Prometheus metrics
- [ ] Backup/restore of SQLite DB
- [ ] Token auth for non-loopback binds

## Stretch (unchanged from the original roadmap)

- Codex / Copilot provider adapters
- Plugin system for custom duties (WASM? Go plugins?)
- Team mode (shared daemon + per-user quotas)
- Cross-repo duties (monorepo-of-monorepos awareness)
- Slack/Discord morning digest

## What exists today (at a glance)

```
nfd binary:
  HTTP: /healthz /readyz /version /openapi.{yaml,json} /docs
        /api/v1/{family,duties,schedule,nights,nights/preview,
                 nights/trigger,runs,prs,stats}
        /ui/dashboard-cards (HTMX fragment)
        /family /duties /plan /runs /nights /prs /docs
  Flags: --addr --log-level --db --repo --base-branch --reviewers
         --signoff --skip-push --skip-pr --auto-trigger
         --provider {mock,claude} --claude-bin --claude-args
         --family-dir

nf CLI: version / family {list,show} / duty {list,show}
        / night {preview,trigger,list,show}
        / run {list,show,start} / pr {list}
```
