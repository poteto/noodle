Back to [[archived_plans/42-requires-approval-gate/overview]]

# Phase 2: Remove Hardcoded Quality from Loop

## Goal

Delete `loop/quality.go` and rewire `handleCompletion` to use the task type's `CanMerge` field from the registry instead of the hardcoded `runQuality` / `reviewEnabled` flow.

## Changes

### Delete `loop/quality.go`

Remove entirely: `runQuality`, `readQualityVerdictFile`, `copyVerdictToRuntime`, `writeDebateVerdict`. These functions are the heart of the hardcoded quality flow.

Verdicts are removed as a noodle-managed concept. Users who want review state can implement it in their skills (write state files to `.noodle/`, have prioritize read them). Noodle itself does not consume or produce verdict files.

### `loop/cook.go` — `handleCompletion`

Current flow:
1. If success AND `reviewEnabled` -> `runQuality()` -> accept/reject
2. If success AND `isPrioritizeItem` -> `skipQueueItem`
3. If success AND `PendingApproval()` -> park in `pendingReview`
4. Otherwise -> `mergeCook()`

New flow:
1. If success AND `isPrioritizeItem` -> `skipQueueItem` (preserve existing bypass — prioritize has no worktree to merge)
2. If success -> look up `TaskType.CanMerge` from registry
3. If `!CanMerge` OR `config.PendingApproval()` -> park in `pendingReview`
4. Otherwise -> `mergeCook()`

The `runQuality` step is gone. The review skill is just another schedulable task now. The `isPrioritizeItem` bypass must be preserved — prioritize cooks have no worktree name and `mergeCook` would crash.

### `mise/` — Remove verdict types and readers

Delete `mise.QualityVerdict` type, `mise.ReadQualityVerdicts`, and any verdict-related fields from `mise.Brief`. The mise builder currently reads `.noodle/quality/` to include historical quality signals — remove this. Verdicts are a userland concept now.

### `loop/cook.go` — `spawnCook`

Remove the `reviewEnabled` field setup (lines 33-36). Remove it from `activeCook` struct.

### `loop/types.go` — `activeCook`

Remove the `reviewEnabled bool` field.

### `loop/types.go` — `QueueItem.Review`

The `QueueItem.Review` bool is a competing approval model — the prioritize skill could mark individual items for review. With `permissions.merge`, approval is per-task-type, not per-queue-item. Remove the `Review` field from `QueueItem` and from `internal/queuex/queue.go` (its canonical twin).

Update all callers:
- `loop/cook.go` — `spawnCook` reads it to set `reviewEnabled` (being removed)
- `internal/schemadoc/specs.go` — remove `items[].review` field doc, update constraints that reference `"review": true`, `execute -> quality (blocking) -> reflect` workflow, and any `blocking` references
- `loop/prioritize.go` — update `buildPrioritizePrompt` which mentions `quality, reflect, meditate` in the synthesize instruction
- `.agents/skills/prioritize/SKILL.md` — remove instructions to set `"review": true`

### `loop/cook.go` — `buildAdoptedCook`

Remove `reviewEnabled` logic (lines 281-286).

### `loop/loop.go` — `planCycleSpawns`

Add `pendingReview` targets to the spawn plan filter. Currently `planCycleSpawns` only filters `activeByTarget`, `failedTargets`, and `adoptedTargets`. Without adding `pendingReview`, parked items will be re-dispatched on the next cycle. Build a `pendingTargets` set from `l.pendingReview` and pass it as `BusyTargets` (or a new filter set) in `spawnPlanInput`.

### `loop/loop_test.go`

Remove or update tests that:
- Assert `runQuality` is called
- Mock quality review verdicts
- Test `reviewEnabled` behavior
- Reference the quality skill by name

### `loop/fixture_test.go` and `loop/testdata/`

Update fixtures that test quality review behavior. Fixtures with quality review steps need to be rewritten to test the `!CanMerge` parking instead.

Check `make bugs` — some bug fixtures may reference quality behavior.

### Persist and rehydrate `pendingReview`

`pendingReview` currently lives only in memory — it's lost on loop restart. With `permissions.merge` as the primary approval mechanism, this must be durable. Persist pending items to a file (e.g. `.noodle/pending-review.json`) when items are parked, and rehydrate on startup in `loop/reconcile.go` alongside adopted session recovery. Without this, a loop restart silently drops all parked items.

### Loop state export for TUI

The TUI currently renders approval items from verdict files (`.noodle/quality/`) and `ActionNeeded`. After removing quality verdict production, there's no data path for the TUI to know about pending-approval items. The loop must export `pendingReview` state so the TUI can render it. The persistence file from above doubles as the TUI's data source — the snapshot loader reads it instead of verdict files. This is the bridge that makes Phases 5-6 (TUI approval flow) work.

## Data Structures

- `activeCook` loses `reviewEnabled bool`
- `pendingReviewCook` unchanged (already exists and works)

## Routing

Provider: `claude` | Model: `claude-opus-4-6` — judgment calls on test rewrites and flow changes.

## Verification

```sh
go test ./loop/...
make fixtures-loop MODE=check
make bugs  # confirm no new unexpected failures
```
