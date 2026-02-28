Back to [[plans/75-channel-ui-redesign/overview]]

# Phase 3: Agent Conversation View

## Goal

When an agent channel is selected, render its event stream as a conversation timeline. Each event becomes a message row with avatar, metadata, and body. Support action events (tool calls), think events, cost events, and ticket protocol events.

## Skills

Invoke `react-best-practices`, `ts-best-practices`, `interaction-design` before starting.

## Routing

- **Provider:** `codex` | **Model:** `gpt-5.3-codex`
- Rendering EventLine data into message rows is mechanical

## Changes

### Create
- `ui/src/components/AgentFeed.tsx` — conversation view for a selected agent session:
  - Header bar: agent name, task badge, model badge, host badge, status text, action buttons (stop, done)
  - Message list: renders `EventLine[]` as chronological message rows
  - Auto-scroll to bottom on new events
  - Input area for steer commands to this specific agent

- `ui/src/components/MessageRow.tsx` — single message in the feed:
  - Avatar (initials from agent name, accent color for manager)
  - Meta line: agent name + timestamp + optional badge
  - Body: monospace text, with code blocks for tool output
  - System messages rendered compact (no avatar)

### Modify
- `ui/src/components/FeedPanel.tsx` — route between `SchedulerFeed` and `AgentFeed` based on active channel
- `ui/src/client/hooks.ts` — `useSessionEvents(sessionId)` already exists, wire it to AgentFeed

## Data Structures

- Map `EventLine.label` to message type: "Read"/"Edit"/"Bash"/"Glob" → tool action, "Think" → think, "Cost" → cost line, "Ticket" → ticket protocol
- `MessageType` — `"action" | "think" | "cost" | "ticket" | "system" | "steer"`

## Tests

- `MessageRow.test.tsx` — renders different message types (action, think, cost, system), renders avatar with correct initials
- `AgentFeed.test.tsx` — renders event stream, auto-scrolls on new events, steer input sends control command

## Verification

### Static
- `pnpm tsc --noEmit` and `pnpm test` pass, EventLine mapping is exhaustive

### Runtime
- Select an active agent → see live event stream updating via SSE
- Tool actions render with output blocks
- Cost lines render as subtle inline text
- Steer input sends control command and appears in feed
- Feed auto-scrolls to latest event
