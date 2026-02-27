Back to [[archived_plans/66-event-trigger-system/overview]]

# Phase 6 — Surface Events in Mise Brief and Spawn Schedule on Change

## Goal

Two connected changes: (1) add `recent_events` to the mise brief so the schedule agent can see lifecycle events, and (2) spawn the schedule agent whenever mise.json content changes, not just when the kitchen is idle.

Currently the schedule agent only spawns when there are no active cooks, no adopted targets, and no non-schedule orders. That means events emitted while cooks are running (stage completions, merges, failures) won't be seen until everything finishes. The schedule agent needs to run whenever there's new information to react to.

**Import cycle note:** `loop` imports `mise`. Therefore `mise` cannot import `loop` or `event` for `LoopEvent`. The builder defines its own minimal struct to unmarshal NDJSON lines. The file format is the API contract between packages — no shared Go type needed.

## Changes

### Mise brief: add recent_events

- **`mise/types.go`** — add `RecentEvent` struct and `RecentEvents []RecentEvent` field to `Brief`.
- **`mise/builder.go`** — add `readRecentEvents(runtimeDir string) []RecentEvent`:
  1. Read `loop-events.ndjson` (tail, max 200 lines for startup truncation).
  2. Find the last `schedule.completed` event's sequence — this is the watermark.
  3. Collect events with sequence > watermark, cap at 50 most recent.
  4. Map each to `RecentEvent` (type, seq, at, summary string derived from payload).
  5. On any read/parse error: return empty slice with a warning (best-effort, never crash the brief build).
- **`mise/builder_test.go`** — tests for watermark logic, truncation, empty file, malformed lines.

### Schedule spawning: react to mise changes

- **`mise/builder.go`** — `Build()` returns an additional `bool` indicating whether mise.json was actually written (content changed). The internal `!bytes.Equal(content, b.lastContent)` check already knows this — expose it.
- **`loop/types.go`** — update `MiseBuilder` interface: `Build(...)` return type changes from `(Brief, []string, error)` to `(Brief, []string, bool, error)` (the new `bool` = content changed).
- **`loop/loop.go`** — `buildCycleBrief()`: propagate the changed signal. `prepareOrdersForCycle()`: when mise.json changed and no schedule cook is active (`l.cooks.activeCooksByOrder` has no schedule entry), inject a schedule order. The existing idle-bootstrap path remains for the case when the kitchen is empty with no orders — this new path adds schedule spawning mid-cycle when events arrive.

## Data Structures

- `RecentEvent` — `Type string`, `Seq uint64`, `At time.Time`, `Summary string`. Deliberately flat and string-typed — the schedule agent reads these as natural language context, not structured data to pattern-match on.

## Routing

Provider: `codex`, Model: `gpt-5.3-codex`

## Verification

```bash
go test ./mise/... ./loop/... && go vet ./...
```

- `mise` package does NOT import `loop` or `event/loop_event.go` (verify with `go vet` and import graph)
- Brief includes `recent_events` with correct post-watermark events
- Empty event file produces empty `recent_events` (no error)
- Malformed lines are skipped without crashing
- Watermark correctly advances when a new `schedule.completed` appears
- Cap of 50 events respected
- `Build()` returns `changed=true` when content differs, `changed=false` when identical
- Schedule agent spawns when mise.json changes and no schedule cook is active
- Schedule agent does NOT double-spawn when one is already running
- Existing idle-bootstrap behavior preserved (kitchen empty → schedule spawns)
