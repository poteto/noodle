---
schema_version: 1
expected_failure: false
bug: false
source_hash: 89ae37b3d56dc9b53063933c86c4466b506ee306add7f3c33c556505f17e4aa0
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
