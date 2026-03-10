Back to [[plans/84-subagent-tracking/overview]]

# Phase 7: Actor Route Agent Tree

## Goal

Expose sub-agent structure inside the existing `/actor/$id` experience so operators can see child agents, status, and current activity without leaving the session they are already steering.

## Changes

**Shared channel state** -- Extend the existing channel-selection model so actor views can optionally carry a selected sub-agent alongside the parent session:
- `ChannelId = { type: "agent", sessionId, agentId? }`
- This keeps `FeedPanel` and `ContextPanel` synchronized without introducing a dedicated per-agent route in v1
- If deep-linking becomes important later, the same shape can move into route search params without redesigning the UI contract
- Target the existing shared-state seams: `ui/src/client/types.ts`, `ui/src/client/hooks.tsx`, and `ui/src/routes/__root.tsx`

**`ui/src/components/ContextPanel.tsx`** -- In the existing agent context panel:
- Add an "Agents" summary section for the current session: total, running, completed, errored
- Render a collapsible agent tree from `session.agents`
- Each row shows name, type, status, current action, and a steerable marker when relevant
- Clicking a row updates the shared active channel's `agentId`
- Include a clear-selection affordance to return to the parent session feed

**`ui/src/components/AgentTree.tsx`** -- New component:
- Takes `agents: AgentNode[]` prop
- Renders as indented list (depth via ParentID)
- Status indicators: green dot (running), gray (completed), red (errored)
- Click handler: `onSelectAgent(agentId: string | null)`

**`ui/src/client/types.ts`** -- Add `AgentNode` type:
- `{id, parent_id, name, type, status, current_action, spawned_at, completed_at, steerable}`
- Extend `ChannelId` so actor views can carry optional `agentId`

**`ui/src/routes/__root.tsx` + `ui/src/components/FeedPanel.tsx`** -- Keep the existing route structure (`/actor/$id`) and make the shared channel state the source of truth for the selected sub-agent.

## Data Structures

- `AgentNode` TypeScript type (8 fields + steerable boolean)
- `ChannelId` actor variant gains optional `agentId`

## Routing

Provider: `claude`, Model: `claude-opus-4-6` -- UI layout and interaction design. Use `react-best-practices` and `interaction-design`.

## Verification

### Static
- `cd ui && pnpm tsc --noEmit && pnpm test`
- AgentTree renders empty gracefully (no agents)
- AgentTree renders 3-level hierarchy
- Context panel selection updates shared channel state
- Feed panel reacts when selected `agentId` changes from the shared channel source of truth

### Runtime
- Visual: `/actor/$id` shows agent summary + tree in the context panel
- Click agent node, verify the selected sub-agent state changes and can be cleared
- Change the selected sub-agent via the shared channel state and verify both `ContextPanel` and `AgentFeed` update together
- Verify status dots and current-action text reflect live agent state
