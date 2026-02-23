Back to [[plans/15-bootstrap-onboarding/overview]]

# Phase 7: Integration Test

## Goal

Prove the full onboarding flow works end-to-end: `noodle start` in a fresh directory scaffolds structure, config validation reports actionable diagnostics, and the system is ready for the agent to configure.

## Changes

- **`startup/firstrun_test.go`** (new, or extend existing) — integration test:
  1. Create a temp directory
  2. Run `EnsureProjectStructure` (from Phase 3)
  3. Assert: `brain/`, `.noodle/`, `.noodle.toml` all created with expected content
  4. Run `config.Load` + `config.Validate` on the generated config
  5. Assert: no fatal diagnostics (repairable diagnostics for missing adapters/skills are expected and fine)
  6. Deliberately delete a required directory, re-run `EnsureProjectStructure`
  7. Assert: directory recreated (idempotency)

- **`generate/skill_noodle_test.go`** — verify the snapshot test from Phase 4 is part of CI. If the noodle skill is out of date, CI fails.

- **CI workflow** (`.github/workflows/test.yml` or equivalent) — ensure both the integration test and the skill snapshot test run on every PR.

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
