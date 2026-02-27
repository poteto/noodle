Back to [[plans/70-scaling-the-loop-redesign/overview]]

# Phase 4: In-memory orders with periodic flush

## Goal

Hold orders in memory as the authoritative state during the cycle. Write to disk once per cycle (or on significant transitions) instead of on every stage status change. Migrate all order-mutating code paths — including control commands — to operate on in-memory state. Reduces disk I/O from 4-6 writes per stage to 1 write per cycle.

## Changes

**`loop/types.go`** — Add `orders orderx.OrdersFile` field to the Loop struct. This is the in-memory authoritative copy. Add explicit ownership rule: **only the loop goroutine mutates orders state**. Background goroutines (completion watchers, merge queue, health observers) communicate via channels only.

**`loop/orders.go`** — `advanceOrder()` and `failStage()` mutate the in-memory orders directly (no read-from-disk). New `flushOrders()` writes the in-memory state to disk atomically. New `loadOrders()` reads from disk into memory (startup only). `consumeOrdersNext()` merges into the in-memory copy and flushes.

**`loop/loop.go`** — At cycle end, call `flushOrders()` once. Also flush before dispatch (write-before-dispatch preserved for crash safety — see below). On startup, call `loadOrders()`. The fsnotify watch on `orders.json` is removed (the loop owns this file — current watch trigger at `loop/loop.go:213-216`). The fsnotify watch on `orders-next.json` and `control.ndjson` remains (external input). Document replacement trigger conditions: the cycle still runs on timer + fsnotify for external inputs + completions channel signal.

**`loop/cook.go`** — `spawnCook()` marks stage active in memory, flushes to disk, then dispatches. If dispatch fails, reverts in memory and flushes again. This preserves the write-before-dispatch invariant. `handleCompletion()` calls `advanceOrder()` or `failStage()` on the in-memory copy. **Critical state transitions flush immediately**: after a successful merge result advances a stage, flush before proceeding — a crash after merge but before flush would leave the stage `"active"` on disk with no session to recover, causing a re-dispatch of already-merged work. The rule: any transition that follows an irreversible side effect (merge, backlog callback) must flush before continuing.

**`loop/schedule.go`** — `spawnSchedule()` must also follow write-before-dispatch: mark the schedule stage active in memory, flush to disk, then dispatch. Without this, a crash after schedule dispatch but before cycle-end flush leaves a running schedule session with no `"active"` stage on disk — `Recover()` can't map the orphaned session back.

**`loop/control.go`** — Migrate **all** control commands to operate on in-memory orders instead of reading/writing `orders.json` directly. Full inventory by `applyControlCommand` switch (`control.go:140`):
- `controlMerge()` (`control.go:228`) — reads pending review state + orders
- `controlReject()` (`control.go:300`) — mutates pending review + orders
- `controlRequestChanges()` (`control.go:333`) — mutates pending review + orders
- `controlEnqueue()` (`control.go:390`) — appends new order
- `controlEditItem()` (`control.go:422`) — mutates stage fields via `activeStageForOrder`
- `controlSkip()` (`control.go:482`) — cancels remaining stages
- `controlRequeue()` (`control.go:498`) — resets failed order
- `controlReorder()` (`control.go:545`) — reorders queue
- `steer()` (`cook.go:743`) — kills session, mutates stage, respawns via `spawnCook` (indirect order mutation)
- `rescheduleForChefPrompt()` (`schedule.go:206`) — rewrites orders via schedule regeneration

Each command mutates `l.orders` in memory. Note: `steer` and `rescheduleForChefPrompt` are cross-file — track by action path through the switch, not just functions in `control.go`.

**Control command durability**: The control processing sequence must be: (1) process commands → mutate in-memory state, (2) flush state to disk, (3) ack + truncate `control.ndjson`. Never ack before flush — a crash after ack but before flush permanently loses accepted commands. The current code persists immediately per-command; the new code batches mutations but must flush before acking.

**Replay idempotency**: Not all commands are replay-safe. `controlEnqueue()` (`control.go:390`) appends a new order — replaying after crash duplicates orders. Fix: assign each control command a unique ID (already present in `ControlCommand.ID` field). Persist the last-processed command ID in `flushState()`. On startup, skip commands with IDs ≤ the last-processed ID. This makes replay safe for all commands including non-idempotent ones.

**`loop/pending_review.go`** — Pending review persistence (`pending-review.json`) is flushed alongside orders at cycle end. Both files are written atomically in `flushState()` to maintain consistency.

**`loop/failures.go`** — `failed.json` is flushed alongside orders at cycle end AND immediately when a failure occurs. The in-memory `failedTargets` map is authoritative, but failure metadata (error reason, session ID, timestamp) must survive crashes for `requeue` to work. Failure is a critical transition — flush immediately after recording it, same as merge advancement. Note: `failed.json` is NOT derivable from `orders.json` — terminal failures remove the order entirely (`failStage`, `loop/orders.go:150-152`) and failure reason survives only in `failed.json` (`loop/failures.go:43-64`). Treat `failed.json` as independently durable state.

**`pendingRetry` persistence**: `pendingRetry` entries are included in `flushState()` — written to `pending-retry.json` alongside orders. On crash recovery, `loadOrders()` also reads `pending-retry.json` and repopulates the in-memory map. Without this, a crash loses retry intent: an `"active"` stage with no live session and no pending retry would become stuck.

**Crash safety:** Write-before-dispatch is preserved — `flushOrders()` is called before `Dispatch()`. This means the stage is marked `"active"` on disk before the session starts. On crash after dispatch but before cycle-end flush, the worst case is that a completion isn't recorded — the stage is still `"active"` on disk, and `Runtime.Recover()` (phase 3) finds the orphaned session. Non-idempotent side effects (backlog `done` callback) are guarded: mark the order as completed in memory first, flush, then call the callback. If the callback fails, the order is already marked done and won't re-trigger.

**`flushState()` atomicity**: Each file is written via write-to-temp + rename (atomic replace). The files (`orders.json`, `pending-review.json`, `failed.json`, `pending-retry.json`, `last-command-id`) are written sequentially in a fixed order. This isn't cross-file atomic, but crash at any point produces recoverable state: `loadOrders()` on restart re-derives `pending-review.json` from orders (which stages are in review state). `failed.json` is independently durable (not derivable from orders — see failures.go note above). `pending-retry.json` is re-derived by checking `"active"` stages with no live session and no pending retry entry.

**Test hooks for crash safety**: Verification of crash-window tests requires deterministic mid-operation interruption. Add test seams: (a) a write barrier hook in `flushState()` that tests can inject to simulate crash between file writes; (b) an ack barrier hook in control processing that tests can inject to simulate crash between flush and ack. These are `func()` fields on the Loop struct, nil in production.

**Full mutation call-site inventory** (verify all switch to in-memory): `spawnCook` (`cook.go:105-149`), `advanceAndPersist` (`cook.go:310-334`), `failAndPersist` (`cook.go:338-359`), `steer` → `spawnCook` (`cook.go:743-782`), all control handlers listed above, `rescheduleForChefPrompt` (`schedule.go:206-213`), `consumeOrdersNext` (`orders.go`).

**Internal sequencing**: (a) Add in-memory orders field + `loadOrders()` + `flushOrders()` with write-rename; (b) add command ID tracking + `last-command-id` persistence; (c) migrate `advanceOrder()`/`failStage()` to in-memory mutation; (d) migrate control commands to in-memory mutation (inventory by `applyControlCommand` switch, including cross-file paths); (e) migrate schedule dispatch to write-before-dispatch; (f) add `pendingRetry` persistence; (g) add `flushState()` writing all files at cycle end; (h) add test barrier hooks for crash-window verification.

## Data structures

- No new types. `orderx.OrdersFile` is the existing type, just held in memory instead of read per-operation.
- `flushState()` writes `orders.json`, `pending-review.json`, `failed.json`, `pending-retry.json`, and `last-command-id` via write-to-temp + rename. Orders file is written first (source of truth). `failed.json` is independently durable.

## Routing

- Provider: `claude` | Model: `claude-opus-4-6` — crash safety reasoning, control command migration, and side-effect ordering require careful analysis

## Verification

### Static
- `go test ./...` — all tests pass
- No `readOrders()` calls remain in control commands (only startup + `consumeOrdersNext`)
- Control command tests verify they mutate in-memory state correctly
- `writeOrdersAtomic()` called in `flushOrders()` and `flushState()` only (no ad-hoc writes)
- Test barrier hooks are nil-checked in production paths

### Runtime
- Integration test: dispatch 5 stages in one cycle, verify only 2 writes to orders.json (1 pre-dispatch flush, 1 cycle-end flush)
- Kill loop mid-cycle, restart, verify orders recover correctly (stages either pending or adopted)
- Race detector: `go test -race ./loop/...`
- Test: `consumeOrdersNext()` merges into in-memory state and triggers a flush
- Test: control `enqueue` command mutates in-memory state, visible in next cycle's dispatch
- Test: pending-review.json is re-derived from orders.json after crash recovery (stages in review state)
- Test: failed.json survives crash independently — requeue works even if orders.json was written but failed.json wasn't
- Test: `"active"` stage with no live session resets to `"pending"` on startup (pendingRetry recovery)
- Test: crash between writing orders.json and pending-review.json — startup re-derives pending reviews from orders
- Test: control command ack happens after flush — crash between mutation and ack replays the command on restart
- Test: replayed `enqueue` command with already-processed ID is skipped (no duplicate order)
- Test: merge-advance flushes immediately — crash after merge but before cycle end does NOT leave stage "active"
- Test: failure flushes immediately — crash after failStage but before cycle end preserves failure metadata for requeue
- Test: schedule dispatch follows write-before-dispatch — crash after schedule dispatch recoverable via Recover()
