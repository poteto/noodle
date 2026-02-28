# Phase 1: Canonical State Model

Back to [[plans/82-backend-v2-first-principles/overview]]

## Goal

Define one canonical backend state model that all loop decisions use, replacing split maps/status derivations as primary orchestration state.

## Changes

- Introduce canonical state package/types for orders, stages, attempts, and run mode
- Consolidate lifecycle enums into typed status sets used across loop/runtime/snapshot
- Define explicit state serialization contract for crash-safe persistence
- Keep adapters for legacy internal call sites only during migration phases

## Data Structures

- `State`
- `OrderNode`
- `StageNode`
- `AttemptNode`
- `RunMode`
- `OrderLifecycleStatus`, `StageLifecycleStatus`, `AttemptStatus`

## Routing

| Phase type | Provider | Model | Why |
|------------|----------|-------|-----|
| Architecture / judgment | `claude` | `claude-opus-4-6` | Canonical type boundaries and status semantics |

## Verification

### Static

- Type checks pass for new canonical types
- No stringly-typed lifecycle status comparisons in migrated files
- `go test ./... && go vet ./...`

### Runtime

- Serialization round-trip tests for canonical state
- State migration tests proving no data loss for active orders/stages
- Edge cases: empty orders, terminal states, partially active pipelines

