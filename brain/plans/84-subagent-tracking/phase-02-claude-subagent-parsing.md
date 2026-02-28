Back to [[plans/84-subagent-tracking/overview]]

# Phase 2: Claude Sub-Agent Parsing

## Goal

Parse Claude's `agent_progress` NDJSON events and Agent `tool_use` blocks into canonical agent events so Noodle can track Claude sub-agents in real-time.

## Changes

**`parse/claude.go`** -- Handle two new patterns:

1. **Agent tool_use detection** (in existing `case "assistant"` handler):
   When a `tool_use` block has `name: "Agent"`, emit `EventAgentSpawn` with:
   - `AgentName` from `input.name` or `input.description`
   - `AgentType` from `input.subagent_type`
   - `AgentID` from the `tool_use` block's `id` field (used as correlation key until the real agentId arrives)

2. **`progress` message type** (new case in `Parse()` switch):
   When `type == "progress"` and `data.type == "agent_progress"`, emit `EventAgentProgress` with:
   - `AgentID` from `data.agentId`
   - `AgentName` from `data.prompt` (first 80 chars as summary)
   - The inner `data.message` parsed into the same canonical action/error events the parent parser would produce, but with `AgentID` set

   This gives real-time visibility into what each sub-agent is doing (every tool call, every text response).

3. **Agent tool_result detection** (in existing `case "user"` handler):
   When a `tool_result` corresponds to an Agent tool_use (match by `tool_use_id`), emit `EventAgentComplete` with:
   - `AgentID` extracted from the result text (pattern: `agentId: {hex}`)

## Data Structures

- `claudeProgressData` struct: `{Type string, AgentID string, Prompt string, Message json.RawMessage}`
- Reuse existing `claudeMessage` / `claudeContent` for parsing the inner agent message

## Routing

Provider: `claude`, Model: `claude-opus-4-6` -- parsing nested message structures requires judgment about what to extract vs skip.

## Verification

### Static
- `go test ./parse/...`
- New test file with fixture NDJSON lines covering:
  - Agent tool_use with subagent_type and name
  - agent_progress with assistant message containing tool_use
  - agent_progress with user message containing tool_result
  - Agent tool_result with agentId in text

### Runtime
- Feed fixture NDJSON through `ClaudeAdapter.Parse()`, verify correct event types and agent metadata
- Verify that non-agent `progress` events (bash_progress, hook_progress) are still ignored
