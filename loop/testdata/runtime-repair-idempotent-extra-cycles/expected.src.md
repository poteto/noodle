---
schema_version: 1
expected_failure: false
bug: false
regression: runtime-repair-idempotent-extra-cycles
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
      "transition": "paused"
    },
    "state-03": {
      "transition": "paused",
      "actions": {
        "normal_task_scheduled": false,
        "repair_task_scheduled": true
      },
      "state": {
        "paused": true,
        "runtime_repair_in_flight": true
      },
      "counts": {
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
      "absence": {
        "normal_task_scheduled": true
      },
      "routing": {
        "runtime_repair_name": {
          "prefix": "repair-runtime-"
        },
        "runtime_repair_prompt": {
          "contains": "Scope: mise.sync"
        },
        "runtime_repair_skill": {
          "equals": "debugging"
        }
      },
      "idempotence": {
        "no_duplicate_runtime_repairs_on_extra_cycles": true,
        "no_new_spawns_on_extra_cycles": true
      }
    }
  }
}
```

