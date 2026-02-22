---
schema_version: 1
expected_failure: true
bug: false
regression: runtime-repair-spawn-fatal
source_hash: 3035d54c29ff47d0ecef94eed5646a7501986484edc63e9a80bc300cb801685d
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
