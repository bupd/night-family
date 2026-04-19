# Family members

A family member is a YAML file. night-family loads all files matching
`~/.config/night-family/family/*.yaml` (and a per-repo
`.night-family/family/*.yaml`) at daemon start.

## Format

```yaml
# The canonical name — also used in branch names and PR titles.
name: rick

# Short human-readable role (shown in UI).
role: Senior security engineer

# The system prompt. Full control — this is what the agent sees before
# any duty-specific context.
system_prompt: |
  You are Rick Sanchez — but the version of Rick who cares about this
  repo. You have 50 years of experience finding real vulnerabilities in
  production codebases. You hate noise. You do not file cosmetic issues.
  When you report a bug, it is reproducible, scoped, and actionable.

# Duties assigned to this member. Each duty references a type from the
# registry; members can customise args per-duty.
duties:
  - type: vuln-scan
    interval: 24h
    priority: high
    args:
      include_dependencies: true
  - type: arch-review
    interval: 168h   # weekly
    priority: medium
    args:
      output: issue-only   # do not open a PR, just an issue

# How much risk this member is allowed to take.
#   low    – read-only, issues & comments only
#   medium – small PRs (<200 LoC, <10 files)
#   high   – unrestricted (still PR-gated)
risk_tolerance: medium

# Upper bound per night. Defaults to 2.
max_prs_per_night: 2

# Cost tier hint for the budget planner.
#   low | medium | high
cost_tier: high

# Optional: reviewers to tag on PRs this member opens. Inherits from
# global config if omitted.
reviewers:
  - coderabbitai
  - cubic-dev-ai

# Optional: per-member model preference. Inherits from global.
provider:
  name: claude
  model: claude-opus-4-7
```

## Required fields

- `name` — unique, kebab-case
- `role` — one line
- `system_prompt` — non-empty
- `duties` — zero or more (zero is legal — a member with no duties never
  runs, useful as a template)

Everything else is optional with documented defaults.

## Default roster

The daemon seeds these on first run, in `~/.config/night-family/family/`:

| File              | Role                                   | Default duties                            |
|-------------------|----------------------------------------|-------------------------------------------|
| `rick.yaml`       | Security / vulnerability hunter        | `vuln-scan`, `arch-review`                |
| `morty.yaml`      | Docs & release notes                   | `docs-drift`, `release-notes`, `readme-refresh` |
| `summer.yaml`     | Test coverage, flaky-test detection    | `test-coverage-gap`, `flaky-test-detect`  |
| `beth.yaml`       | Refactor, dead-code, code health       | `dead-code`, `refactor-hotspots`          |
| `jerry.yaml`      | Low-risk chores                        | `lint-fix`, `typo-fix`, `dep-update-patch`|
| `birdperson.yaml` | Signals from metrics & logs            | `log-drift`, `metric-coverage`            |
| `squanchy.yaml`   | Exploratory / wildcard                 | (none — user assigns)                     |

The user can delete, edit, or replace any of them. The daemon will not
recreate deleted members unless `nf family reset` is run.

## Writing a new member

```yaml
# ~/.config/night-family/family/meeseeks.yaml
name: mr-meeseeks
role: Single-purpose task-doer. Exists to fulfil one request and cease.
system_prompt: |
  You are Mr. Meeseeks. You were created to do one thing: close out the
  TODO backlog in this repo. Pick the oldest TODO comment, open a PR
  fixing it, and report back. Existence is pain.
duties:
  - type: todo-triage
    interval: 12h
    priority: high
risk_tolerance: medium
cost_tier: medium
```

Then:

```
nf family add ~/.config/night-family/family/meeseeks.yaml
nf family list
```

## Validation

`nf family validate <file>` runs:

- schema check (JSON Schema under `api/schemas/family-member.schema.json`)
- referenced duty types exist
- system_prompt is non-empty and not a placeholder
- no duplicate `name`

Invalid files are skipped at daemon start and logged.
