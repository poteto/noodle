Back to [[archive/plans/49-work-orders-redesign/overview]]

# Phase 4: Stage lifecycle functions

## Goal

Build the pure functions that implement stage state transitions. These are the building blocks the loop core (phase 5) will call. Keeping them as pure functions makes them easy to test in isolation.

## Changes

**`loop/orders.go`** (extend from phase 3) — Add lifecycle functions:

- `advanceOrder(orders OrdersFile, orderID string) (OrdersFile, bool, error)` — Find the order, mark its current active/first-pending stage as `completed`. For `"active"` orders: if all main stages are now completed, remove the order from the OrdersFile and return `removed=true`. For `"failing"` orders: advance through OnFailure stages instead of main stages; when the last OnFailure stage completes, remove the order and return `removed=true`. The caller uses `removed` + order status to decide whether to fire adapter "done" (active orders) or call markFailed (failing orders). Error if order not found.
- `failStage(orders OrdersFile, orderID string, reason string) (OrdersFile, bool, error)` — Mark the current active stage as `failed`. Then check: if the order has `OnFailure` stages and is not already `"failing"`, mark remaining main stages as `cancelled`, set order status to `"failing"`, and reset OnFailure stages to `"pending"` so the loop dispatches them next cycle. **Do NOT call markFailed or add to failedTargets** — the order is still alive (running OnFailure stages). If no `OnFailure` stages (or already in the `"failing"` pipeline), mark all remaining stages as `cancelled` and **remove the order from the OrdersFile**. Only at this terminal point does the caller add the order ID to `failed.json`. Returns updated OrdersFile and a `terminal bool` indicating whether the order was removed (so the caller knows whether to call markFailed).
  - **Quality verdict flow:** `accept=false` → `failStage` → if OnFailure present: order becomes `"failing"`, OnFailure stages dispatch next cycle → when OnFailure completes or fails: order is terminally removed, caller calls markFailed.
- `cancelOrder(orders OrdersFile, orderID string) (OrdersFile, error)` — Mark all non-completed stages as `cancelled`, **remove the order from the OrdersFile**. For user-initiated cancellation.
- `dispatchableStages(orders OrdersFile, busy, failed, adopted, ticketed sets) []dispatchCandidate` — Iterate orders, find each order's first pending stage where the order is active (or `"failing"` — OnFailure stages are dispatchable) and not in busy/failed/adopted/ticketed sets. For `"failing"` orders, dispatch from `OnFailure` stages instead of main `Stages`. Return candidates in order priority (array position). A `dispatchCandidate` is `{OrderID, StageIndex, Stage, IsOnFailure bool}`.
  - **`busy` set:** Keyed by **order ID** (from `activeByTarget` in the loop). One cook per order at a time — if an order has an active stage, the entire order is busy.
  - **`failed` set:** Keyed by **order ID** (from `failedTargets`). Orders in `"failing"` status are **exempt** from the failed check — OnFailure stages must dispatch even though the order has a failure. Only terminally failed orders (removed from OrdersFile and added to `failedTargets`) are blocked.
  - **Degenerate orders:** Orders with `stages: []` should never reach `dispatchableStages` (dropped by validation in phase 3). If one somehow does, skip it — return no candidate.
- `activeStageForOrder(order Order) (int, *Stage)` — Return the index and pointer to the currently active or first pending stage. Nil if order is completed/failed.

## Data structures

- `dispatchCandidate` — lightweight struct: `OrderID string`, `StageIndex int`, `Stage Stage`, `IsOnFailure bool`
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
- Unit test: advanceOrder on final stage removes the order from OrdersFile, returns removed=true
- Unit test: advanceOrder on already-completed/missing order returns error
- Unit test: failStage with no OnFailure stages — removes order from OrdersFile, marks remaining cancelled
- Unit test: failStage with OnFailure stages — sets order status to `"failing"`, cancels remaining main stages, OnFailure stages become pending
- Unit test: failStage during OnFailure pipeline (already failing) — removes order from OrdersFile (no double-failure recursion)
- Unit test: advanceOrder on a `"failing"` order advances through OnFailure stages; when last OnFailure stage completes, removes order from OrdersFile and returns removed=true (caller calls markFailed, not adapter "done")
- Unit test: cancelOrder on order with mix of completed and pending stages — completed stages preserved, pending cancelled, **order removed from OrdersFile**
- Unit test: cancelOrder returns updated OrdersFile without the cancelled order (persisted removal verified)
- Unit test: dispatchableStages skips orders in busy/failed/adopted/ticketed sets
- Unit test: dispatchableStages returns first pending stage per order, respects array ordering
- Unit test: dispatchableStages skips orders whose current stage is active (already dispatched)
- Unit test: dispatchableStages dispatches OnFailure stages for `"failing"` orders (with `IsOnFailure=true`)
- Unit test: dispatchableStages exempts `"failing"` orders from the failed set — OnFailure stages dispatch even if orderID is in failedTargets
- Unit test: dispatchableStages skips degenerate order with empty stages array (no candidate returned, no crash)
