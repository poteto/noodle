Back to [[plans/70-scaling-the-loop-redesign/overview]]

# Phase 4: In-memory orders with periodic flush

## Goal

Hold orders in memory as the authoritative state during the cycle. Write to disk once per cycle (or on significant transitions) instead of on every stage status change. Migrate all order-mutating code paths — including control commands — to operate on in-memory state. Reduces disk I/O from 4-6 writes per stage to 1 write per cycle.

## Changes

**`loop/types.go`** — Add `orders orderx.OrdersFile` field to the Loop struct. This is the in-memory authoritative copy. Add explicit ownership rule: **only the loop goroutine mutates orders state**. Background goroutines (completion watchers, merge queue, health observers) communicate via channels only.

**`loop/orders.go`** — `advanceOrder()` and `failStage()` mutate the in-memory orders directly (no read-from-disk). New `flushOrders()` writes the in-memory state to disk atomically. New `loadOrders()` reads from disk into memory (startup only). `consumeOrdersNext()` merges into the in-memory copy and flushes.

**`loop/loop.go`** — At cycle end, call `flushOrders()` once. Also flush before dispatch (write-before-dispatch preserved for crash safety — see below). On startup, call `loadOrders()`. The fsnotify watch on `orders.json` is removed (the loop owns this file). The fsnotify watch on `orders-next.json` remains (external input).

**`loop/cook.go`** — `spawnCook()` marks stage active in memory, flushes to disk, then dispatches. If dispatch fails, reverts in memory and flushes again. This preserves the write-before-dispatch invariant. `handleCompletion()` calls `advanceOrder()` or `failStage()` on the in-memory copy (flushed at cycle end).

**`loop/control.go`** — Migrate all control commands to operate on in-memory orders instead of reading/writing `orders.json` directly. Commands affected: `controlMerge()` (`control.go:242`), `controlReject()` (`control.go:313`), `controlEnqueue()` (`control.go:401`), `controlSkip()` (`control.go:487`), `controlSteer()` (`control.go:558`), `controlReschedule()` (`schedule.go:206`). Each command mutates `l.orders` and the flush happens at cycle end (or immediately for commands that need the write visible to external readers).

**`loop/pending_review.go`** — Pending review persistence (`pending-review.json`) is flushed alongside orders at cycle end. Both files are written atomically in `flushState()` to maintain consistency.

**`loop/failures.go`** — `failed.json` is flushed alongside orders. The in-memory `failedTargets` map is already authoritative; the file becomes a periodic snapshot.

**`pendingRetry` persistence**: `pendingRetry` entries are included in `flushState()` — written to `pending-retry.json` alongside orders. On crash recovery, `loadOrders()` also reads `pending-retry.json` and repopulates the in-memory map. Without this, a crash loses retry intent: an `"active"` stage with no live session and no pending retry would become stuck.

**Crash safety:** Write-before-dispatch is preserved — `flushOrders()` is called before `Dispatch()`. This means the stage is marked `"active"` on disk before the session starts. On crash after dispatch but before cycle-end flush, the worst case is that a completion isn't recorded — the stage is still `"active"` on disk, and `Runtime.Recover()` (phase 3) finds the orphaned session. Non-idempotent side effects (backlog `done` callback) are guarded: mark the order as completed in memory first, flush, then call the callback. If the callback fails, the order is already marked done and won't re-trigger.

**`flushState()` atomicity**: Each file is written via write-to-temp + rename (atomic replace). The files (`orders.json`, `pending-review.json`, `failed.json`, `pending-retry.json`) are written sequentially in a fixed order. This isn't cross-file atomic, but crash at any point produces recoverable state: `loadOrders()` on restart re-derives consistency from the orders file (source of truth) plus `Runtime.Recover()`.

**Internal sequencing**: (a) Add in-memory orders field + `loadOrders()` + `flushOrders()` with write-rename; (b) migrate `advanceOrder()`/`failStage()` to in-memory mutation; (c) migrate control commands to in-memory mutation; (d) add `pendingRetry` persistence; (e) add `flushState()` writing all files at cycle end.

## Data structures

- No new types. `orderx.OrdersFile` is the existing type, just held in memory instead of read per-operation.
- `flushState()` writes `orders.json`, `pending-review.json`, `failed.json`, and `pending-retry.json` via write-to-temp + rename. Orders file is written first (source of truth).

## Routing

- Provider: `claude` | Model: `claude-opus-4-6` — crash safety reasoning, control command migration, and side-effect ordering require careful analysis

## Verification

### Static
- `go test ./...` — all tests pass
- No `readOrders()` calls remain in control commands (only startup + `consumeOrdersNext`)
- Control command tests verify they mutate in-memory state correctly
- `writeOrdersAtomic()` called in `flushOrders()` and `flushBeforeDispatch()` only

### Runtime
- Integration test: dispatch 5 stages in one cycle, verify only 2 writes to orders.json (1 pre-dispatch flush, 1 cycle-end flush)
- Kill loop mid-cycle, restart, verify orders recover correctly (stages either pending or adopted)
- Race detector: `go test -race ./loop/...`
- Test: `consumeOrdersNext()` merges into in-memory state and triggers a flush
- Test: control `enqueue` command mutates in-memory state, visible in next cycle's dispatch
- Test: pending-review.json and failed.json are re-derived from orders.json after crash recovery
- Test: `"active"` stage with no live session resets to `"pending"` on startup (pendingRetry recovery)
- Test: crash between writing orders.json and pending-review.json — startup re-derives pending reviews from orders
