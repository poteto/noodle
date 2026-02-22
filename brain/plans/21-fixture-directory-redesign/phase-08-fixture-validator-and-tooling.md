Back to [[plans/21-fixture-directory-redesign/overview]]

# Phase 8 — Fixture Validator and Tooling

## Goal
Add minimal tooling that keeps fixture quality high and reduces duplication churn.

## Implementation Notes
- Execute this phase in the shared plan worktree (single-worktree model for this plan).
- Keep commits small and logical (minimum one commit for this phase).
- Before completion, validate this phase against both the phase doc and plan overview.

## Changes
- Add a shared fixture validator test/helper used by all fixture packages.
- Add small helper utilities for common fixture reads to reduce repeated boilerplate.
- Add a lightweight fixture scaffold helper (script or helper function) for creating new directory fixtures consistently.
- Keep tooling minimal and local; no heavy CLI surface expansion.

## Data Structures
- `FixtureValidationIssue` — path + severity + message for contract violations.
- `FixtureReadHelpers` — shared helpers for common expected/input parsing patterns.

## Verification
### Static
- Validator catches malformed fixtures and passes all valid fixtures.
- Repeated fixture parsing code shrinks across package tests.

### Runtime
- Create one new fixture via the scaffold helper and run its package test suite successfully.
- Run one negative-path validation with a deliberately malformed fixture directory and assert the expected `FixtureValidationIssue`.
