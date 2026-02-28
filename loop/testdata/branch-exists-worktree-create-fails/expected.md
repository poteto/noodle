---
schema_version: 1
expected_failure: false
bug: false
source_hash: 72a272016b96a52f9f2a8922bac4f190a1a29b95c087129bf66384bc47401016
---

## Runtime Dump

```json
{
  "states": {
    "state-01": {
      "transition": "running",
      "normal_task_scheduled": true,
      "spawn_calls": 1,
      "normal_spawn_calls": 1,
      "created_worktrees": 0,
      "first_spawn": {
        "name": "schedule",
        "skill": "schedule",
        "provider": "codex",
        "model": "gpt-5.3-codex"
      }
    }
  }
}
```
