# Phase 10: Scale and Resilience Verification

Back to [[plans/82-backend-v2-first-principles/overview]]

## Goal

Prove at runtime that the redesign holds under scale, crashes, retries, and control pressure.

## Changes

- Add/expand high-concurrency and crash-window integration fixtures
- Validate convergence properties under replay and restart
- Validate control command correctness under concurrent event traffic
- Validate operator-diagnosis workflows using only `.noodle/` files + snapshot outputs
- Record hard acceptance metrics for release readiness

## Data Structures

- Test fixtures describing event streams, crash boundaries, and expected converged state
- Acceptance metric report artifact

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

- End-of-phase e2e smoke test: `pnpm test:smoke`
- If this phase changes externally observable behavior, update smoke assertions in this same phase.
- Smoke gate: pass required unless an Expected Smoke Failure Contract is declared below.
- Expected Smoke Failure Contract (default): none for this phase.
- 100+ simulated session scale run with mixed runtimes
- End-to-end black-box harness: `orders-next.json` + `control.ndjson` + runtime terminal events -> projections + websocket outputs
- Crash-in-window scenarios:
  - after event persistence, before state/effect ledger commit
  - after external effect success, before effect-result persistence
  - after projection write temp, before atomic rename
- Control pressure scenario: rapid mode/dispatch/stop actions during active load
- Acceptance criteria:
  - duplicate dispatches = `0`
  - lost terminal states = `0`
  - deterministic replay hash match = `100%`
  - projection version monotonicity violations = `0`
