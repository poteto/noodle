Back to [[plans/115-canonical-state-convergence/overview]]

# Phase 3 — Dispatch Planner Cutover

## Goal

Make canonical state the only planner input for “what can run next” and the only lifecycle authority for `pending -> dispatching -> running`, while keeping `spawnCook()` as the execution shell.

## Changes

- **`loop/loop_cycle_pipeline.go`** — replace legacy candidate selection with canonical-plan consumption
- **`loop/cook_spawn.go`** — move dispatch-start ownership onto canonical `dispatch_requested` / `dispatch_completed` transitions instead of pre-dispatch `orders.json` writes
- **`loop/orders.go`** — remove or demote legacy dispatchability helpers once callers move
- **`loop/state_orders.go` / `loop/reconcile.go`** — delete `persistOrderStageStatus`-style authority and restart repair logic for active stages in the same cutover wave
- **`internal/dispatch/dispatch.go`** — become the only planner boundary for stage dispatchability and blocked reasons

## Data structures

- `dispatch.DispatchPlan` — authoritative plan output consumed by the loop shell
- `state.State.OrderBusyIndex()` — canonical busy-target index used by the planner
- dispatch-start event payloads that fully describe the running attempt without relying on a prior legacy `orders.json` write

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `claude` | `claude-opus-4-6` | Source-of-truth boundary redesign |

## Verification

### Static
- `pnpm test:smoke`
- `go test ./internal/dispatch/... ./internal/integration/... ./loop/...`
- `sh scripts/lint-arch.sh`

### Runtime
- prove one dispatch attempt per eligible stage under normal load and replay pressure
- prove capacity limits, busy targets, adopted sessions, and failed orders produce the same blocked/candidate behavior as before
- prove restart and adopted-session recovery no longer depend on `StageStatusActive` being written to legacy `orders.json` before dispatch
- confirm the superseded planner helpers and dispatch-start legacy writes are deleted or unreachable on the hot path once canonical planning is enabled
- rerun `pnpm test:smoke` after dispatch-start ownership moves and treat unexpected failures as blockers
