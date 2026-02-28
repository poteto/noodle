# Phase 9: Cutover and Legacy Deletion

Back to [[plans/82-backend-v2-first-principles/overview]]

## Goal

Perform an atomic backend cutover to V2 contracts and remove legacy paths to prevent long-lived dual behavior.

## Changes

- Remove legacy mode/oversight code paths and compatibility branches
- Remove obsolete control handlers and status/snapshot fields
- Remove duplicated state-tracking maps superseded by canonical state
- Enforce startup validation for incompatible old runtime files

## Data Structures

- None new; subtraction/deletion phase centered on canonical model adoption

## Routing

| Phase type | Provider | Model | Why |
|------------|----------|-------|-----|
| Implementation | `codex` | `gpt-5.3-codex` | Large mechanical deletion and caller migration cleanup |

## Verification

### Static

- Repo builds without removed symbols/fields
- No dead references to legacy contracts
- `go test ./... && go vet ./...`
- `sh scripts/lint-arch.sh`

### Runtime

- Fresh-start smoke tests on new runtime state
- Restart tests from crash-recovery snapshots created by V2 paths
- Edge cases: old state file detection and explicit failure guidance
