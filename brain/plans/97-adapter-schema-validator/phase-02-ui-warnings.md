Back to [[plans/97-adapter-schema-validator/overview]]

# Phase 2: Surface Warnings in UI via LoopState

## Goal

Make adapter warnings visible in the web UI. Today, `server.warnings` is static (config validation warnings set at startup). Dynamic adapter warnings from each cycle need to reach the snapshot.

## Changes

### `loop/state_snapshot.go`

- Add `Warnings []string` field to `LoopState`
- In `buildLoopStateSnapshot()`: populate from `l.lastMiseWarnings` (new field on Loop)
- In `cloneLoopState()`: clone the warnings slice (`append([]string(nil), state.Warnings...)`)

### `loop/types.go` (or wherever Loop fields live)

- Add `lastMiseWarnings []string` field to `Loop` struct

### `loop/loop_cycle_pipeline.go`

- After `buildCycleBrief()` returns warnings: store them on `l.lastMiseWarnings`
- This must happen **before** `stampStatus()` is called, so `publishState()` picks up the current warnings

### `loop/stamp_status.go` — warning-change trigger

**Problem:** `stampStatus` compares `Active`, `LoopState`, `Mode`, `MaxConcurrency` to skip redundant file writes (line 33). If only warnings change, the status file is not rewritten, the fsnotify watcher is not triggered, and `loadAndBroadcast` never runs — the UI stales.

**Fix:** Add a `Warnings` field to `statusfile.Status` and include it in the equality check:

```go
if slices.Equal(l.lastStatus.Active, status.Active) &&
    l.lastStatus.LoopState == status.LoopState &&
    l.lastStatus.Mode == status.Mode &&
    l.lastStatus.MaxConcurrency == status.MaxConcurrency &&
    slices.Equal(l.lastStatus.Warnings, status.Warnings) {
```

This ensures a warning-only change triggers a file write → watcher → `loadAndBroadcast` → WS broadcast. Follows the existing `stampStatus` pattern exactly.

### `server/server.go` and `server/ws_hub.go`

- In `loadAndBroadcast()` (line 249): merge `warnings` (static config) with `snap.LoopState.Warnings` (dynamic adapter)
- In `loadInitialSnapshot()` (line 326): same merge
- In `handleSnapshot()` (line 173): same merge
- **Dedup:** Use `sort.Strings` + `slices.Compact` on the merged slice to produce deterministic, deduplicated output. Do NOT use a `map[string]bool` set — unstable iteration order would produce nondeterministic JSON serialization, defeating the SHA256 hash gate in `loadAndBroadcast` (line 252-267) and causing spurious WS broadcasts.

### `internal/snapshot/types.go`

- No changes needed — `Snapshot.Warnings` is already `[]string`

## Data structures

- `LoopState.Warnings []string` — new field, ephemeral per cycle
- `statusfile.Status.Warnings []string` — new field for change detection

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | Cross-layer wiring between loop and server, needs judgment on merge strategy and change-detection path |

## Verification

### Static
- `go test ./loop/... ./server/...` passes
- `go vet ./loop/... ./server/...` clean

### Runtime
- Test: loop with adapter warnings → `LoopState.Warnings` populated
- Test: server snapshot includes both config warnings and adapter warnings
- Test: when adapter is fixed (no warnings), `LoopState.Warnings` is empty
- Test: duplicate warnings between config and adapter are deduped
- Test: warning-only change (no other state change) still triggers WS broadcast
- Test: identical warnings across cycles do NOT trigger spurious broadcasts (hash gate)
