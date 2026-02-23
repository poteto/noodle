---
schema_version: 1
expected_failure: false
bug: false
source_hash: f217117c013350bc1f6f3f00268a4ebf088600de98106aa68b9d73bdc5f748b1
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
        "model": "claude-sonnet-4-6"
      }
    },
    "state-02": {
      "transition": "running",
      "runtime_repair_in_flight": false,
      "repair_task_scheduled": true,
      "oops_task_scheduled": true,
      "normal_task_scheduled": true,
      "spawn_calls": 2,
      "runtime_repair_spawn_calls": 1,
      "normal_spawn_calls": 1,
      "created_worktrees": 1,
      "first_spawn": {
        "name": "prioritize",
        "skill": "prioritize",
        "provider": "claude",
        "model": "claude-sonnet-4-6"
      },
      "runtime_repair_spawn": {
        "name": "repair-runtime-*",
        "skill": "oops",
        "provider": "claude",
        "model": "claude-sonnet-4-6"
      }
    }
  }
}
```
