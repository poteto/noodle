---
schema_version: 1
expected_failure: false
bug: false
regression: bug-routing-stale-queue-config
---

## Expected

```json
{
  "states": {
    "state-01": {
      "error": {
        "absent": true
      },
      "transition": "running",
      "actions": {
        "normal_task_scheduled": true,
        "repair_task_scheduled": false
      },
      "state": {
        "running": true,
        "runtime_repair_in_flight": false
      },
      "counts": {
        "normal_spawn_calls": {
          "eq": 1
        },
        "runtime_repair_spawn_calls": {
          "eq": 0
        },
        "spawn_calls": {
          "eq": 1
        }
      },
      "routing": {
        "first_spawn_name": {
          "equals": "10"
        },
        "first_spawn_provider": {
          "equals": "codex"
        },
        "first_spawn_model": {
          "equals": "gpt-5.3-codex"
        }
      }
    }
  }
}
```
