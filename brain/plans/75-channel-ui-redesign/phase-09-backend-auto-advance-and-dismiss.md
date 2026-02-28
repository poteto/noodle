Back to [[plans/75-channel-ui-redesign/overview]]

# Phase 9: Backend — Auto-advance and Dismiss

## Goal

Mechanical stages (schedule, quality, etc.) should auto-advance when they complete — no human intervention needed. Execute stages require explicit dismiss (human reviews and merges/rejects). This distinction partially exists today but is incomplete: ALL non-schedule stages get parked in approve mode, even non-mergeable ones like quality/reflect.

## Skills

Invoke `go-best-practices` before starting.

## Routing

- **Provider:** `claude` | **Model:** `claude-opus-4-6`
- Lifecycle policy change, needs judgment on classification and flow

## What already exists (do NOT reimplement)

- `schedule` is already special-cased to auto-advance on success regardless of autonomy mode (`cook_completion.go:124-129`)
- In approve mode, successful non-schedule stages are parked for review BEFORE quality-gate/mergeability checks (`cook_completion.go:135`)
- In auto mode, mergeable stages go through merge metadata + merge (inline or queued) before advancing (`cook_completion.go:162-173`)
- Non-mergeable stages advance directly in auto mode (`cook_completion.go:157`)
- The approve-mode quality gate also runs in `controlMerge` (`control.go:275`), not just `handleCompletion`
- `CanMerge` is resolved from skill frontmatter via registry metadata (`cook_merge.go:97`, `taskreg/registry.go`)
- **`pending_reviews` already exists** — `pending_review.go` tracks parked cooks in `pending-review.json`, snapshot exposes them as `pending_reviews` and `pending_review_count`

## Key insight from review

**Stage classification comes from `canMerge` metadata — but the default matters.** `skill/frontmatter.go:33` defaults `CanMerge()` to `true` when `permissions.merge` is omitted. Currently, schedule/quality/reflect all omit this field, so they resolve as mergeable — which is wrong. They don't produce worktrees that need review.

**Fix the data, not the logic.** Add explicit `permissions.merge: false` to schedule, quality, and reflect skill frontmatter. This is the correct fix per fix-root-causes: the problem is missing metadata, not missing classification logic.

**Don't add a `"review"` stage status.** Three independent reviewers confirmed this cascades into 6+ call sites: `ValidateStageStatus` (`orderx/types.go:74`), `advanceOrder` (`orders.go:127` — only advances `active|merging|pending`), `busyTargets` (`orders.go:73`), `dispatchableStages` (`orders.go:293`), and the frontend `StageStatus` enum (`ui/src/client/enums.ts:9`). The existing `pending_reviews` mechanism already solves this — the channel UI derives review state from the snapshot's `pending_reviews` field.

## Changes

### Modify
- `.agents/skills/schedule/SKILL.md` — add `permissions.merge: false` to noodle frontmatter
- `.agents/skills/quality/SKILL.md` — add `permissions.merge: false` to noodle frontmatter
- `.agents/skills/reflect/SKILL.md` — add `permissions.merge: false` to noodle frontmatter
- `loop/cook_completion.go` — in `handleCompletion`, change the approve-mode path: instead of parking ALL non-schedule stages, only park stages where `canMerge = true`. Non-mergeable stages auto-advance even in approve mode. This is a small change to the existing conditional.

### Feed events

`readLoopEvents` in `internal/snapshot/snapshot.go:458` currently maps only 5 loop event types and drops the rest. The channel UI needs lifecycle transitions in the scheduler feed. Map ALL defined loop event types from `event/loop_event.go:17`:

- `stage.completed` — label: "Completed", category: "stage_lifecycle"
- `stage.failed` — label: "Failed", category: "stage_lifecycle"
- `order.completed` — label: "Completed", category: "order_lifecycle"
- `order.failed` — label: "Failed", category: "order_lifecycle"
- `order.requeued` — label: "Requeued", category: "order_lifecycle"
- `quality.written` — label: "Quality", category: "quality"
- `schedule.completed` — label: "Scheduled", category: "schedule"
- `worktree.merged` — label: "Merged", category: "merge"
- `merge.conflict` — label: "Conflict", category: "merge"

Also: populate `FeedEvent.task_type` (currently always empty per `snapshot.go` constructors) and normalize scheduler feed identity — steer events use `session_id="chef"`, loop events use `session_id="loop"`. Pick one canonical scheduler channel ID.

### What NOT to change
- Don't add a `StageKind` type or hardcoded key-to-kind mapping
- Don't add a `"review"` stage status — use existing `pending_reviews`
- Don't touch the `controlMerge` path — it already handles approve-mode quality gating correctly
- Don't touch the schedule special case — it already works
- Don't touch `advanceOrder` — it already handles the relevant stage statuses

## Data Structures

- No new types or statuses
- Skill frontmatter gains explicit `permissions.merge: false` where appropriate

## Verification

### Static
- `go test ./...` passes (including new tests)

### Tests
- Unit test: non-mergeable stage (`canMerge=false`) auto-advances in approve mode (not parked for review)
- Unit test: mergeable stage (`canMerge=true`) is parked for review in approve mode
- Unit test: all stages auto-advance in auto mode (existing behavior preserved)
- Unit test: schedule, quality, reflect skills resolve `CanMerge() = false` after frontmatter fix
- Unit test: `readLoopEvents` maps all defined loop event types to feed events
- Unit test: `FeedEvent.task_type` is populated

### Runtime
- Submit an order with schedule → execute → quality pipeline in approve mode
- Schedule auto-advances (pre-existing)
- Execute parks for review (mergeable, approve mode)
- Quality auto-advances (non-mergeable, approve mode) — this is the NEW behavior
- Full pipeline completes with only one human touchpoint (execute review)
