Back to [[archived_plans/70-scaling-the-loop-redesign/overview]]

# Phase 7: Async merge queue

## Goal

Decouple worktree merges from the loop cycle. A background goroutine processes merges serially outside the cycle. The loop enqueues merge requests and drains results — merges no longer block stage advancement.

## Changes

**`loop/merger.go`** (new) — `MergeQueue` struct with a mutex-guarded request slice (not a channel), `results` queue, and a single background goroutine. **Deadlock prevention**: using a channel for requests is unsafe because `controlSetMaxCooks` can change concurrency at runtime (`loop/control.go:612-625`) and control commands are processed before completion handling (`loop/loop.go:283-288`), so a channel-capacity proof based on `MaxCooks` doesn't hold. Instead: the loop appends requests to a mutex-guarded slice (non-blocking), and the worker goroutine locks, swaps the slice, unlocks, then processes. Results use a mutex-guarded slice too (non-blocking append by worker, non-blocking drain by loop). The queue tracks `pending` and `inFlight` counts separately. The loop drains results before appending new requests each cycle. For cloud runtimes that push branches rather than local worktrees, the merge queue handles `MergeRemoteBranch()` transparently.

**Merge backpressure**: add `MergeBackpressureThreshold` config. When `pending + inFlight` exceeds threshold, planner suppresses new stage dispatches for that cycle (completions/merge processing still continue). This prevents unbounded merge backlog growth.

**Shutdown**: `MergeQueue.Close()` signals the goroutine to stop. If the goroutine is blocked waiting for the merge lock, it must respect context cancellation. Add context-aware lock acquisition to `worktree/lock.go` — the lock `Acquire()` method accepts a context and returns early if cancelled, instead of sleeping in an uninterruptible retry loop.

**Merge metadata persistence**: Before enqueueing a `MergeRequest`, persist merge metadata to the stage's `Extra` field in orders (worktree name, branch name, merge mode local/remote, merge generation token). This is flushed to disk as part of `flushState()`. Without persisted metadata, crash recovery can't determine what branch to check or re-merge.

**Crash recovery**: In-flight merge requests are not persisted (they're transient). On crash, the stage is in `"merging"` state on disk with merge metadata in `Extra`. On restart, `loadOrders()` detects `"merging"` stages and applies a **deterministic** recovery algorithm:
1. Read merge metadata from stage `Extra`. If missing → fail the stage (unrecoverable — shouldn't happen if persistence is correct).
2. Check if the worktree branch is already merged to main (`git merge-base --is-ancestor <branch> main`). If yes → advance the stage to `"completed"`.
3. If the worktree branch exists but is not merged → keep status as `"merging"` and re-enqueue a `MergeRequest`. Do NOT set to `"completed"` — that would allow the next stage to dispatch before merge finishes, violating order sequencing.
4. If the worktree branch doesn't exist and no session is alive → fail the stage.
5. If a session is still alive for this stage (recovered via `Runtime.Recover()`) → keep status as `"active"` (not `"merging"` — the session hasn't completed yet). This handles the edge case where the loop crashed between dispatch and completion.
6. If a pending review exists for this stage (`pending-review.json`) → keep as `"merging"` but route to pending review on merge result (conflict or review-required).
Each case maps to exactly one action — no ambiguity.

**`loop/cook.go`** — `handleCompletion()` enqueues a `MergeRequest` instead of calling `WorktreeManager.Merge()` directly. Stage status moves to `"merging"` intermediate state. Each request carries `MergeGeneration` copied from the active handle generation for that stage attempt. When the merge result arrives, apply it only if generation matches the current stage metadata; stale results are discarded.

**`loop/loop.go`** — `runCycleMaintenance()` drains `MergeQueue.results` and processes them: successful merges → `advanceOrder()` + immediate flush, failed merges (conflict) → park in pending review or fail stage.

**`loop/control.go`** — `controlMerge()` (manual merge via control command) must enqueue through the `MergeQueue` instead of calling `WorktreeManager.Merge()` directly. This ensures manual merges respect the same serialization as automatic merges. Without this, manual and automatic merges can race on the merge lock and produce inconsistent state transitions.

**`loop/types.go`** — Add `StageStatusMerging` as a typed constant (alongside existing status constants). Add MergeQueue to Dependencies. All stage status comparisons use typed constants, not bare strings.

**`loop/orders.go`** — Update stage state machine: `StageStatusActive` → `StageStatusMerging` → `StageStatusCompleted` or `StageStatusFailed`. `dispatchableStages()` (`orders.go:208`) must treat `StageStatusMerging` as busy (not dispatchable, not failed). `activeStageForOrder()` (`orders.go:38`) must recognize `"merging"` as an active-like state. `advanceOrder()` must handle the merging → completed transition. `failStage()` must handle merging → failed.

**`internal/orderx/queue.go`** — Add `StageStatusMerging` to valid stage statuses (`queue.go:68-77`). Update validation to accept it.

**`ui/src/client/types.ts`** — Add `"merging"` to the `StageStatus` type union (currently `"pending" | "active" | "completed" | "failed" | "cancelled"` at line 50).

**Full stage-status consumer inventory** (all must handle `"merging"`):
- `dispatchableStages` (`loop/orders.go:208`)
- `activeStageForOrder` (`loop/orders.go:40`)
- `controlEditItem` (`loop/control.go:445`)
- Stage status validators (`loop/types.go:88-98`, `internal/orderx/queue.go:68-77`)
- UI type union (`ui/src/client/types.ts:50`)
- Snapshot/board components that render stage status

**`internal/orderx/queue.go`** — Add `StageStatusMerging` to valid stage statuses. Update validation to accept it.

**Shutdown quiescence extension**: The drain-exit condition from phase 2 is extended: `watcherWG` signals done AND `mergeQueue.Pending() == 0` AND `mergeQueue.InFlight() == 0`. After both pending and in-flight merges reach zero, do one final drain of merge results before exit.

**`worktree/lock.go`** — The current `acquireMergeLock()` (`lock.go:91`) is called from `Merge()` and `MergeRemoteBranch()` (`commands.go:83,164`) with no context parameter. Add context-aware variants: either `MergeContext(ctx, name)` / `MergeRemoteBranchContext(ctx, name)` that thread context through to `acquireMergeLock`, or refactor `acquireMergeLock` to accept context directly. The merge queue worker passes its context so shutdown cancellation unblocks lock acquisition. Update `loop.WorktreeManager` interface if it exists to include the context-aware methods.

**Internal sequencing**: (a) Add `StageStatusMerging` typed constant + update validators in orderx + update TS type union; (b) add merge metadata persistence to stage `Extra`; (c) implement `MergeQueue` with mutex-guarded slices + context-aware lock; (d) wire into cook.go completion path + loop.go drain; (e) wire `controlMerge()` through queue; (f) update all stage-status consumers; (g) implement deterministic crash recovery for merging stages; (h) extend shutdown quiescence to include merge queue.

## Data structures

- `MergeRequest` — `OrderID string`, `StageIndex int`, `WorktreeName string`, `IsRemoteBranch bool`, `BranchName string`, `MergeGeneration uint64` (all fields also persisted in stage `Extra` for crash recovery)
- `MergeResult` — `OrderID string`, `StageIndex int`, `MergeGeneration uint64`, `Success bool`, `Error error`, `ConflictFiles []string`

## Routing

- Provider: `claude` | Model: `claude-opus-4-6` — merge queue goroutine lifecycle, context-aware locking, crash recovery for "merging" state, stage state machine changes

## Verification

### Static
- `go test ./...` — all tests pass
- `WorktreeManager.Merge()` not called directly from cook.go or control.go
- Merge lock acquisition only happens in merger.go goroutine (including manual merge path)
- `controlMerge()` enqueues through MergeQueue, not direct merge call
- `"merging"` handled correctly in `dispatchableStages()`, `activeStageForOrder()`, `advanceOrder()`, `failStage()`, `controlEditItem()`
- `StageStatus` TS type includes `"merging"`
- Merge metadata persisted in stage `Extra` before enqueue

### Runtime
- Integration test: complete 3 sessions simultaneously, verify merges are serialized but don't block the loop cycle
- Test: merge conflict parks order in pending review
- Test: remote branch merge (cloud runtime) flows through same queue
- Test: shutdown cancels merge lock wait within 5s
- Test: crash with stage in "merging" state → restart recovers correctly
- Race detector: `go test -race ./loop/...`
- Measure: cycle time with 3 concurrent merges should be <50ms (merges happen in background)
- Test: mutex-guarded slices — no blocking on enqueue or drain regardless of concurrency level
- Test: `controlSetMaxCooks` changes concurrency mid-flight — no deadlock
- Test: manual merge via control command goes through queue, serialized with automatic merges
- Test: crash recovery — stage in "merging" with merge metadata in Extra → branch merged → advances to "completed"
- Test: crash recovery — stage in "merging" with merge metadata → branch exists but not merged → re-enqueues, stays "merging"
- Test: crash recovery — stage in "merging" but live session found via Recover() → resets to "active"
- Test: crash recovery — stage in "merging" with no merge metadata → fails the stage
- Test: stale merge result (generation N) arrives after replacement merge (generation N+1) — stale result is ignored
- Test: shutdown waits for `Pending()==0` and `InFlight()==0` before final drain
- Test: merge backlog above `MergeBackpressureThreshold` suppresses new dispatches until queue depth drops
