Back to [[archive/plans/21-fixture-directory-redesign/overview]]

# Phase 4 — Parse Adapter Stamp Migration

## Goal
Migrate the parser-style fixture suites to directory fixtures with minimal behavior change.

## Implementation Notes
- Execute this phase in the shared plan worktree (single-worktree model for this plan).
- Keep commits small and logical (minimum one commit for this phase).
- Before completion, validate this phase against both the phase doc and plan overview.

## Changes
- Migrate `parse`, `adapter`, and `stamp` fixture tests to the shared directory loader.
- Convert existing fixtures into directory structure while preserving expected outputs.
- Keep test assertions behaviorally equivalent to current tests.
- Keep one stable fixture naming strategy to minimize file churn.

## Data Structures
- `FixtureInputBlob` — normalized input payload loaded from fixture state files.
- `FixtureExpectedEvents` — canonical expected event list.
- `FixtureExpectedStamped` — stamped-output expected object list.

## Verification
### Static
- `go test ./parse ./adapter ./stamp` passes.
- No direct `*.fixture.md` parsing remains in these packages.
- Global suite stability is not required in this intermediate phase; cross-package breakage is acceptable until Phase 9.
- Migration parity check shows all previously existing fixtures in these packages are migrated and represented.

### Runtime
- Execute one representative fixture from each migrated package and compare outputs to pre-migration baseline.
