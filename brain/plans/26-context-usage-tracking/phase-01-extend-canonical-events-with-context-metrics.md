Back to [[plans/26-context-usage-tracking/overview]]

# Phase 1: Extend canonical events with context metrics

## Goal

Add context-window fields to `CanonicalEvent` and teach both provider adapters to populate them. This is the foundational data layer — everything else builds on it.

## Design note: compression as session-level, not per-turn

Compression/compacted events are separate NDJSON lines from result events. Claude has no turn ID; Codex has `turn_id` in `turn_context` but it's only embedded in message text. Correlating compression to a specific turn is unreliable.

**Decision:** Emit compression as a standalone `EventAction` canonical event (Codex already does this: `"text:context compacted"`). Count them at the session level in phase 2. Do NOT add a `Compressed` field to `CanonicalEvent` — it's a separate event, not a per-turn attribute.

## Changes

- **`parse/canonical.go`** — Add field to `CanonicalEvent`:
  - `ContextTokens int` — input tokens for this turn (reflects context window usage at that point)

- **`parse/claude.go`** — Extract from Claude NDJSON:
  - `result` events already provide `tokens_in` via `extractClaudeUsage()` — map to `ContextTokens`
  - Ensure `system` events with compression indicators emit an `EventAction` with a recognizable message prefix (e.g., `"text:context compacted"`) matching the Codex convention

- **`parse/codex.go`** — Already emits `EventAction` with `"text:context compacted"` for `compacted` events. Add `ContextTokens` extraction from `turn_context` or `event_msg` cost data if token counts are available.

## Data structures

- `CanonicalEvent` gains `ContextTokens int` field

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical field additions and adapter logic |

## Verification

- `go test ./parse/...` — existing tests still pass
- New test: Claude compression event → emits `EventAction` with `"text:context compacted"` message
- New test: Codex `compacted` event → emits `EventAction` with `"text:context compacted"` (already works, add explicit assertion)
- New test: `ContextTokens` populated from both providers' result events
- New test: absent/partial provider context data → `ContextTokens` is 0, no crash
