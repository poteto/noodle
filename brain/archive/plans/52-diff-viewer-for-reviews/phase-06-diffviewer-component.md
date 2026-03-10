Back to [[archive/plans/52-diff-viewer-for-reviews/overview]]

# Phase 6: DiffViewer component

## Goal

Create the DiffViewer component that renders a stat summary header and the full unified diff with syntax highlighting. This is the core visual component of the feature.

## Changes

**`ui/src/components/CodeHighlight.tsx`**
- Add `"diff"` case to `getScopeFromLang()` → returns `"source.diff"`. The `source.diff` scope is included in starry-night's `common` grammar set.

**`ui/src/components/DiffViewer.tsx`** (new file)
- Props: `diff: string`, `stat: string`, `isLoading?: boolean`
- Layout:
  1. **Stat header**: render the stat summary in a monospace block at the top with subtle background. Each line of the stat shows a filename and +/- counts. This gives a quick overview before the full diff.
  2. **Diff body**: render the full unified diff using `HighlightedCode` with `lang="diff"`. The starry-night `source.diff` grammar handles coloring `+` lines, `-` lines, `@@` hunk headers, and file headers.
  3. **Loading state**: show a skeleton or "Loading diff..." placeholder while the API call is in flight.
  4. **Empty state**: if both diff and stat are empty strings, show "No changes" message.
  5. **Error state**: accept an optional `error` prop and render it.
- Scrollable: the diff body should scroll independently (the panel might be shorter than the diff).

Style the stat header distinctly from the diff body — lighter background, slightly smaller font, clear visual separation. The diff body should use the existing code block styling from `HighlightedCode` but full-width within the panel.

## Data structures

- `DiffViewerProps { diff: string; stat: string; isLoading?: boolean; error?: string }`

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.4` | UI component with clear spec, uses existing HighlightedCode |

Invoke `frontend-design` skill for styling the diff viewer.

## Verification

### Static
- `cd ui && npx tsc --noEmit`

### Runtime
- Render DiffViewer with sample diff text — verify stat header renders, diff body is syntax-highlighted with green/red coloring for additions/deletions
- Verify loading state shows placeholder
- Verify empty state shows "No changes"
- Verify scroll works for long diffs
- Verify the `source.diff` scope is handled by starry-night (green `+` lines, red `-` lines, styled `@@` headers)
