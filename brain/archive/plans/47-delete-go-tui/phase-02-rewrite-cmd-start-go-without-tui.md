Back to [[archive/plans/47-delete-go-tui/overview]]

# Phase 2 — Rewrite cmd_start.go without TUI

## Goal

Redesign `cmd_start.go` from first principles as if TUI never existed. The `start` command launches the runtime loop, optionally a web server, and streams log output to the terminal so the operator has visibility into what's happening.

## Changes

**`cmd_start.go`:**
- Remove imports: `tea "charm.land/bubbletea/v2"`, `"github.com/poteto/noodle/tui"`
- Remove `--headless` flag and `startOptions.headless` field
- Remove `runTUI()` function entirely
- Remove `runStartTUI` variable
- Remove `runStartWithTUI()` function
- Remove `isInteractiveTerminal()` usage for TUI decision (keep the function in `main.go` — it's used by config repair)
- Rewrite `runStart()`: after `--once` check, start loop + optionally web server + log streamer, wait for all. `shouldStartServer` should default to true when `server.enabled` is nil (no longer gated on interactive terminal).
- Gate `openBrowser` on `terminalInteractiveCheck()` (the injectable var in `main.go:43`) — the server can run in CI/cron but should not try to open a browser there. Note: `cmd_start.go` currently calls `isInteractiveTerminal()` directly; switch to using the `terminalInteractiveCheck` package-level var so tests can mock it.
- Preserve `defer runtimeLoop.Shutdown()` — this kills active sessions on exit. Losing it leaves zombie tmux sessions.

**Cancellation and shutdown:**
`runStart` manages three goroutines (loop, server, streamer). Follow the existing pattern from `runStartWithTUI`:
- All share a `context.Context` from `signal.NotifyContext`
- Loop error or SIGINT cancels the context
- Server and streamer treat context cancellation as clean shutdown (non-fatal)
- Loop errors propagate; server/streamer errors log to stderr but don't propagate

**Terminal log streaming:**
The loop already writes diagnostic events to stderr (skill registry changes, session failures, queue drops, bootstrap progress). It also writes NDJSON event files per session and queue events.

Add a log streamer to `runStart` that tails key runtime events to stderr. Data sources:

- **Session events** — watch `.noodle/sessions/` directory for new subdirs, then tail each `events.ndjson` for `spawned`, `exited`, `state_change` events. Note: session completion comes as `exited` events with an outcome payload, not a separate `cook_completed` type.
- **Queue events** — tail `.noodle/queue-events.ndjson` for `queue_drop`, `registry_rebuild`, `bootstrap_complete`
- **Status file** — watch `.noodle/status.json` for loop state changes (running/paused/idle/draining)
- **Pending review** — watch `.noodle/pending-review.json` for new review items

Implementation notes:
- fsnotify watches are not recursive. The streamer must watch the `sessions/` directory itself for `Create` events (new session dirs), then add a watch on each new `sessions/<id>/events.ndjson`. Clean up watches when sessions complete.
- `status.json` and `pending-review.json` are written via atomic rename (`WriteAtomic`). fsnotify sees `Create` events (the rename target), not `Write`. Handle accordingly.
- `.noodle/sessions/` may not exist at streamer startup (created by `Loop.Run`). The streamer must handle missing directories gracefully — retry/poll until the dir appears, or watch `.noodle/` for the `sessions` dir creation.

Print human-readable one-liners to stderr for key lifecycle events:
  - Session spawned: `cook: spawned "task-name" (claude/claude-opus-4-6) in worktree xyz`
  - Session completed: `cook: completed "task-name" (45s, $0.12)`
  - Session failed: `cook: failed "task-name": <reason>`
  - Review pending: `review: "task-name" awaiting approval`
  - Loop state: `loop: paused`, `loop: running`, `loop: idle`

Keep it simple — just a goroutine that tails files and formats lines. No TUI framework, no raw mode, no cursor manipulation. The `--once` path should NOT stream (it's a one-shot cycle).

**`cmd_start_test.go`:**
- Remove `TestRunStartWithTUIStopsLoopOnExit`
- Remove `TestRunStartWithTUIPropagatesLoopError`
- Remove `TestRunStartWithTUIExitsWhenLoopFailsBeforeTUIExits`
- Remove `TestRunStartWithTUIPropagatesTUIError`
- Keep `TestRunStartOnceUsesLoopCycle`
- Keep `TestShouldStartServer` — update: nil should now return true regardless of terminal interactivity
- Rename `TestRunStartWithTUIStartsServer` → `TestRunStartStartsServer`, rewrite for new loop+server flow
- Add: test that browser is NOT opened in non-interactive mode (mock `terminalInteractiveCheck`)
- Add: test streamer cancellation (starts, receives context cancel, exits cleanly)

## Data structures

- `startOptions` — remove `headless` field, keep `once`
- Log streamer — a small struct or function that watches NDJSON/JSON files and writes formatted lines to an `io.Writer`

## Routing

| Provider | Model |
|----------|-------|
| `claude` | `claude-opus-4-6` |

Design judgment needed for the log streamer implementation and `shouldStartServer` change.

## Verification

### Static
- `go build ./...` passes
- `go vet ./...` clean
- No remaining references to `tui` package, `bubbletea`, or `runStartTUI`

### Runtime
- `go test ./...` passes
- `noodle start` launches loop + web server, opens browser (interactive terminal), prints lifecycle events to stderr
- `noodle start --once` runs one cycle and exits (no streaming)
- `noodle start --headless` produces "unknown flag" error (clean failure)
- Manually verify: start the loop, enqueue a task, confirm terminal shows spawn/complete lines
