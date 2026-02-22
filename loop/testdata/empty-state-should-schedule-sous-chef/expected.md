---
schema_version: 1
expected_failure: true
bug: true
source_hash: 7c9f50281fb9c584febcd263884f7c0035f259ce237f31086231e8624c24cd99
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
      "created_worktrees": 1,
      "first_spawn": {
        "name": "sous-chef",
        "skill": "sous-chef",
        "provider": "codex",
        "model": "gpt-5.3-codex"
      }
    }
  }
}
```
