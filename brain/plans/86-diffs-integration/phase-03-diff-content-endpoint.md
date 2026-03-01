Back to [[plans/86-diffs-integration/overview]]

# Phase 3 — Diff Content REST Endpoint

## Goal

Add two REST endpoints for serving full diff content: a single-event endpoint for the feed's lazy expand, and a batch endpoint for the Changes tab. The feed carries only lightweight stats; these endpoints provide the actual diff content on demand.

## Changes

**`server/server.go`** — add two endpoints:
- `GET /api/sessions/{id}/events/{index}/diff` — single event diff. Uses `EventLine.Index` (stable NDJSON line number from phase 2) to seek directly. Returns one `DiffContent`.
- `GET /api/sessions/{id}/diffs` — batch endpoint. Single pass over the session's `events.ndjson`, extracts `tool_input` from all edit events, returns `DiffContent[]` with each item's `index` matching the `EventLine.Index`. Used by the Changes tab to avoid N individual requests.

**`internal/snapshot/diff_content.go`** (new) — two functions:
- `ReadEventDiffContent(runtimeDir, sessionID string, eventIndex int) (DiffContent, error)` — seeks to the Nth NDJSON line, extracts `tool_input`, returns structured content.
- `ReadAllDiffContent(runtimeDir, sessionID string) ([]DiffContent, error)` — single pass, returns all edit events' diff content with their indices.

**`internal/snapshot/types.go`** — add `DiffContent` struct with a `Type` discriminator:
- `Type string` — `"edit"`, `"write"`, or `"patch"` (discriminator for frontend variant selection)
- `Index int` — matches `EventLine.Index`
- `FilePath string`
- `OldString string` — populated when `Type == "edit"`
- `NewString string` — populated when `Type == "edit"`
- `Content string` — populated when `Type == "write"`
- `Patch string` — populated when `Type == "patch"`

## Data Structures

- `DiffContent` — `{ Type string; Index int; FilePath string; OldString string; NewString string; Content string; Patch string }` — `Type` discriminates which fields are populated
- Single response: `200 OK` with `DiffContent`, `404` if index out of range or event has no tool_input
- Batch response: `200 OK` with `DiffContent[]`

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | API design + NDJSON parsing needs care for correctness |

## Verification

### Static
- `go test ./internal/snapshot/...` — `ReadEventDiffContent` returns correct content for Edit, Write, and apply_patch events with correct `Type` discriminator; returns error for non-edit events and out-of-range indices. `ReadAllDiffContent` returns all edit events in one pass with correct indices.
- `go test ./server/...` — single endpoint returns 200 with content for valid requests, 404 for missing/non-edit events. Batch endpoint returns all diffs for a session.
- `go vet ./...` passes

### Runtime
- Run Noodle, trigger a session with Edit/Write/apply_patch events
- `curl /api/sessions/{id}/events/{index}/diff` — returns `DiffContent` with correct `type` discriminator
- `curl /api/sessions/{id}/diffs` — returns array of all edit diffs in one response
- Same single request for a Read/Bash event — returns 404
- Batch response indices match `EventLine.Index` values from `/api/sessions/{id}/events`
