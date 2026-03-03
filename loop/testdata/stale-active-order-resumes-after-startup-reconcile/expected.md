---
schema_version: 1
expected_failure: false
bug: false
source_hash: d9bdd8cbe1a14d8a0606551517844b249c85bb7c94ca806fa9e5c6f0e4d8b2d1
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
        "model": "gpt-5.3-codex"
      }
    }
  }
}
```
