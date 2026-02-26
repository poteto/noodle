---
schema_version: 1
expected_failure: false
bug: false
source_hash: 0d745d2727716062da960818f3d90babbcaaf2a1299161eb69ddbd095ef82662
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
      "created_worktrees": 1,
      "first_spawn": {
        "name": "42-0-execute",
        "provider": "codex",
        "model": "gpt-5.3-codex"
      }
    }
  }
}
```
