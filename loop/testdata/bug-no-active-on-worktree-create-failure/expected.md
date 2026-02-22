---
schema_version: 1
expected_failure: true
bug: true
regression: bug-no-active-on-worktree-create-failure
source_hash: 96baebab448312c2db9155680fd031ef4840425c1a426ef1b4ba790ef287d9c2
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
