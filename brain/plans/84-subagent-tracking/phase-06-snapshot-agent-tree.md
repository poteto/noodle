Back to [[plans/84-subagent-tracking/overview]]

# Phase 6: Snapshot Agent Tree

## Goal

Add an agent tree to the session snapshot so the UI can render the parent/child hierarchy of agents within a session.

## Changes

**`internal/snapshot/types.go`** -- Add `AgentNode` struct and `Agents` field to `Session`:

```
AgentNode: {ID, ParentID, Name, Type, Status, CurrentAction, SpawnedAt, CompletedAt}
Session.Agents: []AgentNode
```

Status values: `running`, `completed`, `errored`, `shutdown`.

**`internal/snapshot/snapshot_builder.go`** -- `enrichActiveSession()` currently reads the event log and sets `CurrentAction`. Extend it to:
1. Scan for `EventAgentSpawned` events to build the agent list
2. Scan for `EventAgentExited` events to set completion status
3. Scan for `EventAgentProgress` events to set each agent's `CurrentAction` (last action)
4. Build the tree by resolving `ParentID` references

The agent tree is built from the event log on each snapshot rebuild (same pattern as `CurrentAction` today). No separate state storage needed.

**`ui/src/client/generated-types.ts`** -- Add `AgentNode` type and `agents` field to `Session` (generated from Go types or manually kept in sync).

## Data Structures

- `AgentNode` struct with 8 fields (see above)
- `Session.Agents []AgentNode` -- flat list, tree structure implied by ParentID references

## Routing

Provider: `codex`, Model: `gpt-5.3-codex` -- straightforward struct additions and event log scanning.

## Verification

### Static
- `go test ./internal/snapshot/...`
- Unit test: build snapshot from fixture event log containing spawn/progress/exit events, verify agent tree

### Runtime
- Integration test: run stamp processor with fixture NDJSON containing agent events, build snapshot, verify `Session.Agents` is populated with correct hierarchy
