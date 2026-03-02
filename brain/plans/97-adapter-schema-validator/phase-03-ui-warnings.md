Back to [[plans/97-adapter-schema-validator/overview]]

# Phase 3: Surface Warnings in UI via LoopState

## Goal

Make adapter warnings visible in the web UI. Today, `server.warnings` is static (config validation warnings set at startup). Dynamic adapter warnings from each cycle need to reach the snapshot.

## Changes

### `loop/state_snapshot.go`

- Add `Warnings []string` field to `LoopState`
- In `buildLoopStateSnapshot()`: populate from `l.lastMiseWarnings` (new field on Loop)
- In `cloneLoopState()`: clone the warnings slice

### `loop/types.go` (or wherever Loop fields live)

- Add `lastMiseWarnings []string` field to `Loop` struct
- Set in the cycle pipeline after `buildCycleBrief` returns warnings

### `loop/loop_cycle_pipeline.go`

- After `buildCycleBrief()` returns warnings: store them on `l.lastMiseWarnings`

### `server/server.go` and `server/ws_hub.go`

- In `handleSnapshot()`: merge `s.warnings` (config) with loop state warnings
- In `loadAndBroadcast()`: same merge
- In `loadInitialSnapshot()`: same merge
- Dedup: use a set to avoid repeating the same warning from both sources

### `internal/snapshot/types.go`

- No changes needed — `Snapshot.Warnings` is already `[]string`

## Data structures

- `LoopState.Warnings []string` — new field, ephemeral per cycle

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | Cross-layer wiring between loop and server, needs judgment on merge strategy |

## Verification

### Static
- `go test ./loop/... ./server/...` passes
- `go vet ./loop/... ./server/...` clean

### Runtime
- Test: loop with adapter warnings → LoopState.Warnings populated
- Test: server snapshot includes both config warnings and adapter warnings
- Test: when adapter is fixed (no warnings), LoopState.Warnings is empty
- Test: duplicate warnings between config and adapter are deduped
