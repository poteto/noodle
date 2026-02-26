Back to [[plans/59-subtract-go-logic-and-resilience/overview]]

# Phase 8: Merge conflicts to pending review

Covers: #63

## Goal

When a cook's worktree has merge conflicts, park it for pending review with a "merge conflict" reason instead of marking it as a permanent failure. The human can resolve the conflict and merge manually via the web UI.

## Changes

- `loop/pending_review.go` — Add `Reason string` field to `PendingReviewItem`. This surfaces why the item is parked (e.g., "merge conflict", "quality rejected", "approval required"). Existing pending reviews get an empty reason (backward-compatible — empty means normal completion review).
- `loop/types.go` (~line 79) — If `pendingReviewCook` (the in-memory type) is separate from `PendingReviewItem`, extend it with `Reason` too so the reason persists through the park→serialize flow.
- `loop/cook.go` (~lines 302-317) — Rewrite `handleMergeConflict()`: instead of calling `markFailed()` + `skipQueueItem()`, call a variant of `parkPendingReview()` that sets `Reason: "merge conflict: <details>"`. Keep the schedule item exemption (schedule merge conflicts still propagate as errors).
- `loop/loop_test.go` (~line 1146) — Rewrite `TestCycleMergeConflictMarksFailedAndSkips`: assert item parks for pending review with merge-conflict reason instead of being marked failed + skipped.
- `loop/pending_review_test.go` — Add `Reason` round-trip test: park with reason, serialize, load, assert reason preserved. Assert empty reason for normal completion review (backward compat).

## Data structures

`PendingReviewItem` gains `Reason string \`json:"reason,omitempty"\``

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Small behavioral change with clear spec |

## Verification

### Static
- `go test ./loop/...` passes
- `go vet ./...` clean
- New test: merge conflict on cook → item appears in pending review with reason containing "merge conflict"
- New test: merge conflict on cook → item NOT in failed targets
- New test: merge conflict on schedule item → error propagated (existing behavior preserved)
- Existing pending review tests still pass (Reason field is omitempty)

### Runtime
- Create a deliberate merge conflict (two branches editing same line), complete the cook → verify it appears in web UI pending review column with merge conflict reason
- Verify "merge" and "reject" actions still work on conflict-parked items
