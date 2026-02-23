---
name: oops
description: Infrastructure fix for broken builds, failed tests, stale state, and environment drift. Covers both user-project and Noodle-internal failures.
noodle:
  blocking: false
  schedule: "When infrastructure failures are detected (build breaks, test failures, stale Noodle state)"
---

# Oops

Fix infrastructure failures. Applies to both user-project code and Noodle-internal state.

## Fix flow

**Reproduce -> Diagnose -> Fix -> Verify -> Commit**

### 1. Reproduce

Run the failing command. Capture exact error output. If intermittent, run multiple times to confirm.

### 2. Diagnose

**Suspect state before code.** Check in this order:

1. **Noodle state** — `.noodle/` files, queue consistency, config validity
2. **Environment** — Go version, missing deps, stale caches, tmux session health
3. **Config** — `.noodle.toml` (project root), `.agents/skills/` structure, `brain/` integrity
4. **Persistent files** — lock files, temp files, incomplete writes
5. **Code** — only after ruling out state and environment

### Principles

- [[fix-root-causes]]
- [[suspect-state-before-code]]
- [[observe-directly]]

Noodle-internal checklist:
- `.noodle/` state files: corrupt, stale, or inconsistent?
- Queue state: orphaned tasks, stuck sessions?
- Config: valid TOML, referenced paths exist?
- tmux sessions: zombie processes, detached sessions?

### 3. Fix

Trace to root cause. Never paper over symptoms.

After fixing, grep for the same pattern elsewhere — the bug may exist in multiple locations.

### 4. Verify

Run the original failing command. Confirm it passes. Run related tests to check for regressions.

### 5. Commit

Commit message describes the root cause, not symptoms.

```
# Good: fix(queue): stale lock file prevented task dequeue after crash
# Bad:  fix: tasks not running
```
