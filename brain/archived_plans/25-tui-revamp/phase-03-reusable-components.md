Back to [[archived_plans/25-tui-revamp/overview]]

# Phase 3: Reusable Components

## Goal

Build shared UI primitives that all tabs use: bordered cards, pill buttons, section headers, progress bars, and status badges. Pure render functions — no message handling, no state.

## Changes

### `tui/components/card.go` — Bordered card

Renders bordered card with optional title, body, footer. Lipgloss rounded border in gold (#fcd34d) on lifted surface (#24243a). 1-cell padding inside borders.

Key type: `Card` struct with `Title`, `Body`, `Footer` fields and `Render(width int) string`.

### `tui/components/pill.go` — Pill buttons

Inline bordered pill buttons like lipgloss dialog examples. Takes label, icon, color. Used for merge/reject/diff actions and config controls.

Key type: `Pill` struct with `Label`, `Icon`, `Color`, `Focused bool`.

### `tui/components/section.go` — Section header

Renders `Section Title ╌╌╌╌╌╌╌╌` with dotted underline in muted color. Replaces current `sectionLine()`.

### `tui/components/badge.go` — Inline status badges

Colored inline labels for task types and status (APPROVE/FLAG/REJECT). Each badge renders in its task-type or semantic color.

### `tui/components/progress.go` — Budget progress bar wrapper

Wraps `bubbles/progress` with pastel yellow fill shifting to pastel orange above 75%.

### `tui/components/elements.go` — Utility render functions

Pure helpers: `HealthDot(status)`, `AgeLine(now, then)`, `CostLabel(usd)`, `DurationLabel(seconds)`, `StatusIcon(status)`. Consolidates scattered helpers from model_render.go.

## Routing

- Provider: `claude`
- Model: `claude-opus-4-6`

## Verification

### Static
- `go build ./...` compiles
- Each component has a test rendering at multiple widths
- `go test ./tui/...` passes

### Runtime
- Components render correctly in placeholder tabs from Phase 1
- Cards have visible borders and padding at 40-col and 120-col
- Pills render inline with focused/blurred states
- Progress bar shows correct fill and color transition
