---
id: 75
created: 2026-02-27
status: ready
---

# Channel UI Redesign

## Context

The current web UI is a kanban board (Queued → Cooking → Review → Done). This makes sense for queue management but fails as a primary interaction model — users spend most of their time talking to agents, not dragging cards between columns.

The prototype in `prototype/` demonstrates a Slack/Discord-style channel layout: the scheduler is the user's main conversation partner (like a DM), each active cook is a channel in the sidebar, and the user switches between conversations. This is a fundamentally better fit for an agent orchestrator where the primary activity is reading agent output and steering work.

This plan replaces the existing React kanban UI with the channel layout, adds a D3-powered tree view, and makes backend changes so orders advance without waiting for process exit.

## Scope

**In scope:**
- Replace kanban board with three-column channel layout (sidebar | feed | context panel)
- Scheduler as primary conversation channel, cooks as secondary channels
- Agent conversation view with event stream rendering
- Review flow integrated into channel view (not a separate panel)
- D3 tree visualization of order execution graph
- Session history dashboard
- Backend: EventResult canonical event → order completion (no process exit dependency)
- Backend: auto-advance for mechanical stages, explicit dismiss for execute stages
- Brutalist dark theme matching prototype aesthetic

**Out of scope:**
- Deploy/create order page (user talks to scheduler instead)
- Error page (errors shown inline in conversations)
- Drag-and-drop reordering (control commands replace this)
- WebSocket migration (keep SSE, it works)
- Mobile responsive layout

## Constraints

- **Keep existing infrastructure**: Vite 7, TanStack Router, TanStack Query, SSE, stdlib HTTP server
- **Keep snapshot-based data flow**: Backend builds snapshots from filesystem, SSE broadcasts to UI
- **Keep control command model**: POST /api/control for all user actions
- **Tree view**: React + D3 (not raw SVG like the prototype)
- **Theme**: Brutalist dark — near-black (#030303), Inter headings, JetBrains Mono body, zero border-radius, yellow accent (#FBDB24). Reference: `prototype/style.css`
- **TypeScript types**: Generated via tygo from Go structs. Any backend type changes need tygo regeneration.

### Alternatives considered

**Channel layout vs. improved kanban:** Kanban optimizes for queue visibility but the primary user action is conversation, not queue management. Channel layout puts conversations first and relegates queue state to the sidebar. Chose channels.

**D3 vs. raw SVG for tree:** The prototype uses hand-positioned SVG which doesn't scale. D3's force-directed or tree layouts handle dynamic node counts. React-D3 integration via `useRef` + D3 for rendering. Chose D3.

**Replace SSE with WebSocket:** SSE is simpler, unidirectional, and sufficient — control commands go via POST. No benefit to WebSocket for this use case. Kept SSE.

## Applicable Skills

Every frontend phase must invoke these skills:
- `react-best-practices` — component structure, hooks, effects, performance
- `ts-best-practices` — type safety, discriminated unions, exhaustiveness checks
- `interaction-design` — micro-interactions, motion design, transitions, feedback patterns

Additionally:
- `go-best-practices` — backend phases (8, 9)
- `frontend-design` — brutalist theme execution (phase 10)

## Styling

Use **Tailwind CSS** (already installed as Tailwind v4.2 via Vite plugin) for all component styling. Define the brutalist design tokens as Tailwind theme extensions in `app.css`. No inline style objects or separate CSS modules — Tailwind utility classes only.

## Phases

1. [[plans/75-channel-ui-redesign/phase-01-delete-kanban-scaffold-channel-shell]]
2. [[plans/75-channel-ui-redesign/phase-02-sidebar-channels-and-scheduler-chat]]
3. [[plans/75-channel-ui-redesign/phase-03-agent-conversation-view]]
4. [[plans/75-channel-ui-redesign/phase-04-context-panel-and-session-metrics]]
5. [[plans/75-channel-ui-redesign/phase-05-review-flow-inline]]
6. [[plans/75-channel-ui-redesign/phase-06-d3-tree-view]]
7. [[plans/75-channel-ui-redesign/phase-07-dashboard-history-page]]
8. [[plans/75-channel-ui-redesign/phase-08-backend-eventresult-completion-signal]]
9. [[plans/75-channel-ui-redesign/phase-09-backend-auto-advance-and-dismiss]]
10. [[plans/75-channel-ui-redesign/phase-10-brutalist-theme-and-visual-polish]]
11. [[plans/75-channel-ui-redesign/phase-11-verification-and-smoke-tests]]

## Verification

```bash
# TypeScript
cd ui && pnpm tsc --noEmit && pnpm lint

# Unit tests
cd ui && pnpm test

# Go
go test ./... && go vet ./...

# Architecture lint
sh scripts/lint-arch.sh

# Build
pnpm build

# E2E smoke tests (requires running noodle instance)
pnpm test:smoke

# Visual: load localhost:5173, verify channel layout, switch between agents, check tree view
```
