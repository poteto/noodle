# Opus Worker Permissions

## Problem

When spawning `general-purpose` Opus workers for code changes via the `Task` tool, they MUST have `mode: "bypassPermissions"` (or `mode: "acceptEdits"`). Without this, hooks block Write/Edit/Bash tool calls because there is no interactive terminal for the user to approve permissions.

The worker silently stalls or errors on its first tool call that requires approval.

## Fix

Always include `mode: "bypassPermissions"` when spawning Opus workers that need to make code changes:

```
Task(
  subagent_type: "general-purpose",
  model: "opus",
  mode: "bypassPermissions",
  run_in_background: true,
  prompt: "..."
)
```

The `codex-worker` template already specifies `mode: "bypassPermissions"` — this was only missing from the Opus worker path.

## When `acceptEdits` Instead

Use `mode: "acceptEdits"` if you want the worker to auto-accept edit tool calls but still require approval for potentially destructive Bash commands. For most worker tasks, `bypassPermissions` is simpler and correct.

See also [[delegation]], [[principles/boundary-discipline]], [[principles/encode-lessons-in-structure]]
