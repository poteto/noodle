---
schema_version: 1
expected_failure: false
bug: false
regression: bug-no-active-on-worktree-create-failure
source_hash: cd3cac533b6fc0ed78e38ba81aeefb3faee03c5ad82f930a353bf58546b0cf47
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
      "routing": {
        "runtime_repair_name": {
          "prefix": "repair-runtime-"
        },
        "runtime_repair_provider": {
          "equals": "codex"
        },
        "runtime_repair_model": {
          "equals": "gpt-5.3-codex"
        }
      }
    }
  }
}
```
