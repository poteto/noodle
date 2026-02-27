Back to [[archive/plans/27-remote-dispatchers/overview]]

# Phase 11: Integration wiring and end-to-end test

**Routing:** `claude` / `claude-opus-4-6` — judgment needed for test design and integration review

## Goal

Wire everything together: bootstrap constructs the `DispatcherFactory` with configured backends, the loop uses it, and we have an end-to-end test proving the full path works with mock backends.

## Changes

**`cmd_start.go` (or wherever the app bootstraps)**
Build `DispatcherFactory`:
1. Always create `TmuxBackend` → `StreamingDispatcher` (local default)
2. If `config.Runtime.Sprites` has a token: create `SpritesBackend` → `StreamingDispatcher`
3. Cursor stub is **not** registered — it returns "not implemented" and must not be wired into the factory until fully implemented
4. Construct `DispatcherFactory` with registered backends only
5. Pass factory to loop as the `Dispatcher`

**Integration test (new file)**
`dispatcher/integration_test.go`:
- Mock `StreamingBackend` that feeds canned Claude NDJSON through an `io.Pipe`
- Mock `PollingBackend` that returns CREATING → RUNNING → FINISHED with canned conversation
- Verify full lifecycle through `DispatcherFactory`:
  - Streaming: session starts, events flow, cost accumulates, done signals, events.ndjson written
  - Polling: session starts, status transitions emit events, conversation messages appear, done signals
- Verify factory routing: runtime="sprites" goes to streaming, runtime="cursor" returns "runtime not configured", runtime="" goes to tmux/streaming

**Queue item routing verification**
Verify that queue items with `runtime: "sprites"` (set by prioritize agent) flow through the loop to `DispatchRequest.Runtime` and reach the correct backend via the factory.

## Verification

### Static
- `go test ./...` all green
- `go vet ./...` clean
- `sh scripts/lint-arch.sh` clean

### Runtime
- Integration test exercises both dispatcher paths with mocks
- Integration test: context cancellation propagates through factory → dispatcher → session → backend
- Integration test: concurrent dispatches to different runtimes don't interfere
- All integration tests run with `-race` flag
- Manual: configure `.noodle.toml` with `[runtime.sprites]` block, run prioritize, verify queue items get `runtime` field, dispatch via `noodle start --once`, observe session in TUI with badge and streaming events
