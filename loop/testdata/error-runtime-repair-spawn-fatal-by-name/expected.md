---
schema_version: 1
expected_failure: true
bug: true
regression: error-runtime-repair-spawn-fatal-by-name
source_hash: 96629cce17fdc965fef5980f31c852547e903e662b6f615f0fe0610757fda2d6
---

## Expected

```json
{
  "states": {
    "state-01": {
      "error": {
        "contains": "runtime repair unavailable"
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
      },
      "routing": {
        "runtime_repair_name": {
          "prefix": "repair-runtime-"
        }
      }
    }
  }
}
```
