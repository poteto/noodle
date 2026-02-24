Back to [[plans/31-structured-loop-logging/overview]]

# Phase 6: Tests

## Goal

Verify that key lifecycle events produce the expected log output. Use a capturing slog handler injected via `Dependencies.Logger` to assert log lines in existing and new tests without parsing stderr.

## Changes

### Test helper: capturing handler (`loop/loop_test.go`)

Add a test helper that creates a `*slog.Logger` backed by a handler that records log records in memory. The handler stores records in a `[]slog.Record` slice (or a simpler `[]logEntry` struct with level, message, and attrs map). This is a test-only type — define it in the test file.

```
type logEntry struct {
    Level   slog.Level
    Message string
    Attrs   map[string]any
}
```

The handler collects entries into a `[]logEntry` that tests can assert against. Use `slog.NewHandler` with a custom `Handle()` method, or use a simple `slog.Handler` implementation.

### Update existing test infrastructure (`loop/fixture_test.go` or `loop/loop_test.go`)

The existing test helpers construct `Loop` instances with fake deps. Update them to inject a capturing logger by default, so all existing tests continue to pass (they currently get `slog.Default()` or the new stderr default — both are fine, but capturing is better for test isolation).

### Test: dispatch logging

Use an existing dispatch test (or add a minimal one). After a cycle that dispatches a cook, assert that the captured log contains an entry with message "cook dispatched" and the expected item ID and session ID.

### Test: completion logging

After marking a session as done and running a cycle, assert "cook completed" or "cook parked for review" appears in the log.

### Test: state transition logging

Send a pause control command and run a cycle. Assert the log contains "state changed" with `from=running` and `to=paused`. Send resume, assert the reverse transition.

### Test: retry and failure logging

Dispatch a cook, mark it as failed (non-completed status), run a cycle. Assert "cook retrying" appears. Exhaust retries and assert "cook failed permanently".

### Test: runtime repair logging

Trigger a runtime issue (e.g., inject an error from mise build). Assert "runtime issue detected" and "runtime repair spawned" appear. Mark the repair session as completed, run another cycle, assert "runtime repair completed".

### Test: setState no-op

Call `setState` with the loop's current state (e.g., call `setState(StateRunning)` when the loop is already in `StateRunning`). Assert that no "state changed" log entry is produced. This verifies the early-return guard in `setState`.

### Test: control command logging

Write a pause command to control.ndjson, run a cycle. Assert "control command" with `action=pause` appears.

### Test: queue mutation logging

Set up an empty queue with plans present. Run a cycle. Assert "queue empty, bootstrapping prioritize" appears.

## Data structures

- `logEntry` — test-only struct: `Level slog.Level`, `Message string`, `Attrs map[string]any`
- `capturingHandler` — test-only `slog.Handler` implementation that appends to `[]logEntry`. Must use a `sync.Mutex` to protect the `[]logEntry` slice, since `slog.Handler.Handle()` must be safe for concurrent use (required for `-race` safety)

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | Test design requires judgment about what to assert and how to structure the capturing handler |

## Verification

### Static
```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

### Runtime
- All existing tests pass unchanged (the capturing handler is a superset of the default behavior)
- New log-assertion tests pass
- No test produces stderr output from slog (all captured)
