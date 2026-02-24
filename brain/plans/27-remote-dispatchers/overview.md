---
id: 27
created: 2026-02-23
status: ready
---

# Remote Dispatchers

## Context

Noodle dispatches agent sessions exclusively via local tmux. To support cloud execution (Sprites.dev VMs, Cursor Cloud Agent, future providers), we need two generic dispatcher patterns that any backend can implement:

- **StreamingDispatcher** — for backends that provide real-time stdout/stderr (like Sprites). Full event fidelity, parsed in-process using existing `parse/` adapters.
- **PollingDispatcher** — for backends that expose status-polling REST APIs (like Cursor). Synthesizes events from status transitions and conversation history.

Both reuse the existing `Session` interface so the loop, TUI, and monitor work unchanged.

## Scope

**In:** Generic interfaces (`StreamingBackend`, `PollingBackend`), both dispatchers, `SpritesBackend`, `CursorBackend`, dispatcher factory, config, monitor observer abstraction, TUI remote badge, minimal sync-back (remote branch fetch+merge for streaming, PR URL recording for polling).

**Out:** Worktree sync *to* remote (Sprites clones via git — the agent uses its own git clone), Cursor PR lifecycle management (auto-merge, check status), warm Sprite pools, remote-specific skill loading, conflict auto-resolution.

## Constraints

- Go interfaces — backends implement method sets; dispatchers accept any conforming type
- Reuse `SessionEvent`, `EventWriter`, `parse.CanonicalEvent`, existing parse adapters
- Event handling decided per-backend type: streaming parses in-process, polling synthesizes
- Provider validation must relax — Cursor uses its own model names
- `sprites-go` SDK (`github.com/superfly/sprites-go`) mirrors `exec.Cmd` — natural fit
- Cursor API is simple REST (bearer auth, JSON bodies, polling GET endpoints)
- **Runtime is a scheduling decision, not a skill property.** Users configure available runtimes in `.noodle.toml`. The prioritize agent picks runtime per queue item, same as it picks provider/model. Runtime flows: config → mise.json (available_runtimes) → prioritize agent → queue.json item → dispatcher.
- **Command assembly ownership:** StreamingDispatcher owns prompt composition and command pipeline building. Backends receive a prebuilt command and just run it in their environment (tmux session, Sprites VM). PollingBackend receives prompt text — no command pipeline.
- **Runtime is a first-class queue field.** Must flow through all layers: `queuex.Item` → `loop.QueueItem` → `toQueueX`/`fromQueueX` conversions → JSON schema → `DispatchRequest.Runtime`.

## Alternatives considered

1. **Runtime in skill frontmatter** — each skill declares its runtime. Rejected: runtime is an infrastructure choice (what compute is available), not a skill property. The prioritize agent is better positioned to make this decision based on task characteristics, cost, and available backends.
2. **Single RemoteDispatcher** — one dispatcher that handles both streaming and polling. Rejected: the event production patterns are fundamentally different (continuous stream vs periodic poll). Merging them creates a muddled abstraction.
3. **Chosen: two dispatcher types with backend interfaces** — clean separation. StreamingDispatcher handles the io.Reader→event loop. PollingDispatcher handles the poll→synthesize loop. Backends only implement what's natural for their API. TmuxBackend implements StreamingBackend alongside SpritesBackend — all streaming backends share one dispatcher.

## Applicable skills

- `go-best-practices` — non-blocking fanout, ordered shutdown, concurrency testing
- `bubbletea-tui` — for the remote session indicator badge

## Phases

- [[plans/27-remote-dispatchers/phase-01-backend-interfaces-and-shared-types]]
- [[plans/27-remote-dispatchers/phase-02-dispatcher-factory-and-runtime-routing]]
- [[plans/27-remote-dispatchers/phase-03-streamingdispatcher-generic-implementation]]
- [[plans/27-remote-dispatchers/phase-04-pollingdispatcher-generic-implementation]]
- [[plans/27-remote-dispatchers/phase-05-spritesbackend-implementation]]
- [[plans/27-remote-dispatchers/phase-06-cursorbackend-implementation]]
- [[plans/27-remote-dispatchers/phase-07-monitor-observer-abstraction]]
- [[plans/27-remote-dispatchers/phase-08-config-and-provider-validation]]
- [[plans/27-remote-dispatchers/phase-09-tui-remote-session-indicator]]
- [[plans/27-remote-dispatchers/phase-10-minimal-sync-back-for-remote-runtimes]]
- [[plans/27-remote-dispatchers/phase-11-integration-wiring-and-end-to-end-test]]

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

Manual: configure `.noodle.toml` with `[runtime.sprites]` block, run prioritize, verify queue items get `runtime` field, dispatch via `noodle start --once`, verify events appear in TUI with remote badge.
