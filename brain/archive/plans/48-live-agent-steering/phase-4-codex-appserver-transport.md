Back to [[plans/48-live-agent-steering/overview]]

# Phase 4 — Codex App-Server Transport

## Goal

Implement `AgentController` for Codex using `codex app-server --listen stdio://` as a control channel, with rollout JSONL files for event reading.

**Important:** The exact JSON-RPC method names and parameter shapes must be verified against a live `codex app-server` process before implementation. The protocol is not yet stable. This phase begins with a conformance spike.

## Changes

**Step 0: Protocol conformance spike**

Before writing any production code, write a throwaway Go test that:
1. Spawns `codex app-server --listen stdio://`
2. Sends `thread/new` to create a thread
3. Sends `turn/start` with a simple prompt
4. Reads notifications until `turn/completed`
5. Sends `turn/interrupt` and `turn/steer` to verify method names and param shapes
6. Documents the actual wire format in a comment block

This spike produces the ground-truth protocol contract for the rest of the phase.

**New file: `dispatcher/codex_controller.go`**

Codex-specific controller that:
- Holds stdin pipe to `codex app-server --listen stdio://`
- Writes JSON-RPC requests (line-delimited, no `jsonrpc: "2.0"` field per Codex convention):
  - `thread/new` — create a thread, receive thread_id
  - `turn/start` — start a turn with user input
  - `turn/steer` — inject input mid-turn (requires `threadId` + `turnId`)
  - `turn/interrupt` — stop current turn (requires `threadId`, optional `turnId`)
- Reads JSON-RPC responses/notifications from stdout in a goroutine:
  - Track `threadId` from `thread/started` notification
  - Track `turnId` from `turn/started` notification
  - Drain all other notifications (events come from rollout files)
- `SendMessage()`:
  - If idle: send `turn/start`
  - If active: send `turn/steer` with current `turnId`
- `Interrupt()`: send `turn/interrupt` with current `turnId`, wait for `turn/completed` notification
- `Steerable()` returns `true`

**Rollout file discovery:**

Extract rollout file path from the `thread/started` notification payload (contains session/thread metadata). If the path isn't in the notification, derive it from thread_id + date pattern (`~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl`). Wait with bounded retry (up to 5s) for file creation before tailing.

Do NOT watch `~/.codex/sessions/` with fsnotify — race-prone under concurrency.

**Modify: `dispatcher/process_dispatcher.go`**

For Codex provider:
- Build `codex app-server --listen stdio://` command instead of `codex exec --json`
- Map current exec flags to app-server equivalents (see parity matrix in overview)
- Create `codexController` wrapping the process handle's stdin
- Event pipeline: tail rollout JSONL → in-process stamp → canonical events (instead of piping stdout)

**Modify: `stamp/processor.go`**

Add a file-tailing mode: read from a growing file instead of stdin. The processor watches for new lines appended to the rollout file and processes them through the existing adapter pipeline. This is needed because Codex app-server stdout carries JSON-RPC (not the event stream), so stamp reads the rollout file directly.

## Data Structures

- `codexController` — implements `AgentController`, holds stdin pipe, tracks threadID/turnID, request ID counter, mutex
- `jsonrpcRequest` — `{ID int, Method string, Params json.RawMessage}` for writing
- `jsonrpcNotification` — `{Method string, Params json.RawMessage}` for reading

## Routing

Provider: `codex`, Model: `gpt-5.4` — implementation with conformance spike first.

## Verification

### Static
- `go build ./dispatcher/...`
- `go vet ./dispatcher/...`
- Protocol conformance test passes against live app-server

### Runtime
- Integration test: create thread, start turn, receive events via rollout file tailing
- Integration test: send `turn/steer` mid-turn, verify JSON-RPC success response
- Integration test: send `turn/interrupt`, verify turn aborts and `turn/completed` notification received
- Integration test: rollout file discovery — verify correct file found within timeout
- Integration test: concurrent sessions — verify rollout files don't cross-bind
- Manual: `noodle start` with Codex provider, dispatch cook, verify events in web UI
