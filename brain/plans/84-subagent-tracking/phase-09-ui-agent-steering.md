Back to [[plans/84-subagent-tracking/overview]]

# Phase 9: UI Agent Steering

## Goal

Let users send messages to steerable agents (Claude team members, Codex collab agents). Non-steerable agents (Claude sub-agents via Agent tool) show chat read-only.

## Steerable vs Non-Steerable

| Agent Type | Steerable | Mechanism |
|---|---|---|
| Claude team member (TeamCreate + Agent with team_name) | Yes | Write to inbox file |
| Codex sub-agent (collab spawn_agent) | Yes | `codex exec resume` on parent session with prompt to `send_input` to child thread ID |
| Claude sub-agent (Agent tool, no team) | No | Runs to completion, no input channel |
| Claude sub-agent (background, no team) | No | Same as above |

The `AgentNode.Steerable` field is set by the parser during parse (Phase 2/3/4) and carried through the full pipeline to the snapshot. The snapshot builder does not derive it — it reads it from EventAgentSpawned payloads.

## Changes

**Control-plane architecture:** Reuse existing `/api/control` and loop control command processing. Do not introduce a second per-agent endpoint.

Server/loop changes:
1. Extend control request/command schema with optional `agent_id`
2. Add control action `steer-agent`
3. Handle provider-specific steering in the existing control queue path (same validation, retries, and auditability as other control actions)
4. Validate delivery preconditions at execution time (agent still exists, steerable, running, and parent session active) to avoid stale-race sends
5. v1 response contract is terminal and single-path: reuse existing control ack (`status: ok|error`) with typed failure payload (`code`, `message`, `retryable`) for validation/execution failures

**`ui/src/components/AgentChat.tsx`** -- The message input (from phase 8):
- Enabled when `agent.steerable === true` and `agent.status === "running"`
- Disabled with tooltip "This agent cannot receive messages" for non-steerable
- Disabled with tooltip "Agent has completed" for completed steerable agents
- On send: POST to `/api/control` with `{action: "steer-agent", target: sessionId, agent_id, prompt}`
- Optimistically add sent message to the feed as a user message
- On error: show inline error, remove optimistic message

## Data Structures

- `ControlCommand` extension: optional `agent_id string`
- New control action: `steer-agent`
- Claude inbox entry: `{from, text, summary string, timestamp time.Time, read bool}`
- Reuse existing control ack failure shape for steer errors (`code`, `message`, `retryable`) rather than introducing a parallel response contract

## Routing

Provider: `claude`, Model: `claude-opus-4-6` -- requires judgment about error handling, race conditions with inbox file writes, and UX for steering failures.

## Verification

### Static
- `go test ./server/...`
- `go test ./loop/...`
- `cd ui && pnpm tsc --noEmit && pnpm test`
- Control action returns error for non-steerable agents
- Control action succeeds for steerable agents
- UI disables input for non-steerable agents

### Runtime
- Send message to Claude team member via UI, verify inbox file updated
- Send message to completed agent, verify disabled state
- Visual: message input enabled/disabled based on agent type
- Integration: send message, verify it shows in agent chat feed
- Steering race test: agent completes between submit and execution -> deterministic typed failure, no write side effects
- Codex steer integration test: after steer request, observe target-agent `send_input` progress in session events
- Claude inbox write test uses atomic write (temp file + rename) to avoid partial JSON on concurrent reads
