Back to [[plans/76-websocket-streaming/overview]]

# Phase 2 — Wire Sink into Process Dispatcher

## Goal

Thread the `SessionEventSink` through `ProcessDispatcherConfig` → `processSessionConfig` → `processSession`, and call it in `consumeCanonicalLine()` after `eventWriter.Append()`.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| codex | gpt-5.4 | Mechanical wiring, clear spec |

## Changes

**`dispatcher/process_dispatcher.go`**:
- Add `Sink SessionEventSink` field to `ProcessDispatcherConfig` (line ~17)
- Store `sink` on the `processDispatcher` struct (line ~30) — the config is consumed at construction; `Dispatch()` needs the sink later when creating sessions
- In `Dispatch()`, pass `sink` to `processSessionConfig` when creating sessions (line ~172)

**`dispatcher/process_session.go`**:
- Add `sink SessionEventSink` field to `processSessionConfig` (line ~46)
- In `consumeCanonicalLine()` (line ~237), after `eventWriter.Append()` succeeds:
  ```
  if s.sink != nil {
      if el, ok := FormatEventLine(s.id, ce); ok {
          s.sink.PublishSessionEvent(s.id, el)
      }
  }
  ```
- In `emitPromptEvent()` (line ~254), after its own `eventWriter.Append()` call (line ~283): same sink publish pattern. This method writes events directly without going through `consumeCanonicalLine()`, so it needs its own sink call to avoid dropping prompt events from real-time streaming.

## Data Structures

- `processSessionConfig.sink` — optional `SessionEventSink`, nil when no real-time consumers exist

## Verification

### Static
- `go build ./dispatcher/...`
- `go test ./dispatcher/...` — existing tests pass (sink is nil in tests)

### Runtime
- No runtime change yet — sink is wired but nothing provides it
