---
id: 86
created: 2026-03-01
status: active
---

# Plan 86: Inline Diff Viewer

Back to [[plans/index]]

## Context

Code changes are the primary output of Noodle sessions, but the web UI currently shows them as one-line summaries ("Edit src/foo.go") or collapsed groups ("Edit (x5)"). Users can't review what changed without leaving the UI. The `@pierre/diffs` library provides production-grade diff rendering with React components, syntax highlighting via Shiki, and Shadow DOM style isolation.

## Scope

**In scope:**
- Install `@pierre/diffs` and configure theming to match Noodle's dark UI
- Enrich the event pipeline to carry diff metadata (file path + line stats) from both Claude and Codex adapters, through `CanonicalEvent` → `eventFromCanonical()` → `EventLine`
- Add a REST endpoint for lazy-loading full diff content on demand
- Build a collapsible `InlineDiff` component for the activity feed — collapsed by default, showing filename + `+N/-M` stat badge, click to expand fetches and renders the full diff
- Update `ToolGroup` to render inline diffs for grouped Edit events
- Build a dedicated "Changes" tab scoped to agent sessions — chronological list of code-change events, diffs expanded by default
- Replace the existing `DiffViewer` component (review panel) with `@pierre/diffs`-based rendering

**Out of scope:**
- Split-view diffs (unified only — can add later via `diffStyle` prop)
- Interactive features (line annotations, hunk accept/reject) — future work
- Worker pool for background highlighting — premature until perf data says otherwise
- SSR preloading — the UI is fully client-rendered

## Constraints

- **Dark-only theme.** Configure `@pierre/diffs` with a dark Shiki theme and CSS variable overrides to match the existing palette.
- **Shadow DOM isolation.** `@pierre/diffs` uses Shadow DOM — Tailwind/CSS won't leak in. Custom styling via CSS variable overrides or `unsafeCSS`.
- **Stable event identity.** `EventLine` carries an `Index int` field assigned from the NDJSON line number during read. This index is immutable (NDJSON is append-only) and used as the key for lazy diff fetches. The client-side WS dedup bug (drops events with `at <= lastAt`) is a pre-existing issue — the stable index ensures correct fetches regardless.
- **Lazy-load diff content.** `EventLine` carries only file path + `+N/-M` stats (small). Full diff content is fetched on-demand: individual events via `GET /api/sessions/{id}/events/{index}/diff`, batch via `GET /api/sessions/{id}/diffs`. This prevents multi-MB blobs from flowing through WS fanout and snapshot paths.
- **Compute diff metadata at ingestion.** `eventFromCanonical()` computes `{file_path, additions, deletions}` once and stores it as `diff_meta` in the event payload. `formatAction` reads the pre-computed object — never re-parses raw `tool_input`. This keeps the presentation path cheap.
- **Both adapters.** Claude uses `Edit` (old_string/new_string) and `Write` (content). Codex uses `apply_patch` (patch string). Match explicit tool names only: `edit`, `write`, `apply_patch`, `edit_file`, `write_file`. No heuristic matching.
- **Tool grouping.** Consecutive Edit events are grouped by `ToolGroup` into "Edit (x5)". The inline diff must work within both ungrouped `MessageRow` and grouped `ToolGroup` rendering.

## Data Path

The full transformation chain for diff data:

```
parse/claude.go (or codex_items.go)
  → CanonicalEvent { Message, ToolInput }
    → dispatcher/session_helpers.go:eventFromCanonical()
      → computes diff_meta { file_path, additions, deletions } from ToolInput
      → event.Event { Payload: { tool, summary, tool_input, diff_meta } }
        → event/writer.go (persists to NDJSON)
          → internal/snapshot/session_events.go:formatAction()
            → reads pre-computed diff_meta (no re-parsing of tool_input)
            → EventLine { Index, Label, Body, DiffMeta }

Single diff: GET /api/sessions/{id}/events/{index}/diff
  → seeks to NDJSON line {index} → extracts tool_input → returns DiffContent

Batch diffs: GET /api/sessions/{id}/diffs
  → single pass over NDJSON → returns all DiffContent[] (for Changes tab)
```

## Alternatives Considered

1. **Inline full diff content on EventLine** — simpler frontend (no lazy fetch), but pushes multi-MB blobs through WS fanout (drop-on-backpressure), inflates snapshot broadcasts, and stalls backfill for slow clients. Rejected.
2. **Keep `@wooorm/starry-night` for diffs** — handles syntax highlighting but not diff-specific rendering (hunks, line numbers, additions/deletions, expand unchanged). Would require building all diff UI from scratch. Rejected.
3. **Use `@pierre/diffs` with lazy loading (chosen)** — purpose-built diff components, stats-only on EventLine, full content fetched on demand. Best fit for performance and UX.

## Applicable Skills

- `react-best-practices` — component patterns, hooks, performance
- `ts-best-practices` — type safety for the enriched event types
- `frontend-design` — visual polish, interaction design
- `interaction-design` — collapse/expand microinteraction
- `go-best-practices` — backend parser and API changes

## Phases

1. [[plans/86-diffs-integration/phase-01-install-and-theme]]
2. [[plans/86-diffs-integration/phase-02-enrich-event-data]]
3. [[plans/86-diffs-integration/phase-03-diff-content-endpoint]]
4. [[plans/86-diffs-integration/phase-04-inline-diff-component]]
5. [[plans/86-diffs-integration/phase-05-changes-tab]]
6. [[plans/86-diffs-integration/phase-06-replace-diff-viewer]]

## Verification

- `pnpm build` — no TypeScript errors, bundle builds cleanly
- `pnpm check` — full lint + type + format suite
- `go test ./...` — backend parser tests pass
- `go vet ./...` — no vet warnings
- Visual: launch the app, open a session with Edit/Write events, verify collapsed diff badges render in feed and in ToolGroups, click to expand shows syntax-highlighted diff via lazy fetch, Changes tab shows chronological diffs
