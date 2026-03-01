---
schema_version: 1
expected_failure: false
bug: false
source_hash: cde487f299e9a172cbc815aa201447e524c1f4a09aac78a609e23f96cc2d0645
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
