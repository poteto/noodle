---
schema_version: 1
expected_failure: false
bug: false
source_hash: 4bd0f01eba43ebbec624c7f29bb602498ea2db242f7757fae063d138bf8f6d15
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
