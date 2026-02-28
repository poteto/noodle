# Phase 0: Preflight Subtraction and Fencing

Back to [[plans/82-backend-v2-first-principles/overview]]

## Goal

Remove dead/vestigial paths early and establish process/schema fencing before reducer migration starts.

## Changes

- Remove clearly dead mutation paths that are already superseded
- Add schema version marker and runtime writer lockfile contract
- Define immediate deletion policy for replaced code paths (no compat adapters)

## Data Structures

- `SchemaVersion`
- `WriterLock`

## Routing

| Phase type | Provider | Model | Why |
|------------|----------|-------|-----|
| Architecture / judgment | `claude` | `claude-opus-4-6` | Early sequencing and cutover fencing decisions |
| Implementation | `codex` | `gpt-5.3-codex` | Mechanical dead-path deletion |

## Verification

### Static

- Dead paths removed compile-clean
- Startup refuses second writer instance deterministically
- `go test ./... && go vet ./...`

### Runtime

- End-of-phase e2e smoke test: `pnpm test:smoke`
- If this phase changes externally observable behavior, update smoke assertions in this same phase.
- Smoke gate: pass required unless an Expected Smoke Failure Contract is declared below.
- Expected Smoke Failure Contract (default): none for this phase.
- Dual-process startup test: second process fails with explicit failure-state message
- Schema marker round-trip test
- Edge cases: stale lock recovery, interrupted schema write
