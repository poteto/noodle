---
name: testing
description: >
  Test-driven development workflow for Noodle. Use when testing changes, writing
  tests, fixing bugs, creating fixtures, or running the test suite. Triggers:
  "test", "write a test", "fix this bug", "add a fixture", "run tests", "TDD",
  or any task involving verification of code changes.
---

# Testing

Test-driven by default. Prove the problem before solving it.

## Core Workflow: Bug First

1. **Reproduce** — Run the failing scenario and observe the actual behavior
2. **Write a failing test** — Minimal reproduction as a test case
3. **Commit the failing test** — Use `Skill(commit)` with message like `test: reproduce #issue-description`
4. **Fix the code** — Only now attempt the fix
5. **Verify** — Run `pnpm test` and confirm the test passes
6. **Commit the fix** — Separate commit from the test

This applies to all bug fixes. The failing test proves the bug exists and prevents regressions.

## Fixture Tests

Noodle uses directory-based fixtures in `testdata/`. Read [references/fixtures.md](references/fixtures.md) for full format, helper API, and code patterns.

### Creating a New Fixture

```sh
./scripts/scaffold-fixture.sh <package-dir> <fixture-name> [state-count]
```

Fill in the generated `input.json`/`input.ndjson` and `expected.md`, then:

```sh
pnpm fixtures:hash:sync
```

### Bug Fixtures

To add a fixture for a known bug that isn't fixed yet:

1. Scaffold the fixture with the script above
2. Set `expected_failure: true` and `bug: true` in `expected.md` frontmatter
3. Add `## Expected Error` section with `{"any": true}`
4. Verify: `pnpm bugs` lists it
5. Commit: `test: add bug fixture for <description>`

When the bug is later fixed:
1. Record the correct expected output (`pnpm fixtures:loop:record` for loop, or manually update `expected.md`)
2. Set `expected_failure: false` and `bug: false`
3. Sync hashes: `pnpm fixtures:hash:sync`

## Key Commands

| Command | Purpose |
|---------|---------|
| `pnpm test` | Run all tests |
| `pnpm test:short` | Skip integration tests |
| `pnpm check` | Full local CI (test + vet + lint + fixtures) |
| `pnpm bugs` | List fixtures marked as known bugs |
| `pnpm fixtures:loop` | Verify loop fixtures match expected |
| `pnpm fixtures:loop:record` | Regenerate loop fixture expected output |
| `pnpm fixtures:hash` | Verify fixture input hashes |
| `pnpm fixtures:hash:sync` | Update fixture input hashes |
| `pnpm vet` | Run `go vet` |
| `pnpm lint:arch` | Architecture lint (file sizes, legacy patterns) |

## Running Specific Tests

```sh
# Single package
go test ./loop

# Single test
go test ./loop -run TestLoopDirectoryFixtures

# Bypass cache
go test -count=1 ./parse

# With race detector
go test -race ./...

# Verbose
go test -v ./tui
```

## Integration Tests

Long-running or external-dependency tests use `testing.Short()`:

```go
if testing.Short() {
    t.Skip("skipping integration test in short mode")
}
```

`pnpm test:short` skips these. `pnpm test` runs everything.
