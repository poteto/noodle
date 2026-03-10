Back to [[plans/76-websocket-streaming/overview]]

# Phase 1 — Event Sink Interface + FormatEventLine

## Goal

Define the `SessionEventSink` interface and a helper to convert canonical events into `EventLine` structs for real-time broadcasting. This is the foundational type that dispatchers will call and the broker will implement.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| codex | gpt-5.4 | Mechanical type + function extraction |

## Changes

**`dispatcher/types.go`** — Add interface:
- `SessionEventSink` with method `PublishSessionEvent(sessionID string, event snapshot.EventLine)`
- Place alongside existing `Session` interface

**`internal/snapshot/snapshot.go`** — Export single-event formatter:
- Extract the loop body from `mapEventLines()` (line ~181) into `FormatSingleEvent(ev event.Event) (EventLine, bool)`
- `mapEventLines()` calls `FormatSingleEvent` in its loop — no behavior change
- Returns `(EventLine, bool)` since some events may not produce a line (e.g. unknown types)

**`dispatcher/session_helpers.go`** — Add `FormatEventLine`:
- `FormatEventLine(sessionID string, ce parse.CanonicalEvent) (snapshot.EventLine, bool)`
- Converts `CanonicalEvent` → `event.Event` → calls `snapshot.FormatSingleEvent`
- Reuses existing `eventFromCanonical()` logic already in the dispatcher package

## Data Structures

- `SessionEventSink` — single-method interface for publishing session events
- `FormatSingleEvent` — pure function, event.Event → EventLine

## Verification

### Static
- `go build ./dispatcher/... ./internal/snapshot/...`
- `go vet ./...`
- Existing tests pass: `go test ./internal/snapshot/...`

### Runtime
- No runtime change yet — sink is defined but not called
