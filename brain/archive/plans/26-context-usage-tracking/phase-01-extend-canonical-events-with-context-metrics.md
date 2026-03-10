Back to [[archive/plans/26-context-usage-tracking/overview]]

# Phase 1: Extend canonical events with context metrics

## Goal

Add context-window fields to `CanonicalEvent` and teach both provider adapters to populate them. This is the foundational data layer — everything else builds on it.

## Design note: compression as session-level, not per-turn

Compression/compacted events are separate NDJSON lines from result events. Claude has no turn ID; Codex has `turn_id` in `turn_context` but it's only embedded in message text. Correlating compression to a specific turn is unreliable.

**Decision:** Add a new `EventCompression` canonical event type (not `EventAction` with magic string matching — that's brittle). Both adapters emit `EventCompression` for their compression signals. Count at session level in phase 2. Do NOT add a `Compressed` field to `CanonicalEvent` — it's a separate event, not a per-turn attribute.

## Changes

- **`parse/canonical.go`** — Add field to `CanonicalEvent`:
  - `ContextTokens int` — input tokens for this turn (reflects context window usage at that point)

- **`parse/canonical.go`** — Add `EventCompression EventType` constant. This replaces the brittle `"text:context compacted"` string convention.

- **`parse/claude.go`** — Extract from Claude NDJSON:
  - `result` events already provide `tokens_in` via `extractClaudeUsage()` — map to `ContextTokens`
  - `system` events with compression indicators → emit `EventCompression`

- **`parse/codex.go`** — Two changes:
  - `compacted` event type → emit `EventCompression` (currently emits `EventAction` with magic string)
  - Codex emits token/cost data on `task_complete` (`EventComplete`), not `EventResult`. Ensure `ContextTokens` is populated on `EventComplete` events so phase 2 aggregation captures Codex data. Note: phase 2 must aggregate on both `EventResult` and `EventComplete`.

## Data structures

- `CanonicalEvent` gains `ContextTokens int` field

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.4` | Mechanical field additions and adapter logic |

## Verification

- `go test ./parse/...` — existing tests still pass
- New test: Claude compression event → emits `EventCompression`
- New test: Codex `compacted` event → emits `EventCompression`
- New test: `ContextTokens` populated from Claude `result` and Codex `task_complete` events
- New test: absent/partial provider context data → `ContextTokens` is 0, no crash
