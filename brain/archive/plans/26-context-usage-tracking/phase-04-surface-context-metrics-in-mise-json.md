Back to [[archive/plans/26-context-usage-tracking/overview]]

# Phase 4: Surface context metrics in mise.json

## Goal

Expose context metrics in `mise.json` so skills (prioritize, quality, meditate) can consume them without reading raw session files. Add per-skill aggregated stats computed from recent session history.

## Changes

- **`mise/types.go`** — Add to `HistoryItem`:
  - `PeakContextTokens int`
  - `CompressionCount int`
  - `TurnCount int`
  - `Skill string`

- **`mise/types.go`** — Add new type `ContextStats` and a `ContextSummary` field to `Brief`:
  - `ContextStats` per skill: skill name, avg peak context pct, total compressions, session count
  - `ContextSummary` at the brief level: aggregate across recent history
  - `AvgPeakContextPct` should be computed from each session's already-persisted `ContextWindowUsagePct` in meta — do NOT re-derive from raw tokens (avoids duplicating the budget constant from `monitor/types.go`)

- **`mise/builder.go`** — In `readSessionState()`:
  - Copy new meta fields into `HistoryItem`
  - After building history (already bounded to 20), compute `ContextSummary` by grouping `HistoryItem` by `Skill` and aggregating

## Data structures

- `ContextStats`: `Skill string`, `AvgPeakContextPct float64`, `TotalCompressions int`, `SessionCount int`
- `Brief.ContextSummary []ContextStats` — sorted by `AvgPeakContextPct` descending (worst offenders first)

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.4` | Mechanical aggregation and type additions |

## Verification

- `go test ./mise/...` — existing tests pass
- New test: build mise from sessions with context metrics → `ContextSummary` correctly aggregated
- Inspect generated `mise.json` — `context_summary` section present with per-skill stats
