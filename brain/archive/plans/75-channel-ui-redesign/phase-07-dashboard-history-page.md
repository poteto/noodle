Back to [[plans/75-channel-ui-redesign/overview]]

# Phase 7: Dashboard History Page

## Goal

A dedicated route (`/dashboard`) showing session history: summary stats bar, sortable session grid with columns for ID, title, status, host, model, duration, cost. Matches the prototype's `history.html` layout.

## Skills

Invoke `react-best-practices`, `ts-best-practices`, `interaction-design` before starting.

## Routing

- **Provider:** `codex` | **Model:** `gpt-5.4`
- Tabular data rendering is mechanical

## Changes

### Create
- `ui/src/routes/dashboard.tsx` — new TanStack Router route for `/dashboard`
- `ui/src/components/Dashboard.tsx` — two-column layout (sidebar + main):
  - Stats bar: total sessions, active, merged, total cost
  - Session grid: sortable table with columns: ID, TITLE, STATUS, HOST, MODEL, DURATION, COST
  - Status badges: RUNNING (yellow), NEEDS APPROVAL (yellow outline), MERGED (green), FAILED (red)
  - "NEW ORDER" button in header (links to `/` scheduler chat)

### Modify
- `ui/src/routes/__root.tsx` — register dashboard route
- Sidebar nav "Dashboard" link points to `/dashboard`

## Data Structures

- Derive grid data from `snapshot.sessions` (active) + `snapshot.recent` (completed)
- `SessionRow` — flattened session for table display: `{ id, title, status, host, model, duration, cost }`

## Tests

- `Dashboard.test.tsx` — renders stats bar with correct totals, renders session grid rows from snapshot data, status badges show correct variant

## Verification

### Static
- `pnpm tsc --noEmit` and `pnpm test` pass

### Runtime
- Navigate to `/dashboard` → see stats bar and session grid
- Grid populates from live snapshot data
- Status badges render with correct colors
- Click "NEW ORDER" → navigates to scheduler chat
