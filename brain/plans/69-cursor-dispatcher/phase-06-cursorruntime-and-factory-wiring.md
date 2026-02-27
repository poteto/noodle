Back to [[plans/69-cursor-dispatcher/overview]]

# Phase 6: CursorRuntime and Factory Wiring

**Routing:** `codex` / `gpt-5.3-codex` — mechanical wiring following existing patterns

## Goal

Wire the Cursor dispatcher into Noodle's runtime system. Add `CursorRuntime` (thin wrapper like `SpritesRuntime`), update `AvailableRuntimes()` to include cursor when configured, and wire into `defaultDependencies()`.

## Data structures

- No new types — reuses `DispatcherRuntime` via `NewCursorRuntime` constructor function

## Changes

**`runtime/cursor.go` (new)**
- `NewCursorRuntime(d dispatcher.Dispatcher, runtimeDir string, maxConcurrent int) Runtime` — same pattern as `NewSpritesRuntime`: wraps `PollingDispatcher` in `DispatcherRuntime`, sets max concurrent

**`config/config.go`**
- `AvailableRuntimes()` — add cursor when `cursorDefined` and `APIKey()` is non-empty (same pattern as sprites)

**`loop/defaults.go`**
- In `defaultDependencies()`, add cursor runtime block (after sprites block):
  - Guard: `runtimeEnabled(cfg.AvailableRuntimes(), "cursor")`
  - Check: `cfg.Runtime.Cursor.Repository` is non-empty (required for Cursor API)
  - Create: `NewCursorBackend(apiKey, baseURL, repository)` → `NewPollingDispatcher(cfg)` → `NewCursorRuntime(dispatcher, runtimeDir, maxConcurrent)`
  - Register: `runtimes["cursor"] = cursorRuntime`

**`config/config_test.go`**
- Test: `AvailableRuntimes()` includes "cursor" when cursor section is defined and API key env var is set
- Test: `AvailableRuntimes()` excludes "cursor" when API key is missing

## Verification

### Static
- `go vet ./...`
- `go build ./...`

### Runtime
- `go test ./config/... -run TestAvailableRuntimes -race`
- `go test ./runtime/... -run TestCursorRuntime -race`
- `go test ./loop/... -race`
- Full: `go test ./... && sh scripts/lint-arch.sh`
