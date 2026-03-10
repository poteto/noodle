Back to [[plans/84-subagent-tracking/overview]]

# Phase 8: Agent-Scoped Feed Filtering

## Goal

Reuse the existing actor feed instead of building a second chat surface: when a sub-agent is selected, filter the session timeline to that agent's activity while keeping it easy to return to the full parent-session feed.

## Changes

**`ui/src/client/hooks.tsx`** -- Add `useAgentEvents(sessionId, agentId)` (or equivalent selector helper):
- Starts from `useSessionEvents(sessionId)`
- Returns all events when `agentId` is unset
- Returns only rows whose `event.agent_id === agentId` when a sub-agent is selected
- Keeps parent-session rows available again immediately when selection is cleared

**`ui/src/components/AgentFeed.tsx`** -- Read the shared `agentId` from the active channel:
- When no agent is selected, preserve the current top-level session feed behavior
- When an agent is selected, render a scoped header/breadcrumb (session > agent)
- Reuse `VirtualizedFeed` and existing feed input area; do not introduce a second feed component
- Show a lightweight empty state when the selected agent has no visible events yet
- Route the selection through `ui/src/components/FeedPanel.tsx` rather than introducing a parallel agent-only page component

**`ui/src/components/MessageRow.tsx`** -- Make agent-aware rows legible inside the existing timeline:
- Show agent name/type badges when `event.agent_name` / `event.agent_type` are present
- Render agent spawn/exit rows as distinct but still feed-native rows
- Keep the row model compatible with grouped tool output and virtualization

## Data Structures

- `EventLine` gains optional `agent_id`, `agent_name`, `agent_type`
- Shared active channel carries optional `agentId` instead of introducing a separate agent-view route

## Routing

Provider: `claude`, Model: `claude-opus-4-6` -- UI interaction design inside the existing feed architecture. Use `react-best-practices` and `interaction-design`.

## Verification

### Static
- `cd ui && pnpm tsc --noEmit && pnpm test`
- useAgentEvents filters correctly
- AgentFeed switches between full-session and sub-agent-scoped modes
- MessageRow renders agent metadata without breaking existing tool-group rows

### Runtime
- Select an agent from the context tree and verify the feed scopes to that agent's events
- Clear the selection and verify the full parent-session feed returns immediately
- Real-time: new scoped events appear while the selected agent is running
