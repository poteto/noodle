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
   - `Steerable` = `true` (Codex collab agents accept `send_input`)

   When `source` is a string (`"cli"`, `"exec"`), this is a root session -- emit the existing `EventInit` as before (no agent metadata).

2. **`function_call` items in `response_item`** (extend `parseCodexItem()` for `function_call` type):
   Codex sub-agent orchestration uses `response_item` events containing `function_call` with specific function names. Real NDJSON uses this pattern — NOT `collab_tool_call` (which has zero matches in actual session logs):
   - `function_call` with `name: "spawn_agent"` -> `EventAgentSpawn` with `AgentName` from arguments summary; arguments: `{agent_type, message}`
   - `function_call` with `name: "send_input"` -> `EventAgentProgress` on the target agent; arguments: `{id, message}` where `id` is the child thread ID
   - `function_call` with `name: "wait"` -> no event (blocking call)
   - `function_call` with `name: "close_agent"` -> `EventAgentComplete`; arguments: `{id}` where `id` is the child thread ID

   Preserve the existing error handling for failed/error/cancelled status in `parseCodexItem()`.

**Codex file discovery:** Codex sub-agents live in separate JSONL files at `~/.codex/sessions/YYYY/MM/DD/`. The parent session's `function_call` events reference child thread IDs, but the child's NDJSON is a separate file that must be discovered and ingested. Phase 5 addresses how the session manager discovers and consumes these out-of-band files.

## Data Structures

- `codexSessionMeta` struct: `{ID, Source json.RawMessage, AgentNickname, AgentRole string}`
- `codexSubagentSource` struct: `{Subagent struct{ ThreadSpawn struct{ ParentThreadID, AgentNickname, AgentRole string, Depth int }}}`
- `codexFunctionCall` struct: `{Name string, Arguments json.RawMessage, CallID string}`
- Extend existing `codexItem` -- already has `AgentsStates`, `Tool` fields

## Routing

Provider: `codex`, Model: `gpt-5.3-codex` -- mechanical parsing with clear field mappings.

## Verification

### Static
- `go test ./parse/...`
- New test cases with fixture NDJSON:
  - `session_meta` with `source: "cli"` (root, no agent events)
  - `session_meta` with `source.subagent.thread_spawn` (sub-agent spawn)
  - `response_item` with `function_call` name `spawn_agent` -> EventAgentSpawn
  - `response_item` with `function_call` name `send_input` -> EventAgentProgress
  - `response_item` with `function_call` name `close_agent` -> EventAgentComplete
  - `function_call_output` from `spawn_agent` (contains child thread ID)

### Runtime
- Feed fixture lines through `CodexAdapter.Parse()`, verify correct event types and agent metadata
- Verify existing codex tests still pass (backward compat)
