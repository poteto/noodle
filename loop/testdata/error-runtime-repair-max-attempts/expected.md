---
schema_version: 1
expected_failure: false
bug: false
regression: error-runtime-repair-max-attempts
source_hash: a3d444fe72ad845d65cba9b8b989aaf9831700f2eeb13b526c42417fc740416c
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
        "absent": true
      },
      "transition": "paused"
    },
    "state-03": {
      "error": {
        "absent": true
      },
      "transition": "paused"
    },
    "state-04": {
      "error": {
        "contains": "runtime issue unresolved after 3 repair attempt(s)"
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
          "eq": 3
        },
        "normal_spawn_calls": {
          "eq": 0
        },
        "runtime_repair_spawn_calls": {
          "eq": 3
        },
        "spawn_calls": {
          "eq": 3
        }
      }
    }
  }
}
```
