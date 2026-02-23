Back to [[plans/27-remote-dispatchers/overview]]

# Phase 3: StreamingDispatcher generic implementation

**Routing:** `claude` / `claude-opus-4-6` — architectural judgment, goroutine lifecycle design

## Goal

Build the generic `StreamingDispatcher` that works with any `StreamingBackend` (TmuxBackend, SpritesBackend, etc.). This absorbs the session lifecycle logic currently in `TmuxDispatcher` + `tmuxSession`: session directory setup, skill loading, prompt composition, event parsing, monitoring, and publishing.

This is the most architecturally significant phase. The existing `tmuxSession` and `TmuxDispatcher.Dispatch` are refactored into this generic dispatcher. After this phase, `TmuxBackend` is just one of several backends that plug into `StreamingDispatcher`.

## Data structures

- `StreamingDispatcher` struct — holds `StreamingBackend`, runtime dir, noodle bin path, skill resolver
- `streamingSession` struct — implements `Session` interface. Refactored from `tmuxSession` with backend-agnostic monitoring.

## Changes

**`dispatcher/streaming_dispatcher.go` (new)**
`StreamingDispatcher.Dispatch` — absorbs the generic parts of `TmuxDispatcher.Dispatch`:
1. Validate request, generate session ID, create session dir (reuse `sessionPaths`, `generateSessionID`)
2. Load skill bundle, compose prompts (reuse `loadSkillBundle`, `composePrompts`)
3. Call `backend.Start(ctx, config)` → `StreamHandle`
4. Write `spawn.json` metadata (include runtime kind)
5. Return `streamingSession`, launch monitor goroutines

**`dispatcher/streaming_session.go` (new)**
Refactor `tmuxSession` into `streamingSession`:
- Replace `monitorPane` (tmux-specific polling) with `backend.IsAlive(handle)` calls
- Replace `monitorCanonicalEvents` (file polling) with direct stream consumption from `StreamHandle.Stdout` — parse NDJSON in-process using `parse.DetectProvider` + adapters
- Keep: `consumeCanonical`, `eventFromCanonical`, `publish`, `markDone`, `closeEventsWhenDone`, non-blocking event publishing with drop counter
- Still writes `canonical.ndjson` and `events.ndjson` to disk for monitor/TUI compatibility
- `Kill()`: calls `backend.Kill(handle)`, marks done

**Delete `dispatcher/tmux_session.go` and `dispatcher/tmux_dispatcher.go`** after migration. All their logic lives in streaming_dispatcher.go, streaming_session.go, and tmux_backend.go.

## Verification

### Static
- Compiles, passes vet
- `streamingSession` satisfies `Session` interface (compile-time check)

### Runtime
- Unit test with a mock `StreamingBackend` that returns a `bytes.Reader` of canned NDJSON. Verify events appear on the channel, cost accumulates, done signals on EOF.
- Unit test: backend reports not alive mid-stream → session status is "failed"
- Unit test: Kill() calls backend.Kill and closes done channel
- Unit test: context cancellation mid-stream → goroutines exit cleanly, no leaked goroutines
- Unit test: slow consumer (full event channel) → events dropped, drop counter increments, no goroutine deadlock
- Run all tests with `-race` flag
