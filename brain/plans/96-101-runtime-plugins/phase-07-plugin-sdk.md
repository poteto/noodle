Back to [[plans/96-101-runtime-plugins/overview]]

# Phase 7 — Plugin SDK

## Goal

Provide a Go helper library that makes writing a runtime plugin trivial. Plugin authors implement a Go interface and call `sdk.Serve()` — the SDK handles JSON-RPC plumbing, event emission, and stdio management.

## Changes

**New package: `sdk/runtime/`** (published as `github.com/poteto/noodle/sdk/runtime`)

`Plugin` interface — what plugin authors implement:
- `Name() string`
- `Initialize(config json.RawMessage) (Capabilities, error)`
- `Dispatch(ctx context.Context, req DispatchRequest) (Session, error)`
- `Kill(sessionID string) error`
- `Recover(ctx context.Context) ([]RecoveredSession, error)`

`Session` interface — what plugin authors return from Dispatch:
- `ID() string`
- `Events() <-chan Event`
- `Done() <-chan struct{}`
- `TotalCost() float64`

`Serve(plugin Plugin)` — main entry point:
- Reads JSON-RPC from stdin
- Routes to Plugin methods
- Writes JSON-RPC responses to stdout
- Streams session events as NDJSON to stdout
- Handles `SIGTERM` / `SIGINT` gracefully

Helper types:
- `Event` — wraps canonical event fields, JSON-serializable
- `Capabilities` — steerable, polling, remote sync, heartbeat flags
- `DispatchRequest` — mirrors host-side request (prompt, provider, model, skill, etc.)

**New file: `sdk/runtime/serve_test.go`**
- Test Serve() with a minimal mock plugin
- Verify initialize → dispatch → events → done lifecycle
- Verify kill interrupts a running session
- Verify recover returns sessions from a previous run
- Verify graceful shutdown on SIGTERM

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `claude` | `claude-opus-4-6` | API design — developer UX for plugin authors, interface ergonomics |

## Verification

### Static
- `go build ./sdk/runtime/...`
- `go vet ./sdk/runtime/...`
- SDK types are symmetric with host-side protocol types (phase 1)

### Runtime
- Build a test plugin using the SDK, pipe stdio through the host client (phase 2) → verify full round-trip
- Verify SDK handles concurrent dispatch requests (multiple sessions)
- Verify clean shutdown: SIGTERM → in-flight sessions complete or error
