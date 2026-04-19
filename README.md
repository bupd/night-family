# night-family

> *"Wubba lubba dub dub — we've got work to do."*

**night-family** is a crew of AI agents that clock in after you clock out. Each
one has a role, a set of duties, and a system prompt you control. They run
during a configurable nightly window (default **22:00 → 05:00 local**), chew
through the leftover tokens in your Claude session, and — by morning — hand you
a stack of small, reviewable PRs with `@coderabbitai` and `@cubic-dev-ai`
already tagged for review.

Inspired by Rick & Morty. Not affiliated.

---

## Why

You have a codebase. You have a long list of never-quite-urgent chores:

- docs drift further from reality every week
- a release-notes draft that never gets started
- test coverage that slowly rots
- dependency bumps that pile up
- the vague suspicion there's a security issue you'd find if you looked

You also have a Claude subscription that resets every 5 hours whether you use
it or not. night-family is the answer to *"why aren't those tokens working
on my codebase while I sleep?"*

## What it does (at a glance)

- Wakes at a configured time (default 22:00 local)
- Reads the remaining Claude session budget
- Plans the night: which family members run, which duties, in what order
- For each duty: creates a branch, runs the agent, commits, pushes, opens a
  small PR, tags reviewers (configurable)
- Never commits to `main` — every change is PR-gated
- Stops at window-end or when budget is exhausted
- Stores everything (runs, PRs, budget snapshots) in a local SQLite DB

In the morning you open GitHub and triage.

## Who's in the family

Family members are **just YAML files** — a role, a system prompt, a list of
duties, a risk tolerance. The defaults ship with:

| Member      | Role                                   |
|-------------|----------------------------------------|
| Rick        | Security / vulnerability hunter        |
| Morty       | Docs & release notes                   |
| Summer      | Test coverage & flaky-test detection   |
| Beth        | Refactor, dead-code, code health       |
| Jerry       | Low-risk chores: lint, typos, deps     |
| Birdperson  | Signals from metrics & logs            |
| Squanchy    | Exploratory / wildcard                 |

Want a `Mr. Meeseeks` that exists only to clear one backlog of TODO comments
and then cease to be? Drop a YAML file in `~/.config/night-family/family/`.

## Architecture (30-second version)

```
 ┌─── nf (CLI) ───┐   ┌─── nf-web (HTMX) ──┐
 │  trigger/status│   │  dashboard/config   │
 └────────┬───────┘   └──────────┬──────────┘
          │                      │
          └──────────┬───────────┘
                     ▼
            ┌─────────────────┐
            │  nfd (daemon)   │  ← window scheduler
            │  api.v1 (OAS3.1)│  ← REST surface
            └────────┬────────┘
                     │
    ┌────────────────┼────────────────┐
    ▼                ▼                ▼
 agent runner   budget / session    task registry
 (Claude/Codex)  tracker             (built-in + custom)
    │                                 │
    └──── git orchestrator ───────────┘
              │
              ▼
    branch → commit → push → PR → tag reviewers
```

Full design in [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

## Status

**v1 loop runs end-to-end.** The daemon, CLI, API, HTMX dashboard, mock +
Claude providers, git orchestrator, and window scheduler all work. See
[`docs/STATUS.md`](docs/STATUS.md) for what's shipped vs. pending and
[`docs/ROADMAP.md`](docs/ROADMAP.md) for the phased plan.

## Try it in 30 seconds

```
go install github.com/bupd/night-family/cmd/nfd@latest
go install github.com/bupd/night-family/cmd/nf@latest

# Run against the ephemeral in-memory DB, mock provider, no git side-effects.
nfd --db :memory: &
nf night trigger        # fires a full plan through the mock
nf run list             # 12 runs, all succeeded
open http://127.0.0.1:7337
```

For the real deal — a nightly window that opens PRs via Claude — see the
recipe in [`docs/STATUS.md`](docs/STATUS.md).

## Docs

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — system design
- [`docs/AGENTS.md`](docs/AGENTS.md) — family members & persona format
- [`docs/DUTIES.md`](docs/DUTIES.md) — built-in duty catalog
- [`docs/API.md`](docs/API.md) — REST API (OpenAPI 3.1)
- [`docs/ROADMAP.md`](docs/ROADMAP.md) — phases & milestones
- [`docs/IDEAS.md`](docs/IDEAS.md) — scratchpad of future ideas
- [`docs/DECISIONS.md`](docs/DECISIONS.md) — ADRs as we go
- [`docs/STATUS.md`](docs/STATUS.md) — what works today

## License

Apache-2.0 — see [`LICENSE`](LICENSE).
