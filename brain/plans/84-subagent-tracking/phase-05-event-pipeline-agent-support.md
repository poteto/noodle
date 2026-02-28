Back to [[plans/84-subagent-tracking/overview]]

# Phase 5: Event Pipeline Agent Support

## Goal

Wire the new agent canonical events through the full pipeline: event storage, session consumption, and WebSocket broadcast. After this phase, agent events flow end-to-end from parser to browser.

## Changes

**`dispatcher/session_helpers.go`** -- `eventFromCanonical()` already handles the existing event types. Add cases for the 3 new types:
- `parse.EventAgentSpawn` -> `event.EventAgentSpawned` with payload `{agent_id, parent_agent_id, agent_name, agent_type, steerable, message}`
- `parse.EventAgentProgress` -> `event.EventAgentProgress` with payload `{agent_id, agent_name, message, tool, summary}`
- `parse.EventAgentComplete` -> `event.EventAgentExited` with payload `{agent_id, agent_name, outcome, message}`

**`dispatcher/session_base.go`** -- `consumeCanonicalLine()` already routes all canonical events through `eventFromCanonical()` and publishes via sink. No changes needed here -- the existing pipeline handles new event types automatically.

**`server/session_broker.go`** -- No changes needed. The broker fans out all `event.Event` objects regardless of type.

**`server/ws_hub.go`** -- Use `session_event` with agent fields in the data payload. The UI filters by `agent_id`. No new WebSocket message type needed.

**`ui/src/client/types.ts`** -- Extend `EventLine` with optional agent fields:
- `agent_id?: string` — which agent produced this event (empty = root agent)
- `agent_name?: string` — human-readable name
- `agent_type?: string` — role/type

Without these fields, agent metadata is dropped before reaching the UI (EventLine currently only has `{at, label, body, category}`). The UI needs `agent_id` to filter events per-agent in Phase 8.

**Out-of-band file discovery:** Two agent sources live outside the parent's NDJSON stream:
1. **Codex sub-agent session files** — when the parent parser sees a `spawn_agent` function_call with a child thread ID, the session manager starts tailing the corresponding child NDJSON file at `~/.codex/sessions/YYYY/MM/DD/{thread_id}.jsonl`. Child events are tagged with the agent's ID and fed into the same event pipeline.
2. **Claude team inbox files** — polled/watched by the session manager. New messages converted to agent events via the Phase 4 parser.

The session manager needs a method like `discoverSubAgentFiles(parentSessionID)` that maps known child agent IDs to their file paths and starts ingestion.

## Data Structures

- Agent event payload: `{agent_id, parent_agent_id, agent_name, agent_type, steerable, message, tool, summary, outcome}`
- All fields optional/omitempty -- each event type populates the relevant subset
- Extended `EventLine`: `{at, label, body, category, agent_id?, agent_name?, agent_type?}`

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
