Back to [[plans/31-structured-loop-logging/overview]]

# Phase 5: Log retry, failure, and repair events

## Goal

Add log lines for the error-handling paths: retry, permanent failure, merge conflicts, and the runtime repair lifecycle. These are the most important events for diagnosing why work stopped or why an agent was spawned to fix things.

## Changes

### Log retry in retryCook (`cook.go`)

The `fmt.Fprintf` was removed (not replaced) in phase 2. Add structured log lines for both the retry and permanent-failure branches:

```
l.logger.Warn("cook retrying", "item", cook.queueItem.ID, "session", cook.session.ID(), "attempt", nextAttempt, "reason", resolvedReason)
```

When retries are exhausted and the item is marked failed:

```
l.logger.Error("cook failed permanently", "item", cook.queueItem.ID, "reason", resolvedReason, "attempts", nextAttempt)
```

### Log markFailed (`failures.go`)

In `markFailed`, after writing to disk:

```
l.logger.Warn("item marked failed", "item", id, "reason", reason)
```

### Log merge conflicts (`cook.go`)

In `handleMergeConflict`, when a conflict is detected and the item is marked failed:

```
l.logger.Warn("merge conflict", "item", cook.queueItem.ID, "error", err.Error())
```

### Log runtime repair lifecycle (`runtime_repair.go`)

**Issue detected** — in `handleRuntimeIssue`, before calling `ensureRuntimeRepair`:

```
l.logger.Warn("runtime issue detected", "scope", scope, "error", issueMessage(err, warnings))
```

**Repair spawned** — in `ensureRuntimeRepair`, after successful dispatch:

```
l.logger.Info("runtime repair spawned", "session", session.ID(), "scope", issue.Scope, "attempt", attempt)
```

**Repair adopted** — in `adoptRunningRuntimeRepair`, when an existing repair session is found:

```
l.logger.Info("runtime repair adopted", "session", sessionID, "scope", issue.Scope)
```

**Repair completed** — in `advanceRuntimeRepair`, when status is "completed":

```
l.logger.Info("runtime repair completed", "session", inFlight.SessionID, "scope", inFlight.Issue.Scope)
```

**Repair exited (non-completion)** — in `advanceRuntimeRepair`, when status is "exited":

```
l.logger.Error("runtime repair exited", "session", inFlight.SessionID, "scope", inFlight.Issue.Scope)
```

**Max attempts exceeded** — in `ensureRuntimeRepair`, when `attempt > maxRuntimeRepairAttempts`:

```
l.logger.Error("runtime repair exhausted", "scope", issue.Scope, "attempts", maxRuntimeRepairAttempts)
```

## Data structures

No new types.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `codex` | `gpt-5.3-codex` | Adding log lines at known error-handling call sites |

## Verification

### Static
```bash
go test ./... && go vet ./...
```

### Runtime
- Trigger a cook failure with retries remaining → verify "cook retrying" with attempt number
- Exhaust retries → verify "cook failed permanently"
- Trigger a merge conflict → verify "merge conflict" logged
- Trigger a runtime issue → verify "runtime issue detected" and "runtime repair spawned"
- Let a repair complete → verify "runtime repair completed"
- Exhaust repair attempts → verify "runtime repair exhausted"
