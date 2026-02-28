Back to [[plans/75-channel-ui-redesign/overview]]

# Phase 6: D3 Tree View

## Goal

A dedicated route (`/tree`) renders an interactive execution graph using React + D3. Nodes represent the scheduler and active cooks. Edges show order flow: scheduler → cook assignments → quality → merge. The graph updates live as the snapshot changes.

## Skills

Invoke `react-best-practices`, `ts-best-practices`, `interaction-design` before starting.

## Routing

- **Provider:** `claude` | **Model:** `claude-opus-4-6`
- D3 integration with React requires architectural judgment

## Changes

### Install
- `d3` + `@types/d3` — D3 library and TypeScript types

### Create
- `ui/src/routes/tree.tsx` — new TanStack Router route for `/tree`
- `ui/src/components/TreeView.tsx` — main tree visualization component:
  - Uses `useRef` for SVG container, D3 for layout and rendering
  - D3 tree/hierarchy layout (not force-directed — the execution graph is a DAG)
  - Nodes: scheduler at root, cooks as children, stages as leaf detail
  - Edges: solid for active connections, dashed for pending
  - Node cards show: agent name, status badge, current action, cost
  - Animated data packet dots on active edges
  - Dot-grid background pattern
  - Zoom/pan via D3 zoom behavior
- `ui/src/components/TreeNode.tsx` — individual node card rendered as SVG foreignObject (allows HTML/CSS inside SVG nodes)

### Modify
- `ui/src/routes/__root.tsx` — register tree route
- Sidebar nav "Tree" link points to `/tree`

## Data Structures

- `TreeGraphData` — transform snapshot orders + sessions into D3 hierarchy: `{ name, children[], session?, status }`
- D3 `d3.tree()` layout computes x/y positions from hierarchy

## Tests

- `TreeView.test.tsx` — renders SVG container, transforms snapshot data into D3 hierarchy, renders expected number of nodes
- Test the data transform function in isolation: snapshot with 2 orders → correct tree structure

## Verification

### Static
- `pnpm tsc --noEmit` and `pnpm test` pass, D3 types resolve

### Runtime
- Navigate to `/tree` → see execution graph
- Nodes positioned in readable tree layout
- Active edges show animated data packets
- Graph updates live as agents start/complete
- Zoom and pan work smoothly
- Breadcrumb shows "Sessions / {agent} / Tree"
