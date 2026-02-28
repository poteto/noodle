Back to [[plans/84-subagent-tracking/overview]]

# Phase 5: Event Pipeline Agent Support

## Goal

Wire the new agent canonical events through the full pipeline: event storage, session consumption, and WebSocket broadcast. After this phase, agent events flow end-to-end from parser to browser.

## Changes

**`dispatcher/session_helpers.go`** -- `eventFromCanonical()` already handles the existing event types. Two changes:

1. Add cases for the 3 new event types:
   - `parse.EventAgentSpawn` -> `event.EventAgentSpawned` with payload `{agent_id, parent_agent_id, agent_name, agent_type, team_name, steerable, message}`
   - `parse.EventAgentProgress` -> `event.EventAgentProgress` with payload `{agent_id, agent_name, message, tool, summary}`
   - `parse.EventAgentComplete` -> `event.EventAgentExited` with payload `{agent_id, agent_name, outcome, message}`

2. For ALL existing event types (EventAction, EventError, etc.): when `CanonicalEvent.AgentID` is non-empty, include `agent_id` in the event payload. This ensures inner agent actions (tool calls, text output parsed from agent_progress) carry the agent's identity through to the UI.

**`dispatcher/session_base.go`** -- No changes needed. The existing pipeline handles new event types automatically.

**`server/session_broker.go`** -- No changes needed. The broker fans out all `event.Event` objects regardless of type.

**`server/ws_hub.go`** -- Use `session_event` with agent fields in the data payload. The UI filters by `agent_id`. No new WebSocket message type needed.

**`internal/snapshot/types.go`** -- Extend Go `EventLine` struct with optional agent fields:
- `AgentID string json:"agent_id,omitempty"`
- `AgentName string json:"agent_name,omitempty"`
- `AgentType string json:"agent_type,omitempty"`

This is the source of truth — `ui/src/client/generated-types.ts` is generated from this via tygo.

**`internal/snapshot/session_events.go`** -- `FormatSingleEvent()` must extract agent fields from the event payload and populate the new EventLine fields. All event types (not just EventAgent*) may carry `agent_id` in their payload.

Without these changes, agent metadata is lost at the Go -> JSON -> WebSocket boundary. The UI needs `agent_id` on every EventLine to filter events per-agent in Phase 8.

**Out-of-band file discovery:** Two agent sources live outside the parent's NDJSON stream:
1. **Codex sub-agent session files** — when the parent parser sees a `spawn_agent` function_call with a child thread ID, the session manager starts tailing the corresponding child NDJSON file at `~/.codex/sessions/YYYY/MM/DD/{thread_id}.jsonl`. Child events are tagged with the agent's ID and fed into the same event pipeline.
2. **Claude team inbox files** — polled/watched by the session manager. New messages converted to agent events via the Phase 4 parser.

The session manager needs a method like `discoverSubAgentFiles(parentSessionID)` that maps known child agent IDs to their file paths and starts ingestion.

## Data Structures

- Agent event payload: `{agent_id, parent_agent_id, agent_name, agent_type, steerable, message, tool, summary, outcome}`
- All fields optional/omitempty -- each event type populates the relevant subset
- Extended Go `EventLine` struct: `{At, Label, Body, Category, AgentID, AgentName, AgentType}` (generated to TS via tygo)

## Routing

Provider: `codex`, Model: `gpt-5.3-codex` -- mechanical wiring with clear mappings.

## Verification

### Static
- `go test ./dispatcher/... ./server/...`
- `cd ui && pnpm tsc --noEmit`
- Unit test: construct each new canonical event type, verify it maps to the correct event.Event
- Unit test: verify WebSocket message payload includes agent fields
- Unit test: EventLine TypeScript type includes optional agent_id, agent_name, agent_type

### Runtime
- Integration test: feed agent NDJSON through stamp processor -> session_base -> capture WebSocket output, verify agent events arrive with agent metadata
- Verify out-of-band Codex file discovery: parent spawn event triggers child file tailing
