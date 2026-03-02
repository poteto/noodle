Back to [[plans/96-101-runtime-plugins/overview]]

# Phase 2 — JSON-RPC Host Client

## Goal

Implement the host-side JSON-RPC client that launches a plugin subprocess, sends requests, and reads responses. This is the transport layer — no session semantics yet.

## Changes

**New file: `plugin/host.go`**

`PluginHost` struct:
- Launches plugin binary as a child process (`os/exec`)
- Writes JSON-RPC requests to plugin's stdin
- Reads JSON-RPC responses from plugin's stdout (line-delimited)
- Demultiplexes stdout: JSON-RPC responses (by `id`) vs. NDJSON session events (by `event` field)
- Routes session events to per-session channels
- Sends `initialize` on startup, validates protocol version handshake
- Kills subprocess on `Close()`

Key methods:
- `NewPluginHost(binary string, cfg []byte) (*PluginHost, error)` — launch + initialize
- `Call(ctx, method string, params any) (json.RawMessage, error)` — send RPC, await response
- `EventStream(sessionID string) <-chan json.RawMessage` — subscribe to events for a session
- `Close() error` — shutdown plugin process

The demux goroutine reads stdout line-by-line. Lines with `"jsonrpc"` key are matched to pending `Call()` waiters by request ID. Lines with `"event"` key are routed to the session's event channel. Malformed lines are logged and dropped.

**New file: `plugin/host_test.go`**
- Test with a mock plugin binary (small Go program in `testdata/`)
- Verify initialize handshake
- Verify Call() sends correct JSON and receives response
- Verify event demux routes to correct session channel
- Verify Close() terminates subprocess
- Verify timeout on unresponsive plugin

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `claude` | `claude-opus-4-6` | Architectural judgment — concurrency, demux design, error handling |

## Verification

### Static
- `go build ./plugin/...`
- `go vet ./plugin/...`
- No data races: `go test -race ./plugin/...`

### Runtime
- Mock plugin binary in `testdata/` responds to initialize + call + events
- Verify demux correctly separates RPC responses from event streams
- Verify Call() returns error on plugin crash mid-request
- Verify Close() is safe to call multiple times (idempotent)
