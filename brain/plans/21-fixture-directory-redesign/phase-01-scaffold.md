Back to [[plans/21-fixture-directory-redesign/overview]]

# Phase 1 — Scaffold

## Goal
Define and freeze the directory fixture contract before package migrations begin.

## Implementation Notes
- Execute this phase in the shared plan worktree (single-worktree model for this plan).
- Keep commits small and logical (minimum one commit for this phase).
- Before completion, validate this phase against both the phase doc and plan overview.

## Changes
- Add a short fixture contract document that defines required files and ordering.
- Create canonical fixture directory naming and state directory naming (`state-01`, `state-02`, ...).
- Define root vs state-level config precedence (`fixture/noodle.toml` as base, `state-XX/noodle.toml` optional override).
- Add a fixture-shape smoke test that fails fast when required files are missing.

## Data Structures
- `FixtureLayout` — required paths and optional paths for one fixture directory.
- `FixtureStateDir` — ordered state identifier + resolved path.
- `FixtureConfigScope` — base config + optional state override semantics.

## Verification
### Static
- Contract tests pass for valid and invalid fixture directory shapes.
- Existing fixture utility unit tests still pass.

### Runtime
- Create one sample fixture directory and run the shape validator end-to-end.
