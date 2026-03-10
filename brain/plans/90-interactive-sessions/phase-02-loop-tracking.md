Back to [[plans/90-interactive-sessions/overview]]

# Phase 2 — Loop Tracking

## Goal

Track active interactive sessions in the loop alongside order sessions. Handle lifecycle events (completion, failure, termination) using the existing serialized completion infrastructure — no direct goroutine mutation of loop maps.

## Changes

**`loop/types.go`** — Define `chatHandle` struct to track an interactive session (session ID, display name, session handle reference, started-at). Simpler than `cookHandle` — no order/stage fields.

**`loop/loop.go`** (or relevant loop state file) — Add `chats map[string]*chatHandle` to `Loop` struct. Initialize in constructor.

**`loop/chat.go`** — After spawning in `controlChat()`, create a `chatHandle` and store in `l.chats`. Start a session watcher that routes completion through the existing `completionBuffer` (same pattern as `startSessionWatcher` in `cook_watcher.go`). The completion handler removes from `l.chats` on the loop cycle — never from a goroutine directly. Emit lifecycle events (`EventSpawned`, `EventExited`) using existing event infrastructure.

**`loop/control.go`** — Wire `stop` and `kill` actions to also check `l.chats` map (not just cook handles). Allow `stop-all` to terminate interactive sessions too.

**Concurrency accounting** — Include interactive sessions in `atMaxConcurrency` checks. Interactive sessions consume runtime slots just like cook sessions; without counting them, the loop may attempt to spawn order sessions that exceed the runtime's concurrency limit, causing false dispatch failures.

**Completion buffer for non-order sessions** — The existing completion buffer is keyed by order identity. Chat completions have no order. Either extend the buffer to support kind-agnostic completions (add a `Kind` discriminator to completion entries), or add a separate `chatCompletions` channel processed on the loop cycle alongside the existing buffer. Either way, the handler removes from `l.chats` on the loop cycle.

**Critical: no goroutine writes to `l.chats`.** All mutations happen on the loop cycle via the completion buffer/channel, matching the existing cook completion pattern. This prevents concurrent map access panics.

## Data Structures

- `chatHandle` — `{ id string, name string, session runtime.SessionHandle, startedAt time.Time }`

## Routing

- **Provider:** `codex`
- **Model:** `gpt-5.4`

## Verification

### Static
- `go build ./...`
- `go vet ./...`
- Existing `stop`/`stop-all` tests still pass

### Runtime
- Test with `-race`: spawn interactive session, verify it appears in `l.chats`
- Test with `-race`: interactive session completes naturally, verify cleanup from `l.chats` happens on loop cycle (not goroutine)
- Test: `stop` action with interactive session ID, verify session terminated and removed from `l.chats`
- Test: `stop-all` terminates both order sessions and interactive sessions
- Test: concurrent stop-all + session completion — no race (verify with `-race`)
- Test: interactive sessions count against max concurrency — spawn chat at limit, verify order dispatch is blocked
- Test: chat completion flows through buffer/channel and removes from `l.chats` on loop cycle
