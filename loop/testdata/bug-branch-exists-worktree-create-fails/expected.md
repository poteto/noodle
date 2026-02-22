---
schema_version: 1
expected_failure: true
bug: true
regression: bug-branch-exists-worktree-create-fails
source_hash: fe4c05f51271e4cb130408e03817b2228ec381e688f401146e4cee7117ae703c
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
