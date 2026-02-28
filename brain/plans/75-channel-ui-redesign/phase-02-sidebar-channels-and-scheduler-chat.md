Back to [[plans/75-channel-ui-redesign/overview]]

# Phase 2: Sidebar Channels and Scheduler Chat

## Goal

Populate the sidebar with the scheduler as primary channel, active cooks as secondary channels grouped by status, and navigation links. Clicking a channel switches the main feed panel. The scheduler channel is the default view and doubles as the "new order" entry point — user types a prompt and the scheduler creates orders from it.

## Skills

Invoke `react-best-practices`, `ts-best-practices`, `interaction-design` before starting.

## Routing

- **Provider:** `claude` | **Model:** `claude-opus-4-6`
- UX decisions on channel grouping and interaction model

## Changes

### Modify
- `ui/src/components/Sidebar.tsx` — full implementation:
  - Header: logo mark + "NOODLE" + running status dot
  - Nav: Dashboard, Live Feed (active), Tree, Reviews — with SVG icons, uppercase, accent left-border on active
  - Scheduler section: manager item with model name badge
  - Orders section: collapsible order list with stage sub-items (■/□/✓ icons)
  - Footer: session cost
- `ui/src/components/FeedPanel.tsx` — render scheduler feed when scheduler channel selected
- `ui/src/client/hooks.ts` — add `useActiveChannel` hook, derive channel list from snapshot

### Create
- `ui/src/components/SchedulerFeed.tsx` — feed view for scheduler: shows steer history from feed_events, input area at bottom for sending steer commands (which become new orders)

## Data Structures

- `Channel` — `{ id: ChannelId, name: string, status: string, model: string, host: string }`
- Derive channel list from `snapshot.sessions` + a synthetic scheduler entry
- `OrderTreeItem` — `{ orderId: string, title: string, stages: Stage[], expanded: boolean }`

## Tests

- `Sidebar.test.tsx` — renders nav links, renders scheduler item, renders agent channels from snapshot, highlights active channel
- `SchedulerFeed.test.tsx` — renders feed events, steer input sends control command

## Verification

### Static
- `pnpm tsc --noEmit` and `pnpm test` pass

### Runtime
- Sidebar renders scheduler + active agents from live snapshot data
- Clicking scheduler shows scheduler feed
- Clicking an agent highlights it (feed content comes in phase 3)
- Orders section shows collapsible stage lists
- Nav links navigate between pages
