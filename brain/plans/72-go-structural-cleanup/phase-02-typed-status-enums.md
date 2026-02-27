Back to [[plans/72-go-structural-cleanup/overview]]

# Phase 2: Typed status enums

## Goal

Replace `string` status fields with typed enums (`OrderStatus`, `StageStatus`). Parse at boundaries, use typed values internally. Eliminate scattered `strings.ToLower(strings.TrimSpace(...))` normalization throughout the codebase.

Per [[principles/boundary-discipline]]: validate at boundaries, trust the types inside.

## Changes

**`internal/orderx/queue.go`** — Define typed status types:
- `type OrderStatus string` with constants `OrderStatusActive`, etc.
- `type StageStatus string` with constants `StageStatusPending`, etc.
- Update `Order.Status` field from `string` to `OrderStatus`
- Update `Stage.Status` field from `string` to `StageStatus`
- `ValidateOrderStatus` and `ValidateStageStatus` accept and return the typed values

**`internal/orderx/orders.go`** — Update all status comparisons to use typed constants. No more string literals for statuses.

**`loop/*.go`** — Propagate typed statuses through all loop code:
- `cookHandle.orderStatus` → `orderx.OrderStatus`
- `pendingRetryCook.orderStatus` → `orderx.OrderStatus`
- `StageResult.Status` stays as `StageResultStatus` (different enum for result outcomes)
- Keep `strings.ToLower(strings.TrimSpace(...))` calls on **session runtime status** in `cook.go` — these parse external session status at the boundary (e.g., `stageResultStatus()` at line 300, `handleCompletion` at line 381, `readSessionStatus` at line 720). Per [[principles/boundary-discipline]], normalization belongs at system boundaries
- `ensureOrderStageStatus` in `state_orders.go` — change parameter from raw `string` to `orderx.StageStatus`
- `PendingRetryItem.OrderStatus` in `pending_retry.go` — change from `string` to `orderx.OrderStatus`

**`internal/snapshot/snapshot.go`** — Update status comparisons.

## Data structures

```
type OrderStatus string   // "active", "completed", "failed", "failing"
type StageStatus string   // "pending", "active", "merging", "completed", "failed", "cancelled"
```

JSON serialization is unchanged — `OrderStatus` and `StageStatus` marshal/unmarshal as strings. The `DisallowUnknownFields` strict parsing in `ParseOrdersStrict` provides boundary validation.

## Verification

- `go test ./...` — all tests pass
- `go vet ./...` — no issues
- Grep for `strings.ToLower.*Status` and `strings.TrimSpace.*Status` in loop/ — count should drop significantly (only legitimate boundary parsing remains)
- Grep for `== "active"`, `== "pending"`, `== "completed"` string literals in loop/ and orderx/ — should be zero for order/stage status comparisons (session runtime status comparisons at boundaries are allowed)
