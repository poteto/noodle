Back to [[plans/86-diffs-integration/overview]]

# Phase 5 — Dedicated Changes Tab

## Goal

Add a "Changes" tab to the agent session view that shows all code-change events chronologically with diffs expanded by default. Scoped to agent sessions only — scheduler views don't get tabs.

## Changes

**`ui/src/components/AgentFeed.tsx`** (or `FeedPanel.tsx` agent branch) — add a tab bar above the feed content when in agent-session mode. Two tabs: "Feed" (existing activity feed) and "Changes" (new). Feed remains the default. Tab state is local component state, scoped to the current session — no global layout changes.

**`ui/src/components/ChangesPanel.tsx`** (new) — fetches all diffs in one request via `GET /api/sessions/{id}/diffs` (batch endpoint from phase 3). Renders them chronologically as a virtualized list of `InlineDiff` components with `defaultExpanded={true}`. Each entry shows a small timestamp + filename header above the diff. If no code changes exist, shows an empty state message. Uses `@tanstack/react-virtual` for scroll performance with many expanded diffs. The batch fetch avoids N individual requests that would cause O(N^2) IO.

**Do NOT modify `AppLayout.tsx`** — tabs are agent-session-specific, not a global layout concern.

## Data Structures

- `TabId` — `'feed' | 'changes'` discriminated union
- `ChangesPanel` props — `{ sessionId: string; events: EventLine[] }` (filtered to diff-bearing events)

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Data contract and component pattern established by phases 2-4; this is assembly |

## Verification

### Static
- `pnpm build` passes
- `pnpm check` passes

### Runtime
- Open an agent session — Feed tab is active by default
- Switch to Changes tab — see chronological list of all Edit/Write events with expanded diffs
- Empty session (no edits) — Changes tab shows empty state
- Tab state persists within the session view (no flicker on switch)
- Tab switching doesn't re-fetch data — uses same event data as the feed
- Scheduler view — no tab bar appears
- Long session with many edits — Changes panel scrolls smoothly (virtualized)
