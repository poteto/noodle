Back to [[plans/75-channel-ui-redesign/overview]]

# Phase 1: Delete Kanban, Scaffold Channel Shell

## Goal

Remove the kanban board and replace with the three-column channel layout shell. The app renders an empty but correctly structured layout: sidebar (260px) | main feed (flex) | context panel (300px).

## Skills

Invoke `react-best-practices`, `ts-best-practices`, `interaction-design` before starting.

## Routing

- **Provider:** `claude` | **Model:** `claude-opus-4-6`
- Architectural decision on layout structure, needs judgment

## Changes

### Delete
- `ui/src/components/Board.tsx` — the entire kanban board
- `ui/src/components/OrderCard.tsx`, `AgentCard.tsx`, `ReviewCard.tsx`, `DoneCard.tsx` — kanban column cards
- `ui/src/components/TaskEditor.tsx` — deploy modal (replaced by scheduler chat)
- Kanban-specific CSS from `app.css` (poster theme, column styles, card shadows)

### Create
- `ui/src/components/AppLayout.tsx` — three-column CSS grid shell
- `ui/src/components/Sidebar.tsx` — sidebar skeleton (header, nav, channel list area, footer)
- `ui/src/components/FeedPanel.tsx` — main content area skeleton
- `ui/src/components/ContextPanel.tsx` — right panel skeleton

### Modify
- `ui/src/routes/index.tsx` — render `AppLayout` instead of `Board`
- `ui/src/app.css` — strip poster theme, add Tailwind theme extensions for brutalist palette (CSS variables as `@theme` overrides)

## Data Structures

- `ChannelId` — discriminated union: `{ type: "scheduler" }` | `{ type: "agent", sessionId: string }`
- `useActiveChannel()` — hook managing selected channel state (URL param or React state)

## Tests

- Delete old component tests: `Board.test.tsx`, `OrderCard.test.tsx`, `AgentCard.test.tsx`, `ReviewActions.test.tsx`, `TaskEditor.test.tsx`
- Delete `types.test.ts` (tests `deriveKanbanColumns` which no longer exists)
- Add `AppLayout.test.tsx` — renders three-column grid, mounts children
- Add `useActiveChannel.test.ts` — default channel is scheduler, can switch

## Verification

### Static
- `pnpm tsc --noEmit` passes
- `pnpm test` passes (old tests removed, new tests added)
- No references to deleted Board/Card components remain

### Runtime
- App loads at localhost:5173 with three visible columns
- Sidebar shows NOODLE header and nav placeholder
- Main area and context panel render empty shells
- SSE connection still establishes (check console)
