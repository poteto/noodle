Back to [[plans/96-101-runtime-plugins/overview]]

# Phase 2 — JSON-RPC Host Client

## Goal

Implement the host-side JSON-RPC client that launches a plugin subprocess, sends requests, and reads responses. This is the transport layer — no session semantics yet.

## Changes

**New file: `plugin/host.go`**

`PluginHost` struct:
- Launches plugin binary as a child process (`os/exec`)
- Writes JSON-RPC requests to plugin's stdin, serialized through a write mutex (concurrent `Call()` goroutines must not interleave JSON on stdin)
- Reads JSON-RPC responses from plugin's stdout (line-delimited, scanner buffer sized to 64MB to match existing NDJSON large-line handling)
- Demultiplexes stdout: lines with `"jsonrpc"` field → RPC response (matched by request ID); everything else → session event (routed by session ID in payload)
- Routes session events to per-session channels using non-blocking sends with bounded buffers. If a session's buffer is full, drop the event with a warning rather than blocking the demux goroutine (prevents head-of-line blocking across sessions and RPCs)
- Sends `initialize` on startup, validates protocol version handshake
- Kills subprocess on `Close()`
- Protocol corruption heuristic: after 3 consecutive unparseable lines, terminate the plugin host and mark the runtime as failed

Key methods:
- `NewPluginHost(binary string, cfg []byte) (*PluginHost, error)` — launch + initialize
- `Call(ctx, method string, params any) (json.RawMessage, error)` — send RPC, await response
- `PrepareEventStream(sessionID string) <-chan json.RawMessage` — pre-register an event channel for a session ID *before* sending the dispatch RPC (prevents early-event loss race: plugin may emit events immediately after dispatch response)
- `Close() error` — shutdown plugin process

**New file: `plugin/host_test.go`**
- Test with a mock plugin binary (small Go program in `testdata/`)
- Verify initialize handshake
- Verify Call() sends correct JSON and receives response
- Verify concurrent Call() does not interleave stdin writes (write mutex)
- Verify PrepareEventStream pre-registers channel before dispatch
- Verify event demux routes to correct session channel
- Verify slow consumer doesn't block other sessions (non-blocking send)
- Verify Close() terminates subprocess
- Verify timeout on unresponsive plugin
- Verify protocol corruption terminates host after threshold

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
- Verify early events (emitted before PrepareEventStream returns) are buffered and delivered
- Verify head-of-line blocking: stalled consumer on session A doesn't block session B events or RPC responses
