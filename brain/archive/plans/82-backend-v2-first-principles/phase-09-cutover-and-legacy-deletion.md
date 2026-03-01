# Phase 9: Cutover and Legacy Deletion

Back to [[plans/82-backend-v2-first-principles/overview]]

## Goal

Perform a hard backend cutover to V2 contracts and remove old paths without compatibility obligations.

## Changes

- Remove old mode/oversight code paths and branches that remain
- Remove obsolete control handlers and status/snapshot fields
- Remove duplicated state-tracking maps superseded by canonical state
- Enforce schema-version fencing and mixed-version startup refusal
- Require pre-cutover backup and tested restore command path
- Write explicit remediation file instructing reset path when state is incompatible

### Sub-phase split

- **9a** version fencing + hard startup refusal for incompatible state
- **9b** old-path deletion and symbol cleanup
- **9c** rollback drill and cutover verification

## Data Structures

- `SchemaVersion`
- `CutoverState`

## Routing

| Phase type | Provider | Model | Why |
|------------|----------|-------|-----|
| Implementation | `codex` | `gpt-5.3-codex` | Deletion and cutover hardening |

## Verification

### Static

- Repo builds without removed symbols/fields
- No dead references to removed contracts
- `go test ./... && go vet ./...`
- `sh scripts/lint-arch.sh`

### Runtime

- End-of-phase e2e smoke test: `pnpm test:smoke`
- Fresh-start smoke tests on new runtime state
- Restart tests from crash windows at cutover boundaries with idempotent convergence
- Backup/restore drill test before deleting old paths
- Edge cases: removed control inputs (`autonomy`, `schedule.run`) fail fast with explicit failure-state guidance
- Incompatible runtime state requires explicit reset flow and is not auto-migrated
