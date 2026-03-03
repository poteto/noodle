Back to [[plans/86-diffs-integration/overview]]

# Phase 2 — Enrich Event Data with Diff Metadata

## Goal

Thread tool input through the event pipeline so `EventLine` carries a stable index and diff metadata for edit events. Diff metadata is computed once at ingestion time in `eventFromCanonical()` — `formatAction` reads the pre-computed object, never re-parses raw tool input. Full diff content stays in the persisted NDJSON.

## Data Path (what changes at each hop)

1. **`parse/canonical.go`** — add `ToolInput json.RawMessage` field to `CanonicalEvent`. Only populated for edit/write/apply_patch tool actions.

2. **`parse/claude.go`** — for `tool_use` content blocks where `name` is `Edit` or `Write`, set `CanonicalEvent.ToolInput = content.Input`. Other tools continue summary-only.

3. **`parse/codex_items.go`** — for `function_call`/`custom_tool_call` where name matches explicit edit tool names (`apply_patch`, `edit`, `write`, `edit_file`, `write_file`), set `CanonicalEvent.ToolInput` from `item.Arguments` or `item.Input`. No heuristic matching — explicit names only.

4. **`dispatcher/session_helpers.go:eventFromCanonical()`** — in the `EventAction` mapping path:
   - If `canonical.ToolInput` is non-nil, include it in the payload map: `payload["tool_input"] = json.RawMessage(canonical.ToolInput)`.
   - **Compute `diff_meta` here** (not in `formatAction`): extract `file_path` from `ToolInput`, count lines in `old_string`/`new_string`/`content`/`patch` to produce `additions`/`deletions`. Store as `payload["diff_meta"] = map[string]any{...}`. This ensures metadata is computed once at ingestion and persisted in the NDJSON.

5. **`event/writer.go`** — no changes needed. It already marshals any `event.Event.Payload` verbatim.

6. **`internal/snapshot/session_events.go`** — two changes:
   - **`formatAction()`** — read the pre-computed `diff_meta` object from payload (not raw `tool_input`). No re-parsing of large strings. Also add `"apply_patch"` to the tool switch, labeling it `"Edit"`.
   - **`mapEventLines()`** — assign `Index` to each `EventLine` from its position in the NDJSON event list. This is the stable identity used for diff fetches.

7. **`internal/snapshot/types.go`** — add `Index int` and `DiffMeta *DiffMeta` fields to `EventLine`. `DiffMeta` struct: `FilePath string`, `Additions int`, `Deletions int`. The `tygo` generator will produce the corresponding TypeScript types.

## Changes

| File | Change |
|------|--------|
| `parse/canonical.go` | Add `ToolInput json.RawMessage` to `CanonicalEvent` |
| `parse/claude.go` | Set `ToolInput` for Edit/Write tool_use blocks |
| `parse/codex_items.go` | Set `ToolInput` for explicit edit tool names (no heuristic) |
| `dispatcher/session_helpers.go` | Forward `ToolInput` into payload, compute `diff_meta` at ingestion |
| `internal/snapshot/types.go` | Add `Index int`, `DiffMeta` struct and field on `EventLine` |
| `internal/snapshot/session_events.go` | Read pre-computed `diff_meta`, add `apply_patch` to tool switch, assign `Index` |

## Data Structures

- `DiffMeta` — `{ FilePath string; Additions int; Deletions int }`
- `CanonicalEvent.ToolInput` — `json.RawMessage`, raw tool input for edit/write/apply_patch actions only
- `EventLine.Index` — `int`, stable NDJSON line number for diff fetch identity

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | Touches event pipeline across 4 packages with architectural judgment needed |

## Verification

### Static
- `go test ./parse/...` — parser correctly sets `ToolInput` for Edit/Write (Claude), `apply_patch`/`edit_file`/`write_file` (Codex), leaves nil for other tools
- `go test ./dispatcher/...` — `eventFromCanonical` computes `diff_meta` from `ToolInput` and includes both `tool_input` and `diff_meta` in payload
- `go test ./internal/snapshot/...` — `formatAction` reads pre-computed `diff_meta` (not raw `tool_input`), `apply_patch` events labeled as `"Edit"`, `EventLine.Index` assigned correctly
- `go vet ./...` passes

### Runtime
- Run Noodle, trigger sessions with Edit/Write (Claude) and apply_patch (Codex) tool calls
- Hit `GET /api/sessions/{id}/events` — verify Edit events include `diff_meta` with `file_path` and correct `additions`/`deletions` counts, and `index` field is present
- Verify WS `session_event` messages carry `diff_meta` without full content
- Verify `apply_patch` events appear with label `"Edit"` and correct `diff_meta`
