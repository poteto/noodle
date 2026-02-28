Back to [[plans/75-channel-ui-redesign/overview]]

# Phase 9: Backend — Auto-advance and Dismiss

## Goal

Mechanical stages (schedule, quality, etc.) should auto-advance when they complete — no human intervention needed. Execute stages require explicit dismiss (human reviews and merges/rejects). This distinction partially exists today but is incomplete and inconsistent.

## Skills

Invoke `go-best-practices` before starting.

## Routing

- **Provider:** `claude` | **Model:** `claude-opus-4-6`
- Lifecycle policy change, needs judgment on classification and merge flow

## What already exists (do NOT reimplement)

- `schedule` is already special-cased to auto-advance on success regardless of autonomy mode (`cook_completion.go:124-129`)
- In approve mode, successful non-schedule stages are parked for review BEFORE quality-gate/mergeability checks (`cook_completion.go:135`)
- In auto mode, mergeable stages go through merge metadata + merge (inline or queued) before advancing (`cook_completion.go:162-173`)
- Non-mergeable stages advance directly in auto mode (`cook_completion.go:157`)
- The approve-mode quality gate also runs in `controlMerge` (`control.go:275`), not just `handleCompletion`
- `CanMerge` is already resolved from skill frontmatter via registry metadata (`cook_merge.go:97`, `taskreg/registry.go`)

## Key insight from review

**Stage classification should NOT be a hardcoded key list.** Task keys are registry-driven — any skill with `noodle:` frontmatter becomes a task type (`taskreg/registry.go:32`). Beyond `schedule/quality/reflect/execute`, there are also `debate`, `oops`, `meditate`, and potentially user-defined keys. Hardcoding a list of "mechanical" keys is fragile and wrong.

Instead, derive the classification from existing registry metadata:
- **Mechanical:** stages where `CanMerge = false` (no worktree to review) AND the task is not `execute`-type. Schedule already has its own special case.
- **Interactive:** stages where `CanMerge = true` (has a worktree that needs review) OR the task is explicitly `execute`-type.

Actually, the simplest redesign: **the only stages that need human review are mergeable ones in approve mode.** Everything else auto-advances. This is nearly what the code does today — the gap is that ALL non-schedule stages get parked in approve mode, even non-mergeable ones like quality/reflect.

## Changes

### Modify
- `loop/cook_completion.go` — in `handleCompletion`, change the approve-mode path: instead of parking ALL non-schedule stages, only park stages where `canMerge = true`. Non-mergeable stages auto-advance even in approve mode. This is a small change to the existing conditional.
- `loop/pending_review.go` — add a `"review"` status to the stage when it's parked for review, so the UI has an explicit state instead of inferring from a side file. This makes the channel UI's review banner unambiguous.

### What NOT to change
- Don't add a `StageKind` type or hardcoded key-to-kind mapping — use existing `canMerge` metadata
- Don't touch the `controlMerge` path — it already handles approve-mode quality gating correctly
- Don't touch the schedule special case — it already works
- `advanceOrder` — no changes needed, it already handles the relevant stage statuses

## Feed events gap

`readLoopEvents` in `internal/snapshot/snapshot.go:431` currently maps only 4 loop event types (`order.dropped`, `registry.rebuilt`, `bootstrap.*`, `sync.degraded`) and drops the rest. The channel UI needs `stage.completed`, `order.completed`, `stage.failed`, `order.failed`, `quality.written`, and `schedule.completed` mapped to feed events. Add these mappings so the scheduler feed timeline shows lifecycle transitions.

## Data Structures

- `Stage.Status` — add `"review"` as a valid status (alongside pending/active/merging/completed/failed/cancelled)
- No new types — classification derived from existing `canMerge` boolean

## Verification

### Static
- `go test ./...` passes (including new tests)

### Tests
- Unit test: non-mergeable stage auto-advances in approve mode (not parked for review)
- Unit test: mergeable stage is parked for review in approve mode
- Unit test: all stages auto-advance in auto mode (existing behavior preserved)
- Unit test: `readLoopEvents` maps `stage.completed`, `order.completed` etc. to feed events

### Runtime
- Submit an order with schedule → execute → quality pipeline in approve mode
- Schedule auto-advances (pre-existing)
- Execute parks for review (mergeable, approve mode)
- Quality auto-advances (non-mergeable, approve mode) — this is the NEW behavior
- Full pipeline completes with only one human touchpoint (execute review)
