Back to [[archived_plans/66-event-trigger-system/overview]]

# Phase 4 — Emit Lifecycle Events: Completion and Merge

## Goal

Add event emission at the cook completion and merge paths. These are the highest-value events — they represent work finishing and code landing.

## Changes

- **`loop/cook_completion.go`**:
  - `advanceAndPersist` — after successful advance: emit `stage.completed` (order ID, stage index, task key, session ID if available). When order is removed (final stage): also emit `order.completed`.
  - `failAndPersist` — after recording failure: emit `stage.failed` (order ID, stage index, reason, session ID if available). When terminal: also emit `order.failed`.
  - `handleCompletion` — after quality verdict read: emit `quality.written` (order ID, session ID, accept bool, feedback).
  - `scheduleCompleted` — emit `schedule.completed` (session ID). This event's sequence becomes the watermark for the next mise brief.
- **`loop/cook_merge.go`**:
  - `mergeCookWorktree` — after successful merge: emit `worktree.merged` (order ID, stage index, worktree name).
  - `handleMergeConflict` — emit `merge.conflict` (order ID, stage index, worktree name).

## Data Structures

- Payload structs for each event type. Key design choice: `session_id` is `*string` (pointer, omitted when nil) because crash-recovery paths in reconcile (phase 5) advance stages without a session handle.

## Routing

Provider: `codex`, Model: `gpt-5.3-codex`

## Verification

```bash
go test ./loop/... && go vet ./...
```

- Cook completion writes `stage.completed` to `loop-events.ndjson`
- Final stage completion writes both `stage.completed` and `order.completed`
- Failed cook writes `stage.failed`; terminal failure also writes `order.failed`
- Quality verdict gate writes `quality.written`
- Successful merge writes `worktree.merged`; conflict writes `merge.conflict`
- Schedule completion writes `schedule.completed`
- All events have monotonically increasing sequences
