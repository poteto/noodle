Back to [[plans/87-go-codebase-simplification/overview]]

# Phase 9: Decompose `prepareOrdersForCycle` + `applyControlCommand`

## Goal

Break two long functions in the loop package into focused sub-functions.

## Changes

### 9a: `prepareOrdersForCycle` (155 lines, `loop_cycle_pipeline.go:27-181`)

This function mixes: orders-next promotion, validation, bootstrapping, mise change detection, and routing defaults.

Extract only the mechanical read/merge step — leave promotion logic (promoted flag, emptyPromotion, promotion failure classification, cooldown updates, canonical promotion emission) in the orchestrator since these outputs drive scheduler state and cannot be lost behind a simple `(*OrdersFile, error)` return.

Extract:
- `mergeOrdersNext() (*OrdersFile, bool, error)` — read orders-next file, validate, merge into current orders. Returns merged orders, whether a promotion occurred, and error. Does NOT handle promotion side effects.
- `ensureScheduleIfNeeded(brief, orders) error` — check if schedule bootstrap is needed
- `applyRoutingDefaults(orders)` — normalize runtime/provider defaults

Keep `prepareOrdersForCycle` as the orchestrator calling these in sequence and handling promotion metadata (failure classification, cooldown, canonical emission) directly.

Also flatten the nested conditionals at lines 140-157 into early returns.

### 9b: `applyControlCommand` (106 lines, `control.go:179-284`)

Giant switch with 17+ cases. Extract each case handler into its own method:
- `controlPause`, `controlResume`, `controlStop`, etc.

The switch becomes a dispatch table or a series of one-line case bodies calling the extracted methods. Existing per-command methods like `controlStopKill` already follow this pattern — make it consistent.

## Data Structures

No new types. Sub-functions are private methods on `*Loop`.

## Routing

- Provider: `claude`
- Model: `claude-opus-4-6`
- Decomposition requires understanding the function's control flow and choosing good split points.

## Verification

### Static
- `go test ./loop/...` — all loop tests pass
- `go vet ./loop/...` — clean
- No function in modified files exceeds 60 lines

### Runtime
- `go test ./...` — full suite passes
