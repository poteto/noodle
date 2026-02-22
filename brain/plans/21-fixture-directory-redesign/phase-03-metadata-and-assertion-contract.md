Back to [[plans/21-fixture-directory-redesign/overview]]

# Phase 3 — Metadata and Assertion Contract

## Goal
Move all expectation metadata and assertion semantics to `expected.md` and remove filename-based behavior.

## Implementation Notes
- Execute this phase in the shared plan worktree (single-worktree model for this plan).
- Keep commits small and logical (minimum one commit for this phase).
- Before completion, validate this phase against both the phase doc and plan overview.

## Changes
- Define required frontmatter keys in `expected.md` (`expected_failure`, `bug`, `schema_version`).
- Keep explicit expected-error schema (`contains`, `equals`, `absent`, `any`) in `expected.md`.
- Preserve current rule: expected-failure fixtures execute and pass only if they actually fail.
- Remove filename-prefix fallback (`error-*`) as assertion logic.

## Data Structures
- `FixtureMetadata` — parsed frontmatter contract.
- `ErrorExpectation` — structured expected error semantics.
- `AssertionOutcome` — normalized pass/fail mismatch details used by fixture harnesses.

## Verification
### Static
- Metadata parser tests cover missing keys, invalid booleans, unsupported keys, and schema version mismatch.
- Assertion contract tests prove expected-failure behavior is not skip-based.

### Runtime
- Run a known expected-failure fixture and verify it logs observed mismatch while suite remains green.
