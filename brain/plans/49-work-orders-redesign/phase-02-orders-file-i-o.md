Back to [[plans/49-work-orders-redesign/overview]]

# Phase 2: Orders file I/O

## Goal

Build the file I/O layer for `orders.json` and `orders-next.json`. Same atomic write / staging promotion pattern as current queue files, but for the new types.

## Changes

**`internal/queuex/queue.go`** (or new file alongside it) — Add functions:
- `ReadOrders(path) (OrdersFile, error)` — read and parse orders.json
- `ParseOrdersStrict(data) (OrdersFile, error)` — validate bytes without disk I/O
- `WriteOrdersAtomic(path, OrdersFile) error` — atomic write via temp file + rename
- `ApplyOrderRoutingDefaults(OrdersFile, registry, config) (OrdersFile, bool)` — fill missing provider/model on stages from config defaults
- `NormalizeAndValidateOrders(OrdersFile, planIDs, registry, config) (OrdersFile, bool, error)` — validate stage task types against registry, drop orders with no valid stages, normalize IDs, enforce order ID uniqueness (reject duplicates)

**`loop/orders.go`** (new file, parallel to `loop/queue.go`) — Add wrapper functions:
- `readOrders(path) (OrdersFile, error)` — wraps queuex.ReadOrders, converts types
- `writeOrdersAtomic(path, OrdersFile) error` — wraps queuex.WriteOrdersAtomic
- `consumeOrdersNext(nextPath, ordersPath) (bool, error)` — atomically promotes orders-next.json to orders.json. Same pattern as current `consumeQueueNext`: read next, remove next, validate, write to orders.

## Data structures

- Same types from phase 1
- File paths: `.noodle/orders.json`, `.noodle/orders-next.json`

## Routing

| Provider | Model |
|----------|-------|
| `codex` | `gpt-5.3-codex` |

Mechanical — clear spec, mirrors existing queue I/O patterns.

## Verification

### Static
- `go build ./...` passes
- No production callers yet — only tests

### Runtime
- Unit test: WriteOrdersAtomic → ReadOrders round-trip
- Unit test: consumeOrdersNext promotes file and removes staging file
- Unit test: consumeOrdersNext with missing staging file returns false
- Unit test: NormalizeAndValidateOrders drops orders with unknown task types
- Unit test: ApplyOrderRoutingDefaults fills missing provider/model from config
- Unit test: NormalizeAndValidateOrders rejects duplicate order IDs
