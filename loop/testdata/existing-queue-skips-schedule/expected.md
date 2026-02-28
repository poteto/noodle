---
schema_version: 1
expected_failure: false
bug: false
source_hash: 49d2a737cc2e4be48bdadb2a90affa86c99c5ba8bb79ceea6894d394aa445bba
---

## Runtime Dump

```json
{
  "states": {
    "state-01": {
      "transition": "running",
      "normal_task_scheduled": true,
      "spawn_calls": 2,
      "normal_spawn_calls": 2,
      "created_worktrees": 2,
      "first_spawn": {
        "name": "42-0-execute",
        "provider": "codex",
        "model": "gpt-5.3-codex"
      }
    }
  }
}
```
