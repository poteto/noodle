Back to [[archive/plans/27-remote-dispatchers/overview]]

# Phase 9: TUI remote session indicator

**Routing:** `codex` / `gpt-5.4` — small UI change, clear spec

## Goal

Add a subtle badge in the TUI session list showing when a session is running remotely and which backend it uses. Use `bubbletea-tui` skill for styling guidance.

## Data structures

- Add `Runtime string` field to `spawn.json` metadata (already written by `writeDispatchMetadata`)
- Read it back in monitor/TUI when displaying sessions

## Changes

**`dispatcher/streaming_dispatcher.go`**
Include `Runtime` field in the dispatch metadata written to `spawn.json`. Already has provider, model, skill — add runtime kind.

**`monitor/derive.go`**
Pass runtime kind through to `SessionMeta` so the TUI can read it. Add `Runtime string` field to `SessionMeta`.

**`tui/` (session list component)**
When rendering a session row, if `Runtime` is not empty and not "tmux", render a small dimmed badge after the session name: e.g., `session-name  sprites` or `session-name  cursor`. Use the theme's muted color. Invoke `bubbletea-tui` skill for component styling.

## Verification

### Static
- Compiles, passes vet

### Runtime
- Unit test: session with runtime="sprites" renders badge in session row
- Visual: launch TUI with a mock remote session, verify badge appears and doesn't break layout
