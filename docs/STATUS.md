# Status

A periodic snapshot of what's shipped vs. what's still to come.
Updated alongside the roadmap.

*Last updated: 2026-04-20 (post overnight build session — 31 PRs
merged since project inception.)*

## Working today

Everything below runs on `main`. The v1 loop is end-to-end.

```
nfd \
  --db ~/.local/share/night-family/nf.db \
  --family-dir ~/.config/night-family/family \
  --repo /path/to/target \
  --provider claude \
  --claude-args "--dangerously-skip-permissions" \
  --reviewers coderabbitai,cubic-dev-ai \
  --slack-webhook "$NF_SLACK_WEBHOOK" \
  --digest-dir ~/.local/share/night-family/digests \
  --auto-trigger
```

At **22:00 local time**, night-family will:

1. Plan the night from the default seven-member roster overlaid with
   any YAMLs in `--family-dir`, and the 18-entry duty catalogue.
2. Dispatch each slot through the Claude CLI inside `/path/to/target`.
   (`--provider codex` also works if Codex is auth'd.)
3. For each successful run, open a branch, commit the note (and any
   file edits Claude made), push, and open a PR with `@coderabbitai`
   and `@cubic-dev-ai` tagged in the body.
4. Persist every step to SQLite.
5. At FinishNight: render the markdown digest, write it to
   `$digest-dir/<date>-<night-id>.md`, and POST it to the Slack
   webhook.
6. Stamp a budget snapshot row.

By morning, your dashboard (`http://127.0.0.1:7337`) shows:

- Live counts (nights, runs with status pills, PRs with state pills,
  schedule window + in-window indicator) auto-refreshing every 15s.
- `/plan` — what the next night would do.
- `/family`, `/duties`, `/runs`, `/nights`, `/prs`, `/digests` —
  browseable.
- `/docs` — Swagger UI on the OpenAPI 3.1 spec.
- `/metrics` — Prometheus format.

SIGHUP reloads the family-dir overlay without restarting.

## Configuration

`nfd` accepts both flags and a YAML file. Precedence: flag > file >
built-in default. See `docs/INSTALL.md` for the full recipe.

`~/.config/night-family/config.yaml`:

```yaml
addr: 127.0.0.1:7337
provider: claude
claude_args:
  - --dangerously-skip-permissions
reviewers:
  - coderabbitai
  - cubic-dev-ai
repo: /home/bupd/code/target
auto_trigger: true
```

## CLI

```
nf version
nf family {list,show,add,replace,remove,validate}
nf duty {list,show}
nf night {preview,trigger,list,show}
nf run {list,show,start}
nf pr list
nf digest show <night-id>
nf schedule show
nf budget show
nf stats
nf doctor
```

`NF_DAEMON_URL` overrides the daemon base URL (default
`http://127.0.0.1:7337`). All list commands support `--json`.

## Not yet working

- **Real Claude session probe.** The `StatusProber` interface exists
  and the Mock implements it with plausible values; Claude + Codex
  return `Confidence="low"` stubs until we wire a real parser.
  (Planner caps via `--budget` are functionally equivalent for v1.)
- **Branch-protection preflight.** We trust the operator; ADR-0005
  says we'll refuse to run against an unprotected `main` eventually.
- **Worktree-per-run isolation.** Concurrent nights would fight. v1
  runs slots sequentially, so this hasn't bitten anyone yet.
- **Deletion-on-disk via SIGHUP.** Removing a YAML from `--family-dir`
  and sending HUP won't remove the member from the store; use
  `nf family remove <name>` or the DELETE endpoint.
- **Run-detail page with SSE log streaming.** The DB schema has a
  `run_logs` table ready to go; handler + UI not yet implemented.
- **launchd plist for macOS.** systemd on Linux is done
  (`deploy/systemd/nfd.service`); macOS port is a straightforward
  follow-up.

## Demo without a real provider

Zero setup, zero tokens:

```
nfd --db :memory: &
curl -sX POST localhost:7337/api/v1/nights/trigger \
     -H 'content-type: application/json' -d '{}'
# → 12 runs dispatched, 3 skipped at plan time (member caps)
nf night list
nf pr list       # empty unless --repo is set
nf digest show <night-id>
open http://127.0.0.1:7337
```

## Inventory

```
 internal/
   config       YAML file loader (~/.config/night-family/config.yaml)
   digest       Markdown renderer for completed nights
   duty         Built-in catalogue (18 types)
   family       Member type + Store + overlay loader
   gitops       git + gh wrapper (branch → commit → push → PR)
   notify       Slack/Discord webhook shipping
   planner      Stateless plan builder (priority + budget caps)
   provider     Mock, Claude, Codex adapters + StatusProber
   runner       Duty lifecycle + NightResult orchestration
   schedule     Windowed schedule (22:00→05:00, IANA TZ)
   scheduler    Background auto-trigger loop
   server       HTTP surface + HTMX UI + embedded assets
   storage      SQLite (nights, runs, prs, budget_snapshots, …)
   ulid         Prefixed ULID generator (run_, night_, pr_)
 cmd/nfd        The daemon
 cmd/nf         The CLI
 tests/         End-to-end harness that spawns nfd
 deploy/systemd Templated nfd@<user>.service
```
