Back to [[plans/66-event-trigger-system/overview]]

# Phase 6 — Surface Events in Mise Brief

## Goal

Make lifecycle events visible to the schedule agent by adding `recent_events` to the mise brief. The builder reads `loop-events.ndjson`, finds the watermark, and includes post-watermark events.

**Import cycle note:** `loop` imports `mise`. Therefore `mise` cannot import `loop` or `event` for `LoopEvent`. The builder defines its own minimal struct to unmarshal NDJSON lines. The file format is the API contract between packages — no shared Go type needed.

## Changes

- **`mise/types.go`** — add `RecentEvent` struct and `RecentEvents []RecentEvent` field to `Brief`.
- **`mise/builder.go`** — add `readRecentEvents(runtimeDir string) []RecentEvent`:
  1. Read `loop-events.ndjson` (tail, max 200 lines for startup truncation).
  2. Find the last `schedule.completed` event's sequence — this is the watermark.
  3. Collect events with sequence > watermark, cap at 50 most recent.
  4. Map each to `RecentEvent` (type, seq, at, summary string derived from payload).
  5. On any read/parse error: return empty slice with a warning (best-effort, never crash the brief build).
- **`mise/builder_test.go`** — tests for watermark logic, truncation, empty file, malformed lines.

## Data Structures

- `RecentEvent` — `Type string`, `Seq uint64`, `At time.Time`, `Summary string`. Deliberately flat and string-typed — the schedule agent reads these as natural language context, not structured data to pattern-match on.

## Routing

Provider: `codex`, Model: `gpt-5.3-codex`

## Verification

```bash
go test ./mise/... && go vet ./...
```

- `mise` package does NOT import `loop` or `event/loop_event.go` (verify with `go vet` and import graph)
- Brief includes `recent_events` with correct post-watermark events
- Empty event file produces empty `recent_events` (no error)
- Malformed lines are skipped without crashing
- Watermark correctly advances when a new `schedule.completed` appears
- Cap of 50 events respected
