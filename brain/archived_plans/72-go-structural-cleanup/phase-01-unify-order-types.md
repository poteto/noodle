Back to [[plans/72-go-structural-cleanup/overview]]

# Phase 1: Unify order types

## Goal

Eliminate the duplicate Order/Stage/OrdersFile type definitions between `loop/types.go` and `internal/orderx/queue.go`. The `orderx` types become canonical; the loop package uses them directly. Delete the ~120-line conversion layer in `loop/orders.go`.

## Changes

**`loop/types.go`** — Delete the following (they're identical to orderx):
- `Stage` struct (lines 57-66)
- `Order` struct (lines 69-77)
- `OrdersFile` struct (lines 80-84)
- Stage status constants (lines 31-38)
- Order status constants (lines 41-46)
- `ValidateOrderStatus()` (lines 87-96)
- `ValidateStageStatus()` (lines 99-108)

Keep: `State`, `StageResultStatus`, `ControlCommand`, `ControlAck`, `QualityVerdict`, `StageResult`, `cookHandle`, `pendingReviewCook`, `pendingRetryCook`, interfaces, `Dependencies`, `Loop` struct.

**`loop/orders.go`** — Delete the conversion layer:
- `toOrdersFileX`, `fromOrdersFileX` (lines 404-426)
- `toOrderX`, `fromOrderX` (lines 428-450)
- `toStagesX`, `fromStagesX` (lines 452-490)
- Wrapper `NormalizeAndValidateOrders` (lines 493-502) — callers use `orderx.NormalizeAndValidateOrders` directly
- Wrapper `ApplyOrderRoutingDefaults` (lines 505-511) — callers use `orderx.ApplyOrderRoutingDefaults` directly
- `readOrders` / `writeOrdersAtomic` shims (lines 334-344) — callers use `orderx.ReadOrders` / `orderx.WriteOrdersAtomic` directly

**`loop/*.go`** — Replace `loop.Order` → `orderx.Order`, `loop.Stage` → `orderx.Stage`, `loop.OrdersFile` → `orderx.OrdersFile` throughout the loop package. Replace status constant references (e.g., `StageStatusPending` → `orderx.StageStatusPending`). Add `orderx` import where needed.

**`loop/*_test.go`** — Same type migration in test files.

**`internal/snapshot/snapshot.go`** — Uses `loop.StageStatusActive` and `loop.StageStatusMerging`. Migrate to `orderx.StageStatusActive` / `orderx.StageStatusMerging`.

**`internal/snapshot/snapshot_test.go`** — Constructs `loop.Order{}` and `[]loop.Stage{}` literals. Migrate to `orderx` types.

## Data structures

No new types. `orderx.Order`, `orderx.Stage`, `orderx.OrdersFile` are the sole definitions. The `cookHandle`, `pendingReviewCook`, `pendingRetryCook` structs in `loop/types.go` update their field types from `loop.Stage` → `orderx.Stage` (same underlying type, just different package qualifier).

## Verification

- `go test ./...` — all loop and orderx tests pass
- `go vet ./...` — no issues
- Grep for `loop.Order{`, `loop.Stage{`, `loop.OrdersFile{` outside the loop package — should return zero results (confirms no external consumers depended on the loop copies)
- Grep for `toOrdersFileX`, `fromOrdersFileX`, `toOrderX`, `fromOrderX`, `toStagesX`, `fromStagesX` — should return zero results
