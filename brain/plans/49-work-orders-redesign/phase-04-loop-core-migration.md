Back to [[plans/49-work-orders-redesign/overview]]

# Phase 4: Loop core migration

## Goal

Swap the loop from reading `queue.json` (flat items) to reading `orders.json` (orders with stages). This is the big-bang phase — dispatch, completion, retry, pending review, queue audit, reconciliation, and compilation stubs for control/schedule all switch together. Everything that touches `QueueItem` in the hot path must move at once because they share the `activeCook` struct.

## Changes

**`loop/types.go`** — Update `activeCook`, `pendingReviewCook`, `pendingRetryCook`:
- Replace `queueItem QueueItem` with `orderID string` + `stageIndex int` + `stage Stage`
- Add order-level fields needed at completion time (`plan []string` for adapter callbacks)

**`loop/loop.go`** — `prepareQueueForCycle()` → `prepareOrdersForCycle()`:
- Call `consumeOrdersNext()` instead of `consumeQueueNext()`
- Call `readOrders()` instead of `readQueue()`
- Normalize and validate orders instead of queue items
- Filter stale schedule orders (same logic, new types)
- Bootstrap schedule order if no active orders exist

**`loop/loop.go`** — `planCycleSpawns()`:
- Call `dispatchableStages()` (from phase 3) instead of iterating queue items
- Build spawn plan from `dispatchCandidate` list

**`loop/cook.go`** — `spawnCook()`:
- Takes `dispatchCandidate` (or order + stage + index) instead of `QueueItem`
- `buildCookPrompt()` reads from stage fields + order-level plan/rationale
- `cookBaseName()` derives name from `orderID-stageIndex-taskKey` (e.g. `29-0-execute`, `29-1-quality`). Include stage index to avoid collisions if an order ever has repeated task keys.
- `activeCook` stores orderID, stageIndex, stage

**`loop/cook.go`** — `handleCompletion()`:
- On success: call `advanceOrder()` (from phase 3), persist with `writeOrdersAtomic()`
- If order has more stages after advancement, they'll be dispatched next cycle
- On failure: call `failOrder()` if retries exhausted, or retry the same stage
- Merge path: check `canMerge` from the task type registry (same as current code). Mergeable stages merge their worktree, then advance. Non-mergeable stages (debate, schedule) skip merge and just advance. Only call `Adapter.Run("backlog", "done", orderID)` when the final stage of the order completes — not per-stage.
- Schedule special case: schedule stages have no worktree (run in project dir). handleCompletion must detect schedule and skip merge/worktree cleanup, same as current `isScheduleItem` check.

**`loop/cook.go`** — `retryCook()`:
- Retry dispatches the same stage (same orderID, same stageIndex) with incremented attempt
- Pending retry stores orderID + stageIndex + stage

**`loop/cook.go`** — `collectCompleted()`:
- Maps session ID → activeCook unchanged (activeCook struct just has different fields)

**`loop/pending_review.go`** — `PendingReviewItem`:
- Replace QueueItem-mirror fields with orderID + stageIndex + stage fields
- `parkPendingReview()` copies from activeCook's new shape
- `loadPendingReview()` deserializes the new shape. If parsing fails (old-format file from pre-upgrade), log a warning and discard — the worktree still exists for manual resolution. No backward-compat shim.

**`loop/util.go`**:
- `buildCookPrompt()` takes stage + order-level context instead of QueueItem
- `cookBaseName()` takes orderID + stageIndex + stage.TaskKey
- Delete `findQueueItemByTarget()` — replaced by order lookup

**`loop/queue_audit.go`** → `auditOrders()`:
- `auditQueue()` is called from `rebuildRegistry()` during every loop cycle — cannot be deferred. Migrate now.
- Iterate orders, for each order iterate stages, validate stage task types against registry
- Drop orders where no stages resolve. Log `order_drop` events to `queue-events.ndjson`.

**`loop/reconcile.go`** — Update adopted-session recovery:
- Current code parses cook prompts to re-associate sessions after restart (e.g. `work backlog item <id>`)
- `buildCookPrompt()` format changes in this phase — the parser must match the new format
- If prompt parsing fails, sessions become orphaned and may cause duplicate respawns

**`internal/taskreg/registry.go`** — Rename registry APIs:
- `QueueItemInput` → `StageInput`
- `ResolveQueueItem()` → `ResolveStage()`
- Update all callers (queuex, loop, audit)

**`loop/control.go`** — Minimal compilation stubs:
- Control commands must compile against the new types. Add stub implementations that read/write `orders.json` instead of `queue.json`. Full logic refinement happens in phase 5.

**`loop/schedule.go`** — Minimal compilation stubs:
- Schedule functions must compile against the new types. Stub `bootstrapScheduleOrder()`, `isScheduleOrder()`, etc. Full logic refinement happens in phase 6.

**`loop/queue.go`** — Keep for now (deleted in phase 9). Stop calling old functions.

## Data structures

- `activeCook{orderID, stageIndex, stage, session, worktreeName, worktreePath, attempt, displayName}`
- `activeByTarget` keyed by `orderID` (one cook per order at a time)

## Routing

| Provider | Model |
|----------|-------|
| `claude` | `claude-opus-4-6` |

Complex migration with judgment calls about edge cases and state transitions.

## Verification

### Static
- `go build ./...` and `go vet ./...` pass — the entire `loop/` package must compile, including control.go and schedule.go (via stubs)
- No remaining references to `QueueItemInput` or `ResolveQueueItem` in taskreg

### Runtime
- Unit test: spawnCook with a dispatchCandidate creates activeCook with correct orderID/stageIndex
- Unit test: handleCompletion on success advances order stage, order persisted
- Unit test: handleCompletion on success of final stage marks order completed
- Unit test: handleCompletion on failure retries same stage with incremented attempt
- Unit test: handleCompletion on failure with exhausted retries calls failOrder
- Run `go test ./loop/...` — existing tests will break and must be updated in this phase to use the new types
