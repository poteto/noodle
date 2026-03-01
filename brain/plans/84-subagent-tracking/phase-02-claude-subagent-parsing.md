Back to [[plans/84-subagent-tracking/overview]]

# Phase 2: Claude Sub-Agent Parsing

## Goal

Parse Claude's `agent_progress` NDJSON events and Agent `tool_use` blocks into canonical agent events so Noodle can track Claude sub-agents in real-time.

## Changes

**`parse/claude.go`** -- Handle three patterns:

1. **Agent tool_use detection** (in existing `case "assistant"` handler):
   When a `tool_use` block has `name: "Agent"`, record a provisional correlation entry keyed by the `tool_use` block `id`:
   - `AgentName` from `input.name` or `input.description`
   - `AgentType` from `input.subagent_type`
   - `tool_use_id` as provisional correlation only (not canonical `AgentID`)

   Do not emit `EventAgentSpawn` with provisional `tool_use_id`.

   Correlation-map lifecycle requirements:
   - Map scope is per-session parser context (never shared across sessions)
   - Remove entry when matching `EventAgentComplete` is emitted
   - Enforce bounded size (LRU or FIFO cap) and TTL eviction for orphaned entries
   - Ownership contract for v1:
     - One `stamp.Processor`/`parse.Registry` instance is created per active session ingest stream
     - `ClaudeAdapter` reconciliation state is stored on that session-local adapter instance only
     - No package-level/shared singleton adapter may hold reconciliation state
   - Parser state must be concurrency-safe within a session stream (mutex-protected map)

2. **`progress` message type** (new case in `Parse()` switch):
   When `type == "progress"` and `data.type == "agent_progress"`, emit `EventAgentProgress` with:
   - `AgentID` from `data.agentId`
   - `AgentName` from `data.prompt` (first 80 chars as summary)
   - The inner `data.message` parsed into the same canonical action/error events the parent parser would produce, but with `AgentID` set

   If this is the first stable `data.agentId` seen for a pending tool_use, emit `EventAgentSpawn` using that stable `AgentID` before progress events.
   If `agent_progress` arrives with a stable `data.agentId` and no pending tool_use mapping, still emit `EventAgentSpawn` (best-effort metadata) before progress so the node is materialized.

   This gives real-time visibility into what each sub-agent is doing (every tool call, every text response) without provisional-ID nodes.

3. **Agent tool_result detection** (in existing `case "user"` handler):
   When a `tool_result` corresponds to an Agent tool_use (match by `tool_use_id`), emit `EventAgentComplete` with:
   - `AgentID` from reconciliation map first (`tool_use_id -> agentId`)
   - Fallback: parse a single unambiguous ID from result text, accepting both `agentId: {hex}` and `agent_id: {hex}` patterns (Claude uses both camelCase and snake_case across versions)
   - If multiple IDs or no ID can be resolved, do not emit `EventAgentComplete` for that line; emit a non-lifecycle warning/action event so the failure is observable

**ID reconciliation:** The Agent tool_use `id` field is a temporary correlation key. The real `agentId` arrives later in `agent_progress` events. The parser maintains a map from `tool_use_id` -> `agentId` so that when the tool_result arrives, it can emit the correct canonical `AgentID`.

Do not fall back to `tool_use_id` as canonical `AgentID`; keep provisional IDs internal-only.

All Claude sub-agents are non-steerable (`Steerable: false`) — they run to completion with no input channel.

## Data Structures

- `claudeProgressData` struct: `{Type string, AgentID string, Prompt string, Message json.RawMessage}`
- `tool_use_id` -> `agentId` reconciliation map (built during parse, not persisted) on session-local `ClaudeAdapter` instance:
  - `mu sync.Mutex`
  - `pending map[string]agentCorrelation`
  - `agentCorrelation`: `{ToolUseID, AgentName, AgentType, FirstSeenAt, CanonicalAgentID}`
- Canonical identity rule: only stable `agentId` is serialized as `AgentID`.
- Reuse existing `claudeMessage` / `claudeContent` for parsing the inner agent message
- `claudeContent` must include `ID string json:"id"` for tool_use correlation
- `claudeLine` must include `Data json.RawMessage json:"data"` for `progress` parsing

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
  - Assert no canonical event uses provisional `tool_use_id` as `AgentID`
  - agent_progress with stable `agentId` but no pending tool_use still materializes a spawn node
  - map lifecycle tests: completion cleanup, TTL/size eviction, and no cross-session contamination
  - tool_result text with multiple IDs does not emit ambiguous `EventAgentComplete`
  - session ownership test: two concurrent session processors cannot observe each other's tool_use correlations

### Runtime
- Feed fixture NDJSON through `ClaudeAdapter.Parse()`, verify correct event types and agent metadata
- Verify that non-agent `progress` events (bash_progress, hook_progress) are still ignored
