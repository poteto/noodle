Back to [[plans/113-deterministic-completion-detection/overview]]

# Phase 8: Cleanup Verification Gate

## Goal

After a session exits with "completed" status, verify that the claimed deliverable was actually produced. A session that reports success but produced no artifact should be downgraded to failed.

## Changes

**`loop/cook_completion.go`** — update `handleCompletion()`:
- After determining `StageResultCompleted`, check `SessionOutcome.HasDeliverable`
- If `HasDeliverable` is false AND no `stage_yield` was emitted → downgrade to `StageResultFailed` with reason "completed with no deliverable"
- **Do NOT check worktree git changes** — many valid stages (review, quality, reflect) intentionally produce no diff. The existing `completeWithoutMerge` path handles no-change completions correctly. The verification gate only checks whether the agent completed at least one turn with output.

**`dispatcher/types.go`** — `SessionOutcome.HasDeliverable` field (defined in phase 1) is the primary input for this check

## Data Structures

- No new types — uses `SessionOutcome.HasDeliverable` from phase 1

## Design Notes

With the updated CompletionTracker (phase 2), `sawAction` alone now maps to `StatusFailed`, so most false completions are already caught. This gate provides an additional check: even if the tracker says "completed" (because `sawResult` was seen), verify via `HasDeliverable` that the session actually produced output worth advancing.

The `HasDeliverable` flag (set when `EventResult` or `EventComplete` is seen) is the only check. No git-diff gate — review/quality/reflect stages legitimately complete with no code changes, and the existing `completeWithoutMerge` path already handles this correctly.

## Routing

- Provider: `codex`, Model: `gpt-5.3-codex`

## Verification

- Unit test: session with HasDeliverable=true → advances normally
- Unit test: session with HasDeliverable=false and no stage_yield → downgraded to failed
- Unit test: session with HasDeliverable=false but stage_yield emitted → advances (yield overrides)
- Unit test: no-change worktree session with HasDeliverable=true → advances normally (no git-diff gate)
- `go test ./loop/... -race`
