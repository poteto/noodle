Back to [[plans/31-structured-loop-logging/overview]]

# Phase 3: Log dispatch and completion events

## Goal

Add log lines for when the loop spawns agents and when they finish. These are the highest-value events for observability — an operator can see what's cooking and what completed.

## Changes

### Log dispatch in spawnCook (`cook.go`)

After successful dispatch (after `l.deps.Dispatcher.Dispatch` returns without error and the `activeCook` is stored), log:

```
l.logger.Info("cook dispatched", "item", item.ID, "session", session.ID(), "worktree", name, "attempt", attempt)
```

### Log dispatch in spawnSchedule (`schedule.go`)

After successful dispatch, log:

```
l.logger.Info("schedule dispatched", "session", session.ID(), "attempt", attempt)
```

### Log completion in handleCompletion (`cook.go`)

At the top of `handleCompletion`, log the outcome. The function branches on success vs. failure — log at each branch entry:

- Success + schedule: `l.logger.Info("schedule completed", "session", cook.session.ID())`
- Success + park for review: `l.logger.Info("cook parked for review", "item", cook.queueItem.ID, "session", cook.session.ID())`
- Success + merge: `l.logger.Info("cook completing", "item", cook.queueItem.ID, "session", cook.session.ID())` — logged at the branch entry, before `mergeCook` is called. The actual success is confirmed by the "cook merged" log inside `mergeCook`. Do NOT log "cook completed" here since `mergeCook` can still fail after this point
- Failure (falls through to `retryCook`): logged in phase 5

### Log merge result in mergeCook (`cook.go`)

After successful merge, log:

```
l.logger.Info("cook merged", "item", item.ID, "worktree", worktreeName)
```

### Log adopted completions in collectAdoptedCompletions (`cook.go`)

When an adopted session reaches a terminal status, log:

```
l.logger.Info("adopted session completed", "item", targetID, "session", sessionID, "status", status)
```

When an adopted session is dropped (not processable), log:

```
l.logger.Info("adopted session dropped", "item", targetID, "session", sessionID)
```

## Data structures

No new types.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Adding log lines at known call sites |

## Verification

### Static
```bash
go test ./... && go vet ./...
```

### Runtime
- Dispatch a cook item → verify "cook dispatched" in log with item ID, session ID, worktree name
- Complete a cook → verify "cook completing" then "cook merged", or "cook parked for review"
- Dispatch a schedule item → verify "schedule dispatched"
- Complete a schedule → verify "schedule completed"
