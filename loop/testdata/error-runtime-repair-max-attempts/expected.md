---
schema_version: 1
expected_failure: false
bug: false
regression: error-runtime-repair-max-attempts
source_hash: 72f602b043f9aaaa70d22e3e2b9f8bcd2cdc1417a04199d30727048716ebe7cb
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
