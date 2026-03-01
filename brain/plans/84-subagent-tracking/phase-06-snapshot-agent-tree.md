Back to [[plans/84-subagent-tracking/overview]]

# Phase 6: Snapshot Agent Tree

## Goal

Add an agent tree to the session snapshot so the UI can render the parent/child hierarchy of agents within a session.

## Changes

**`internal/snapshot/types.go`** -- `AgentNode` struct is already defined (Phase 1). Add the `Agents` field to `Session`:

```
Session.Agents: []AgentNode
```

Status values: `running`, `completed`, `errored`, `shutdown`.

**`internal/snapshot/snapshot_builder.go`** -- `enrichActiveSession()` currently reads the event log and sets `CurrentAction`. Extend it to:
1. Scan for `EventAgentSpawned` events to build the agent list
2. Scan for `EventAgentExited` events to set completion status
3. Scan for `EventAgentProgress` events to set each agent's `CurrentAction` (last action)
4. Build the tree by resolving `ParentID` references

v1 Codex fallback materialization rule:
- if an agent-scoped `EventAgentProgress`/`EventAgentExited` exists with `agent_id` but no prior `EventAgentSpawned`, materialize a placeholder node keyed by that canonical `agent_id`
- metadata precedence for placeholder upsert: spawn payload (if later discovered) > progress/exit payload > defaults (`AgentName=agent_id`, `AgentType="codex-subagent"`, `ParentID=""`)
- placeholder nodes remain valid for v1 and merge into canonical spawn metadata when child `session_meta` is later ingested

Identity/dedupe rule: materialize nodes by canonical `AgentID` only. If multiple spawn-like signals refer to the same canonical agent, merge metadata into one node (no duplicates). Provisional correlation IDs are never used as node keys.

Performance guardrail: keep per-snapshot tree rebuild linear in event count with a target budget of <50ms per active session at 10k events; if exceeded, add incremental/cache-based materialization in v2.

The agent tree is built from the event log on each snapshot rebuild (same pattern as `CurrentAction` today). No separate state storage needed.

**`ui/src/client/generated-types.ts`** -- Add `AgentNode` type and `agents` field to `Session` (generated from Go types or manually kept in sync).

## Data Structures

- `AgentNode` struct defined in Phase 1 (9 fields including Steerable)
- `Session.Agents []AgentNode` -- flat list, tree structure implied by ParentID references

## Routing

Provider: `codex`, Model: `gpt-5.3-codex` -- straightforward struct additions and event log scanning.

## Verification

### Static
- `go test ./internal/snapshot/...`
- Unit test: build snapshot from fixture event log containing spawn/progress/exit events, verify agent tree and no duplicate nodes for a single canonical agent
- Unit test: Codex orchestration-only fixture (progress/exit without spawn) still materializes one placeholder node keyed by `agent_id`

### Runtime
- Integration test: run stamp processor with fixture NDJSON containing agent events, build snapshot, verify `Session.Agents` is populated with correct hierarchy
- Benchmark-style runtime check on large fixture to confirm rebuild budget is met or flagged
