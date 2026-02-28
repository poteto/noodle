Back to [[plans/84-subagent-tracking/overview]]

# Phase 7: UI Agent Tree and Dashboard Stats

## Goal

Show sub-agent information in two places: the dashboard/tree view for the overview, and the right-panel session detail for per-agent stats. The live feed continues to focus on the top-level agent.

## Changes

**Dashboard / Tree View** -- The existing tree view or dashboard shows top-level agents (Noodle sessions). No change to what's displayed at this level -- sub-agents don't appear as top-level rows. Instead, top-level sessions that have sub-agents show a count badge (e.g., "3 agents").

**Session Detail (right panel)** -- When clicking a session in the dashboard:
- Add an "Agents" stats section showing: total sub-agents, active count, completed count, errored count
- Below the stats, render a collapsible agent tree for the session's `agents` array
- Each node: name/type badge, status dot, current action text
- Click a node to navigate into that agent's chat (phase 8)

**`ui/src/components/AgentTree.tsx`** -- New component:
- Takes `agents: AgentNode[]` prop
- Renders as indented list (depth via ParentID)
- Status indicators: green dot (running), gray (completed), red (errored)
- Click handler: `onSelectAgent(agentId: string)`

**`ui/src/components/AgentStats.tsx`** -- New component:
- Takes `agents: AgentNode[]` prop
- Shows summary: `{total} agents ({active} active, {completed} done, {errored} failed)`
- Compact single-line display

**`ui/src/client/types.ts`** -- Add `AgentNode` type:
- `{id, parent_id, name, type, status, current_action, spawned_at, completed_at, steerable}`

## Data Structures

- `AgentNode` TypeScript type (8 fields + steerable boolean)
- `selectedAgentId: string | null` in session detail state

## Routing

Provider: `claude`, Model: `claude-opus-4-6` -- UI layout and interaction design. Use `frontend-design` skill.

## Verification

### Static
- `cd ui && pnpm tsc --noEmit && pnpm test`
- AgentTree renders empty gracefully (no agents)
- AgentTree renders 3-level hierarchy
- AgentStats shows correct counts

### Runtime
- Visual: session with agents shows badge and stats
- Click agent node, verify selection callback fires
- Verify status dots reflect agent state
