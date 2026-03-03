Back to [[plans/90-interactive-sessions/overview]]

# Phase 1 — Session Kind + Control Action + Spawn

## Goal

Introduce `SessionKind` as a foundational type, add the `chat` control action, and implement the spawn path for interactive sessions. This is the core backend capability: user sends `{ action: "chat", provider: "claude", model: "...", prompt: "..." }` and gets a running session.

## Changes

**`dispatcher/types.go`** — Add `SessionKind` type and constants (`KindOrder`, `KindInteractive`). Add `Kind` field to `DispatchRequest`. Session kind is derived from the control action — `chat` implies `KindInteractive`, `enqueue` implies `KindOrder`. No `Kind` field on `ControlCommand`.

**`dispatcher/session_base.go`** — Add `kind` field to `sessionBase`. Expose via accessor method. Set from `DispatchRequest.Kind` during construction.

**`server/ws_hub.go`** — Add `"chat"` to `validActions` map.

**`server/server.go`** — Map `chat` request fields in `parseControlRequest()`. The `chat` action uses existing `ControlCommand` fields: `Provider`, `Model`, `Prompt`, `Name` (optional display name).

**`loop/control.go`** — Add `case "chat"` to `dispatchControlCommand()`, call `l.controlChat(cmd)`.

**`loop/chat.go`** (new) — Implement `controlChat()`:
1. Resolve provider/model (use defaults if not specified; restrict to steerable providers)
2. Build `DispatchRequest` with `Kind: KindInteractive`, set `WorktreePath` to the primary checkout path (same pattern as scheduler — `DispatchRequest.Validate()` requires a non-empty path)
3. No skill bundle resolution — use provider default system prompt
4. Call `l.dispatchSession()` to get a session handle
5. Return ack — ack carries the control command's ID as a correlation ID. The UI discovers the spawned session via the next snapshot update (not via session ID in the ack, since the control protocol is append-only and acks fire before execution).

## Data Structures

- `SessionKind string` — `"order"` or `"interactive"`
- `DispatchRequest.Kind SessionKind`
- `sessionBase.kind SessionKind`
- No new fields on `ControlCommand` — derive kind from action name

## Routing

- **Provider:** `claude`
- **Model:** `claude-opus-4-6`

## Verification

### Static
- `go build ./...` — compiles with new types
- `go vet ./...`
- Existing tests pass unchanged (default kind is empty or `"order"`, existing paths unaffected)

### Runtime
- Integration test: send `chat` control command with steerable provider, verify session spawns with `KindInteractive` and `WorktreePath` set to primary checkout
- Integration test: send `chat` without provider/model, verify defaults are used
- Integration test: send `chat` with non-steerable provider, verify error ack
- Integration test: verify ack does NOT contain session ID (correlation ID only)
- Unit test: `DispatchRequest` with `Kind: KindInteractive` creates session with correct kind
