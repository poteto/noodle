Back to [[plans/49-work-orders-redesign/overview]]

# Phase 3: Orders file I/O

## Goal

Build the file I/O layer for `orders.json` and `orders-next.json`. Same atomic write / staging promotion pattern as current queue files, but for the new types.

## Changes

**`internal/queuex/queue.go`** (or new file alongside it) — Add functions:
- `ReadOrders(path) (OrdersFile, error)` — read and parse orders.json
- `ParseOrdersStrict(data) (OrdersFile, error)` — validate bytes without disk I/O
- `WriteOrdersAtomic(path, OrdersFile) error` — atomic write via temp file + rename
- `ApplyOrderRoutingDefaults(OrdersFile, registry, config) (OrdersFile, bool)` — fill missing provider/model on stages from config defaults. Apply to both `Stages` and `OnFailure` stages. **Must preserve `Stage.Extra`** — update fields in-place on the existing stage struct, do not reconstruct stages from known fields.
- `NormalizeAndValidateOrders(OrdersFile, planIDs, registry, config) (OrdersFile, bool, error)` — validate stage task types against registry (both `Stages` and `OnFailure`), drop orders with no valid main stages, strip invalid OnFailure stages (clear OnFailure if none remain — order still valid without it), normalize IDs, enforce order ID uniqueness (reject duplicates). `Extra` fields are opaque — pass through without validation.

**`loop/orders.go`** (new file, parallel to `loop/queue.go`) — Add wrapper functions:
- `readOrders(path) (OrdersFile, error)` — wraps queuex.ReadOrders, converts types
- `writeOrdersAtomic(path, OrdersFile) error` — wraps queuex.WriteOrdersAtomic
- `consumeOrdersNext(nextPath, ordersPath) (bool, error)` — atomically promotes orders-next.json to orders.json. **Do NOT replicate the current `consumeQueueNext` pattern** (which deletes next, then writes orders — crash between the two loses data). Instead: read and validate orders-next.json, merge into existing orders.json via `WriteOrdersAtomic` (atomic temp+rename), THEN delete orders-next.json. If the loop crashes after writing orders.json but before deleting orders-next.json, the next cycle re-promotes idempotently (orders-next content is re-merged — duplicates rejected by ID uniqueness).

## Data structures

- Same types from phase 2
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
- Unit test: ApplyOrderRoutingDefaults fills defaults on OnFailure stages too
- Unit test: Round-trip preserves Stage.Extra (arbitrary JSON in, same JSON out)
- Unit test: Round-trip preserves Order.OnFailure (including empty/nil case)
- Unit test: NormalizeAndValidateOrders validates OnFailure stage task types (strips invalid OnFailure stages; if all OnFailure stages invalid, clears OnFailure but keeps order)
- Unit test: ApplyOrderRoutingDefaults preserves Stage.Extra through the defaults pipeline (Extra in, same Extra out)
- Unit test: consumeOrdersNext crash safety — write orders.json, leave orders-next.json present, re-run consumeOrdersNext → idempotent (no duplicate orders, no data loss)
- Unit test: ReadOrders on empty file → returns empty OrdersFile (not error)
- Unit test: ReadOrders on malformed JSON → returns descriptive error
- Unit test: NormalizeAndValidateOrders on order with `stages: []` → drops the order (no valid stages)
- Unit test: NormalizeAndValidateOrders on order with only OnFailure stages (empty main stages) → drops the order
