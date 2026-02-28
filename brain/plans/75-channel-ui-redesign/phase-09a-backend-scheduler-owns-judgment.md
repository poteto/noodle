Back to [[plans/75-channel-ui-redesign/overview]]

# Phase 9: Backend — Scheduler Owns Judgment

## Goal

Move three categories of hardcoded judgment from the Go loop to the scheduler agent. The Go loop becomes dumb plumbing that reports events; the scheduler — visible in the chat UI — makes decisions the user can see and influence.

**Failures become conversations.** Today a stage failure silently triggers an OnFailure pipeline or terminates the order. The user sees an opaque status change. With this phase, the scheduler gets the failure in its chat, explains what happened, and decides — retry, add an oops stage, or abort. The user can jump in and steer recovery.

**Merge conflicts get explained.** Today a merge conflict silently parks for review. The user has to dig into the worktree to understand what conflicted. With this phase, the scheduler explains the conflict and offers resolution strategies in the chat.

**Retries are contextual.** Today the system retries up to 3 times regardless of whether retrying makes sense. A flaky network error and a fundamental design mistake get the same treatment. With this phase, the scheduler reads the failure, decides if retrying will help, and tells the user what it's doing.

Depends on phase 8 (stage message mechanism).

## Prerequisites

**Persistent scheduler with direct message path.** This phase forwards events to the scheduler via `controller.SendMessage()` — the direct message path established in phase 2. The current `steer("schedule", ...)` path in `cook_steer.go:41` calls `rescheduleForChefPrompt`, which rewrites `orders.json` — it does NOT message a live session. Phase 2 must provide: (1) persistent scheduler session, (2) direct message delivery bypassing `rescheduleForChefPrompt`.

## Skills

Invoke `go-best-practices` before starting.

## Routing

- **Provider:** `claude` | **Model:** `claude-opus-4-6`
- Lifecycle policy change with failure semantics, needs careful reasoning

## Subtract

### Failure pipeline management
- `loop/orders.go` — rewrite `failStage` (lines 162-200). Currently: marks stage failed, then either transitions to OnFailure pipeline or removes the order entirely. New behavior: marks stage failed, persists — does NOT remove the order, does NOT transition to OnFailure. The order stays in the orders file with a failed stage until the scheduler decides.
- `loop/cook_completion.go` — remove `failAndPersist` (line 252-270) calling `failStage`. Replace with: mark stage failed, persist, emit `stage.failed`, forward to scheduler.
- `loop/cook_completion.go` — remove OnFailure completion interpretation (lines 236-237).

### All `failStage` call sites (4 total)
1. `loop/cook_completion.go:258` — `failAndPersist` helper (removed above)
2. `loop/control.go:284` — quality verdict gate in `controlMerge` (already removed in phase 8)
3. `loop/control.go:424` — `controlRequestChanges` calls `failStage` for rejected reviews. Update to use new simplified failure path (mark failed, persist, no OnFailure routing).
4. `loop/reconcile.go:217` — `failMergingStage` for crash recovery of stuck "merging" stages. Update to use new simplified failure path + forward to scheduler.

### OnFailure pipeline
- `internal/orderx/types.go` — remove `OnFailure []Stage` field (line 52) from Order struct. `ParseOrdersStrict` uses `DisallowUnknownFields()` (`internal/orderx/orders.go:39`), so any persisted `orders.json` still containing `on_failure` will fail to parse. Per CLAUDE.md: no backward compatibility — drain active orders before deploying.
- `loop/cook_merge.go:70-71` — remove `isOnFailure` branch that selects OnFailure stages
- `loop/reconcile.go:87-112` — remove `isOnFailure` tracking in crash recovery scan
- `loop/types.go:93,115,139` — remove `isOnFailure`/`IsOnFailure` fields from cook identity types
- `loop/cook_steer.go:127` — remove `IsOnFailure` from steer context

### Merge conflict auto-parking
- `loop/cook_merge.go` — remove auto-parking on merge conflict (lines 135-155) and the `parkPendingReview` call (line 146). Replace with: emit `merge.conflict` loop event, forward to scheduler. The scheduler decides (retry with rebase, park for review, abort).

### Hardcoded retry limits
- `loop/cook_retry.go` — remove `retryCook` function's max retry threshold (lines 139-147) and automatic retry spawning. Remove retry deferral logic (lines 151-161) and `pendingRetry` state.
- `loop/cook_retry.go` — remove `processPendingRetries` (lines 85-129) and its call in the loop cycle. No automatic retry queue.
- Remove `pendingRetry` map from `cookTracker` and `writePendingRetry`/`loadPendingRetry` persistence.

### Pending review mechanism update
- `parkPendingReview` (`pending_review.go:77`) has exactly 2 callers — both being removed:
  1. `cook_completion.go:137` (approve-mode parking, removed in phase 8)
  2. `cook_merge.go:146` (merge conflict parking, removed above)
- Keep `pendingReview` map and `controlMerge`/`controlReject` resolution commands — the scheduler creates pending reviews via the new `park-review` control command (see Add section). The creation path changes, the resolution path stays.

## Add

### Control commands for scheduler decisions
Existing commands in `control.go:177-256` that the scheduler can already use: `steer` (message an agent), `skip` (cancel an order — calls `cancelOrder`), `kill`/`stop` (process management), `enqueue` (add new orders), `requeue` (re-add a failed order).

New commands the scheduler needs:

- **`advance`** — advance past a stage blocked by a `stage_message` (from phase 8). Takes `order_id`. Calls `advanceOrder` on the order, allowing the pipeline to continue. Currently no control command exposes `advanceOrder` externally.
- **`add-stage`** — insert a stage into an existing order's pipeline. Takes `order_id`, `task_key`, `provider`, `model`, `skill`. The scheduler uses this to dynamically add recovery stages (e.g., "oops" fix stage after a failure). Insert after the current active stage.
- **`park-review`** — park an order for human review. Takes `order_id`, `reason`. Calls `parkPendingReview` — this becomes the sole creation path for pending reviews, replacing the removed auto-parking callers.

## Modify

### Failure path
- `loop/cook_completion.go` — when a stage fails: mark stage as `failed`, persist orders, emit `stage.failed` loop event (already exists), forward failure details to scheduler via `controller.SendMessage()`. The failure message includes: order ID, stage task key, exit status, and the last few lines of agent output for context.
- If the scheduler is not alive or doesn't respond, the stage stays failed and the order stays in the orders file — safe default. No order removal. The scheduler or user can recover later via control commands.

### Merge conflict path
- `loop/cook_merge.go` — when merge conflicts: emit `merge.conflict` loop event (already exists), forward conflict details to scheduler via `controller.SendMessage()`. Message includes: order ID, worktree name, conflicted files. Scheduler decides: "retry with rebase," "park for human review," or "abort."

### Retry path
- Retries are now scheduler-initiated. The scheduler receives a failure message, decides to retry, and issues a control command (`add-stage` with the same task key, or `requeue`). The Go loop executes it mechanically.
- Remove the concept of "pending retry" as a loop-managed state.

### What NOT to change
- `advanceOrder` — still mechanical state transition (now exposed via `advance` control command)
- Merge execution — the Go loop still performs the actual git merge
- Process lifecycle — spawn, kill, watch are still Go's job
- Capacity limits — MaxCooks is infrastructure, not judgment
- `canMerge` — still determines whether a worktree exists to merge
- `controlMerge`, `controlReject` — still resolve pending reviews created by `park-review`

## Data Structures

- Remove `OnFailure []Stage` from `orderx.Order` and `OrderStatusFailing` from order statuses
- Remove `isOnFailure` from cook identity types (`cookHandle`, `pendingRetryCook`, `dispatchCandidate`)
- Remove `pendingRetry` map from `cookTracker` and pending retry persistence
- `StageFailedPayload` — add `AgentOutput string` field (last lines of agent output for scheduler context)
- `MergeConflictPayload` — add `ConflictedFiles []string` field
- `ControlCommand` — no struct changes needed (existing fields cover new commands: `Action`, `OrderID`, `TaskKey`, `Provider`, `Model`, `Skill`, `Prompt`)

## Verification

### Static
- `go test ./...` passes (including new tests)
- `go vet ./...` — no issues

### Tests
- Unit test: stage failure marks stage failed, forwards to scheduler, does NOT remove the order or trigger OnFailure
- Unit test: merge conflict forwards to scheduler, does NOT auto-park
- Unit test: no automatic retries — no `processPendingRetries`, scheduler issues control commands
- Unit test: if scheduler is not alive, failed stage and order stay in orders file (safe default)
- Unit test: `advance` control command calls `advanceOrder` on a blocked order
- Unit test: `add-stage` inserts a stage after the current active stage
- Unit test: `park-review` creates a pending review (sole creation path)
- Unit test: `controlRequestChanges` uses simplified failure path (no OnFailure routing)
- Unit test: `failMergingStage` crash recovery uses simplified failure path

### Runtime
- Stage fails → message appears in scheduler chat with failure details and agent output
- Scheduler responds in chat: "Adding oops stage to fix test failures" → `add-stage` command
- User sees the decision in the scheduler chat and can intervene
- Merge conflict → scheduler explains conflict in chat, offers options
- Scheduler retries a failed stage → `add-stage` with same task key, new cook spawns with context
- Scheduler parks for human review → `park-review`, user resolves via existing merge/reject commands
