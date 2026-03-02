Back to [[plans/96-101-runtime-plugins/overview]]

# Phase 3 — Plugin Session Bridge

## Goal

Bridge a plugin's JSON-RPC dispatch + NDJSON event stream into Noodle's internal `runtime.Runtime` interface. After this phase, a plugin runtime is indistinguishable from a built-in runtime to the rest of the system.

## Changes

**New file: `plugin/runtime.go`**

`PluginRuntime` struct — implements `runtime.Runtime`:
- `Dispatch(ctx, req) (SessionHandle, error)` — calls plugin's `dispatch` RPC, returns a `pluginSessionHandle` that reads from the event stream
- `Kill(handle) error` — calls plugin's `kill` RPC
- `Recover(ctx) ([]RecoveredSession, error)` — calls plugin's `recover` RPC, maps results to `RecoveredSession`

`pluginSessionHandle` struct — implements `runtime.SessionHandle`:
- Wraps `PluginHost.EventStream(sessionID)` channel
- Translates raw JSON events into canonical `SessionEvent` types
- Exposes `Done()` channel (closed when event stream ends or plugin reports completion)
- Tracks `TotalCost()` from cost events
- `Kill()` delegates to `PluginRuntime.Kill()`
- `Controller()` returns `NoopController` (steering via plugin is future work)
- `VerdictPath()` returns the runtime dir path for this session

**New file: `plugin/runtime_test.go`**
- Test Dispatch → session handle → event stream → Done
- Test Kill propagates to plugin
- Test Recover returns session descriptors
- Test event translation (plugin JSON → canonical SessionEvent)
- Test session handle reports correct cost from cost events

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `codex` | `gpt-5.3-codex` | Mechanical bridge — clear contract on both sides |

## Verification

### Static
- `go build ./plugin/...`
- `go vet ./plugin/...`
- `PluginRuntime` satisfies `runtime.Runtime` interface (compile-time check)
- `pluginSessionHandle` satisfies `runtime.SessionHandle` interface

### Runtime
- End-to-end test: mock plugin dispatches a session, streams 5 events, completes → verify SessionHandle sees all events and Done() fires
- Kill test: kill mid-stream → verify Done() fires and event channel drains
- Cost tracking: verify TotalCost() accumulates from cost events
