Back to [[plans/113-deterministic-completion-detection/overview]]

# Phase 8: Cleanup Verification Gate

## Goal

After a session exits with "completed" status, verify that the claimed deliverable was actually produced. A session that reports success but produced no artifact should be downgraded to failed.

## Changes

**`loop/cook_completion.go`** — update `handleCompletion()`:
- After determining `StageResultCompleted`, run a verification check before advancing
- For worktree sessions: verify the worktree has changes (`git diff --stat` or equivalent) OR a stage_yield was emitted
- For non-worktree sessions: verify at least one `EventResult` canonical event exists (the CompletionTracker's `HasDeliverable` flag)
- If verification fails, downgrade to `StageResultFailed` with reason "completed with no deliverable"

**`dispatcher/types.go`** — `SessionOutcome.HasDeliverable` field (defined in phase 1) is the primary input for this check

## Data Structures

- No new types — uses `SessionOutcome.HasDeliverable` from phase 1

## Design Notes

This gate catches the case where `terminalStatus()` (or now `CompletionTracker`) marks a session as "completed" because it saw lifecycle events, but the session didn't actually produce output. Example: agent initialized, read some files, encountered an error, and exited — the tracker sees `EventInit` + `EventAction` and marks completed, but no useful work was produced.

The `HasDeliverable` flag (set when `EventResult` or `EventComplete` is seen) distinguishes "did work with output" from "started but produced nothing."

The worktree check is defense-in-depth: even if the tracker says "has deliverable," confirm the worktree actually has changes before attempting merge.

## Routing

- Provider: `codex`, Model: `gpt-5.3-codex`

## Verification

- Unit test: session with HasDeliverable=true → advances normally
- Unit test: session with HasDeliverable=false → downgraded to failed
- Unit test: worktree session with HasDeliverable=true but no git changes → downgraded
- `go test ./loop/... -race`
