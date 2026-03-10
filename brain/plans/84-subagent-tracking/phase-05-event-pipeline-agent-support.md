Back to [[plans/84-subagent-tracking/overview]]

# Phase 5: Event Pipeline Agent Support

## Goal

Wire the new agent canonical events through the full pipeline: event storage, session consumption, and WebSocket broadcast. After this phase, agent events flow end-to-end from parser to browser.

## Changes

**`dispatcher/session_helpers.go`** -- in the canonical-to-event mapping path (`eventFromCanonical()` and adjacent helpers), apply two changes:

1. Add cases for the 3 new event types:
   - `parse.EventAgentSpawn` -> `event.EventAgentSpawned` with payload `{agent_id, parent_agent_id, agent_name, agent_type, team_name, steerable, message}`
   - `parse.EventAgentProgress` -> `event.EventAgentProgress` with payload `{agent_id, agent_name, message, tool, summary}`
   - `parse.EventAgentComplete` -> `event.EventAgentExited` with payload `{agent_id, agent_name, outcome, message}`

2. For ALL existing event types (EventAction, EventError, etc.): when `CanonicalEvent.AgentID` is non-empty, include `agent_id` in the event payload. This ensures inner agent actions (tool calls, text output parsed from agent_progress) carry the agent's identity through to the UI.

**`dispatcher/session_base.go`** -- No changes needed. The existing pipeline handles new event types automatically.

**`server/session_broker.go`** -- No broker-specific branching changes are expected. Verification in this phase must confirm that the expanded `EventLine` payload shape survives `session_event` serialization unchanged once `agent_id` / `agent_name` / `agent_type` fields are added.

**`server/ws_hub.go`** -- Keep using `session_event` with the expanded `EventLine` payload. The UI filters by `agent_id`. No new WebSocket message type needed.

**`internal/snapshot/types.go`** -- Extend Go `EventLine` struct with optional agent fields:
- `AgentID string json:"agent_id,omitempty"`
- `AgentName string json:"agent_name,omitempty"`
- `AgentType string json:"agent_type,omitempty"`

This is the source of truth — `ui/src/client/generated-types.ts` is generated from this via tygo.

**`dispatcher/session_helpers.go` + `internal/snapshot/session_events.go`** -- Ensure the existing canonical->event formatting path and snapshot event-line formatting both carry agent fields end-to-end. All event types (not just EventAgent*) may carry `agent_id` in payload.

Without these changes, agent metadata is lost at the Go -> JSON -> WebSocket boundary. The UI needs `agent_id` on every EventLine to filter events per-agent in Phase 8.

**Deferred follow-up:** Out-of-band discovery for Codex child session files and Claude team inbox files is intentionally excluded from phase 5. V1 ends at in-band pipeline wiring plus explicit orchestration-only fallback rows; plan 88 owns discovery, bounded polling, lifecycle cleanup, and replay behavior for external child streams.

v1 fallback contract for Codex child visibility (explicit):
- retain parent-stream orchestration events (`spawn_agent`, `send_input`, `close_agent`) as normal session events
- when child thread ID is known, those events must carry `agent_id` so downstream snapshot/UI can key agent rows
- mark orchestration-only rows with summary text `"orchestration-only: child transcript pending ingestion"`
- do not drop/merge away orchestration-only rows even when no `EventAgentSpawned` exists yet

## Data Structures

- Agent event payload: `{agent_id, parent_agent_id, agent_name, agent_type, steerable, message, tool, summary, outcome}`
- All fields optional/omitempty -- each event type populates the relevant subset
- Extended Go `EventLine` struct: `{At, Label, Body, Category, AgentID, AgentName, AgentType}` (generated to TS via tygo)

## Routing

Provider: `codex`, Model: `gpt-5.4` -- mechanical wiring with clear mappings.

## Verification

### Static
- `go test ./dispatcher/... ./server/...`
- `cd ui && pnpm tsc --noEmit`
- Unit test: construct each new canonical event type, verify it maps to the correct event.Event
- Unit test: verify WebSocket message payload includes agent fields
- Unit test: EventLine TypeScript type includes optional agent_id, agent_name, agent_type
- Unit test: orchestration-only Codex rows preserve `agent_id` and summary marker when no child transcript exists

### Runtime
- Integration test: feed agent NDJSON through stamp processor -> session_base -> capture WebSocket output, verify agent events arrive with agent metadata
- Out-of-band ingest tests belong to plan 88, not this phase
