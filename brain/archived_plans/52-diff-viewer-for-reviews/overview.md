---
id: 52
created: 2026-02-26
status: done
---

# Diff Viewer for Reviews

## Context

When an agent completes work and it enters the Review column, the user currently sees a card with the task title, prompt snippet, worktree label, and merge/reject/changes buttons — but no way to see *what actually changed*. The user must leave the UI to inspect the diff, which breaks the review flow. This feature adds a diff viewer to the review side panel so the user can read the code changes and act on them in one place.

## Scope

**In scope:**
- Go endpoint returning unified diff + stat for a pending review item
- Resizable side panel (shared by chat and review panels)
- DiffViewer component with stat summary header and diff-level syntax highlighting
- ReviewPanel that opens from clicking a ReviewCard in the kanban board
- Panel defaults wider when showing diffs

**Out of scope:**
- Per-file language-aware syntax highlighting (diff-level coloring via `source.diff` is sufficient for v1)
- Inline comments or annotations on the diff
- Side-by-side diff view (unified only)
- Diff for done/completed items (only pending review)

## Constraints

- `PendingReviewItem` already carries `worktree_path` (absolute) and `worktree_name` — no new data plumbing needed
- `@wooorm/starry-night` is installed and used in `CodeHighlight.tsx` — add `source.diff` scope
- Server uses `net/http` stdlib with `http.NewServeMux` — follow existing endpoint patterns
- UI uses Tailwind CSS with poster theme — no CSS modules
- No new npm dependencies for the resizable panel — custom drag handle

### Alternatives considered

1. **Dedicated ReviewPanel with shared SidePanel base (chosen)** — ReviewPanel and ChatPanel both compose a shared SidePanel wrapper. Clean separation, each panel purpose-built for its content type.
2. **Unified SidePanel with content slots** — single panel that switches between chat and diff content. Less code but more complex state management and harder to customize layout per content type.
3. **Server-rendered HTML diff** — server returns pre-highlighted HTML. Simpler client but breaks with theme changes, harder to style, and adds server complexity.

## Applicable Skills

- `react-best-practices` — for React component design, hooks, data fetching
- `go-best-practices` — for the Go diff endpoint
- `ts-best-practices` — for TypeScript type safety
- `interaction-design` — for the resize handle interaction
- `frontend-design` — for the diff viewer styling

## Phases

- [[archived_plans/52-diff-viewer-for-reviews/phase-01-worktree-diff-function]]
- [[archived_plans/52-diff-viewer-for-reviews/phase-02-diff-api-endpoint]]
- [[archived_plans/52-diff-viewer-for-reviews/phase-03-client-diff-api-hook]]
- [[archived_plans/52-diff-viewer-for-reviews/phase-04-resizable-sidepanel-component]]
- [[archived_plans/52-diff-viewer-for-reviews/phase-05-refactor-chatpanel-onto-sidepanel]]
- [[archived_plans/52-diff-viewer-for-reviews/phase-06-diffviewer-component]]
- [[archived_plans/52-diff-viewer-for-reviews/phase-07-reviewpanel-component]]
- [[archived_plans/52-diff-viewer-for-reviews/phase-08-board-integration]]

## Verification

- `go test ./... && go vet ./...`
- `sh scripts/lint-arch.sh`
- `cd ui && npx tsc --noEmit`
- Visual: open web UI, park a review item, click ReviewCard, verify diff renders in side panel with stat header. Resize the panel. Merge/reject/request-changes still work from within the panel.
