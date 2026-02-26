Back to [[plans/70-scaling-the-loop-redesign/overview]]

# Phase 7: Async merge queue

## Goal

Decouple worktree merges from the loop cycle. A background goroutine processes merges serially outside the cycle. The loop enqueues merge requests and drains results — merges no longer block stage advancement.

## Changes

**`loop/merger.go`** (new) — `MergeQueue` struct with `requests chan MergeRequest`, `results chan MergeResult`, and a single background goroutine. The goroutine dequeues requests, acquires the merge lock, performs the rebase+merge, and sends the result. For cloud runtimes that push branches rather than local worktrees, the merge queue handles `MergeRemoteBranch()` transparently.

**Shutdown**: `MergeQueue.Close()` signals the goroutine to stop. If the goroutine is blocked waiting for the merge lock, it must respect context cancellation. Add context-aware lock acquisition to `worktree/lock.go` — the lock `Acquire()` method accepts a context and returns early if cancelled, instead of sleeping in an uninterruptible retry loop.

**Crash recovery**: In-flight merge requests are not persisted. On crash, the stage is in `"merging"` state on disk. On restart, `loadOrders()` detects `"merging"` stages and applies a **deterministic** recovery algorithm (not heuristic):
1. Check if the worktree branch is already merged to main (`git merge-base --is-ancestor <branch> main`). If yes → advance the stage to `"completed"`.
2. If the worktree branch exists but is not merged → re-enqueue for merge (reset status to `"completed"`, enqueue `MergeRequest`).
3. If the worktree branch doesn't exist and no session is alive → fail the stage.
Each case maps to exactly one action — no ambiguity.

**`loop/cook.go`** — `handleCompletion()` enqueues a `MergeRequest` instead of calling `WorktreeManager.Merge()` directly. Stage status moves to `"merging"` intermediate state. When the merge result arrives (next cycle), the loop advances or fails the stage.

**`loop/loop.go`** — `runCycleMaintenance()` drains `MergeQueue.results` and processes them: successful merges → `advanceOrder()`, failed merges (conflict) → park in pending review or fail stage.

**`loop/types.go`** — Add `StageStatusMerging` as a typed constant (alongside existing status constants). Add MergeQueue to Dependencies. All stage status comparisons use typed constants, not bare strings.

**`loop/orders.go`** — Update stage state machine: `StageStatusActive` → `StageStatusMerging` → `StageStatusCompleted` or `StageStatusFailed`. `dispatchableStages()` must treat `StageStatusMerging` as busy (not dispatchable, not failed). `advanceOrder()` must handle the merging → completed transition. `failStage()` must handle merging → failed.

**`internal/orderx/queue.go`** — Add `StageStatusMerging` to valid stage statuses. Update validation to accept it.

**Shutdown quiescence extension**: The drain-exit condition from phase 2 is extended: `len(activeCooksByOrder) == 0 AND mergeQueue.Pending() == 0`. After producer count reaches zero and the merge queue is empty, do one final drain of merge results before exit.

**`worktree/lock.go`** — Add `AcquireContext(ctx context.Context)` that returns `context.Canceled` if the context is cancelled while waiting.

**Internal sequencing**: (a) Add `StageStatusMerging` typed constant + update validators in orderx; (b) implement `MergeQueue` with context-aware lock; (c) wire into cook.go completion path + loop.go drain; (d) implement deterministic crash recovery for merging stages; (e) extend shutdown quiescence to include merge queue.

## Data structures

- `MergeRequest` — `OrderID string`, `StageIndex int`, `WorktreeName string`, `IsRemoteBranch bool`, `BranchName string`
- `MergeResult` — `OrderID string`, `StageIndex int`, `Success bool`, `Error error`, `ConflictFiles []string`

## Routing

- Provider: `claude` | Model: `claude-opus-4-6` — merge queue goroutine lifecycle, context-aware locking, crash recovery for "merging" state, stage state machine changes

## Verification

### Static
- `go test ./...` — all tests pass
- `WorktreeManager.Merge()` not called directly from cook.go
- Merge lock acquisition only happens in merger.go goroutine
- `"merging"` handled correctly in `dispatchableStages()`, `advanceOrder()`, `failStage()`

### Runtime
- Integration test: complete 3 sessions simultaneously, verify merges are serialized but don't block the loop cycle
- Test: merge conflict parks order in pending review
- Test: remote branch merge (cloud runtime) flows through same queue
- Test: shutdown cancels merge lock wait within 5s
- Test: crash with stage in "merging" state → restart recovers correctly
- Race detector: `go test -race ./loop/...`
- Measure: cycle time with 3 concurrent merges should be <50ms (merges happen in background)
