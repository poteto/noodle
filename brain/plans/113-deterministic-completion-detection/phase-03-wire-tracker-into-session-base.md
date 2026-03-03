Back to [[plans/113-deterministic-completion-detection/overview]]

# Phase 3: Wire CompletionTracker into sessionBase

## Goal

Integrate the CompletionTracker into `sessionBase` so that every session type (process and sprites) automatically tracks completion state as events flow through.

## Changes

**`dispatcher/session_base.go`**:
- Add `tracker *CompletionTracker` field to `sessionBase`
- Initialize tracker in the session constructor
- In `consumeCanonicalLine()` (or the canonical event interceptor), call `tracker.Observe(event)` for each parsed canonical event
- Add `Outcome() SessionOutcome` method to `sessionBase` that delegates to `tracker`
- The tracker receives events through the same in-process path that writes to `canonical.ndjson` — no additional file I/O

**`dispatcher/session_base.go`** — update `markDone()`:
- After setting the done status, also call `tracker.Resolve()` with exit code and context state to finalize the outcome
- Store the resolved `SessionOutcome` so `Outcome()` returns it after `Done()` closes

## Data Structures

- No new types — `sessionBase` gains a `tracker` field and `outcome` field

## Design Notes

The key insight: the canonical event interceptor (`io.MultiWriter(canonicalFile, interceptor)`) already processes every event in-process. Adding `tracker.Observe()` to this path is zero-cost — no file reading, no parsing overhead, just a function call per event.

## Routing

- Provider: `codex`, Model: `gpt-5.3-codex`

## Verification

- Integration test: create a `sessionBase` with mock events, verify `Outcome()` returns correct status after `markDone()`
- Verify tracker receives events in correct order
- `go test ./dispatcher/... -race`
