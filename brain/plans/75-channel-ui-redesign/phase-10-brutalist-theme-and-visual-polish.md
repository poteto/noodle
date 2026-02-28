Back to [[plans/75-channel-ui-redesign/overview]]

# Phase 10: Brutalist Theme and Visual Polish

## Goal

Apply the brutalist dark theme from the prototype to all React components. Ensure visual consistency across every page: channel layout, tree view, dashboard. Add micro-interactions and polish.

## Skills

Invoke `frontend-design`, `react-best-practices`, `ts-best-practices`, `interaction-design` before starting.

## Routing

- **Provider:** `claude` | **Model:** `claude-opus-4-6`
- Visual design decisions, needs creative judgment

## Changes

### Modify
- `ui/src/app.css` — complete Tailwind theme overhaul:
  - `@theme` extensions for brutalist palette: `--color-bg-depth: #030303`, `--color-bg-surface: #0A0A0A`, `--color-accent: #FBDB24`, etc.
  - Font imports: Inter + JetBrains Mono as Tailwind font families
  - Base layer: `border-radius: 0` on everything via `@layer base`
  - Typography system: Inter for headings/labels, JetBrains Mono for body/data
  - Square status dots everywhere (no circles)
  - Remove all poster theme overrides

- All components — apply consistent styling:
  - Sidebar: dark bg, accent left-border on active nav, uppercase section labels
  - Feed: message rows with monospace body, code blocks with border
  - Context panel: metric cards with monospace values
  - Tree view: dark nodes with accent borders, dashed/solid edges
  - Dashboard: dark table with accent header row, status badges
  - Buttons: square, monospace, uppercase
  - Inputs: square textarea, monospace font, accent border on focus

### Polish
- Smooth transitions on channel switch (fade or slide)
- Auto-scroll behavior with "new messages" indicator
- Keyboard shortcuts: up/down to switch channels, `n` to focus input
- Loading skeletons matching dark theme
- SSE connection indicator in sidebar header

## Verification

### Static
- `pnpm tsc --noEmit` passes
- `pnpm build` produces clean output

### Runtime
- All pages visually match prototype aesthetic
- Zero border-radius on every element (except tables if needed)
- Inter headings, JetBrains Mono body text consistently applied
- Yellow accent (#FBDB24) used sparingly: active states, badges, buttons
- Transitions feel smooth, no jank
- Works in Chrome and Firefox
