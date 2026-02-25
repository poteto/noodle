---
schema_version: 1
expected_failure: false
bug: false
source_hash: f6538f767e4494c5316293434e73a6249fbf0dd345f17cdff87c9957ae74a50c
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
        "name": "42-task-42",
        "skill": "execute",
        "provider": "codex",
        "model": "gpt-5.3-codex"
      }
    }
  }
}
```
