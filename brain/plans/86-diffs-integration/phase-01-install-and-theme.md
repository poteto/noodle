Back to [[plans/86-diffs-integration/overview]]

# Phase 1 — Install `@pierre/diffs` and Configure Theme

## Goal

Install the library and create a theme configuration that matches Noodle's dark UI, so all subsequent phases can import a pre-configured component wrapper.

## Changes

**`ui/package.json`** — add `@pierre/diffs` dependency.

**`ui/src/components/DiffTheme.tsx`** (new) — thin wrapper that provides Noodle-themed `PatchDiff` and `MultiFileDiff` components. Maps Noodle CSS variables to `@pierre/diffs` CSS variable overrides. Selects a dark Shiki theme (e.g., `github-dark` or register a custom one). Exports a `DiffOptions` object with shared defaults: `diffStyle: 'unified'`, `showLineNumbers: true`, `overflow: 'scroll'`, `indicators: 'bars'`.

## Data Structures

- `DiffOptions` — shared config object type with theme, style, and display defaults

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.4` | Mechanical setup — install dep, create wrapper component |

## Verification

### Static
- `pnpm build` passes — library resolves, types are correct
- `pnpm check` passes

### Runtime
- Render a hardcoded `PatchDiff` with a sample unified diff string in a throwaway route or Storybook-style test page
- Confirm syntax highlighting renders, colors match Noodle palette
