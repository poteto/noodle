Back to [[plans/95-orders-json-ownership/overview]]

# Phase 1 — In-Memory Orders Merge

## Goal

`consumeOrdersNext` stops reading orders.json from disk and instead receives the current orders from memory. The caller handles disk persistence and orders-next.json cleanup with correct crash-safe ordering.

## Changes

- **`loop/orders.go`** — refactor `consumeOrdersNext` signature and body
- **`loop/loop_cycle_pipeline.go`** — update `mergeOrdersNext` and `handlePromotionResult`

## Data structures

- `mergeResult` — struct with `Orders OrdersFile`, `Promoted bool`, `EmptyPromotion bool`

## Details

### consumeOrdersNext refactor

Current signature:
```
func consumeOrdersNext(nextPath, ordersPath string) (bool, bool, error)
```

New signature:
```
func consumeOrdersNext(nextPath string, existing OrdersFile) (mergeResult, error)
```

Changes:
- Accept `existing OrdersFile` parameter (from `l.orders`) instead of `ordersPath`
- Return `mergeResult` with merged `OrdersFile` instead of writing to disk
- Still reads `orders-next.json` from disk (agents are supposed to write this)
- Still renames invalid files to `.bad`
- **Do NOT delete orders-next.json** — the caller handles deletion after writing merged state (crash safety)
- Remove `orderx.ReadOrders(ordersPath)` call and `orderx.WriteOrdersAtomic(ordersPath, existing)` call

### mergeOrdersNext refactor

Pass in-memory state, receive merged result:
```go
func (l *Loop) mergeOrdersNext() (mergeResult, error) {
    existing, err := l.currentOrders()
    ...
    return consumeOrdersNext(l.deps.OrdersNextFile, existing)
}
```

### handlePromotionResult refactor

Current: calls `loadOrdersState()` (line 60) to re-read merged result from disk.

New: receives merged orders from `mergeResult`, then:
1. Write merged state to disk + memory via `writeOrdersState(result.Orders)`
2. **Then** delete orders-next.json (crash-safe: delete only after successful persist)
3. Remove the `loadOrdersState()` call entirely

This preserves the current crash-safety invariant: if the loop crashes after writing orders.json but before deleting orders-next.json, the next cycle re-promotes idempotently.

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `claude` | `claude-opus-4-6` | Function boundary redesign, crash-safety judgment |

## Verification

### Static
- `go vet ./loop/...`
- `go build ./...`

### Runtime
- Existing loop tests pass (merge behavior unchanged, only the data source changed)
- `pnpm check`
