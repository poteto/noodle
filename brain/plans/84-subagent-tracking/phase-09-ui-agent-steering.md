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

**`server/agent_steering.go`** -- New file, new API endpoint:

`POST /api/sessions/{id}/agents/{agent_id}/message`
- Request: `{text: string}`
- Response: `{ok: bool, error?: string}`

Dispatch logic:
1. Look up agent in session snapshot by agent_id
2. If not steerable, return 400
3. **Claude team member**: Read agent's team config from `~/.claude/teams/{team}/config.json` to find the inbox path. Append message to inbox JSON array with `{from: "user", text, summary, timestamp, read: false}`. Use atomic write (write to temp file, rename) to avoid corrupting the inbox if multiple writers race.
4. **Codex sub-agent**: The live steering mechanism is `send_input` — a function_call the parent agent makes to deliver input to a running sub-agent. Noodle triggers this by invoking `codex exec resume` on the parent session with a prompt that instructs it to send_input to the target child thread ID. Note: Codex stdin is closed after initial prompt write, so direct stdin is not available — `codex exec resume` is the only path.

**`server/server.go`** -- Register the new endpoint in the existing `mux` route table (there is no separate routes.go).

**`ui/src/components/AgentChat.tsx`** -- The message input (from phase 8):
- Enabled when `agent.steerable === true` and `agent.status === "running"`
- Disabled with tooltip "This agent cannot receive messages" for non-steerable
- Disabled with tooltip "Agent has completed" for completed steerable agents
- On send: POST to `/api/sessions/{id}/agents/{agent_id}/message`
- Optimistically add sent message to the feed as a user message
- On error: show inline error, remove optimistic message

## Data Structures

- `AgentSteerRequest`: `{text string}` (Go)
- `AgentSteerResponse`: `{ok bool, error string}` (Go)
- Claude inbox entry: `{from, text, summary string, timestamp time.Time, read bool}`

## Routing

Provider: `claude`, Model: `claude-opus-4-6` -- requires judgment about error handling, race conditions with inbox file writes, and UX for steering failures.

## Verification

### Static
- `go test ./server/...`
- `cd ui && pnpm tsc --noEmit && pnpm test`
- API returns 400 for non-steerable agents
- API returns 200 and writes to inbox for Claude team member
- UI disables input for non-steerable agents

### Runtime
- Send message to Claude team member via UI, verify inbox file updated
- Send message to completed agent, verify disabled state
- Visual: message input enabled/disabled based on agent type
- Integration: send message, verify it shows in agent chat feed
