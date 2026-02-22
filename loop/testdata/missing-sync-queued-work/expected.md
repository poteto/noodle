---
schema_version: 1
expected_failure: false
bug: false
regression: missing-sync-queued-work
source_hash: d265bfb10878b1efc9e2511e3eec4c72f983f68b9f42d46d504287c8f12a79bd
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
        "created_worktrees": {
          "eq": 1
        },
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
        "first_spawn_model": {
          "equals": "claude-sonnet-4-6"
        },
        "first_spawn_name": {
          "equals": "42"
        },
        "first_spawn_provider": {
          "equals": "claude"
        }
      }
    }
  }
}
```
