# Contributing

Thanks for poking around. night-family is small enough that the
contribution story is simple.

## Environment

You need:

- Go 1.23 or newer.
- `git` and `gh` on PATH (the orchestrator shells out to both).
- A working Claude Code CLI if you want to exercise the real provider.
  The mock provider is enabled by default and needs nothing.

Nothing else. No npm, no make, no containers.

## Build / test

```bash
task build        # build bin/nf and bin/nfd
task test         # go test -race ./...
task check        # fmt-check + vet + test + build (fast CI proxy)
task cover        # coverage report
```

## Code conventions

### Commits

Conventional Commit style (`feat:`, `fix:`, `docs:`, `test:`, …), with
a scope that matches the package you're touching:

```
feat(nfd): serve /api/v1/budget
docs(roadmap): mark phase 7 complete
```

All commits carry a DCO `Signed-off-by` trailer. `git commit -s` is
enough; the CI doesn't enforce it today but it will eventually.

### Small PRs

The project's history is a stream of small PRs on purpose. When you
touch more than one logical thing, split them. Hundreds of
lines-changed is fine; "this PR changes storage *and* the scheduler
*and* also adds a CLI command" isn't.

Exception: if your change genuinely spans the stack (say, an OpenAPI
edit + the matching handler + the matching CLI subcommand + a test),
keep it in one PR but structure the commits so each commit is a
clean logical step.

### Tests

- Anything new under `internal/` should have test coverage for the
  happy path + the obvious error branches.
- Tests use the stdlib `testing` package (+ `testify` is fine if you
  genuinely need it; prefer not).
- Avoid touching the network. `tests/e2e_test.go` spawns nfd as a
  subprocess; that's the only integration path.
- Never write a test that writes to the real `~/.config`, `~/.local`,
  or `~/.claude` directories. Use `t.TempDir()` and XDG overrides.

### Comments

Default to none. Exception: a line that says *why* something is the
way it is when the answer isn't obvious from the code. Don't document
*what* — the code does that.

### New duties

If you're adding a built-in duty type:

1. Append an `Info{}` entry to `internal/duty.Builtins()`.
2. Add the row to `docs/DUTIES.md`.
3. Pair it with a test in `internal/duty/duty_test.go` that ensures
   it's included in the default catalogue and has valid enum values.

### New family members

Default family members are YAML files under
`internal/family/defaults/`. To add one:

1. Drop a YAML file with a unique `name:`.
2. Add a row to the roster table in `docs/AGENTS.md`.
3. Verify `go test ./internal/family/...` covers it (the existing
   `TestLoadDefaultsRosterLoads` auto-picks-up the file).

## API changes

`internal/server/apiassets/openapi.yaml` is the source of truth. Any
API change means editing the spec first, then the handler, then the
tests. The CI doesn't regenerate types yet (the roadmap has it), so
hand-written requests/responses should match the schema exactly.

Breaking changes bump `/api/v1` → `/api/v2`. Additive ones stay on
v1.

## Reviewing

- If your PR opens a PR against the night-family repo itself (e.g.
  from a nightly dogfood run), tag a human in addition to the
  configured bots.
- PRs are **never** merged directly to `main` by automation; the
  reviewer is a human (see ADR-0005).

## Release

Not yet formalised. For now: tag `vX.Y.Z`, write a release note
pointing at the merged PRs for that range, publish the tag. Automated
release binaries are TBD.
