# Phase 10: Scale and Resilience Verification

Back to [[plans/82-backend-v2-first-principles/overview]]

## Goal

Prove at runtime that the redesign holds under scale, crashes, retries, and control pressure.

## Changes

- Add/expand high-concurrency and crash-window integration fixtures
- Validate convergence properties under replay and restart
- Validate control command correctness under concurrent event traffic
- Document operational acceptance criteria for backend V2

## Data Structures

- Test fixtures describing event streams, crash boundaries, and expected converged state

## Routing

| Phase type | Provider | Model | Why |
|------------|----------|-------|-----|
| Implementation | `codex` | `gpt-5.3-codex` | Test scaffolding and fixture expansion |

## Verification

### Static

- Full test suite and vet clean
- Fixture hashes and expected outputs synchronized
- `go test ./... && go vet ./...`

### Runtime

- 100+ simulated session scale run with mixed runtimes
- Crash-in-window scenarios:
  - after state write, before effect completion
  - after effect completion, before projection write
- Control pressure scenario: rapid mode/dispath/stop actions during active load
- Acceptance criteria: no duplicate dispatch, no lost terminal state, deterministic convergence after replay
