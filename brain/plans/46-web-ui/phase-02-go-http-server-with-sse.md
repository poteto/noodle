Back to [[plans/46-web-ui/overview]]

# Phase 2: Go HTTP Server with SSE

## Goal

Build the HTTP server that serves snapshot data via SSE and accepts control commands via POST. This is the backend the web UI will consume.

## Changes

- **New `server/` package** — `server.go` with `Server` struct. Uses stdlib `net/http`. Key endpoints:
  - `GET /api/events` — SSE stream. On connect, sends full snapshot immediately. Then watches `.noodle/` via fsnotify (debounced ~1s) and pushes snapshot diffs. Each message: `data: {json}\n\n`.
  - `GET /api/snapshot` — Current snapshot as JSON (for initial page load before SSE connects).
  - `GET /api/sessions/{id}/events` — Event log for a session. Optional `?after=` query param for incremental fetches.
  - `POST /api/control` — Accepts `loop.ControlCommand` JSON, appends to `control.ndjson` (same mechanism as `tui.sendControlCmd`). Returns `ControlAck` JSON in the response body so the UI gets immediate feedback.
  - `GET /api/config` — Returns default provider, model, skill, and available options from `.noodle.toml`. Used by the task editor to populate dropdowns.
  - `GET /` — Serves embedded SPA (wired in phase 3, placeholder for now).
- **SSE client management** — Mutex-protected client list. Each client gets a channel. Broadcast goroutine fans out snapshots. Clean up on client disconnect. Skip push if snapshot hasn't changed since last send (diff-gate) to avoid churn under rapid `.noodle/` writes.
- **CORS** — Allow `localhost:*` origins for dev (Vite on different port).

## Data structures

- `server.Server` — `runtimeDir string`, `httpServer *http.Server`, `clients` map, `mu sync.RWMutex`, cached `*snapshot.Snapshot`
- `server.Config` — `Port int`, `RuntimeDir string`

## Routing

Provider: `claude` | Model: `claude-opus-4-6` — SSE fan-out, fsnotify debounce, graceful shutdown, concurrent client management.

## Verification

### Static
- `go test ./server/...` — unit tests for SSE message encoding, control command appending, snapshot JSON shape
- `go vet ./...` clean

### Runtime
- Start server, `curl localhost:PORT/api/snapshot` returns valid JSON
- `curl -N localhost:PORT/api/events` receives SSE messages on file changes
- `curl -X POST localhost:PORT/api/control -d '{"action":"pause"}'` appends to `control.ndjson`
