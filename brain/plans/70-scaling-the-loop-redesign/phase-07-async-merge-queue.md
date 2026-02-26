Back to [[plans/70-scaling-the-loop-redesign/overview]]

# Phase 7: Async merge queue

## Goal

Decouple worktree merges from the loop cycle. A background goroutine processes merges serially outside the cycle. The loop enqueues merge requests and drains results — merges no longer block stage advancement.

## Changes

**`loop/merger.go`** (new) — `MergeQueue` struct with `requests chan MergeRequest`, `results chan MergeResult`, and a single background goroutine. The goroutine dequeues requests, acquires the merge lock, performs the rebase+merge, and sends the result. For cloud runtimes that push branches rather than local worktrees, the merge queue handles `MergeRemoteBranch()` transparently.

**Shutdown**: `MergeQueue.Close()` signals the goroutine to stop. If the goroutine is blocked waiting for the merge lock, it must respect context cancellation. Add context-aware lock acquisition to `worktree/lock.go` — the lock `Acquire()` method accepts a context and returns early if cancelled, instead of sleeping in an uninterruptible retry loop.

**Crash recovery**: In-flight merge requests are not persisted. On crash, the stage is in `"merging"` state on disk. On restart, `loadOrders()` detects `"merging"` stages and resets them to `"completed"` (re-enqueue for merge) or derives the right action from the worktree state (if the worktree was already merged to main, advance; if not, re-enqueue).

**`loop/cook.go`** — `handleCompletion()` enqueues a `MergeRequest` instead of calling `WorktreeManager.Merge()` directly. Stage status moves to `"merging"` intermediate state. When the merge result arrives (next cycle), the loop advances or fails the stage.

**`loop/loop.go`** — `runCycleMaintenance()` drains `MergeQueue.results` and processes them: successful merges → `advanceOrder()`, failed merges (conflict) → park in pending review or fail stage.

**`loop/types.go`** — Add `"merging"` as a valid stage status in the order model. Add MergeQueue to Dependencies.

**`loop/orders.go`** — Update stage state machine: `"active"` → `"merging"` → `"completed"` or `"failed"`. `dispatchableStages()` must treat `"merging"` as busy (not dispatchable, not failed). `advanceOrder()` must handle the `"merging"` → `"completed"` transition. `failStage()` must handle `"merging"` → `"failed"`.

**`internal/orderx/queue.go`** — Add `"merging"` to valid stage statuses. Update validation to accept it.

**`worktree/lock.go`** — Add `AcquireContext(ctx context.Context)` that returns `context.Canceled` if the context is cancelled while waiting.

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
