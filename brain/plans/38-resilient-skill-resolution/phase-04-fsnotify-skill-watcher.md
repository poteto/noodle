Back to [[plans/38-resilient-skill-resolution/overview]]

# Phase 4: fsnotify skill watcher with debounce

**Routing:** `claude` / `claude-opus-4-6` — goroutine lifecycle, channel patterns, shutdown coordination

## Goal

Watch configured skill paths for filesystem changes. When skills are added, modified, or deleted, trigger a registry rebuild. Uses the same debounce pattern as `monitor/` (200ms).

## Changes

**New file: `skill/watcher.go`**
- `SkillWatcher` struct with fsnotify watcher, debounce timer, callback, tracked directory set
- `NewSkillWatcher(paths []string, onChange func()) (*SkillWatcher, error)` — creates watcher. For each path: if directory exists, add watch and scan for subdirectories (each skill is a subdirectory). If path doesn't exist yet (bare repo), watch the nearest existing parent directory for Create events so that when e.g. `.agents/skills/` is created mid-run, the watcher picks it up and adds a proper watch.
- `Run(ctx context.Context)` — event loop goroutine:
  - Reads fsnotify events
  - Filters out Chmod-only events (noise from editors)
  - On Create events for directories: add new watch (same pattern as `monitor/maybeAddSessionDirWatch`). If the created directory matches a configured skill path that was absent at startup, add recursive watches for its contents.
  - On Remove/Rename of directories: remove stale watch from tracked set
  - Debounce via `time.Timer`: on any qualifying event, reset timer to 200ms. When timer fires, call `onChange`. This coalesces rapid-fire events (editor atomic writes: create temp → write → rename) into a single callback.
- `Close()` — shuts down watcher and goroutine

**New method: `loop/loop.go` — `rebuildRegistry()`**
- Re-runs `discoverRegistry()`, replaces `l.registry`, clears `l.registryErr`
- Defined here (not in phase 5) because the watcher integration needs it. Phase 5 adds queue audit on top.

**Integrate into loop:**
- **Important:** Do NOT create the watcher in `New()`. `New()` is used broadly in tests and by `--once` mode, which returns after `Cycle()` without calling `Shutdown()`. Creating the watcher there leaks fd/goroutines.
- Instead, create and start the watcher in `Run()` (the long-running loop entry point), alongside where the monitor goroutine is started. This is the correct lifecycle boundary — `Run()` owns the goroutine, `Run()` cleans it up on context cancellation.
- `loop/loop.go` — in `Run()`: create `SkillWatcher`, if creation fails log warning to stderr and continue without it. If creation succeeds, start `go watcher.Run(ctx)` and defer `watcher.Close()`. The `onChange` callback sets `l.registryStale`.
- `loop/loop.go` — in `Cycle()`, if `registryStale` is true, call `rebuildRegistry()` and reset the flag. Registry rebuild happens synchronously at cycle start, not in the fsnotify goroutine (avoids concurrent registry access).
- For `--once` mode and tests: no watcher is created (they call `Cycle()` directly, not `Run()`). The dispatch-time re-scan (phase 6) handles staleness in these paths.

## Data structures

- `SkillWatcher` — new struct in `skill/watcher.go`
- `Loop.registryStale atomic.Bool` — flag set by watcher callback, read by cycle
- `rebuildRegistry()` — new method on Loop

## Verification

```bash
go test ./skill/... && go test ./loop/... && go vet ./...
```

Unit tests in `skill/watcher_test.go`:
- Create temp directory, start watcher, add a file, verify callback fires exactly once within debounce window
- Rapid-fire 5 events within 100ms, verify callback fires once after debounce settles
- Chmod-only event does not trigger callback
- Create new subdirectory, verify it gets watched (add file inside, callback fires)
- Remove watched subdirectory, verify no error/panic
- Context cancellation shuts down cleanly with no goroutine leak
- Constructor with non-existent path: watcher watches parent; creating the path triggers callback
- `rebuildRegistry()`: verify registry is replaced and `registryStale` is reset
- Verify `New()` does NOT create a watcher (no fd leak in tests/--once)
