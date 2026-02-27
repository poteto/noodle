Back to [[archive/plans/52-diff-viewer-for-reviews/overview]]

# Phase 4: Resizable SidePanel component

## Goal

Create a shared SidePanel component with a drag handle for resizing. This becomes the layout primitive that both ChatPanel and ReviewPanel compose.

## Changes

**`ui/src/components/SidePanel.tsx`** (new file)
- Props: `defaultWidth: number`, `minWidth?: number` (default ~400), `maxWidth?: number` (default ~1200), `onClose: () => void`, `children: ReactNode`
- Layout: fixed overlay with semi-transparent backdrop (same as current ChatPanel), panel slides from right
- Drag handle: a narrow vertical strip (4-6px) on the left edge of the panel. On mousedown, track mousemove to adjust width. On mouseup, stop tracking. Use `cursor: col-resize` on hover/drag.
- Width stored in component state. No localStorage persistence for now (keep it simple).
- Escape key and backdrop click to close (extract from ChatPanel)
- Animations: reuse `animate-fade-in` and `animate-slide-right` from current ChatPanel

Design the drag handle to be subtle — a thin line that highlights on hover, not a chunky grip. The poster theme uses 2-3px borders, so a 2px handle line with hover highlight fits.

## Data structures

- `SidePanelProps { defaultWidth: number; minWidth?: number; maxWidth?: number; onClose: () => void; children: ReactNode }`

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | UI component with clear spec |

Invoke `interaction-design` skill for the resize handle interaction pattern.

## Verification

### Static
- `cd ui && npx tsc --noEmit`

### Runtime
- Render SidePanel standalone, verify:
  - Panel appears with correct default width
  - Drag handle resizes panel smoothly (no jank, no layout thrashing)
  - Escape and backdrop click close the panel
  - Min/max width constraints respected
  - Panel doesn't resize when clicking (only on drag)
