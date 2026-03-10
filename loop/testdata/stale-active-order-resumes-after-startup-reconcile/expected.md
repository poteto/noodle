---
schema_version: 1
expected_failure: false
bug: false
source_hash: 569641aa4c368f9a419dd522528b69d0fcdd8d6cc24675fc5e6da394e0d4984d
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
      "created_worktrees": 1,
      "active_summary_total": 2,
      "first_spawn": {
        "name": "fix-uncommitted-ui-changes-0-oops",
        "skill": "oops",
        "provider": "codex",
        "model": "gpt-5.4"
      }
    }
  }
}
```
