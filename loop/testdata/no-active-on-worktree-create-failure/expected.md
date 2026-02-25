---
schema_version: 1
expected_failure: false
bug: false
source_hash: a19b2031334b5a1ca67b6eea42e8e50e02fb21756b5a9d1d49a406687383e851
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
        "name": "prioritize",
        "skill": "prioritize",
        "provider": "codex",
        "model": "gpt-5.3-codex"
      }
    }
  }
}
```
