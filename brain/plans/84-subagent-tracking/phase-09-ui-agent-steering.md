Back to [[plans/84-subagent-tracking/overview]]

# Phase 9: UI Agent Steering

## Goal

Let users send messages to currently steerable sub-agents from the existing actor feed input. In v1, that means Claude team members; Codex child-agent steering stays disabled until the runtime exposes a live control path.

## Steerable vs Non-Steerable

| Agent Type | Steerable | Mechanism |
|---|---|---|
| Claude team member (TeamCreate + Agent with team_name) | Yes | Write to inbox file |
| Codex sub-agent (collab spawn_agent) | No in v1 | Current process runtime closes stdin after launch; steering deferred until runtime/controller support exists |
| Claude sub-agent (Agent tool, no team) | No | Runs to completion, no input channel |
| Claude sub-agent (background, no team) | No | Same as above |

The `AgentNode.Steerable` field is set by the parser during parse (Phase 2/3/4) and carried through the full pipeline to the snapshot. The snapshot builder does not derive it — it reads it from EventAgentSpawned payloads.

## Changes

**Control-plane architecture:** Reuse existing `/api/control` and loop control command processing. Do not introduce a second per-agent endpoint.

Server/loop changes:
1. Extend control request/command schema with optional `agent_id`
2. Add control action `steer-agent`
3. Handle provider-specific steering in the existing control queue path (same validation, retries, and auditability as other control actions); v1 implementation only succeeds for Claude team members
4. Validate delivery preconditions at execution time (agent still exists, steerable, running, and parent session active) to avoid stale-race sends
5. Reuse the existing typed failure taxonomy for steering failures (`not found`, `not steerable`, `already completed`, backend dispatch failure) instead of inventing a parallel error contract

**`ui/src/components/AgentFeed.tsx`** -- Reuse the existing input area:
- When no sub-agent is selected, preserve current session-level `steer` behavior
- When a steerable sub-agent is selected and running, send `/api/control` with `{action: "steer-agent", target: sessionId, agent_id, prompt}`
- When a selected agent is non-steerable or completed, disable the input with explicit copy explaining why
- Show optimistic user rows in the scoped feed, then reconcile with either later in-band team activity or a typed control failure

V1 feedback-loop note: there is no inbox-delivery acknowledgment path in plan 84. A steer request is considered observable only when later in-band team activity appears in the parent session stream; inbox ingestion/ack semantics are deferred to [[plans/88-subagent-tracking-v2/overview]]. Until that activity arrives, the optimistic row remains visibly pending and must not be auto-upgraded to success by timeout or absence of errors.

## Data Structures

- `ControlCommand` extension: optional `agent_id string`
- New control action: `steer-agent`
- Claude inbox entry: `{from, text, summary string, timestamp time.Time, read bool}`
- Reuse existing control ack failure projection metadata rather than introducing a parallel response contract

## Routing

Provider: `claude`, Model: `claude-opus-4-6` -- requires judgment about error handling, race conditions with inbox file writes, and UX for steering failures.

## Verification

### Static
- `go test ./server/...`
- `go test ./loop/...`
- `cd ui && pnpm tsc --noEmit && pnpm test`
- Control action returns error for non-steerable agents
- Control action succeeds for steerable agents
- UI disables input for non-steerable or completed selected agents
- Typed failure metadata is preserved on steer-agent acks

### Runtime
- Select a Claude team member in `/actor/$id`, send a message, and verify the inbox file is updated
- Select a completed or non-steerable agent and verify the input stays disabled with the correct explanation
- Integration: send message to a steerable agent and verify it shows in the scoped feed
- Steering race test: agent completes between submit and execution -> deterministic typed failure, no write side effects
- Codex child agent selection test: verify the input is disabled with copy explaining that steering is not available for this provider/runtime yet
- Claude inbox write test uses atomic write (temp file + rename) to avoid partial JSON on concurrent reads
