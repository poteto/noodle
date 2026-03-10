---
id: 84
created: 2026-02-28
status: active
---

# Sub-Agent Tracking

Back to [[plans/index]]

## Context

Claude Code and Codex both spawn sub-agents, but Noodle currently has no visibility into them. A single Noodle "session" may spawn dozens of sub-agents (Claude's Agent tool, Claude teams, Codex collab threads) with no way to see what they're doing, how they relate to each other, or to steer them.

The goal: parse sub-agent lifecycle into canonical events, track agent trees in session state, stream activity to the UI, and let users send messages to steerable agents without leaving the existing actor view.

## Scope

**In scope:**
- Claude sub-agents (Agent tool) -- detected via `agent_progress` NDJSON events
- Claude teams (TeamCreate/SendMessage) -- detected via in-band tool_use blocks and teammate-message activity already visible in session logs
- Codex sub-agents -- detected via `function_call` orchestration in the parent stream plus stable child IDs when available; full child transcript ingestion is follow-up work
- New canonical event types for agent lifecycle (spawn, progress, complete)
- Agent tree state in session snapshot (parent/child relationships, status, identity)
- UI: actor-route agent tree, per-agent activity filtering inside the existing session feed, and agent steering through the existing control plane for providers that already expose a live input channel
- Real-time streaming of sub-agent activity to UI via WebSocket

**Out of scope:**
- Out-of-band ingestion for Codex child session files and Claude team inbox files (deferred to [[plans/88-subagent-tracking-v2/overview]])
- A dedicated per-agent page or second chat surface if the existing `/actor/$id` view can host selection and steering cleanly
- Changing how Noodle dispatches sessions (that's backend v2 territory)
- Claude Code SDK integration (we read NDJSON logs, not the SDK API)
- Codex TypeScript SDK integration
- Multi-session orchestration (team lead coordinating across Noodle sessions)

## Constraints

- The canonical event pipeline (`parse.CanonicalEvent` -> `event.Event` -> WebSocket -> UI) is the established pattern. Extend it, don't replace it.
- Claude sub-agents share the parent's session ID. Their transcript lives at `~/.claude/projects/{project}/{sessionId}/subagents/agent-{agentId}.jsonl`. Their activity streams in-band via `progress` messages with `data.type = "agent_progress"`.
- Codex sub-agents get their own JSONL session files with `session_meta.source.subagent.thread_spawn.parent_thread_id` linking back to the parent, but plan 84 only promises parent-stream orchestration visibility; out-of-band child ingestion is plan 88.
- Claude teams are separate processes. Inbox writes are still the steering mechanism, but inbox ingestion/hardening is plan 88 scope.
- The 64MB NDJSON line buffer in stamp is already in place.
- `CanonicalEvent` is intentionally flat (no nested types). Agent metadata should be optional fields, not a nested struct hierarchy.
- The current UI shell is `Sidebar -> FeedPanel -> ContextPanel` with `/actor/$id` as the session route. Sub-agent UX should land there first; dashboard/topology polish is optional follow-up, not the v1 entrypoint.
- Codex child-session steering is not currently available in the process runtime: the dispatcher writes stdin once and closes it, so plan 84 can track Codex children in v1 but should only promise steering where the runtime already supports it.
- The failure taxonomy from [[archive/plans/83-error-recoverability-taxonomy/overview]] already exists. Plan 84 should reuse that typed failure model for steer-agent validation/execution errors instead of reopening taxonomy design.

## Alternatives Considered

1. **Extend CanonicalEvent with agent fields (chosen)** -- Add `AgentID`, `ParentAgentID`, `AgentName`, `AgentType` optional fields to `CanonicalEvent`. Add new `EventType` variants (`EventAgentSpawn`, `EventAgentProgress`, `EventAgentComplete`). Parsers emit these alongside existing events. Minimal new plumbing.

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
3. [[plans/84-subagent-tracking/phase-03-codex-subagent-parsing]] -- Parse Codex `session_meta.source.subagent` and `function_call` items into agent events
4. [[plans/84-subagent-tracking/phase-04-claude-team-parsing]] -- Parse Claude team tool calls (TeamCreate, SendMessage) and inbox files
5. [[plans/84-subagent-tracking/phase-05-event-pipeline-agent-support]] -- Wire agent events through event storage, snapshot builder, and WebSocket broadcast
6. [[plans/84-subagent-tracking/phase-06-snapshot-agent-tree]] -- Add agent tree to session snapshot (parent/child relationships, identity, status)
7. [[plans/84-subagent-tracking/phase-07-ui-agent-tree]] -- Add agent summary/tree to the existing actor context panel and shared channel state
8. [[plans/84-subagent-tracking/phase-08-ui-agent-feed]] -- Filter the existing actor feed by selected agent and surface agent-aware rows
9. [[plans/84-subagent-tracking/phase-09-ui-agent-steering]] -- Reuse the existing actor feed input for steerable sub-agents through typed control actions
10. [[plans/84-subagent-tracking/phase-10-smoke-tests]] -- End-to-end tests with fixture NDJSON covering all agent patterns

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
cd ui && pnpm tsc --noEmit && pnpm test
```
