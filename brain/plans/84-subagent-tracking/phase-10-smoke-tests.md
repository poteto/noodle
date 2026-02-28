Back to [[plans/84-subagent-tracking/overview]]

# Phase 10: Smoke Tests

## Goal

End-to-end tests using fixture NDJSON files that exercise the full agent tracking pipeline: parse -> event -> snapshot -> (optionally) WebSocket.

## Changes

**`parse/testdata/` or `testdata/fixtures/`** -- Create fixture NDJSON files:

1. `claude_subagent.ndjson` -- A Claude session that spawns 2 sub-agents via the Agent tool:
   - System init
   - Assistant message with Agent tool_use (Explore type)
   - Multiple agent_progress events showing sub-agent tool calls
   - Agent tool_result with agentId
   - Second Agent tool_use (general-purpose type, run_in_background)
   - Its agent_progress events interleaved with first agent
   - Both complete

2. `claude_team.ndjson` -- A Claude session that creates a team:
   - TeamCreate tool_use (with team_name, description)
   - Agent tool_use with team_name (spawning teammate, Steerable=true)
   - SendMessage tool_use (type: message) -> EventAgentProgress
   - `<teammate-message>` user message (XML wrapper with teammate_id, color, summary)
   - SendMessage tool_use (type: shutdown_request)
   - Agent tool_result (teammate completed, agent_id: name@team)

3. `codex_subagent.ndjson` -- A Codex parent session with collab:
   - session_meta with source: "exec" (root)
   - response_item with function_call name: "spawn_agent" (arguments: {agent_type, message})
   - function_call_output from spawn_agent (child thread ID)
   - response_item with function_call name: "send_input" (arguments: {id, message})
   - response_item with function_call name: "close_agent" (arguments: {id})

4. `codex_subagent_child.ndjson` -- A Codex sub-agent session file:
   - session_meta with source.subagent.thread_spawn (depth 1, nickname, role)
   - Regular tool calls and completion

**`parse/agent_test.go`** -- Test each fixture through the adapter, verify:
- Correct number and types of agent events emitted
- Agent IDs, names, types populated correctly
- Parent-child relationships correct
- Non-agent events still parsed normally alongside agent events

**`internal/snapshot/agent_tree_test.go`** -- Test snapshot builder:
- Feed agent events into event log
- Build snapshot
- Verify Session.Agents tree is correct

## Data Structures

- Fixture files are plain NDJSON (one JSON object per line)
- Test assertions on `[]CanonicalEvent` and `[]AgentNode`

## Routing

Provider: `codex`, Model: `gpt-5.3-codex` -- mechanical test writing from clear fixture specs.

## Verification

### Static
- `go test ./parse/... ./internal/snapshot/...`
- All fixture files are valid NDJSON (parse without error)

### Runtime
- All test assertions pass
- Coverage: each agent event type has at least one fixture line and one assertion
- `go test -race ./...` passes (no data races in agent event handling)
