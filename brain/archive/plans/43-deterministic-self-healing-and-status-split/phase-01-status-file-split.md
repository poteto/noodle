Back to [[archive/plans/43-deterministic-self-healing-and-status-split/overview]]

# Phase 1: Status file split

## Goal

Move loop runtime state out of queue.json into a dedicated `.noodle/status.json`. After this phase, queue.json is purely scheduling intent (agent-written) and status.json is runtime state (noodle-written).

## Changes

### New status type and file (`internal/statusfile/`)

New package with a `Status` struct and read/write functions:
- `Status` — `Active []string`, `LoopState string`, `Autonomy string`
- `Read(path)` / `WriteAtomic(path, status)` — same atomic write pattern as queuex

### Remove loop state from queue structs

- **`internal/queuex/queue.go`** — Remove `Active`, `LoopState`, `Autonomy` fields from `Queue` struct. Keep `ActionNeeded` — it's scheduling intent written by the prioritize agent, not loop state.
- **`loop/types.go`** — Remove same fields from loop's `Queue` struct
- **`loop/queue.go`** — Delete `stampLoopState()`. Add `stampStatus()` that writes to status.json. Remove Active/Autonomy/LoopState from `toQueueX()`/`fromQueueX()`.
- **`loop/loop.go`** — Update all `stampLoopState()` call sites (lines 178, 186, 194, 201) to `stampStatus()`.

### Update TUI to read status.json

- **`tui/model_snapshot.go`** — `loadSnapshot()` reads `status.json` alongside `queue.json`. The `queueResult` struct loses Active/LoopState/Autonomy fields; those come from the status file. The TUI uses poll-based refresh (tick → snapshot), so no new file watchers are needed.

### Update CLI commands

- **`cmd_status.go`** — Currently derives loop state from session metadata via `readSessionSummary()`, not from queue.json. After this change, read from status.json as the authoritative source (it's written by the loop every cycle). Fall back to session-derived state if status.json doesn't exist (loop not running).
- **`cmd_debug.go`** — Currently derives loop state from session metadata via `readDebugSessions()`. Include status.json in the debug dump as an additional data source.

## Data structures

- `statusfile.Status` — new struct: `Active []string`, `LoopState string`, `Autonomy string`

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Mechanical struct changes, file read/write boilerplate |

## Verification

### Static
```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

### Runtime
- Run `noodle start`, verify TUI renders loop state, active spinners, autonomy mode
- Verify queue.json no longer contains `active`, `loop_state`, or `autonomy` fields
- Verify status.json is created with those fields
- Write to queue-next.json as prioritize agent — confirm loop promotes it without state stamping race

### Degraded states (must be covered by automated tests)
- **status.json absent** (loop not yet started, or first run): `cmd_status` and `cmd_debug` fall back to session-derived loop state. TUI `loadSnapshot()` returns sensible defaults (loop_state "running", empty active list, default autonomy).
- **status.json stale** (loop crashed, file left behind): TUI and CLI commands show last-known state — verify no crash or hang. `statusfile.Read()` returns the stale data; consumers don't assume freshness.
- **status.json malformed**: `statusfile.Read()` returns an error. Consumers fall back to defaults, don't crash.
