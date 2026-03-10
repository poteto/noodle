Back to [[archive/plans/31-structured-loop-logging/overview]]

# Phase 4: Log queue mutations and state transitions

## Goal

Add log lines for queue changes and loop state transitions. These events explain why the loop's behavior changed — queue-next promotion, bootstrap schedule, item skip, normalization, and state changes between Running/Idle/Paused/Draining.

## Changes

### Log queue-next consumption (`loop.go` or `queue.go`)

In `prepareQueueForCycle`, after `consumeQueueNext` succeeds and actually promoted a file (not a no-op), log:

```
l.logger.Info("queue-next promoted")
```

Since `consumeQueueNext` returns nil both when the file doesn't exist and when it succeeds, add a boolean return value or check for the file before/after. The simplest approach: log inside `consumeQueueNext` is not possible (no logger access). Instead, check if the queue changed after the call by comparing state, or add a return value indicating whether promotion happened. The cleanest option is a `promoted bool` return: `consumeQueueNext(nextPath, queuePath string) (bool, error)`.

### Log queue normalization (`loop.go`)

In `prepareQueueForCycle`, when `normalizeAndValidateQueue` returns `changed == true`, log:

```
l.logger.Info("queue normalized")
```

### Log bootstrap schedule (`loop.go`)

In `prepareQueueForCycle`, when the queue is empty and a schedule bootstrap is created, log:

```
l.logger.Info("queue empty, bootstrapping schedule")
```

### Log skipQueueItem (`cook.go`)

In `skipQueueItem`, after successful write, log:

```
l.logger.Info("queue item skipped", "item", id)
```

### Log state transitions (`loop.go`, `control.go`)

State changes happen in multiple places. Add a helper method:

```
func (l *Loop) setState(next State) {
    if l.state == next {
        return
    }
    l.logger.Info("state changed", "from", string(l.state), "to", string(next))
    l.state = next
}
```

Replace all direct `l.state = ...` assignments with `l.setState(...)`, except the initial assignment in `New()`. Call sites:
- `loop.go:74` — initial `StateRunning` in `New()`. **Keep as direct assignment** (`state: StateRunning` in the struct literal). Do NOT convert to `setState()` — the zero value `""` would produce a noisy `from="" to="running"` log line. Add a comment: `// Direct assignment; setState() is not used here to avoid logging the initial state.`
- `loop.go:164` — `StateRunning` when plans dir changes (currently `StateIdle → StateRunning`)
- `loop.go:242` — `StateIdle → StateRunning` in `buildCycleBrief`
- `loop.go:277` — `StateIdle` when queue empty
- `control.go:145` — `StatePaused` from pause command
- `control.go:147` — `StateRunning` from resume command
- `control.go:149` — `StateDraining` from drain command

### Log control commands (`control.go`)

In `applyControlCommand`, after the switch statement resolves, log the command and outcome:

```
l.logger.Info("control command", "action", cmd.Action, "status", ack.Status)
```

For error cases, include the message:

```
l.logger.Warn("control command failed", "action", cmd.Action, "message", ack.Message)
```

## Data structures

- `consumeQueueNext` signature changes to return `(bool, error)` — the bool is `true` only when data was actually promoted to the queue file. It is `false` when the queue-next file does not exist (`os.IsNotExist` → `false, nil`) and `false` when the file exists but fails validation (file is removed but nothing is promoted → `false, nil` or `false, error`)
- New `setState(State)` method on `Loop`

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.4` | Mechanical log insertions and a small helper method |

## Verification

### Static
```bash
go test ./... && go vet ./...
```

### Runtime
- Write a queue-next.json, run a cycle → verify "queue-next promoted" in log
- Start with empty queue and plans present → verify "queue empty, bootstrapping schedule"
- Send a pause control command → verify "control command" and "state changed" logged
- Send a skip control command → verify "queue item skipped" logged
- Verify idle cycles produce no log output (no spam)
