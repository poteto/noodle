---
schema_version: 1
expected_failure: true
bug: false
regression: runtime-repair-spawn-fatal
source_hash: c8e4a8e5f5bf202b906de3f2cc932127819ad0960533e64e6e7b39424702e52c
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
