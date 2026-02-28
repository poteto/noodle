Back to [[plans/84-subagent-tracking/overview]]

# Phase 8: UI Agent Chat and Inline Sub-Agent Messages

## Goal

Two things: (1) a new message type in the parent chat showing "agent spawned sub-agents" that links to sub-agent chats, and (2) a per-agent chat view when clicking through.

## Changes

**Inline sub-agent messages in parent chat:**

**`ui/src/components/MessageRow.tsx`** (or new `SubAgentMessageRow.tsx`) -- New message row variant for `EventAgentSpawned` events:
- Renders as a distinct card/chip in the parent's chat timeline: "[Agent] Spawned 'Feynman' (awaiter)"
- Clicking the card navigates to that sub-agent's chat view
- Shows agent name, type, and status badge inline
- When the sub-agent completes, the card updates with outcome

**Per-agent chat view:**

**`ui/src/components/AgentChat.tsx`** -- New component for viewing a single agent's messages:
- Shows the agent's events as a chat-like feed (tool calls, text, errors)
- Header: agent name, type, status, back button to parent session
- For steerable agents (`agent.steerable === true`): show message input at bottom (phase 9)
- For non-steerable agents: show messages read-only, input disabled with explanatory text
- Events filtered by `agent_id` from the session event stream

**Navigation:**
- From parent chat: click sub-agent message card -> navigate to AgentChat for that agent
- From agent tree: click agent node -> navigate to AgentChat for that agent
- AgentChat back button -> return to parent session chat
- URL structure: `/sessions/{id}/agents/{agentId}` or similar (or just component state if no deep linking needed yet)

**`ui/src/client/hooks.tsx`** -- Add `useAgentEvents(sessionId, agentId)`:
- Filters `useSessionEvents(sessionId)` by `event.agent_id === agentId`
- Returns filtered event list (EventLine already has `agent_id` from Phase 5)

## Data Structures

- Extend `EventLine` type with optional `agent_id`, `agent_name`, `agent_type` fields
- Navigation state: `{view: 'session' | 'agent', agentId?: string}`

## Routing

Provider: `claude`, Model: `claude-opus-4-6` -- UI interaction design, chat layout. Use `react-best-practices` and `interaction-design` skills.

## Verification

### Static
- `cd ui && pnpm tsc --noEmit && pnpm test`
- SubAgentMessageRow renders with agent metadata
- AgentChat renders events for a specific agent
- useAgentEvents filters correctly

### Runtime
- Visual: sub-agent spawn message appears in parent chat as clickable card
- Click card -> navigates to agent chat showing only that agent's events
- Back button returns to parent chat
- Non-steerable agent chat shows disabled input
- Real-time: new agent events appear in agent chat while agent is running
