Back to [[archived_plans/25-tui-revamp/overview]]

# Phase 2: Scaffold — Layout Framework and Pastel Palette

## Goal

Replace the current single-pane TUI with a split layout (left rail + right pane) and update the color palette to the pastel theme. After this phase, the TUI renders the new layout skeleton with placeholder content in each tab.

## Changes

### `tui/styles.go` — Rewrite color palette

Replace current color definitions with the pastel palette. Keep semantic naming but update every hex value. Add new colors for card backgrounds, task types, and lifted surface.

Key type: `Theme` struct holding all colors as `lipgloss.Color`, referenced by all components.

### `tui/layout.go` — New file: split layout renderer

Core layout function dividing terminal into left rail and right pane. Rail at fixed width (20 chars), remainder to the active tab.

Key type: `Layout` with `RenderRail(height)` and `RenderPane(width, height)` methods.

### `tui/model.go` — Replace surface system with tab model

Replace `Surface` type and overlay model with `Tab` enum (`TabFeed`, `TabQueue`, `TabBrain`, `TabConfig`). Model keeps `activeTab Tab` instead of `surface Surface`. View() calls layout renderer.

### `tui/tab_bar.go` — New file: tab bar component

Horizontal tab bar: renders tab names, highlights active with pastel yellow underline, dims inactive. No bubbles tabs component exists — lightweight custom component.

### `tui/rail.go` — New file: left rail component

Renders: agent list (dot + name + model + duration), stats section (active count, queued, budget, cost/hr). Receives Snapshot data via methods.

### `tui/model_render.go` — Gut and replace

Remove all existing render functions. Replace View() with new layout composition: rail on left, tab bar + active tab content on right.

## Routing

- Provider: `claude`
- Model: `claude-opus-4-6`

## Verification

### Static
- `go build ./...` compiles
- `go vet ./...` passes
- `go test ./tui/...` passes (update existing tests)

### Runtime
- Launch `noodle start` — TUI renders split layout with rail + placeholder tabs
- Resize terminal — layout adapts (rail fixed, right pane fills)
- Number keys 1-4 switch tabs (tab bar highlight moves)
- Colors render correctly on dark background
