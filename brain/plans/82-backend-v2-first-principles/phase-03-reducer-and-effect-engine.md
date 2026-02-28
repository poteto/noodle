# Phase 3: Reducer and Effect Engine

Back to [[plans/82-backend-v2-first-principles/overview]]

## Goal

Split pure state transitions from side effects so loop behavior becomes deterministic, testable, and replayable.

## Changes

- Implement reducer that maps `(State, StateEvent) -> (State, []Effect)`
- Introduce effect executor pipeline for dispatch, merge, cleanup, and file projections
- Ensure effect execution is idempotent and retry-safe
- Move existing imperative transition logic into reducer transitions

## Data Structures

- `Reducer`
- `Effect`
- `EffectDispatch`, `EffectMerge`, `EffectWriteProjection`, `EffectAck`
- `EffectResult`

## Routing

| Phase type | Provider | Model | Why |
|------------|----------|-------|-----|
| Implementation | `codex` | `gpt-5.3-codex` | Deterministic reducer/effect migration |

## Verification

### Static

- Reducer package has no side-effect imports
- Effect executors consume typed effect payloads only
- `go test ./... && go vet ./...`

### Runtime

- Table-driven reducer tests for all lifecycle transitions
- Effect retry tests proving no double-merge/double-dispatch
- Edge cases: effect failure mid-cycle, partial executor outage, stale lock recovery
