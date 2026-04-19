# Duties

A duty is a typed unit of nightly work. The duty type is a string; the
registry maps type → implementation. Members reference duties by type in
their YAML.

## Built-in duty catalog (v1 target)

| Type                   | Output     | Cost | Risk | Notes                                                 |
|------------------------|------------|------|------|-------------------------------------------------------|
| `docs-drift`           | PR         | med  | low  | Finds stale docs referring to renamed/removed code    |
| `release-notes`        | PR         | low  | low  | Drafts release notes from git log since last tag      |
| `readme-refresh`       | PR         | low  | low  | Updates READMEs where code/behaviour has changed      |
| `changelog-groom`      | PR         | low  | low  | Normalises CHANGELOG entries                          |
| `test-coverage-gap`    | PR         | high | med  | Adds unit tests for low-coverage exported functions   |
| `flaky-test-detect`    | Issue      | med  | low  | Analyses CI history, flags flakes                     |
| `vuln-scan`            | Issue      | med  | low  | Dep audit + code-pattern scan                         |
| `dead-code`            | PR         | med  | med  | Removes unused exports (PR requires human merge)      |
| `refactor-hotspots`    | Issue      | high | low  | Identifies high-churn, high-complexity files          |
| `lint-fix`             | PR         | low  | low  | Runs formatter + linter auto-fixes                    |
| `typo-fix`             | PR         | low  | low  | `typos`/`codespell` auto-fixes                        |
| `dep-update-patch`     | PR         | low  | low  | Patch-level bumps only                                |
| `dep-update-minor`     | PR         | med  | med  | Minor bumps, one dep per PR                           |
| `todo-triage`          | Issue + PR | med  | low  | Converts TODO comments to tracked issues              |
| `arch-review`          | Issue      | high | low  | Long-form architectural commentary                    |
| `log-drift`            | Issue      | low  | low  | Finds log lines referencing removed/renamed fields    |
| `metric-coverage`      | Issue      | low  | low  | Flags endpoints without metrics/traces                |
| `ci-signal-noise`      | Issue      | med  | low  | Spots noisy CI jobs / flaky notifications             |

## Duty contract

Every duty implementation is a Go type that implements:

```go
type Duty interface {
    Type() string
    Describe() DutyInfo
    Plan(ctx context.Context, env Env) (*Plan, error)
    Run(ctx context.Context, env Env, plan *Plan) (*Result, error)
}
```

- `Describe` returns static metadata (cost tier default, risk, whether it
  emits a PR or an issue, etc.).
- `Plan` is cheap: it inspects the repo and says "yes I have work, here's the
  estimated scope." The scheduler uses this to decide whether to dispatch.
- `Run` is the heavy call — it's what invokes the agent and produces output.

## Writing a custom duty

Two paths:

### 1. "Prompt-only" duty (no Go code)

Drop a file in `~/.config/night-family/duties/<type>.yaml`:

```yaml
type: harbor-patch-health
name: Patch Stack Health Check
output: issue-only
cost_tier: medium
risk: low
prompt: |
  Validate that all patches in 8gcr-ee/patches/ apply cleanly against the
  current main branch. Report any patches that fail. Do NOT commit anything.
```

The daemon picks it up and runs it through the generic prompt-runner duty
handler.

### 2. "Go plugin" duty

For duties that need custom pre/post logic (e.g. run coverage tool, parse
output, synthesise prompt), implement the `Duty` interface and register it
via `duty.Register`. Plugin distribution TBD — see ADR-0004.

## Output types

- **PR** — agent produced file changes; orchestrator opens a PR
- **Issue** — agent produced a markdown report; orchestrator opens a GitHub
  issue
- **Issue + PR** — both (e.g. `todo-triage` files issues for each TODO and
  opens a PR that removes the TODO comments)
- **Note** — internal-only; stored in the DB, shown in the UI, no external
  artefact

## Interval semantics

`interval: 24h` means "don't run this duty more than once per 24h per
member." Multiple members can run the same duty type — they each keep their
own clock.

If a duty fails, the interval clock still advances (to avoid hot-looping on
broken duties). Three consecutive failures disable the duty until manually
re-enabled via `nf duty enable`.
