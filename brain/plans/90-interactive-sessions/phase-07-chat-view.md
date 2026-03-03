Back to [[plans/90-interactive-sessions/overview]]

# Phase 7 — Chat View

## Goal

Render interactive sessions as a chat conversation rather than an event log. User messages and agent responses are clearly distinguished with chat-style presentation. Differentiate rendering by session kind, not event type.

## Changes

**`ui/src/components/AgentFeed.tsx`** — Detect when the session has `kind === "interactive"`. When interactive, switch to chat rendering mode:
- User messages (`label === "User"` — same events as steer, no new event type) render as distinct user bubbles (right-aligned or visually separated)
- Agent output (thinking, tool use, deltas) groups into assistant turns
- Tool calls within an assistant turn are collapsible (existing `ToolGroup` behavior)
- Streaming delta shows as the agent "typing"

Alternatively, create a new `ChatFeed.tsx` component if the divergence from `AgentFeed` is large enough to warrant separation. The route at `/actor/:id` checks session kind and renders the appropriate component.

**`ui/src/components/MessageRow.tsx`** — Add chat-specific rendering when session kind is `"interactive"` and label is `"User"`. Render as a user message bubble instead of the standard event row.

**`ui/src/noodle.css`** — Chat-specific styles:
- User message bubbles (aligned right or distinct background)
- Assistant response sections (left-aligned, grouped)
- Chat input area styling (distinct from the steer input)

The input area at the bottom of the chat view sends messages via `useSendControl({ action: "steer", target: sessionId, prompt: message })`.

Invoke `interaction-design` and `frontend-design` skills.

## Data Structures

- No new data structures — rendering changes only. Uses session `kind` to determine view mode.

## Routing

- **Provider:** `claude`
- **Model:** `claude-opus-4-6`

## Verification

### Static
- `cd ui && pnpm tsc --noEmit`
- Visual inspection

### Runtime
- Start interactive session, send messages, verify chat-style rendering
- Verify user messages appear as user bubbles
- Verify agent responses (thinking + tools + output) group as assistant turns
- Verify streaming output shows as agent "typing"
- Verify tool calls within turns are collapsible
- Verify existing order session AgentFeed is unaffected (still renders as event log)
- Verify reconnect backfills conversation history correctly (existing backfill + event replay)
