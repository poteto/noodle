Back to [[plans/86-diffs-integration/overview]]

# Phase 6 — Replace Existing DiffViewer

## Goal

Replace the existing `DiffViewer` component (used in the review panel) with `@pierre/diffs`-based rendering for visual consistency across the entire UI. Delete the old `DiffViewer` and its `@wooorm/starry-night` diff dependency if no longer used elsewhere.

## Changes

**`ui/src/components/DiffViewer.tsx`** — rewrite to use `PatchDiff` from `@pierre/diffs/react` with the `DiffTheme` config from phase 1. The existing component takes `{ diff: string; stat: string }` — `PatchDiff` accepts a patch string directly, so the interface stays the same (just drop `stat` into a header and pass `diff` to `PatchDiff`).

**`ui/src/components/CodeHighlight.tsx`** — check if `@wooorm/starry-night` is still used for non-diff syntax highlighting (markdown code blocks, etc.). If it's only used by `DiffViewer`, remove the dependency. If used elsewhere, keep it.

**`ui/package.json`** — remove `@wooorm/starry-night` if no longer referenced.

## Data Structures

No new types — reuses existing `DiffViewerProps` interface.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.4` | Mechanical replacement — swap one component implementation for another |

## Verification

### Static
- `pnpm build` passes
- `pnpm check` passes
- No remaining imports of old diff rendering code (if removed)

### Runtime
- Open a session with a pending review
- Verify the review diff panel renders correctly with `@pierre/diffs`
- Compare visual quality with the inline diffs from phase 4 — they should use the same theme
- Verify no `@wooorm/starry-night` references remain (if removed) — `grep` the codebase
