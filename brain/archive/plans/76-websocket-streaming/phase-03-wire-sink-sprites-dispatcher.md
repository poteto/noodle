Back to [[plans/76-websocket-streaming/overview]]

# Phase 3 — Wire Sink into Sprites Dispatcher

## Goal

Same wiring as Phase 2 but for the sprites dispatcher path. Identical pattern.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| codex | gpt-5.3-codex | Mechanical, same pattern as Phase 2 |

## Changes

**`dispatcher/sprites_dispatcher.go`**:
- Add `Sink SessionEventSink` field to `SpritesDispatcherConfig` (or equivalent config struct)
- Store `sink` on the `spritesDispatcher` struct (line ~34) — same pattern as Phase 2
- In `Dispatch()`, pass `sink` to `spritesSessionConfig` when creating sessions (line ~197)

**`dispatcher/sprites_session.go`**:
- Add `sink SessionEventSink` field to `spritesSessionConfig`
- In `consumeCanonicalLine()` (line ~228), after `eventWriter.Append()`:
  - Same pattern as Phase 2: call `FormatEventLine` then `sink.PublishSessionEvent`

## Data Structures

- `spritesSessionConfig.sink` — same pattern as process session

## Verification

### Static
- `go build ./dispatcher/...`
- `go test ./dispatcher/...`

### Runtime
- No runtime change yet
