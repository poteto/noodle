---
schema_version: 1
expected_failure: false
bug: false
regression: runtime-repair-adopt-running-session
source_hash: 84321288798e53f891cca164a5661c49aa124980a96532f2adfe27ef1ecea378
---

## Expected

```json
{
  "states": {
    "state-01": {
      "error": {
        "absent": true
      },
      "transition": "paused",
      "actions": {
        "normal_task_scheduled": false,
        "repair_task_scheduled": false
      },
      "state": {
        "paused": true,
        "runtime_repair_in_flight": true
      },
      "counts": {
        "created_worktrees": {
          "eq": 0
        },
        "normal_spawn_calls": {
          "eq": 0
        },
        "runtime_repair_spawn_calls": {
          "eq": 0
        },
        "spawn_calls": {
          "eq": 0
        }
      }
    }
  }
}
```
