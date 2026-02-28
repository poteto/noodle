Back to [[plans/84-subagent-tracking/overview]]

# Phase 1: Canonical Agent Events

## Goal

Extend `CanonicalEvent` and `event.Event` with agent-specific types and metadata fields so parsers can emit agent lifecycle events.

## Changes

**`parse/canonical.go`** -- Add new event types and optional agent fields:
- `EventAgentSpawn` -- a sub-agent was created
- `EventAgentProgress` -- a sub-agent produced output (tool call, text, etc.)
- `EventAgentComplete` -- a sub-agent finished (success or error)
- `EventAgentMessage` -- an inter-agent message (team inbox, collab send_input)

Add optional fields to `CanonicalEvent`:
- `AgentID string` -- unique agent identifier
- `ParentAgentID string` -- parent's agent ID (empty for root)
- `AgentName string` -- human-readable name (slug, nickname)
- `AgentType string` -- role/type (Explore, worker, awaiter, team member name)

**`event/types.go`** -- Add corresponding event types:
- `EventAgentSpawned`
- `EventAgentProgress`
- `EventAgentExited`
- `EventAgentMessage`

**`dispatcher/session_helpers.go`** -- Extend `eventFromCanonical()` to map the new parse event types to the new event types, carrying agent metadata in the payload.

## Data Structures

- `CanonicalEvent` gains 4 string fields (AgentID, ParentAgentID, AgentName, AgentType) -- all `omitempty`
- New `EventType` constants (4 values)
- Agent payload shape: `{agent_id, parent_agent_id, agent_name, agent_type, message}`

## Routing

Provider: `codex`, Model: `gpt-5.3-codex` -- mechanical type additions with clear spec.

## Verification

### Static
- `go test ./parse/... ./event/... ./dispatcher/...`
- `go vet ./...`
- Existing tests still pass (no breaking changes to CanonicalEvent)

### Runtime
- Unit test: construct CanonicalEvent with agent fields, verify JSON round-trip
- Unit test: `eventFromCanonical()` maps each new event type correctly
