---
schema_version: 1
expected_failure: false
bug: false
regression: bug-branch-exists-worktree-create-fails
source_hash: 00120439dae410850aebf79df651b47ce425cd5c1e359e502d9b8bbdabf141a1
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
