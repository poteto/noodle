---
schema_version: 1
expected_failure: true
bug: true
regression: runtime-repair-spawn-fatal
source_hash: 8365ffe64953a40d7428aae563f23e3b91ca3aadda9d6ec0f65bf25e06201332
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
