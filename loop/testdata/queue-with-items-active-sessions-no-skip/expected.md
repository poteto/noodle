---
schema_version: 1
expected_failure: false
bug: false
source_hash: 3814b1ed5d17b6c976f80bd3f2f61df5a9fb2b3f28450a09400519e9ad334073
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
        "name": "42-task-42",
        "skill": "execute",
        "provider": "codex",
        "model": "gpt-5.3-codex"
      }
    }
  }
}
```
