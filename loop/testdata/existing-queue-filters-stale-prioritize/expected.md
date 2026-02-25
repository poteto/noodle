---
schema_version: 1
expected_failure: false
bug: false
source_hash: ea06ae777faaebeb1e5d6279467afad21fe27104f90cacd2bbcc703b5b902117
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
