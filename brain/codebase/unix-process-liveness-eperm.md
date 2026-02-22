# Unix Process Liveness: EPERM Means Alive

## Problem

On Unix, `kill(pid, 0)` is the standard way to check if a process exists without sending a signal. It returns:

- **0** — process exists and we have permission to signal it
- **ESRCH** — process does not exist
- **EPERM** — process exists but we lack permission to signal it

If you only check for `err == nil` (success) as "alive", you will misclassify EPERM as "dead". This means live processes owned by other users (or with different privilege levels) get incorrectly treated as dead — leading to premature lock removal, resource cleanup, or other incorrect state transitions.

## Fix

In Go:

```go
err := syscall.Kill(pid, 0)
if err == nil || errors.Is(err, syscall.EPERM) {
    // Process is alive
} else {
    // Process is dead (ESRCH or other error)
}
```

## Where This Bit Us

The worktree merge lockfile uses PID-based stale lock detection. Without EPERM handling, a lock held by a process running as a different user would be incorrectly classified as stale and removed — allowing concurrent merges that the lockfile was designed to prevent.

See also [[codebase/worktree-gotchas]], [[principles/observe-directly]], [[principles/fix-root-causes]], [[delegation/codex-review-value]]
