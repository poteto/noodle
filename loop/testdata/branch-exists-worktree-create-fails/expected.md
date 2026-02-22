---
schema_version: 1
expected_failure: false
bug: false
regression: branch-exists-worktree-create-fails
source_hash: 00120439dae410850aebf79df651b47ce425cd5c1e359e502d9b8bbdabf141a1
---

## Runtime Dump

```json
{
  "states": {
    "state-01": {
      "transition": "paused",
      "runtime_repair_in_flight": true,
      "repair_task_scheduled": true,
      "oops_task_scheduled": false,
      "normal_task_scheduled": false,
      "spawn_calls": 1,
      "runtime_repair_spawn_calls": 1,
      "normal_spawn_calls": 0,
      "created_worktrees": 2,
      "runtime_repair_spawn": {
        "name": "repair-runtime-*",
        "skill": "debugging",
        "provider": "codex",
        "model": "gpt-5.3-codex"
      }
    }
  }
}
```
