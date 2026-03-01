---
schema_version: 1
expected_failure: false
bug: false
source_hash: ab2c7c17d75bb2121dbbf63d3b2be8aa97c5e931b76b5059b71be5f82b585615
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
