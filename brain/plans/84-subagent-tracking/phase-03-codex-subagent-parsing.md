Back to [[plans/84-subagent-tracking/overview]]

# Phase 3: Codex Sub-Agent Parsing

## Goal

Parse Codex sub-agent metadata from `session_meta` and `collab_tool_call` items into canonical agent events.

## Changes

**`parse/codex.go`** -- Handle two patterns:

1. **`session_meta` with subagent source** (extend existing `case "session_meta"` handler):
   When `payload.source` is an object containing `subagent.thread_spawn`, emit `EventAgentSpawn` with:
   - `AgentID` from `payload.id` (the sub-agent's own session/thread ID)
   - `ParentAgentID` from `source.subagent.thread_spawn.parent_thread_id`
   - `AgentName` from `payload.agent_nickname` (e.g., "Feynman", "Planck")
   - `AgentType` from `payload.agent_role` (e.g., "awaiter", "worker", "explorer")

   When `source` is a string (`"cli"`, `"exec"`), this is a root session -- emit the existing `EventInit` as before (no agent metadata).

2. **`collab_tool_call` items** (extend existing `case "collab_tool_call"` in `parseCodexItem()`):
   Currently only emits on error. Extend to also emit:
   - `item.started` with `tool: "spawn_agent"` -> `EventAgentSpawn` with `AgentName` from prompt summary
   - `item.started` with `tool: "send_input"` -> `EventAgentMessage` with target agent IDs from `receiver_thread_ids`
   - `item.completed` with `tool: "wait"` -> no event (just blocks)
   - `item.completed` with `tool: "close_agent"` -> `EventAgentComplete`
   - `item.completed` with `agents_states` map -> update agent status from state values

   Preserve the existing error handling for failed/error/cancelled status.

## Data Structures

- `codexSessionMeta` struct: `{ID, Source json.RawMessage, AgentNickname, AgentRole string}`
- `codexSubagentSource` struct: `{Subagent struct{ ThreadSpawn struct{ ParentThreadID, AgentNickname, AgentRole string, Depth int }}}`
- Extend existing `codexItem` -- already has `AgentsStates`, `Tool` fields

## Routing

Provider: `codex`, Model: `gpt-5.3-codex` -- mechanical parsing with clear field mappings.

## Verification

### Static
- `go test ./parse/...`
- New test cases with fixture NDJSON:
  - `session_meta` with `source: "cli"` (root, no agent events)
  - `session_meta` with `source.subagent.thread_spawn` (sub-agent spawn)
  - `collab_tool_call` item.started with `tool: "spawn_agent"`
  - `collab_tool_call` item.completed with `agents_states` showing running/completed/errored
  - `collab_tool_call` item.completed with `tool: "close_agent"`

### Runtime
- Feed fixture lines through `CodexAdapter.Parse()`, verify correct event types and agent metadata
- Verify existing codex tests still pass (backward compat)
