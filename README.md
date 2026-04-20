# night-family

> *"Wubba lubba dub dub — we've got work to do."*

**night-family** is a crew of AI agents that clock in after you clock out.
Each one has a role, a set of duties, and a system prompt you control. They
run during a configurable nightly window (default **22:00 → 05:00 local**),
chew through the leftover tokens in your Claude session, and — by morning —
hand you a stack of small, reviewable PRs with `@coderabbitai` and
`@cubic-dev-ai` already tagged.

Inspired by Rick & Morty. Not affiliated.

---

## Why this exists

You have a codebase. You have a long list of never-quite-urgent chores:

- docs drift further from reality every week
- a release-notes draft that never gets started
- test coverage that slowly rots
- dependency bumps that pile up
- the vague suspicion there's a security issue you'd find if you looked

You also have a Claude (or Codex) subscription that resets every 5 hours
whether you use it or not. night-family is the answer to *"why aren't those
tokens working on my codebase while I sleep?"*

**The bet**: small, persona-scoped, clearly-bounded AI agents — each one
doing one thing — produce better overnight PRs than one unbounded agent
told to "improve the codebase." A repo tended by Jerry's typo-fix bot, a
dedicated security reviewer, and a docs-drift hunter is in a different
place each morning than a repo tended by an LLM on a firehose.

## What it does (30-second version)

```
                ┌──────────────────────────────┐
                │         nf (CLI)             │
                │   nf night trigger, etc.     │
                └──────────────┬───────────────┘
                               │
                ┌──────────────▼───────────────┐      ┌──────────────────┐
                │   nfd  (HTTP:7337, HTMX)     │◀────▶│  SQLite store    │
                │   /api/v1/*, /openapi.yaml   │      └──────────────────┘
                └──────┬───────────────────────┘
                       │                  │
         ┌─────────────▼────┐    ┌────────▼────────┐
         │ Scheduler loop   │    │ Planner         │
         │ 22:00→05:00 auto │    │ (family+duties) │
         └─────────┬────────┘    └────────┬────────┘
                   │                      │
             ┌─────▼───────┐       ┌──────▼──────┐       ┌─────────────────┐
             │   Runner    │──────▶│  Provider   │  ===▶ │  Claude / Codex │
             │             │       │ (interface) │       │   subprocess    │
             └─────┬───────┘       └─────────────┘       └─────────────────┘
                   │
             ┌─────▼────────┐      ┌──────────────────────────────────┐
             │  Git orch.   │═════▶│ branch→commit→push→gh pr create  │
             └─────┬────────┘      │ with @coderabbitai @cubic-dev-ai │
                   │               └──────────────────────────────────┘
             ┌─────▼────────┐
             │  Digest +    │──────▶ on-disk .md + Slack/Discord webhook
             │  Notifier    │
             └──────────────┘
```

At 22:00 local:

1. Read the family YAMLs (embedded defaults + anything in
   `~/.config/night-family/family/`).
2. Build a plan: (member, duty) slots ordered by priority, capped by
   `max_prs_per_night` per member and your total token budget.
3. For each slot: spawn a provider subprocess, capture the result, and
   land it as a branch + commit + PR with reviewers tagged.
4. Write a markdown digest and (optionally) ship it to Slack/Discord.

Every step persists to SQLite; the morning dashboard shows counts, PR
status, last night's digest, and the schedule.

## Who's in the family (defaults)

Family members are **just YAML files** — a role, a system prompt, a duty
list, a risk tolerance. The defaults ship with:

| Member      | Role                                         |
|-------------|----------------------------------------------|
| Rick        | Security / vulnerability hunter              |
| Morty       | Docs, release notes, changelog               |
| Summer      | Test coverage & flaky-test detective         |
| Beth        | Refactor, dead-code, code-health steward     |
| Jerry       | Low-risk chores: lint, typos, dep bumps      |
| Birdperson  | Logs, metrics, observability drift           |
| Squanchy    | Exploratory / wildcard                       |

Want a `Mr. Meeseeks` that exists only to clear the TODO backlog and then
cease? Drop a YAML file in `~/.config/night-family/family/`, send the
daemon a SIGHUP, and you're live. Full format in
[`docs/AGENTS.md`](docs/AGENTS.md).

---

## Install

### Quick: dev install (try it, zero commitment)

```bash
go install github.com/bupd/night-family/cmd/nfd@latest
go install github.com/bupd/night-family/cmd/nf@latest

# Start the daemon with the mock provider (no tokens burned, no git touched).
nfd --db :memory: &

# Trigger a full night's plan through the mock.
nf night trigger
nf run list                 # 12 succeeded runs
open http://127.0.0.1:7337  # the dashboard
```

Stop with `Ctrl-C` or `kill %1`. Nothing is persisted (in-memory DB); nothing
is written to your filesystem; no network calls made.

### From source

```bash
git clone https://github.com/bupd/night-family
cd night-family
task build                  # or: go build ./cmd/nf ./cmd/nfd
./bin/nfd --help
```

Requirements: **Go 1.23+**, `git`, and `gh` (only when using `--repo`).
The Claude CLI is required if you use `--provider=claude`; Codex CLI if
`--provider=codex`. The Mock provider needs nothing.

### Production: systemd (Linux)

The repo ships a templated systemd unit so multiple users on one box can
each run their own daemon:

```bash
git clone https://github.com/bupd/night-family && cd night-family
task build
sudo install -m 0755 bin/nfd /usr/local/bin/nfd
sudo install -m 0755 bin/nf  /usr/local/bin/nf

# Auth as the user who will run the daemon (NOT root):
gh auth login
claude login     # only if you want --provider=claude

# Install the unit:
sudo cp deploy/systemd/nfd.service /etc/systemd/system/nfd@.service
sudo systemctl daemon-reload
sudo systemctl enable --now nfd@$USER.service
sudo journalctl -u nfd@$USER.service -f
```

The unit reads `/etc/default/nfd` for env-var overrides. Full recipe
(including macOS/launchd TODO and uninstall steps) in
[`docs/INSTALL.md`](docs/INSTALL.md).

---

## Use

### 1. Configure (optional — flags work standalone)

Drop a config file at `~/.config/night-family/config.yaml`:

```yaml
addr: 127.0.0.1:7337
db:   /home/bupd/.local/share/night-family/nf.db

# Provider + how to call it.
provider: claude
claude_args:
  - --dangerously-skip-permissions

# Git orchestrator: enables PR creation. Leave unset to run without git.
repo:           /home/bupd/code/target-project
base_branch:    main
reviewers:
  - coderabbitai
  - cubic-dev-ai
signoff: true

# Scheduler + output.
auto_trigger:   true
family_dir:     /home/bupd/.config/night-family/family
```

CLI flags always override the file. Precedence is **flag > config file >
built-in default**.

### 2. Run the daemon

Foreground:

```bash
nfd
```

Or via systemd (see install section).

### 3. Drive it from the terminal

```bash
# Inspect
nf family list
nf duty list
nf schedule show
nf budget show
nf stats
nf doctor                                   # Liveness probes

# Plan + trigger nights
nf night preview                            # What would run now
nf night preview --budget 20000             # With a token cap
nf night trigger                            # Dispatch now (honours window)
nf night trigger --only-members jerry       # Restrict
nf night trigger --dry-run                  # Plan without dispatch

# Manual single-duty dispatch
nf run start --member jerry --duty lint-fix
nf run list
nf run show <run-id>

# PRs opened by the family
nf pr list

# Morning digest
nf digest show <night-id> > morning.md

# Manage the roster
nf family add     ./my-meeseeks.yaml
nf family replace ./rick.yaml
nf family remove  mr-meeseeks
nf family validate ./rick.yaml
```

All list commands accept `--json`. The daemon URL can be overridden with
`NF_DAEMON_URL` (default `http://127.0.0.1:7337`).

### 4. Or drive it from the browser

- `/` — live dashboard (auto-refreshes every 15s)
- `/family`, `/duties`, `/plan`, `/runs`, `/nights`, `/prs`, `/digests`
- `/docs` — Swagger UI on the OpenAPI 3.1 spec
- `/openapi.yaml` · `/openapi.json` — the raw spec
- `/metrics` — Prometheus text format

### 5. Or from anywhere with curl

Every page has a matching JSON endpoint. A tiny sample:

```bash
curl -sX POST localhost:7337/api/v1/nights/trigger \
     -H 'content-type: application/json' -d '{}'

curl -s localhost:7337/api/v1/runs | jq '.items | length'

# Raw markdown digest — pipe it into email, Slack, etc.
curl -s localhost:7337/api/v1/nights/$ID/digest > morning.md
```

### 6. Add a custom family member

```yaml
# ~/.config/night-family/family/meeseeks.yaml
name: mr-meeseeks
role: Single-purpose task-doer. Exists to fulfil one request and cease.
system_prompt: |
  You are Mr. Meeseeks. You were created to do one thing: close out the
  TODO backlog in this repo. Pick the oldest TODO, open a PR fixing it,
  and report back. Existence is pain.
duties:
  - type: todo-triage
    interval: 12h
    priority: high
risk_tolerance: medium
cost_tier: medium
```

Then:

```bash
kill -HUP $(pidof nfd)    # No restart needed.
nf family list            # mr-meeseeks is live.
```

---

## Safety

- **Never commits to `main`.** Every duty outputs a branch named
  `night-family/<member>/<duty>-<shortid>`.
- **Git orchestrator is opt-in.** Without `--repo` set, nothing git-related
  happens — even successful runs just produce Run records.
- **Providers run in the configured repo dir.** They never see anything
  outside it.
- **Reviewers are tagged in the PR body**, not auto-requested, because
  review-bots typically don't accept formal review requests from
  non-org accounts. A body mention reliably pings them.
- **Body truncation + budget caps + per-member PR caps** keep runaway
  nights bounded. Every rejected slot is recorded in the plan's
  `skipped` list with a reason.
- **SQLite is the only persistent state.** Blow it away to forget.

---

## Docs

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — system design
- [`docs/AGENTS.md`](docs/AGENTS.md) — family-member YAML format
- [`docs/DUTIES.md`](docs/DUTIES.md) — duty catalogue
- [`docs/API.md`](docs/API.md) — REST API (OpenAPI 3.1)
- [`docs/STATUS.md`](docs/STATUS.md) — what works today
- [`docs/INSTALL.md`](docs/INSTALL.md) — dev + systemd install
- [`docs/ROADMAP.md`](docs/ROADMAP.md) — phases + what's next
- [`docs/IDEAS.md`](docs/IDEAS.md) — scratchpad
- [`docs/DECISIONS.md`](docs/DECISIONS.md) — ADRs
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — commit / PR conventions

## License

Apache-2.0 — see [`LICENSE`](LICENSE).
