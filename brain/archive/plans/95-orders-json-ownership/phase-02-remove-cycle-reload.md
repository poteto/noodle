Back to [[archive/plans/95-orders-json-ownership/overview]]

# Phase 2 — Remove Per-Cycle Disk Reload

## Goal

Make in-memory state truly authoritative during operation. The loop stops re-reading orders.json from disk on every cycle and stops watching it via fsnotify.

## Changes

- **`loop/loop.go`** — remove `loadOrdersState()` call in `Cycle()`, update fsnotify filter
- **`loop/state_orders.go`** — collapse `flushState()` orders write through `writeOrdersState`

## Details

### Remove per-cycle reload

Current (`loop.go:396-407`):
```go
func (l *Loop) Cycle(ctx context.Context) error {
    ...
    if err := l.loadOrdersState(); err != nil {  // <-- remove this
        ...
    }
```

After startup, in-memory state carries across cycles. All mutation paths already update both memory and disk:
- `writeOrdersState` — writes disk + sets `l.orders`
- `mutateOrdersState` — calls `writeOrdersState`
- `setOrdersState` — sets memory only (used after in-memory-only transforms)

Audit all callers to confirm no path depends on the per-cycle re-read.

### Stop watching orders.json

Current (`loop.go:382-387`): fsnotify triggers `Cycle()` on orders.json, orders-next.json, and control.ndjson changes.

Remove `orders.json` from the filter. The loop writes orders.json itself and doesn't need to react to its own writes. Only watch:
- `orders-next.json` — agent write triggers promotion
- `control.ndjson` — control commands trigger processing

### Collapse orders write paths

Current: `flushState()` calls `writeOrdersAtomic(l.deps.OrdersFile, l.orders)` directly (state_orders.go:87), bypassing `writeOrdersState`. This creates a second write path that future stamping would need to cover.

Refactor `flushState()` to delegate its orders write through `writeOrdersState` (or extract the shared stamping logic into a helper both call). This ensures a single write boundary for orders.json.

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `claude` | `claude-opus-4-6` | Architectural change, needs judgment on mutation path audit |

## Verification

### Static
- `go vet ./loop/...`
- `go build ./...`

### Runtime
- All existing loop tests pass
- Manual: tamper orders.json while loop is running → loop ignores it, flushState overwrites
- `pnpm check`
