# Phase 4: Runtime Capability Contract

Back to [[plans/82-backend-v2-first-principles/overview]]

## Goal

Unify runtime handling under one capability contract so process/sprites/cursor behavior is explicit and consistent.

## Changes

- Define runtime capabilities surfaced by each runtime implementation
- Standardize recovery, interruption, terminal cleanup, and heartbeat semantics
- Replace runtime-name branch logic with capability checks
- Implement two-phase launch persistence (`launching` -> `launched`) with stable attempt token
- Reconcile `launching` records on startup before dispatch retries

## Data Structures

- `RuntimeCapabilities`
- `RuntimeSessionHandle`
- `RecoveredAttempt`
- `LaunchMetadata`
- `LaunchRecordState`

## Routing

| Phase type | Provider | Model | Why |
|------------|----------|-------|-----|
| Architecture / judgment | `claude` | `claude-opus-4-6` | Runtime contract and behavior semantics |
| Implementation | `codex` | `gpt-5.3-codex` | Mechanical adapter migration |

## Verification

### Static

- Runtime adapters conform to one interface contract
- Capability checks replace string branching in dispatch/recovery paths
- `go test ./... && go vet ./...`

### Runtime

- End-of-phase e2e smoke test: `pnpm test:smoke`
- If this phase changes externally observable behavior, update smoke assertions in this same phase.
- Smoke gate: pass required unless an Expected Smoke Failure Contract is declared below.
- Expected Smoke Failure Contract (default): none for this phase.
- Per-runtime contract tests for dispatch/recover/stop/delete
- Restart recovery tests with in-flight process and remote sessions
- Launch-gap crash tests: crash after remote launch but before `launched` persistence
- Edge cases: terminal API errors (401/403/404/410), retryable API errors (429/5xx)
