Back to [[plans/84-subagent-tracking/overview]]

# Phase 5: Event Pipeline Agent Support

## Goal

Wire the new agent canonical events through the full pipeline: event storage, session consumption, and WebSocket broadcast. After this phase, agent events flow end-to-end from parser to browser.

## Changes

**`dispatcher/session_helpers.go`** -- `eventFromCanonical()` already handles the existing event types. Add cases for the 4 new types:
- `parse.EventAgentSpawn` -> `event.EventAgentSpawned` with payload `{agent_id, parent_agent_id, agent_name, agent_type, message}`
- `parse.EventAgentProgress` -> `event.EventAgentProgress` with payload `{agent_id, agent_name, message, tool, summary}`
- `parse.EventAgentComplete` -> `event.EventAgentExited` with payload `{agent_id, agent_name, outcome, message}`
- `parse.EventAgentMessage` -> `event.EventAgentMessage` with payload `{agent_id, agent_name, message, direction}` (direction: sent/received)

**`dispatcher/session_base.go`** -- `consumeCanonicalLine()` already routes all canonical events through `eventFromCanonical()` and publishes via sink. No changes needed here -- the existing pipeline handles new event types automatically.

**`server/session_broker.go`** -- No changes needed. The broker fans out all `event.Event` objects regardless of type.

**`server/ws_hub.go`** -- The WebSocket hub broadcasts `session_event` messages. Add a new message type `agent_event` that carries agent metadata alongside the event, so the UI can route events to the correct agent node. Alternatively, just send them as `session_event` with agent fields in the data payload (simpler, less plumbing).

Decision: use `session_event` with agent fields in the payload. The UI can filter by `agent_id`.

## Data Structures

- Agent event payload: `{agent_id, parent_agent_id, agent_name, agent_type, message, tool, summary, outcome, direction}`
- All fields optional/omitempty -- each event type populates the relevant subset

## Routing

Provider: `codex`, Model: `gpt-5.3-codex` -- mechanical wiring with clear mappings.

## Verification

### Static
- `go test ./dispatcher/... ./server/...`
- Unit test: construct each new canonical event type, verify it maps to the correct event.Event
- Unit test: verify WebSocket message format includes agent fields

### Runtime
- Integration test: feed agent NDJSON through stamp processor -> session_base -> capture WebSocket output, verify agent events arrive
