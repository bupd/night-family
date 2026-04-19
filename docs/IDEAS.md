# Ideas

A scratchpad. Things we might want eventually. Not commitments.

## Agent / persona

- **Evil Morty**: the contrarian reviewer. Reads PRs opened by other family
  members during the night and posts critical comments. Helps catch
  hallucinated fixes before the human reviewer.
- **Unity**: consensus mode. Two family members work on the same duty; we
  take the intersection of their suggested changes.
- **Noob Noob**: the "explain this PR to a junior" persona. Adds an
  educational comment to every PR so the human reviewer understands context
  fast in the morning.

## Duties

- **`flaky-quarantine`** — go beyond detect; actually quarantine via a test
  tag and open a PR.
- **`budget-autotune`** — observes how often the nightly budget is exhausted
  or left on the table, recommends config changes.
- **`prompt-self-improve`** — family member reviews its own past runs, notes
  failure modes, proposes an edit to its own system_prompt. Opens a PR
  against its own YAML for the human to approve.
- **`doc-linkcheck`** — dead internal/external links in docs.
- **`spec-conformance`** — given an OpenAPI spec in the repo, finds handler
  drift (hat-tip to harbor nightshift).
- **`cross-repo`** — point night-family at a GitHub org; each repo gets a
  rotation slot per night.

## Scheduler

- **Energy-aware**: skip if laptop is on battery < 50%.
- **Session-aware cascading**: if the first run used less budget than
  expected, opportunistically add a bonus duty instead of sleeping.
- **Morning preview**: at 05:00 send a local notification summarising
  tonight's output.

## UI

- A "wall of PRs" page showing all night-family PRs across all repos.
- Timeline visualisation of a night (gantt-style).
- Per-member "health" score: % of PRs merged vs closed unmerged, as a
  feedback signal.

## Integrations

- **Linear / Jira**: auto-file issues instead of (or in addition to) GitHub
  issues.
- **Slack morning digest**.
- **Git providers other than GitHub**: GitLab, Gitea, Codeberg.
- **IDE extension**: VS Code panel that tails `/runs/current/logs` live.

## Safety

- **Rollback helper**: `nf revert <pr-id>` to auto-open a revert PR for any
  night-family PR, in case something slipped through.
- **"Don't touch" markers**: files/directories can be tagged in a
  `.night-family/NOTOUCH` file; agents are told about them in the system
  prompt.
- **Dry-run night**: produce the plan + summaries without actually opening
  PRs. Useful for new users evaluating the tool.

## Cost / budget

- **Multi-provider fallback**: if Claude budget is exhausted, fall back to
  Codex/Copilot for low-risk duties.
- **Cost accounting report**: per-duty average cost over time, so users can
  see which duties are expensive and which are cheap.

## Developer experience

- `nf doctor`: diagnostics (provider reachable? git auth ok? gh auth ok?
  branch protection on main? write perms to repo?).
- `nf new-member`: interactive scaffold.
- Testable agents: record/replay mode so we can unit-test prompts against
  fixed fixtures.
