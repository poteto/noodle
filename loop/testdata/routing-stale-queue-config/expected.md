---
schema_version: 1
expected_failure: false
bug: false
source_hash: ee591e74094e4001f51839ba57603e198fbfe09c310fd37918e82ed79be3c4c9
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
