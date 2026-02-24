---
schema_version: 1
expected_failure: false
bug: false
source_hash: 321b4f97442976039320961132a5cd946c2c55491b3779a64f771e05dc331b16
---

## Runtime Dump

```json
{
  "states": {
    "state-01": {
      "transition": "paused",
      "runtime_repair_in_flight": true,
      "repair_task_scheduled": true,
      "oops_task_scheduled": true,
      "normal_task_scheduled": false,
      "spawn_calls": 1,
      "runtime_repair_spawn_calls": 1,
      "normal_spawn_calls": 0,
      "created_worktrees": 1,
      "runtime_repair_spawn": {
        "name": "repair-runtime-*",
        "skill": "oops",
        "provider": "claude",
        "model": "claude-opus-4-6"
      }
    }
  }
}
```
