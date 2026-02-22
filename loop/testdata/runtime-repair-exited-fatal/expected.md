---
schema_version: 1
expected_failure: false
bug: false
regression: runtime-repair-exited-fatal
source_hash: 28e460a81fa66762f44ec53baba1e9d34c3c2a752bed9cc892cdcd3e0f50b3d8
---

## Expected

```json
{
  "states": {
    "state-01": {
      "error": {
        "absent": true
      },
      "transition": "paused"
    },
    "state-02": {
      "error": {
        "contains": "exited before completion"
      },
      "transition": "paused",
      "actions": {
        "normal_task_scheduled": false,
        "repair_task_scheduled": true
      },
      "state": {
        "paused": true,
        "runtime_repair_in_flight": false
      },
      "counts": {
        "created_worktrees": {
          "eq": 1
        },
        "normal_spawn_calls": {
          "eq": 0
        },
        "runtime_repair_spawn_calls": {
          "eq": 1
        },
        "spawn_calls": {
          "eq": 1
        }
      }
    }
  }
}
```
