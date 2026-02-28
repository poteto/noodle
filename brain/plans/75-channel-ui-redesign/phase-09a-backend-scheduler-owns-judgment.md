Back to [[plans/75-channel-ui-redesign/overview]]

# Phase 9: Backend — Scheduler Owns Judgment

## Goal

Move three categories of hardcoded judgment from the Go loop to the scheduler agent. The Go loop becomes dumb plumbing that reports events; the scheduler — visible in the chat UI — makes decisions the user can see and influence.

**Failures become conversations.** Today a stage failure silently triggers an OnFailure pipeline or terminates the order. The user sees an opaque status change. With this phase, the scheduler gets the failure in its chat, explains what happened, and decides — retry, add an oops stage, or abort. The user can jump in and steer recovery.

**Merge conflicts get explained.** Today a merge conflict silently parks for review. The user has to dig into the worktree to understand what conflicted. With this phase, the scheduler explains the conflict and offers resolution strategies in the chat.

**Retries are contextual.** Today the system retries up to 3 times regardless of whether retrying makes sense. A flaky network error and a fundamental design mistake get the same treatment. With this phase, the scheduler reads the failure, decides if retrying will help, and tells the user what it's doing.

Depends on phase 8 (stage message mechanism).

## Skills

Invoke `go-best-practices` before starting.

## Routing

- **Provider:** `claude` | **Model:** `claude-opus-4-6`
- Lifecycle policy change with failure semantics, needs careful reasoning

## Subtract

### Failure pipeline management
- `loop/orders.go` — remove `failStage` OnFailure transition logic (lines 162-200). The Go loop no longer decides whether to run an OnFailure pipeline or terminate. It marks the stage as failed and reports to the scheduler.
- `loop/cook_completion.go` — remove `failAndPersist` calling `failStage`. Replace with: mark stage failed, persist, forward failure to scheduler via steer.
- `loop/cook_completion.go` — remove OnFailure completion interpretation (lines 236-237). The scheduler decides what a failure pipeline completion means.

### Merge conflict auto-parking
- `loop/cook_merge.go` — remove auto-parking on merge conflict (lines 135-155). Replace with: emit `merge.conflict` loop event, forward to scheduler via steer. The scheduler decides (retry with rebase, park for review, abort).

### Hardcoded retry limits
- `loop/cook_retry.go` — remove max retry threshold (lines 139-147) and automatic retry spawning. Retries are now initiated by the scheduler issuing a control command, not by the Go loop automatically.
- `loop/cook_retry.go` — remove retry deferral logic (lines 151-161). No automatic retry queue.

## Modify

### Failure path
- `loop/cook_completion.go` — when a stage fails: mark stage as `failed`, persist orders, emit `stage.failed` loop event (already exists), forward failure details to scheduler via `steer("schedule", failureMessage)`. The failure message includes: order ID, stage task key, exit status, and the last few lines of agent output for context.
- If the scheduler is not alive or doesn't respond, the stage stays failed — safe default. The scheduler adds recovery intelligence on top.

### Merge conflict path
- `loop/cook_merge.go` — when merge conflicts: emit `merge.conflict` loop event (already exists), forward conflict details to scheduler via steer. Message includes: order ID, worktree name, conflicted files. Scheduler decides: "retry with rebase," "park for human review," or "abort."

### Retry path
- Retries are now scheduler-initiated. The scheduler receives a failure message, decides to retry, and issues a control command (e.g., the existing steer respawn mechanism). The Go loop executes it mechanically.
- Remove the concept of "pending retry" as a loop-managed state. If the scheduler wants to retry later, it can re-issue the command.

### Control commands
- Ensure the scheduler has the control commands it needs: advance, cancel/abort an order, add a stage to an order. Some of these may already exist via `/api/control`. Audit what's available and add any missing commands the scheduler needs to express its decisions.

### What NOT to change
- `advanceOrder` — still mechanical state transition
- Merge execution — the Go loop still performs the actual git merge
- Process lifecycle — spawn, kill, watch are still Go's job
- Capacity limits — MaxCooks is infrastructure, not judgment
- `canMerge` — still determines whether a worktree exists to merge

## Data Structures

- Remove or simplify `OnFailure` pipeline from order spec — the scheduler dynamically adds recovery stages instead of pre-defining a failure pipeline
- Remove pending retry state from `cookTracker`
- `StageFailedPayload` already has `Reason` — add last agent output lines for scheduler context

## Verification

### Static
- `go test ./...` passes (including new tests)
- `go vet ./...` — no issues

### Tests
- Unit test: stage failure forwards to scheduler via steer, does NOT auto-trigger OnFailure
- Unit test: merge conflict forwards to scheduler via steer, does NOT auto-park
- Unit test: no automatic retries — scheduler must issue control command to retry
- Unit test: if scheduler is not alive, failed stage stays failed (safe default)
- Unit test: scheduler can issue advance/cancel/add-stage control commands

### Runtime
- Stage fails → message appears in scheduler chat with failure details
- Scheduler responds in chat: "Adding oops stage to fix test failures"
- User sees the decision in the scheduler chat and can intervene
- Merge conflict → scheduler explains conflict in chat, offers options
- Scheduler retries a failed stage → new cook spawns with context from the failure
