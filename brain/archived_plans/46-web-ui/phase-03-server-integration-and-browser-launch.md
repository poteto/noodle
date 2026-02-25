Back to [[archived_plans/46-web-ui/overview]]

# Phase 3: Server Integration and Browser Launch

## Goal

Wire the server into `noodle start` so it runs alongside the loop and opens the browser.

## Changes

- **`cmd_start.go`** — In `runStartWithTUI`, add a third goroutine for `server.Start(ctx)`. After the server is listening, open `localhost:PORT` in the default browser. Browser open uses `exec.Command("open", url)` on macOS, `xdg-open` on Linux, `cmd /c start` on Windows. The TUI still runs — both coexist for now.
- **`config/config.go`** — Add `Server ServerConfig` to `Config` struct. Fields: `Port int` (default 3000), `Enabled *bool` (nil = auto: enabled in interactive, disabled in headless).
- **`server/embed.go`** — `//go:embed` directive for `ui/dist/` (the built SPA). Falls back to a placeholder HTML page if `ui/dist/` doesn't exist yet.
- **Port finding** — If configured port is busy, try incrementing (3001, 3002...) up to 10 attempts.

## Data structures

- `config.ServerConfig` — `Port int`, `Enabled *bool`

## Routing

Provider: `codex` | Model: `gpt-5.3-codex`

## Verification

### Static
- `go test ./...` passes
- `go vet ./...` clean

### Runtime
- `noodle start` opens browser (serves placeholder or embedded SPA)
- `noodle start --headless` does not start the server
- Server shuts down cleanly on Ctrl+C
- Port collision handled gracefully
