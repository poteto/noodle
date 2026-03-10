Back to [[plans/76-websocket-streaming/overview]]

# Phase 7 — Loop + Main Wiring

## Goal

Thread the event sink from server's broker through the loop dependencies to the dispatchers. This is the final backend connection: agent events now flow from dispatcher → broker → WS clients.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| codex | gpt-5.4 | Mechanical wiring through config structs |

## Changes

**`loop/types.go`**:
- Add `EventSink dispatcher.SessionEventSink` to `Dependencies` struct (line ~157)

**`loop/defaults.go`**:
- `defaultDependencies()` currently takes no parameters (line ~26) and is called from `loop.New` (line ~28). It creates dispatchers without EventSink.
- Pass `EventSink` as parameter to `defaultDependencies(sink SessionEventSink)` and thread it to dispatcher configs at construction time. Do not set sink after construction — dispatchers must have the sink from the start so every `Dispatch()` call passes it to sessions.

**`cmd_start.go`**:
- The current startup order creates the loop before the server (`cmd_start.go:77` creates `runtimeLoop`, `cmd_start.go:93` creates server in a goroutine). The broker must be created *before* both so it can be passed to each.
- The factory `newStartRuntimeLoop` (line 29) hardcodes `loop.Dependencies{}`. Change its signature to accept `Dependencies` so the caller can pass the broker.
- Create `broker := server.NewSessionEventBroker()` early in `runStart()`
- Pass broker to loop deps: `newStartRuntimeLoop(cwd, noodleBin, cfg, loop.Dependencies{EventSink: broker})`
- Pass broker to server: `server.New(server.Options{..., Broker: broker})`
- Update test mocks that use `newStartRuntimeLoop` to match new signature

## Data Structures

- `Dependencies.EventSink` — optional, nil means no real-time broadcasting (tests, CLI)
- `SessionEventBroker` created independently, shared by reference

## Verification

### Static
- `go build ./...`
- `go test ./loop/...`

### Runtime
- Start noodle, trigger an agent session
- Events should flow from agent → dispatcher → broker → WS clients
- Verify via `wscat` or browser devtools that session events arrive in real-time
