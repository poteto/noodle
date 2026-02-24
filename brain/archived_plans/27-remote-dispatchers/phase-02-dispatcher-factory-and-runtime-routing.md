Back to [[archived_plans/27-remote-dispatchers/overview]]

# Phase 2: Dispatcher factory and runtime routing

**Routing:** `claude` / `claude-opus-4-6` — architectural refactor, judgment on decomposition

## Goal

Decompose `TmuxDispatcher` into `TmuxBackend` (implements `StreamingBackend`) + generic `StreamingDispatcher`. Introduce `DispatcherFactory` that routes based on the `runtime` field from the queue item.

Runtime comes from the queue item (set by the prioritize agent), not from skill frontmatter. The loop reads `item.Runtime` and passes it through to `DispatchRequest.Runtime`.

## Data structures

- `TmuxBackend` struct — extracted from `TmuxDispatcher`. Holds tmux-specific logic: building the command pipeline, launching `tmux new-session`, checking `tmux has-session`, killing sessions.
- `DispatcherFactory` struct — holds a map of runtime kind → `StreamingBackend` or `PollingBackend`. Routes based on `DispatchRequest.Runtime`.

## Changes

**`dispatcher/tmux_backend.go` (new)**
Extract tmux-specific logic from `TmuxDispatcher` into `TmuxBackend` implementing `StreamingBackend`:
- `Start`: receives a prebuilt command pipeline from the dispatcher, wraps it in `tmux new-session -d`, returns a `StreamHandle` with stdout pipe. Command assembly stays in the dispatcher (Phase 3), not the backend.
- `IsAlive`: runs `tmux has-session`
- `Kill`: runs `tmux kill-session`

**`dispatcher/factory.go` (new)**
`DispatcherFactory` implements `Dispatcher`. Only registers backends that are fully implemented and configured. Routes: `"tmux"` or `""` → `StreamingDispatcher` with `TmuxBackend`, `"sprites"` → `StreamingDispatcher` with `SpritesBackend` (when configured). Cursor stub is **not** registered in the factory — requesting `"cursor"` returns "runtime not configured" error. Backends are registered at bootstrap time based on `Config.AvailableRuntimes()`.

**`dispatcher/types.go`**
Add `Runtime` field documentation: set by prioritize agent via queue item, not skill frontmatter.

**`internal/queuex/queue.go`**
Add `Runtime string \`json:"runtime,omitempty"\`` to `queuex.Item`. This is the canonical serialization layer — runtime must be here or it gets silently dropped during JSON round-trips.

**`loop/queue.go`**
Add `Runtime` to the `toQueueX`/`fromQueueX` conversion functions. Without this, runtime is silently lost when the loop reads/writes queue.json.

**`loop/types.go`**
Add `Runtime string` field to `loop.QueueItem`.

**`loop/cook.go`**
Change line 54: `Runtime: taskType.Runtime` → `Runtime: nonEmpty(item.Runtime, "tmux")`. Runtime now comes from the queue item with "tmux" as default.

**`internal/queuex/validate.go` (or wherever schema validation lives)**
Accept `runtime` as a valid queue item field. Add to JSON schema output from `noodle schema queue`.

**Remove `runtime` from skill/task registry path:**
- `skill/frontmatter.go`: remove `Runtime` from `NoodleMeta`
- `internal/taskreg/registry.go`: remove `Runtime` from `TaskType`

## Verification

### Static
- `go build ./...` compiles
- All existing tests pass (queue items without `runtime` field default to "tmux")

### Runtime
- Unit test: factory routes "" and "tmux" to streaming dispatcher with TmuxBackend
- Unit test: factory returns error for unconfigured runtime kind
- Unit test: QueueItem with `runtime: "sprites"` flows through to DispatchRequest
- Migrate existing `tmux_dispatcher_test.go` and `tmux_command_test.go` tests to cover `TmuxBackend` — Start builds the correct command pipeline, IsAlive checks `tmux has-session`, Kill calls `tmux kill-session`
- `var _ StreamingBackend = (*TmuxBackend)(nil)` compile-time check
