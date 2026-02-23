---
schema_version: 1
expected_failure: false
bug: false
source_hash: 8d1622a9fdcdc657d20b9325587fdbd1a221b0fb4e5bac47f6d3195cfe7d7d9b
---

## Runtime Dump

```json
{
  "states": {
    "state-01": {
      "transition": "running",
      "runtime_repair_in_flight": false,
      "repair_task_scheduled": false,
      "oops_task_scheduled": false,
      "normal_task_scheduled": true,
      "spawn_calls": 1,
      "runtime_repair_spawn_calls": 0,
      "normal_spawn_calls": 1,
      "created_worktrees": 0,
      "first_spawn": {
        "name": "prioritize",
        "skill": "prioritize",
        "provider": "codex",
        "model": "gpt-5.3-codex"
      }
    }
  }
}
```
