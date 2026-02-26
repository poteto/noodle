---
schema_version: 1
expected_failure: false
bug: false
source_hash: 7e5a78f0e72ca8bc200bdce158b1bde179212fb8e76cc1391b07e3ef8bde4446
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
        "name": "42:0:execute",
        "provider": "codex",
        "model": "gpt-5.3-codex"
      }
    }
  }
}
```
