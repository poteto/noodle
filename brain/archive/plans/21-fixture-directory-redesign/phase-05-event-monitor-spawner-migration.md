Back to [[archive/plans/21-fixture-directory-redesign/overview]]

# Phase 5 — Event Monitor Spawner Migration

## Goal
Unify remaining non-loop fixture suites on the same directory-fixture contract.

## Implementation Notes
- Execute this phase in the shared plan worktree (single-worktree model for this plan).
- Keep commits small and logical (minimum one commit for this phase).
- Before completion, validate this phase against both the phase doc and plan overview.

## Changes
- Migrate `event`, `monitor`, and `spawner` fixture suites to shared loader utilities.
- Delete duplicated per-package markdown section scanners once migrated.
- Keep package-specific assertions intact while using shared fixture IO boundaries.

## Data Structures
- `FixtureSessionEvents` — event fixture input by session.
- `FixtureExpectedClaims` — monitor expected claim struct.
- `FixtureExpectedCommand` — spawner command contains/omits contract.

## Verification
### Static
- `go test ./event ./monitor ./spawner` passes.
- Duplication in fixture parsing utilities is removed or reduced to package-specific decoding only.
- Migration parity check shows all previously existing fixtures in these packages are migrated and represented.

### Runtime
- Run one fixture per package and verify identical assertion outcomes before/after migration.
