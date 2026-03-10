Back to [[plans/96-101-runtime-plugins/overview]]

# Phase 8 — Extract Sprites to Plugin

## Goal

Move the Sprites runtime implementation out of the Noodle core into `noodle-runtime-sprites` — a separate Go module that implements the plugin protocol using the SDK from phase 7.

## Changes

**New repo/module: `noodle-runtime-sprites`**

Structure:
```
noodle-runtime-sprites/
├── go.mod          # depends on noodle/sdk/runtime + sprites-go
├── main.go         # sdk.Serve(&SpritesPlugin{})
├── plugin.go       # SpritesPlugin implementing sdk/runtime.Plugin
├── dispatch.go     # dispatch logic (from dispatcher/sprites_dispatcher.go)
├── session.go      # session lifecycle (from dispatcher/sprites_session.go)
├── git.go          # git push/clone/sync helpers (extracted from sprites_dispatcher.go)
└── plugin_test.go  # tests
```

Migration from core:
- `dispatcher/sprites_dispatcher.go` → `plugin.go` + `dispatch.go` + `git.go`
- `dispatcher/sprites_session.go` → `session.go`
- Adapt to use SDK's `Session` interface instead of embedding `sessionBase`
- Emit events through SDK's `Event` type instead of internal `SessionEventSink`
- Parse config from raw JSON (from `initialize` RPC) instead of typed `SpritesConfig`

The plugin reads skill bundles via the dispatch request params (skill content is passed by the host, not resolved by the plugin). Git operations (push worktree, clone on sprite, sync back) stay in the plugin — they're Sprites-specific.

**Comprehensive test suite:**

`plugin_test.go` — unit tests:
- Initialize with valid config → capabilities returned
- Initialize with missing token → clear error
- Initialize with missing sprite_name → clear error
- Initialize with malformed config JSON → clear error

`dispatch_test.go` — dispatch lifecycle:
- Dispatch → session ID returned → events stream → done signal
- Dispatch with invalid provider → error before any sprite launch
- Dispatch concurrent sessions → each gets independent event stream
- Dispatch when sprite launch fails → error propagated, no leaked goroutines

`git_test.go` — git sync operations:
- Push worktree branch to remote
- Clone on sprite with authenticated URL
- Sync back changes to `noodle/SESSION_ID` branch
- Handle remote URL parsing (HTTPS, SSH)
- Handle auth token injection edge cases

`session_test.go` — session lifecycle:
- Events translate correctly to SDK event format
- Cost tracking accumulates from cost events
- Kill mid-stream terminates cleanly
- Session done after all events consumed

`integration_test.go` — end-to-end via plugin protocol:
- Build plugin binary, launch via host client, dispatch, verify events, kill
- Verify the plugin binary matches the protocol spec from phase 1

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `codex` | `gpt-5.4` | Mechanical extraction — moving code between modules |

## Verification

### Static
- `go build ./...` in the plugin repo
- `go vet ./...`

### Runtime
- Build the plugin binary, run it with the host client from phases 2-3
- Verify initialize handshake returns capabilities (steerable: true)
- Mock sprites-go SDK calls, verify dispatch→events→done lifecycle
- Verify git sync-back writes correct branch
- Integration: `noodle plugin install` the built binary, configure `[runtime.sprites]`, dispatch a stage
