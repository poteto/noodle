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
- **Simplify filtering (#60):** Don't port `filterStaleScheduleItems`/`hasNonScheduleItems` nested conditionals. Simplify to: if no non-schedule orders exist and work is available (plans or needs_scheduling), bootstrap a schedule order. If no work exists, go idle. Keep validation (reject malformed orders) but stop making scheduling judgments — the schedule skill owns queue composition.

**`loop/loop.go`** — `planCycleSpawns()`:
- Call `dispatchableStages()` (from phase 3) instead of iterating queue items
- Build spawn plan from `dispatchCandidate` list

**`loop/cook.go`** — `spawnCook()`:
- Takes `dispatchCandidate` (or order + stage + index) instead of `QueueItem`
- `buildCookPrompt()` reads from stage fields + order-level plan/rationale
- `cookBaseName()` derives name from `orderID-stageIndex-taskKey` (e.g. `29-0-execute`, `29-1-quality`). Include stage index to avoid collisions if an order ever has repeated task keys.
- `activeCook` stores orderID, stageIndex, stage

**`loop/cook.go`** — `handleCompletion()`:
- On success: check quality verdict before merging (see below), then call `advanceOrder()` (from phase 3), persist with `writeOrdersAtomic()`. `advanceOrder` returns `removed bool` — if true and order was `"active"`, fire adapter "done"; if true and order was `"failing"`, call `markFailed` instead.
- If `removed` is false, more stages remain — they'll be dispatched next cycle
- On failure: call `failStage()` if retries exhausted (which triggers OnFailure stages if present), or retry the same stage
- Merge path: check `canMerge` from the task type registry (same as current code). Mergeable stages merge their worktree, then advance. Non-mergeable stages (debate, schedule) skip merge and just advance. Only call `Adapter.Run("backlog", "done", orderID)` when the final stage of a **non-failing** order completes — not per-stage. For `"failing"` orders, when the last OnFailure stage completes, `advanceOrder` removes the order and the caller calls `markFailed` (the OnFailure pipeline is remediation, not recovery — the original failure stands). Do NOT fire adapter "done" for failing orders.
- Schedule special case: schedule stages have no worktree (run in project dir). handleCompletion must detect schedule and skip merge/worktree cleanup, same as current `isScheduleItem` check.
- **Quality verdict check (#65):** After a stage completes successfully but before merging, read `.noodle/quality/<session-id>.json`. If a verdict file exists and `accept == false`, treat the stage as failed (call `failStage()` instead of advancing). This makes quality verdicts enforceable. Add a `QualityVerdict` struct to `loop/types.go`: `{Accept bool, Feedback string}` (only read the fields the loop needs). Verdict reading is at the boundary — validate at read time, trust internally.
- **Pending approval interaction:** If `config.PendingApproval()` is true, park for review as before (human sees the verdict in the review UI). Verdict check only applies in `auto` autonomy mode where the loop would otherwise merge without human review.

**`loop/cook.go`** — `retryCook()`:
- Retry dispatches the same stage (same orderID, same stageIndex) with incremented attempt. If `IsOnFailure` is true, retry within the OnFailure pipeline.

**`loop/cook.go`** — `retryCook()` (already updated above).

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

- `activeCook{orderID, stageIndex, stage, isOnFailure, orderStatus, plan, session, worktreeName, worktreePath, attempt, displayName}`
- `activeByTarget` keyed by `orderID` (one cook per order at a time)
- `QualityVerdict{Accept bool, Feedback string}` — minimal struct for reading verdict files at the merge boundary

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
- Unit test: handleCompletion on success of final stage removes order from OrdersFile and fires adapter "done"
- Unit test: handleCompletion on failure retries same stage with incremented attempt
- Unit test: handleCompletion on failure with exhausted retries calls failStage — when terminal=false, order stays (OnFailure will dispatch); when terminal=true, calls markFailed
- Unit test: handleCompletion on final OnFailure stage completion calls markFailed (not adapter "done")
- Unit test: handleCompletion with quality verdict `accept=false` treats stage as failed (calls failStage)
- Unit test: handleCompletion with quality verdict `accept=true` proceeds normally
- Unit test: handleCompletion with no verdict file proceeds normally (verdict is optional)
- Unit test: handleCompletion in `approve` autonomy mode parks for review regardless of verdict
- Unit test: handleCompletion for schedule stage skips merge/worktree cleanup
- Unit test: loadPendingReview discards old-format files gracefully (logs warning, no crash)
- Unit test: auditOrders drops orders with unresolvable stages, emits `order_drop` event
- Unit test: reconcile re-associates adopted sessions using new prompt format
- Run `go test ./loop/...` — existing tests will break and must be updated in this phase to use the new types
