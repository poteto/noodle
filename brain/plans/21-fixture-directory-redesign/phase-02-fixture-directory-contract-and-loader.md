Back to [[plans/21-fixture-directory-redesign/overview]]

# Phase 2 — Fixture Directory Contract and Loader

## Goal
Implement a shared directory-fixture loader that replaces `*.fixture.md` discovery.

## Implementation Notes
- Execute this phase in the shared plan worktree (single-worktree model for this plan).
- Keep commits small and logical (minimum one commit for this phase).
- Before completion, validate this phase against both the phase doc and plan overview.

## Changes
- Design and implement a clean-slate directory fixture utility (do not preserve old markdown-section parsing architecture).
- Add a shared directory walker in fixture test utilities that discovers fixture directories deterministically.
- Load required fixture artifacts (`expected.md`, ordered `state-*` directories, optional fixture-level `noodle.toml`).
- Normalize cross-platform paths and stable ordering for deterministic CI behavior.
- Keep current assertion helper call sites stable by returning typed fixture payloads from the new loader.
- Define a migration mapping artifact (or test table) that tracks old fixture cases to new directory fixture paths.

## Data Structures
- `FixtureCase` — fixture name, absolute paths, loaded metadata, loaded states.
- `FixtureState` — state id, files, optional config override.
- `FixtureInventory` — sorted fixture case list for a package.

## Verification
### Static
- Loader tests cover ordering, missing required files, and malformed state directory names.
- `go test ./internal/testutil/...` passes.
- Mapping coverage check proves all pre-existing fixtures in the migrated package set are accounted for.

### Runtime
- Run at least one package fixture suite using the new loader while old content remains unchanged.
