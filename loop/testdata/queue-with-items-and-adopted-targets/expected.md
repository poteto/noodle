---
schema_version: 1
expected_failure: false
bug: false
source_hash: 11ddfb90277bd9b048d7bbb93d23e02d34e3b52e3542bb790bb00eec6fecd807
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
