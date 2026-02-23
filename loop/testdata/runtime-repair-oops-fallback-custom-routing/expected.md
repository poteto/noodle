---
schema_version: 1
expected_failure: false
bug: false
source_hash: 29c6e58dad96311e25883fb9716c5e61c1426eb6ad9aaa5744d70b9d29f59255
---

## Runtime Dump

```json
{
  "states": {
    "state-01": {
      "transition": "paused",
      "runtime_repair_in_flight": true,
      "repair_task_scheduled": true,
      "oops_task_scheduled": true,
      "normal_task_scheduled": false,
      "spawn_calls": 1,
      "runtime_repair_spawn_calls": 1,
      "normal_spawn_calls": 0,
      "created_worktrees": 1,
      "runtime_repair_spawn": {
        "name": "repair-runtime-*",
        "skill": "oops",
        "provider": "codex",
        "model": "gpt-5.3-codex"
      }
    }
  }
}
```
