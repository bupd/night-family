# Status

A periodic snapshot of what's shipped vs. what's still to come.
Updated alongside the roadmap.

*Last updated: 2026-04-20*

## Working today

You can, on `main`, run:

```
nfd \
  --db ~/.local/share/night-family/nf.db \
  --repo /path/to/target \
  --provider claude \
  --claude-args "--dangerously-skip-permissions" \
  --reviewers coderabbitai,cubic-dev-ai \
  --auto-trigger
```

and at **22:00 local time**, night-family will:

1. Build a plan from the default seven-member family + 18-duty catalogue.
2. Dispatch each slot through the Claude CLI inside `/path/to/target`.
3. For each successful run, open a branch, commit a note (and any file
   edits Claude made), push, and open a PR with `@coderabbitai` and
   `@cubic-dev-ai` tagged in the body.
4. Persist every step to SQLite so the morning dashboard
   (`http://127.0.0.1:7337`) shows what happened.

## Not yet working

- Real branch-protection preflight (we trust the operator).
- Worktree-per-run isolation (concurrent nights would fight).
- Budget snapshots / session remaining-tokens probe (the planner caps
  at an explicit `--budget`, which is good enough for v1).
- Per-duty Go implementations (everything is prompt-only via Claude).
- SIGHUP reload.
- Linear/Jira/Slack integrations.

## Demo without a real provider

The mock provider is zero side-effect and useful for trying the
surface without burning tokens:

```
nfd --db :memory:    # default provider=mock
curl -sX POST localhost:7337/api/v1/nights/trigger -H 'content-type: application/json' -d '{}'
# → 12 runs dispatched, 2 skipped at plan time (cap)
open http://127.0.0.1:7337
```
