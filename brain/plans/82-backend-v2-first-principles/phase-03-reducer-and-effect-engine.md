# Phase 3: Reducer and Effect Engine

Back to [[plans/82-backend-v2-first-principles/overview]]

## Goal

Split pure state transitions from side effects so loop behavior becomes deterministic, testable, and replayable.

## Changes

- Implement reducer that maps `(State, StateEvent) -> (State, []Effect)`
- Introduce effect executor pipeline for dispatch, merge, cleanup, and file projections
- Ensure effect execution is idempotent and retry-safe via durable effect ledger
- Move existing imperative transition logic into reducer transitions

## Data Structures

- `Reducer`
- `Effect`
- `EffectDispatch`, `EffectMerge`, `EffectWriteProjection`, `EffectAck`
- `EffectResult`
- `EffectLedgerRecord` (`pending`, `running`, `done`, `failed`, `cancelled`, `deferred`)

### Concurrency Model (Required)

- Single reducer goroutine owns canonical state mutation.
- Effect workers run concurrently but never mutate canonical state directly.
- Effect workers emit effect-result events back to ingestion arbiter.
- Reducer commits effect state transitions from those events.

### Crash-Consistency Protocol

1. Persist ingested event with sequence.
2. Reduce event to next state and effect set.
3. Persist canonical state snapshot + embedded effect ledger in a single file atomically (`write temp` + `rename`).
4. Execute effects with deterministic `effect_id` idempotency key. For launch effects, this step depends on Phase 4's two-phase launch contract (`launching` -> `launched`) and must not be implemented independently.
5. Persist effect result.
6. Emit ack/projection updates after durable commit.

## Routing

| Phase type | Provider | Model | Why |
|------------|----------|-------|-----|
| Implementation | `codex` | `gpt-5.3-codex` | Deterministic reducer/effect migration |

## Verification

### Static

- Reducer package has no side-effect imports
- Effect executors consume typed effect payloads only
- Lint rule forbids canonical-state writes outside reducer package
- `go test ./... && go vet ./...`

### Runtime

- End-of-phase e2e smoke test: `pnpm test:smoke`
- If this phase changes externally observable behavior, update smoke assertions in this same phase.
- Smoke gate: pass required unless an Expected Smoke Failure Contract is declared below.
- Expected Smoke Failure Contract (default): none for this phase.
- Table-driven reducer tests for all lifecycle transitions
- Effect retry tests proving no double-merge/double-dispatch
- Crash-window tests between each protocol step above
- Edge cases: partial executor outage, stale lock recovery
