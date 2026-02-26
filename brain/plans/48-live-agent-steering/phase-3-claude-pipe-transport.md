Back to [[plans/48-live-agent-steering/overview]]

# Phase 3 — Claude Pipe Transport

## Goal

Implement `AgentController` for Claude Code using `--input-format stream-json` / `--output-format stream-json` over stdin/stdout pipes.

## Changes

**New file: `dispatcher/claude_controller.go`**

Claude-specific controller wrapping a `ProcessHandle`'s stdin pipe:

- Writes NDJSON messages to stdin:
  - User message: `{"type":"user","message":{"role":"user","content":"..."}}\n`
  - Interrupt: `{"type":"control_request","request_id":"<uuid>","request":{"subtype":"interrupt"}}\n`
- Tracks turn state via a goroutine reading provider stdout (alongside stamp):
  - On `{"type":"system","subtype":"init"}` — session ready, send first user message
  - On `{"type":"result"}` — turn complete, mark idle, ready for next message
  - On interrupt acknowledgment — mark idle
- `SendMessage()`:
  - If idle: write user message immediately
  - If active: queue the message, it's sent after the current turn completes or after interrupt
- `Interrupt()`:
  - Send control_request, wait for turn to stop (with timeout)
  - On timeout: return error (caller decides whether to fall back to Kill)
- `Steerable()` returns `true`

**Steering edge cases:**
- Interrupt during tool call: Claude stops after the current tool completes (not mid-tool). The interrupt may take seconds. Timeout at 30s, fall back to Kill.
- Concurrent steers: per-session mutex serializes `SendMessage`/`Interrupt`. Second steer waits for first to complete.
- Stale state: if stdout stops producing events (process hung), the interrupt timeout catches it.

**Modify: `dispatcher/process_dispatcher.go`**

For Claude provider:
- Add `--input-format stream-json --output-format stream-json` flags
- Remove `< prompt.txt` file redirection (prompt sent as first user message over stdin)
- Create `claudeController` wrapping the process handle's stdin
- Pass controller to `processSession`

**Modify: `dispatcher/process_session.go`**

Accept an `AgentController` at construction. `Controller()` returns it instead of `noopController`.

## Data Structures

- `claudeController` — implements `AgentController`, holds `io.WriteCloser` (stdin), mutex, turn state
- `turnState` — `idle` | `active` | `interrupting`

## Routing

Provider: `codex`, Model: `gpt-5.3-codex` — implementation against well-documented NDJSON protocol.

## Verification

### Static
- `go build ./dispatcher/...`
- `go vet ./dispatcher/...`

### Runtime
- Integration test: spawn Claude with pipe transport, send prompt, verify `system/init` → events → `result` lifecycle
- Integration test: send interrupt mid-turn, verify turn stops within timeout, new message can be sent
- Integration test: send two steers rapidly, verify serialization (second waits for first)
- Integration test: interrupt timeout → verify error returned (caller can Kill)
- Manual: `noodle start`, dispatch a Claude cook, verify events stream correctly in web UI
