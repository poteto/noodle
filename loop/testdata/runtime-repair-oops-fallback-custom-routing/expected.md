---
schema_version: 1
expected_failure: false
bug: false
source_hash: b5d5bfa9f180c4708062e4c3541f2053eacc5df120c6c2707d1518af2e70e085
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
