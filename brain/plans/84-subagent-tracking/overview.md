---
id: 84
created: 2026-02-28
status: active
---

# Sub-Agent Tracking

Back to [[plans/index]]

## Context

Claude Code and Codex both spawn sub-agents, but Noodle currently has no visibility into them. A single Noodle "session" may spawn dozens of sub-agents (Claude's Agent tool, Claude teams, Codex collab threads) with no way to see what they're doing, how they relate to each other, or to steer them.

The goal: parse sub-agent lifecycle into canonical events, track agent trees in session state, stream activity to the UI, and let users send messages to agents.

## Scope

**In scope:**
- Claude sub-agents (Agent tool) -- detected via `agent_progress` NDJSON events
- Claude teams (TeamCreate/SendMessage) -- detected via tool_use blocks + inbox files on disk
- Codex sub-agents -- detected via `session_meta.source.subagent` in session JSONL files + `collab_tool_call` items
- New canonical event types for agent lifecycle (spawn, progress, complete)
- Agent tree state in session snapshot (parent/child relationships, status, identity)
- UI: agent tree view, per-agent activity feed, agent steering (send message)
- Real-time streaming of sub-agent activity to UI via WebSocket

**Out of scope:**
- Changing how Noodle dispatches sessions (that's backend v2 territory)
- Claude Code SDK integration (we read NDJSON logs, not the SDK API)
- Codex TypeScript SDK integration
- Multi-session orchestration (team lead coordinating across Noodle sessions)

## Constraints

- The canonical event pipeline (`parse.CanonicalEvent` -> `event.Event` -> WebSocket -> UI) is the established pattern. Extend it, don't replace it.
- Claude sub-agents share the parent's session ID. Their transcript lives at `~/.claude/projects/{project}/{sessionId}/subagents/agent-{agentId}.jsonl`. Their activity streams in-band via `progress` messages with `data.type = "agent_progress"`.
- Codex sub-agents get their own JSONL session files with `session_meta.source.subagent.thread_spawn.parent_thread_id` linking back to the parent. Their activity is in their own file, not the parent's stream.
- Claude teams are separate processes. Each teammate has its own session. Communication is via file-based mailboxes at `~/.claude/teams/{name}/inboxes/{agent}.json`.
- The 64MB NDJSON line buffer in stamp is already in place.
- `CanonicalEvent` is intentionally flat (no nested types). Agent metadata should be optional fields, not a nested struct hierarchy.

## Alternatives Considered

1. **Extend CanonicalEvent with agent fields (chosen)** -- Add `AgentID`, `ParentAgentID`, `AgentName`, `AgentRole` optional fields to `CanonicalEvent`. Add new `EventType` variants (`EventAgentSpawn`, `EventAgentProgress`, `EventAgentComplete`). Parsers emit these alongside existing events. Minimal new plumbing.

2. **Separate AgentEvent type** -- Create a parallel event type with its own pipeline. More isolated but doubles the plumbing (writer, reader, broker, WebSocket message type, React Query cache). Not worth it for what amounts to a few extra fields.

3. **Watch filesystem for Codex/Claude sub-agent files** -- Instead of parsing NDJSON, watch `~/.codex/sessions/` and `~/.claude/projects/` for new files. Simpler but loses real-time streaming and requires polling. Also couples Noodle to provider-specific file layouts rather than NDJSON protocol.

## Applicable Skills

- `go-best-practices` -- all backend phases
- `react-best-practices` -- UI phases
- `ts-best-practices` -- TypeScript type definitions
- `testing` -- all phases

## Phases

1. [[plans/84-subagent-tracking/phase-01-canonical-agent-events]] -- Add agent event types and fields to CanonicalEvent
2. [[plans/84-subagent-tracking/phase-02-claude-subagent-parsing]] -- Parse Claude `agent_progress` and Agent tool_use into agent events
3. [[plans/84-subagent-tracking/phase-03-codex-subagent-parsing]] -- Parse Codex `session_meta.source.subagent` and `collab_tool_call` into agent events
4. [[plans/84-subagent-tracking/phase-04-claude-team-parsing]] -- Parse Claude team tool calls (TeamCreate, SendMessage) and inbox files
5. [[plans/84-subagent-tracking/phase-05-event-pipeline-agent-support]] -- Wire agent events through event storage, snapshot builder, and WebSocket broadcast
6. [[plans/84-subagent-tracking/phase-06-snapshot-agent-tree]] -- Add agent tree to session snapshot (parent/child relationships, identity, status)
7. [[plans/84-subagent-tracking/phase-07-ui-agent-tree]] -- Agent tree component in UI with expand/collapse and status indicators
8. [[plans/84-subagent-tracking/phase-08-ui-agent-feed]] -- Per-agent activity feed (click an agent to see its tool calls, text, errors)
9. [[plans/84-subagent-tracking/phase-09-ui-agent-steering]] -- Send messages to agents (Claude team inbox, Codex send_input, Claude interrupt+resume)
10. [[plans/84-subagent-tracking/phase-10-smoke-tests]] -- End-to-end tests with fixture NDJSON covering all agent patterns

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
cd ui && pnpm tsc --noEmit && pnpm test
```
