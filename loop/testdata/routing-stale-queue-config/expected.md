---
schema_version: 1
expected_failure: false
bug: false
source_hash: f629b644bf7332ec54d843c07ed78dd98f5c7ec9d7fc806d673f9a278c26abd7
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
