Back to [[plans/84-subagent-tracking/overview]]

# Phase 3: Codex Sub-Agent Parsing

## Goal

Parse Codex sub-agent metadata from `session_meta` and `function_call` items into canonical agent events.

## Changes

**`parse/codex.go`** -- Handle two patterns:

1. **`session_meta` with subagent source** (extend existing `case "session_meta"` handler):
   When `payload.source` is an object containing `subagent.thread_spawn`, emit `EventAgentSpawn` with:
   - `AgentID` from `payload.id` (the sub-agent's own session/thread ID)
   - `ParentAgentID` from `source.subagent.thread_spawn.parent_thread_id`
   - `AgentName` from `payload.agent_nickname` (e.g., "Feynman", "Planck")
   - `AgentType` from `payload.agent_role` (e.g., "awaiter", "worker", "explorer")
   - `Steerable` = `false` in v1 (tracking only)

   When `source` is a string (`"cli"`, `"exec"`), this is a root session -- emit the existing `EventInit` as before (no agent metadata).

   This is the canonical spawn source for Codex agent nodes.
   Unknown `source` shapes must emit an observable warning/action event (not silent drop) so protocol drift is detectable.

2. **`function_call` items in `response_item`** (extend `parseCodexResponseItem()` for `function_call` type):
   Codex sub-agent orchestration uses `response_item` events containing `function_call` with specific function names. Real NDJSON uses this pattern — NOT `collab_tool_call` (which has zero matches in actual session logs):
   - `function_call` with `name: "spawn_agent"` -> `EventAgentProgress` intent event only (request to spawn), not canonical spawn. Include child thread ID in `agent_id` when present in arguments/output so downstream snapshot can key a placeholder node deterministically.
   - `function_call` with `name: "send_input"` -> `EventAgentProgress` on the target agent; arguments: `{id, message}` where `id` is the child thread ID
   - `function_call` with `name: "wait"` -> no event (blocking call)
   - `function_call` with `name: "close_agent"` -> `EventAgentComplete`; arguments: `{id}` where `id` is the child thread ID

   Preserve the existing error handling for failed/error/cancelled status in `parseCodexItem()`.

**Codex file discovery:** Codex sub-agents live in separate JSONL files at `~/.codex/sessions/YYYY/MM/DD/`. The parent session's `function_call` events reference child thread IDs, but the child's NDJSON is a separate file that must be discovered and ingested. Phase 5 addresses how the runtime ingest path discovers and consumes these out-of-band files.

v1 behavior note: until out-of-band ingestion is enabled, Codex child nodes may be represented by orchestration-only activity (spawn/send_input/close) without full child transcript. Contract:
- parser emits agent-scoped orchestration events with stable `agent_id` when child thread ID is known
- event summary includes `"orchestration-only: child transcript pending ingestion"` so UI can distinguish this from "no activity"

Steering note: current process-dispatch Codex sessions are not live-steerable because stdin is closed after the initial prompt and there is no controller path for per-child `send_input`. Plan 84 should surface Codex child visibility in v1 and reserve Codex child steering for a future runtime/control-plane change.

## Data Structures

- `codexSessionMeta` struct: `{ID, Source json.RawMessage, AgentNickname, AgentRole string}`
- `codexSubagentSource` struct: `{Subagent struct{ ThreadSpawn struct{ ParentThreadID, AgentNickname, AgentRole string, Depth int }}}`
- `codexFunctionCall` struct: `{Name string, Arguments json.RawMessage, CallID string}`
- Extend existing `codexItem` with function_call name dispatch

## Routing

Provider: `codex`, Model: `gpt-5.4` -- mechanical parsing with clear field mappings.

## Verification

### Static
- `go test ./parse/...`
- New test cases with fixture NDJSON:
  - `session_meta` with `source: "cli"` (root, no agent events)
  - `session_meta` with `source.subagent.thread_spawn` (sub-agent spawn)
  - `response_item` with `function_call` name `spawn_agent` -> EventAgentProgress intent (no spawn)
  - `response_item` with `function_call` name `send_input` -> EventAgentProgress
  - `response_item` with `function_call` name `close_agent` -> EventAgentComplete
  - `function_call_output` from `spawn_agent` (contains child thread ID)
  - When both `session_meta` and `spawn_agent` exist, only one canonical spawn node is produced
  - Unknown `session_meta.source` shape emits warning/action event

### Runtime
- Feed fixture lines through `CodexAdapter.Parse()`, verify correct event types and agent metadata
- Verify existing codex tests still pass (backward compat)
- Assert Codex child nodes materialize as non-steerable in v1
