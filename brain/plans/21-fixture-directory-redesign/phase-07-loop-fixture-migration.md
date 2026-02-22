Back to [[plans/21-fixture-directory-redesign/overview]]

# Phase 7 — Loop Fixture Migration

## Goal
Migrate loop fixtures to the new directory/state model and preserve high-signal regression coverage.

## Implementation Notes
- Execute this phase in the shared plan worktree (single-worktree model for this plan).
- Keep commits small and logical (minimum one commit for this phase).
- Before completion, validate this phase against both the phase doc and plan overview.

## Changes
- Convert loop `*.fixture.md` cases into directory fixtures with explicit states and `expected.md`.
- Keep and migrate known bug fixtures for routing-default override and missing-planning scheduling behavior.
- Preserve existing assertions for actions, counts, routing, transitions, and idempotence.
- Add at least one additional multi-state regression fixture derived from real runtime state transitions.

## Data Structures
- `LoopFixtureCase` — loop-specific fixture payload assembled from shared loader + loop decoding.
- `LoopAssertionSet` — typed expected values for loop behavior dimensions.
- `LoopExecutionTrace` — captured per-cycle observed values used for assertions.

## Verification
### Static
- `go test ./loop` passes with migrated fixture suite.
- Expected-failure loop fixtures execute and pass only when mismatches are observed.
- Migration parity check shows all previously existing loop fixture scenarios are migrated and represented.

### Runtime
- Run targeted loop fixture tests in verbose mode and verify observed mismatch logs for expected-failure bug fixtures.
