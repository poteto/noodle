Back to [[plans/113-deterministic-completion-detection/overview]]

# Phase 5: Sprites Session Alignment

## Goal

Update `spritesSession.waitAndSync()` to use the CompletionTracker instead of exit-code-only detection. After this phase, both session types use the same completion logic.

## Changes

**`dispatcher/sprites_session.go`**:
- `waitAndSync()`: replace the manual exit-code → status mapping. After sync-back completes, call `s.resolveAndMarkDone(exitCode, ctx.Err() != nil)` (from phase 3) which waits for stream completion, resolves the tracker, and marks done
- The tracker already receives events through `sessionBase.consumeCanonicalLine()` (wired in phase 3), so sprites sessions get the same incremental tracking as process sessions
- The sync-back logic (push changes from sprite to git remote) remains unchanged — it runs before `resolveAndMarkDone()`

## Data Structures

- No new types

## Design Notes

Sprites sessions stream stdout through the same stamp processor pipeline as local sessions. The CompletionTracker in `sessionBase` already observes these events. The only change is replacing the manual exit-code switch in `waitAndSync()` with the tracker's output.

This means a Sprites session that does work but exits non-zero (e.g., remote OOM) will now correctly be classified as "completed" if it emitted lifecycle events — matching local session behavior.

## Routing

- Provider: `codex`, Model: `gpt-5.3-codex`

## Verification

- Unit test: mock sprites session with events + non-zero exit, verify "completed" outcome
- Unit test: mock sprites session with zero events + non-zero exit, verify "failed" outcome
- `go test ./dispatcher/... -race`
