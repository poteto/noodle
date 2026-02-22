---
schema_version: 1
expected_failure: false
bug: false
regression: bug-branch-exists-worktree-create-fails
source_hash: 1234c05cbb1dd1a9e6e378442ce94b61614f778254ccc499f2b895b5ac20aa6d
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
