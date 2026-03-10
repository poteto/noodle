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

**`dispatcher/session_base.go`** — new `resolveAndMarkDone(exitCode int, ctxCancelled bool)` method:
- **Critical ordering:** Wait for stream processing to complete (`<-s.streamDone` or equivalent barrier) BEFORE calling `tracker.Resolve()`. The stream goroutine may still be consuming the final stdout lines when the process exits — resolving before stream completion will miss terminal events.
- Call `tracker.Resolve(exitCode, ctxCancelled)` to produce the final outcome
- Store the resolved `SessionOutcome` on the session
- Then call `markDone(outcome.Status.String())` to close the Done channel

Both `processSession.waitForExit()` and `spritesSession.waitAndSync()` should call `resolveAndMarkDone()` instead of `markDone()` directly.

## Data Structures

- No new types — `sessionBase` gains a `tracker` field and `outcome` field

## Design Notes

The key insight: the canonical event interceptor (`io.MultiWriter(canonicalFile, interceptor)`) already processes every event in-process. Adding `tracker.Observe()` to this path is zero-cost — no file reading, no parsing overhead, just a function call per event.

**Stream completion barrier:** The stream processing goroutine (`processStream`) must signal completion via a channel or WaitGroup. `resolveAndMarkDone()` waits on this signal before resolving. This prevents the race where the process exits, `Resolve()` runs, but the final `EventResult`/`EventComplete` hasn't been observed yet. This ordering constraint is non-negotiable — it's the foundation of the "deterministic" claim.

## Routing

- Provider: `codex`, Model: `gpt-5.4`

## Verification

- Integration test: create a `sessionBase` with mock events, verify `Outcome()` returns correct status after `markDone()`
- Verify tracker receives events in correct order
- `go test ./dispatcher/... -race`
