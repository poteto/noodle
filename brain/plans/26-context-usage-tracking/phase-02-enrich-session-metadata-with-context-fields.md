Back to [[plans/26-context-usage-tracking/overview]]

# Phase 2: Enrich session metadata with context fields

## Goal

Aggregate per-turn context metrics from canonical events into session-level metadata. Replace the crude `ContextWindowUsagePct` calculation with richer fields.

## Changes

- **`monitor/types.go`** — Add fields to `SessionClaims`:
  - `PeakContextTokens int` — highest `ContextTokens` value seen across all turns
  - `CompressionCount int` — count of `EventCompression` events
  - `TurnCount int` — total number of result/complete events (for computing averages)

- **`monitor/claims.go`** — Update `accumulateClaim()` to track:
  - Max of `ContextTokens` → `PeakContextTokens` (on `EventResult` AND `EventComplete` — Codex emits tokens on complete, not result)
  - Count of `EventCompression` events → `CompressionCount`
  - Increment `TurnCount` on each `EventResult` or `EventComplete`

- **`internal/sessionmeta/sessionmeta.go`** — Replace `ContextWindowUsagePct float64` with:
  - `PeakContextTokens int`
  - `CompressionCount int`
  - `TurnCount int`
  - `ContextWindowUsagePct float64` — recomputed from `PeakContextTokens / contextTokenBudget` (same field name, new semantics: peak-turn instead of cumulative)

- **`monitor/derive.go`** — Map new claims fields into `Meta`. Replace the old `(tokensIn+tokensOut)/budget` computation with `PeakContextTokens/budget`. Update the health threshold check (currently `>= 80%` → `HealthYellow`) to use the new semantics.

- **Caller migration** — update all readers of `ContextWindowUsagePct`:
  - `tui/model_snapshot.go` (~line 86, 282) — reads for display/health
  - `monitor/derive.go` (~line 57-59) — health threshold check
  - Any other consumers found via grep

## Data structures

- `SessionClaims` gains `PeakContextTokens`, `CompressionCount`, `TurnCount`
- `Meta`: `ContextWindowUsagePct` semantics change from cumulative to peak-turn; add `PeakContextTokens`, `CompressionCount`, `TurnCount`

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical aggregation logic |

## Verification

- `go test ./monitor/...` — existing tests pass
- New test: feed canonical events with varying `ContextTokens` → `PeakContextTokens` is the max
- New test: feed `EventAction` with `"text:context compacted"` → `CompressionCount` incremented
- New test: health threshold still triggers at 80% with new peak-turn semantics
- Grep for all `ContextWindowUsagePct` consumers — verify all updated
