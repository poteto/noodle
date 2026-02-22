---
schema_version: 1
expected_failure: false
bug: false
regression: runtime-repair-oops-fallback-custom-routing
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
        "oops_task_scheduled": true,
        "repair_task_scheduled": true
      },
      "state": {
        "paused": true,
        "runtime_repair_in_flight": true
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
        "runtime_repair_model": {
          "equals": "gpt-5.3-codex"
        },
        "runtime_repair_name": {
          "prefix": "repair-runtime-"
        },
        "runtime_repair_provider": {
          "equals": "codex"
        },
        "runtime_repair_skill": {
          "equals": "oops"
        }
      }
    }
  }
}
```

