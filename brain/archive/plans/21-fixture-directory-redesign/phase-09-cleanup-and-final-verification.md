Back to [[archive/plans/21-fixture-directory-redesign/overview]]

# Phase 9 — Cleanup and Final Verification

## Goal
Finalize the redesign by removing legacy fixture paths and validating the full suite.

## Implementation Notes
- Execute this phase in the shared plan worktree (single-worktree model for this plan).
- Keep commits small and logical (minimum one commit for this phase).
- Before completion, validate this phase against both the phase doc and plan overview.

## Changes
- Remove legacy `*.fixture.md` assumptions and dead helper code.
- Delete the old single-file fixture util implementation and any compatibility scaffolding that only exists for legacy markdown fixtures.
- Ensure docs/reference notes describe directory fixture usage and expected metadata semantics.
- Confirm every fixture has consistent frontmatter keys and stable section structure in `expected.md`.
- Close the loop by verifying known bug fixtures remain encoded and high-signal.

## Data Structures
- `LegacyFixtureCompat` (deleted) — remove old markdown-file-only compatibility logic.
- `FixtureSchemaVersion` — stabilized version marker used in all `expected.md` files.

## Verification
### Static
- `go test ./...`
- `go vet ./...`
- Contract check: all fixtures satisfy required metadata and directory layout.
- `rg`-based check confirms no production/test code still imports or depends on the legacy fixture util path.

### Runtime
- Run focused fixture suites for previously reported runtime regressions.
- Validate one end-to-end multi-state fixture from initial state through final transition assertions.
