---
schema_version: 1
expected_failure: false
bug: false
source_hash: d265bfb10878b1efc9e2511e3eec4c72f983f68b9f42d46d504287c8f12a79bd
---

## Runtime Dump

```json
{
  "states": {
    "state-01": {
      "transition": "running",
      "runtime_repair_in_flight": false,
      "repair_task_scheduled": false,
      "oops_task_scheduled": false,
      "normal_task_scheduled": true,
      "spawn_calls": 1,
      "runtime_repair_spawn_calls": 0,
      "normal_spawn_calls": 1,
      "created_worktrees": 1,
      "first_spawn": {
        "name": "42",
        "provider": "claude",
        "model": "claude-sonnet-4-6"
      }
    }
  }
}
```
