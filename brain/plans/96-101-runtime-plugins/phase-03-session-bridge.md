Back to [[plans/96-101-runtime-plugins/overview]]

# Phase 3 — Plugin Session Bridge

## Goal

Bridge a plugin's JSON-RPC dispatch + NDJSON event stream into Noodle's internal `runtime.Runtime` interface. After this phase, a plugin runtime is indistinguishable from a built-in runtime to the rest of the system.

## Changes

**New file: `plugin/runtime.go`**

`PluginRuntime` struct — implements `runtime.Runtime`:
- Constructed with a `SessionEventSink` (same as process/sprites dispatchers today) so plugin events reach the canonical event pipeline that the UI, monitor, and scheduler consume
- `Dispatch(ctx, req) (SessionHandle, error)` — pre-registers event channel via `PrepareEventStream(sessionID)`, then calls plugin's `dispatch` RPC, returns a `pluginSessionHandle`
- `Kill(handle) error` — calls plugin's `kill` RPC
- `Recover(ctx) ([]RecoveredSession, error)` — calls plugin's `recover` RPC (passing runtime dir for project scoping), maps results to `RecoveredSession`

`pluginSessionHandle` struct — implements `runtime.SessionHandle`:
- Reads from the pre-registered event channel (no early-event loss)
- Translates raw JSON events into canonical `SessionEvent` types and publishes them to the `SessionEventSink`
- Synthesizes session artifact files (`meta.json`, `spawn.json`, event log) from plugin events, matching the contract that `loop/reconcile.go`, `monitor/`, and `loop/cook_completion.go` depend on
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
| `codex` | `gpt-5.4` | Mechanical bridge — clear contract on both sides |

## Verification

### Static
- `go build ./plugin/...`
- `go vet ./plugin/...`
- `PluginRuntime` satisfies `runtime.Runtime` interface (compile-time check)
- `pluginSessionHandle` satisfies `runtime.SessionHandle` interface

### Runtime
- End-to-end test: mock plugin dispatches a session, streams 5 events, completes → verify SessionHandle sees all events and Done() fires
- Verify events published to SessionEventSink (mock the sink, assert events received)
- Verify session artifacts written: `meta.json`, `spawn.json`, event log exist in runtime dir after session completes
- Verify Recover scopes to project runtime dir (mock plugin returns sessions from two dirs, only matching ones adopted)
- Kill test: kill mid-stream → verify Done() fires and event channel drains
- Cost tracking: verify TotalCost() accumulates from cost events
