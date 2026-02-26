Back to [[plans/49-work-orders-redesign/overview]]

# Phase 3: Stage lifecycle functions

## Goal

Build the pure functions that implement stage state transitions. These are the building blocks the loop core (phase 4) will call. Keeping them as pure functions makes them easy to test in isolation.

## Changes

**`loop/orders.go`** (extend from phase 2) — Add lifecycle functions:

- `advanceOrder(orders OrdersFile, orderID string) (OrdersFile, error)` — Find the order, mark its current active/first-pending stage as `completed`. If all stages are now completed, **remove the order from the OrdersFile** (mirrors current `skipQueueItem` behavior — completed work leaves the file). Returns updated OrdersFile. Error if order not found.
- `failOrder(orders OrdersFile, orderID string, reason string) (OrdersFile, error)` — Mark the current active stage as `failed`, mark all remaining pending stages as `cancelled`, **remove the order from the OrdersFile**. The caller adds the order ID to `failed.json` separately. Returns updated OrdersFile.
- `cancelOrder(orders OrdersFile, orderID string) (OrdersFile, error)` — Mark all non-completed stages as `cancelled`, **remove the order from the OrdersFile**. For user-initiated cancellation.
- `dispatchableStages(orders OrdersFile, busy, failed, adopted, ticketed sets) []dispatchCandidate` — Iterate orders, find each order's first pending stage where the order is active and not in busy/failed/adopted/ticketed sets. Return candidates in order priority (array position). A `dispatchCandidate` is `{OrderID, StageIndex, Stage}`.
- `activeStageForOrder(order Order) (int, *Stage)` — Return the index and pointer to the currently active or first pending stage. Nil if order is completed/failed.

## Data structures

- `dispatchCandidate` — lightweight struct: `OrderID string`, `StageIndex int`, `Stage Stage`
- All functions take and return `OrdersFile` (value semantics, no mutation) — the caller persists

## Routing

| Provider | Model |
|----------|-------|
| `claude` | `claude-opus-4-6` |

State machine semantics require judgment about edge cases.

## Verification

### Static
- `go build ./...` and `go vet ./...` pass

### Runtime
- Unit test: advanceOrder on order with 3 stages — advances through pending → completed for each stage, order completes when last stage completes
- Unit test: advanceOrder on final stage removes the order from OrdersFile
- Unit test: advanceOrder on already-completed/missing order returns error
- Unit test: failOrder removes the order from OrdersFile
- Unit test: failOrder marks current stage failed, remaining cancelled, order failed
- Unit test: cancelOrder on order with mix of completed and pending stages — completed stages preserved, pending cancelled
- Unit test: dispatchableStages skips orders in busy/failed/adopted/ticketed sets
- Unit test: dispatchableStages returns first pending stage per order, respects array ordering
- Unit test: dispatchableStages skips orders whose current stage is active (already dispatched)
