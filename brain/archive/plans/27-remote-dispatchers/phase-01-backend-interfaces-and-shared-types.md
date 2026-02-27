Back to [[archive/plans/27-remote-dispatchers/overview]]

# Phase 1: Backend interfaces and shared types

**Routing:** `codex` / `gpt-5.3-codex` — clear spec, mechanical types

## Goal

Define the two backend interfaces and the shared types that both dispatchers use. This is the foundation — get these right and everything else follows.

## Data structures

- `StreamingBackend` interface — `Start`, `IsAlive`, `Kill` methods. Tmux, Sprites, and any future streaming backend all implement this.
- `PollingBackend` interface — `Launch`, `PollStatus`, `GetConversation`, `Stop`, `Delete` methods
- `StreamHandle` struct — wraps `io.Reader` (stdout), process/session identifier, provider name
- `RemoteStatus` enum-like — `Running`, `Completed`, `Failed`, `Expired`, `Unknown`
- `ConversationMessage` struct — role, text, timestamp

## Changes

**`dispatcher/backend.go` (new)**
Define `StreamingBackend` and `PollingBackend` interfaces. `StreamingBackend.Start` takes a `StreamStartConfig` (session ID, command pipeline, env, working dir) and returns a `StreamHandle`. `PollingBackend.Launch` takes a `PollLaunchConfig` (prompt text, repo URL, model, branch) and returns a remote ID string.

Design `StreamingBackend.Start` so it works for both tmux (start a tmux session, pipe stdout) and Sprites (exec a command, pipe stdout). The config carries the fully-built command pipeline — the backend just runs it in its environment.

**`dispatcher/backend_types.go` (new)**
`StreamHandle`, `RemoteStatus`, `ConversationMessage`. Keep these minimal — backends extend with their own internal state.

## Verification

### Static
- `go build ./dispatcher/...` compiles
- Interfaces have no concrete dependencies (pure method signatures)
- No import cycles

### Runtime
- Compile-time interface checks: `var _ StreamingBackend = (*TmuxBackend)(nil)`, `var _ StreamingBackend = (*SpritesBackend)(nil)`, `var _ PollingBackend = (*CursorBackend)(nil)` (placeholder stubs)
