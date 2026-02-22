# Observe Directly

**Principle:** When determining the state of a system, read the actual state directly rather than inferring it from secondary signals. Indirect indicators introduce ambiguity and false positives.

## Why

Indirect observation (file mtimes, output freshness, CWD matching, screenshot captures of cached compositor frames) feels cheaper than direct observation, but the cost of acting on a wrong inference dwarfs the cost of reading the real state. Every noodle bug and MCP verification failure in this project traced back to the same root cause: trusting a proxy instead of checking the source.

## The Pattern

- **Check process liveness directly** (PID, process table), not indirectly (tmux pane status, output freshness, file mtime).
- **Read the actual value**, not a cached or derived representation. Screenshots capture the OS compositor's cached frame, not a fresh render. `sessions` infers status from file metadata, not process state.
- **"It compiles," "it looks right," or "the file was created" is not verification.** Run it and exercise the actual feature path.
- **When verification fails, suspect the observation method** before suspecting the system.

## Evidence

- Noodash-cli bugs: three of four bugs shared the root cause — process liveness inferred from indirect signals (file mtime, CWD matching via lsof) instead of checked directly. Led to spawning a duplicate manager on a live worktree.
- MCP verification gotchas: `xcap` captures the OS compositor's cached frame, not a fresh render. Sequential screenshots can't verify incremental changes. Hit test results are unreliable.
- Director operational discipline: "Before spawning a replacement manager, verify the process is truly dead" — check monitor state, NDJSON timestamps, tmux pane process.

## Relationship to Other Principles

[[principles/verify-runtime]] says *verify your work*. This principle says *how* to verify: prefer direct observation over indirect inference. The dashboard-cli bugs happened because the system *did* verify — it just used indirect signals.

See also [[principles/verify-runtime]]
