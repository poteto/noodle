Back to [[archive/plans/15-bootstrap-onboarding/overview]]

# Phase 7: Integration Test

## Goal

Prove the full onboarding flow works end-to-end: `noodle start` in a fresh directory scaffolds structure, config validation reports actionable diagnostics, and the system is ready for the agent to configure.

## Changes

- **`startup/firstrun_test.go`** (new, or extend existing) — unit test for `EnsureProjectStructure`:
  1. Create a temp directory
  2. Run `EnsureProjectStructure`
  3. Assert: `brain/`, `.noodle/`, `.noodle.toml` all created with expected content
  4. Run `config.Load` + `config.Validate` on the generated config. Stub environment to ensure tmux is on PATH (e.g., prepend a temp dir with a fake `tmux` binary to `$PATH`) so the test is deterministic regardless of host.
  5. Assert: no fatal diagnostics and no repairable diagnostics (scaffolded config has no adapter entries, so no adapter diagnostics; tmux is stubbed, so no runtime.tmux fatal)
  6. Deliberately delete a required directory, re-run `EnsureProjectStructure`
  7. Assert: directory recreated (idempotency)

- **CLI integration test** (new) — test the real command path, not just the library function:
  1. Build the `noodle` binary to a temp path
  2. Run `noodle start --once` in a fresh temp directory (no existing config, no brain, no skills)
  3. Assert: `PersistentPreRunE` in `root.go` triggers scaffolding before `config.Load`
  4. Deterministic exit expectations:
     - **tmux on PATH**: scaffolded files exist, config parses correctly. Exit may be non-zero because a fresh project has no skills installed (the agent installs them after scaffolding). If non-zero, assert the failure reason is the expected missing-skill error, not a config or scaffolding problem.
     - **tmux not on PATH**: exit non-zero with `runtime.tmux` fatal diagnostic message in stderr (current output format is severity/field/message, not diagnostic codes)
     Test both paths explicitly.
  5. Run again — assert idempotent (no files recreated, same output, same exit behavior)
  This tests the boundary path where `PersistentPreRunE` gates the `start` command, not just the library function in isolation.

- **`generate/skill_noodle_test.go`** — verify the snapshot test from Phase 4 is part of CI. If the noodle skill is out of date, CI fails.

- **CI workflow** (`.github/workflows/test.yml` or equivalent) — ensure the unit test, CLI integration test, and skill snapshot test all run on every PR.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Test writing against a clear spec |

## Verification

### Static
- `go test ./...` passes, including the new integration test
- CI runs the skill snapshot test

### Runtime
- Run the integration test locally — passes
- Change a config default, push — CI fails on skill snapshot
- Run `go generate`, push — CI passes
