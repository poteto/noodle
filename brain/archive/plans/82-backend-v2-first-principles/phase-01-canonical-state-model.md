# Phase 1: Canonical State Model

Back to [[plans/82-backend-v2-first-principles/overview]]

## Goal

Define one canonical backend state model that all loop decisions use, replacing split maps/status derivations as primary orchestration state.

## Changes

- Introduce canonical state package/types for orders, stages, attempts, and run mode
- Consolidate lifecycle enums into typed status sets used across loop/runtime/snapshot
- Define explicit state serialization contract for crash-safe persistence
- Define lifecycle invariants and access-pattern indexes up front
- Delete replaced paths directly when callers are switched (no compatibility adapters)

## Data Structures

- `State`
- `OrderNode`
- `StageNode`
- `AttemptNode`
- `RunMode`
- `OrderLifecycleStatus`, `StageLifecycleStatus`, `AttemptStatus`
- `OrderBusyIndex`, `AttemptBySessionIndex`, `PendingEffectIndex`

### Access-Pattern Matrix

| Access need | Primary structure |
|-------------|-------------------|
| Active stage lookup by order | `OrderBusyIndex` |
| Session completion -> attempt lookup | `AttemptBySessionIndex` |
| Retry candidate scan | `AttemptNode` status + retry index |
| Projection build | canonical `State` + projection reducer |

## Routing

| Phase type | Provider | Model | Why |
|------------|----------|-------|-----|
| Architecture / judgment | `claude` | `claude-opus-4-6` | Canonical type boundaries and invariants |

## Verification

### Static

- Type checks pass for new canonical types
- No stringly-typed lifecycle status comparisons in migrated files
- Index definitions align with declared access-pattern matrix
- `go test ./... && go vet ./...`

### Runtime

- End-of-phase e2e smoke test: `pnpm test:smoke`
- Serialization round-trip tests for canonical state
- State persistence tests proving no data loss for active orders/stages/attempts
- Edge cases: empty orders, terminal states, partially active pipelines
