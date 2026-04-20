# Roadmap

Iterative. Each phase is a set of small, independently-mergeable PRs.

The first overnight build session landed **31 PRs** — v1 runs
end-to-end through the mock *and* the real Claude + Codex providers.
Phases marked ✅ are on `main`.

## Phase 0 — Planning ✅

- [x] Repo exists
- [x] Planning docs (PR #2)
- [x] ADR-0001 through 0007
- [x] License file

## Phase 1 — Skeleton ✅

- [x] Go module, `Taskfile.yml`, `cmd/{nf,nfd}`, `internal/`
- [x] `nf version`, `nfd` with `/healthz`
- [x] `html/template` + HTMX landing page
- [x] CI: gofmt + vet + race test + build + smoke

## Phase 2 — Storage & config ✅

- [x] SQLite (`modernc.org/sqlite`) with auto-migration
- [x] `--db` + XDG default path
- [x] YAML config loader (`internal/config`) with `--config` override
      + XDG default; flag > file > built-in precedence
- [x] `/etc/default/nfd` EnvironmentFile pattern in the systemd unit

## Phase 3 — Family members ✅

- [x] JSON Schema + OpenAPI schema
- [x] `nf family` CRUD (list/show/add/replace/remove/validate)
- [x] Seed default seven-member roster
- [x] User YAML overlay via `--family-dir` (+ XDG default)
- [x] **SIGHUP reloads the overlay**
- [x] `/family` page
- [x] `POST /api/v1/family/validate` endpoint

## Phase 4 — OpenAPI & API v1 ✅

- [x] `internal/server/apiassets/openapi.yaml` — full v1 surface
- [x] Every endpoint wired: family, duties, runs, nights, prs,
      schedule, budget, stats, provider, digest
- [x] Swagger UI at `/docs`, JSON spec at `/openapi.json`

## Phase 5 — Provider & runner ✅

- [x] Mock provider (zero side-effect)
- [x] Claude adapter (spawns `claude --print`)
- [x] Codex adapter (spawns `codex`)
- [x] Optional `StatusProber` interface + stub status for real
      providers (Confidence="low")
- [x] Runner lifecycle + persisted Run rows

## Phase 6 — Git orchestration ✅

- [x] `internal/gitops` wraps git + gh
- [x] Branch naming, conventional commit messages
- [x] Reviewer tagging in PR body (defaults to @coderabbitai +
      @cubic-dev-ai per ADR-0006)
- [x] Note-PR fallback when provider produced no file changes

## Phase 7 — Scheduler & budget ✅

- [x] Window loop (22:00 → 05:00 configurable)
- [x] `--auto-trigger` background loop
- [x] Planner with priority ordering + budget ceiling + member cap
- [x] `storage.BudgetSnapshot` at FinishNight
- [x] `GET /api/v1/budget` serving the latest snapshot

## Phase 8 — Built-in duties ⚠️ *(metadata only; prompt-driven exec)*

- [x] 18-entry catalogue (`lint-fix`, `docs-drift`, `vuln-scan`, …)
- [x] Exec via the Claude/Codex adapters using `Info.Description` as
      the duty prompt
- [ ] Concrete per-duty Go implementations for hot paths
- [ ] Custom prompt-only duty loader from `~/.config/night-family/duties/`

## Phase 9 — Web UI ✅

- [x] Dashboard with live HTMX counts + schedule indicator
- [x] `/family`, `/duties`, `/plan`, `/runs`, `/nights`, `/prs`,
      `/digests`, `/digests/{id}`, `/docs`
- [ ] Run-detail page with SSE log streaming
- [ ] Family-member editor (currently: `nf family replace <file.yaml>`)

## Phase 10 — Hardening ⚠️ *(partial)*

- [x] `/metrics` in Prometheus text format
- [x] Slack/Discord webhook (`--slack-webhook` / `NF_SLACK_WEBHOOK`)
- [x] End-to-end test harness (spawns nfd in-process)
- [x] Systemd unit (templated, user-scoped, hardened)
- [ ] `main` branch-protection preflight
- [ ] Blast-radius caps (files-changed, lines-changed)
- [ ] launchd plist for macOS
- [ ] SQLite backup/restore
- [ ] Token auth for non-loopback binds

## Stretch

- Real Claude session probe (parse `~/.claude/` or shell `/status`)
- Worktree-per-run isolation
- Plugin system for custom duties (WASM? Go plugins?)
- Team mode (shared daemon + per-user quotas)
- Cross-repo duties (monorepo rotation)
- Morning digest attachments via Slack `blocks`

## Summary of what's reachable today

```
Endpoints: /healthz /readyz /version /openapi.{yaml,json} /docs /metrics
           /api/v1/{family,duties,schedule,budget,stats,provider,
                    nights,nights/preview,nights/trigger,runs,prs,
                    nights/{id}/digest}
           /ui/dashboard-cards    (HTMX fragment)
           /family /duties /plan /runs /nights /prs /digests /digest

nfd flags: --addr --log-level --db --config --repo --base-branch
           --reviewers --signoff --skip-push --skip-pr
           --auto-trigger --family-dir
           --provider {mock,claude,codex}
           --claude-bin --claude-args --codex-bin --codex-args
           --digest-dir --slack-webhook

nf CLI:    version family duty night run pr digest schedule budget
           stats doctor  (+ --json on every list command)

Signals:   SIGINT/SIGTERM = graceful shutdown
           SIGHUP         = reload family-dir overlay
```
