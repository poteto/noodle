Back to [[plans/72-go-structural-cleanup/overview]]

# Phase 6: Cleanup exports and packages

## Goal

Tighten the API surface: unexport symbols that are only used within their package, move `recover` to `internal/` since it has a single consumer.

Per [[principles/subtract-before-you-add]]: reduce surface area to reveal essential structure.

## Changes

### Unexport loop-internal symbols

These are exported but have zero external consumers (verified by grep):

**`loop/orders.go`:**
- `ActiveOrderIDs` → `activeOrderIDs` (used only in loop/state_snapshot.go and loop/orders.go)
- `BusyTargets` → `busyTargets` (used only in loop/loop.go and loop/orders.go)

Note: `NormalizeAndValidateOrders`, `ApplyOrderRoutingDefaults`, `ValidateOrderStatus`, `ValidateStageStatus` will be deleted in Phase 1 (callers migrated to orderx directly).

### Move recover to internal/

**`recover/`** → **`internal/recover/`**

Single consumer: `loop/cook.go` (or `loop/cook_retry.go` after Phase 5). Update the import path.

### Do NOT move parse or stamp

`parse` is imported by `dispatcher` (sprites_session.go, tmux_session.go), `monitor`, and `stamp` (3 packages, 4 production files). `stamp` is imported by `cmd_stamp.go` and `dispatcher`. Both have legitimate multi-consumer usage — they stay at top level.

## Verification

- `go test ./...` — all tests pass
- `go vet ./...` — no issues
- `go build ./...` — confirms no broken imports
- Grep for `loop.ActiveOrderIDs`, `loop.BusyTargets` — zero external results
