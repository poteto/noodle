Back to [[plans/96-101-runtime-plugins/overview]]

# Phase 1 — Plugin Protocol Types

## Goal

Define the Go types that form the JSON-RPC plugin protocol. These are the foundational data structures — get them right before any implementation.

## Changes

**New file: `plugin/protocol.go`**

Define the JSON-RPC 2.0 message envelope and the four RPC methods:

- `initialize` — plugin declares its name, capabilities, and protocol version. Host sends raw config bytes.
- `dispatch` — host sends a dispatch request, plugin returns a session ID and begins streaming NDJSON events.
- `kill` — host requests session termination by ID.
- `recover` — host asks plugin to enumerate recoverable sessions from previous runs.

**Key types:**
- `Request` / `Response` / `ErrorResponse` — JSON-RPC 2.0 envelope
- `InitializeParams` / `InitializeResult` — config bytes in, capabilities + version out
- `DispatchParams` / `DispatchResult` — mirrors existing `DispatchRequest` fields, returns session ID
- `KillParams` / `KillResult` — session ID in, ack out
- `RecoverParams` / `RecoverResult` — returns list of recoverable session descriptors

Event streaming is not a separate RPC — after `dispatch` returns, the plugin writes NDJSON session events to stdout interleaved with future RPC responses. The host demultiplexes by checking whether each line is a JSON-RPC response (has `jsonrpc` field) or a session event (has `event` field).

**New file: `plugin/protocol_test.go`**
- Round-trip marshal/unmarshal for every type
- Verify JSON field names match the protocol spec

**New file: `plugin/protocol_spec.md`** (checked into repo)
- Human-readable protocol spec document — the contract between host and plugin
- Versioned (start at v1) so future protocol changes are explicit
- Covers: message format, method signatures, event interleaving, error codes, shutdown sequence
- This is the reference doc for non-Go plugin authors

## Routing

| Provider | Model | Why |
|----------|-------|-----|
| `claude` | `claude-opus-4-6` | Protocol design is architectural — get the interface right before anything builds on it |

## Verification

### Static
- `go build ./plugin/...`
- `go vet ./plugin/...`
- Types are JSON-serializable with correct field tags

### Runtime
- All protocol types round-trip through `json.Marshal` / `json.Unmarshal`
- Error response codes match JSON-RPC 2.0 spec (-32600, -32601, -32602, -32603)
