# Loop Fixture: In-Flight Runtime Repair Suppresses Queued Work

## Setup
```json
{
  "queue_items": [
    {
      "id": "42",
      "provider": "codex",
      "model": "gpt-5.3-codex"
    }
  ],
  "mise_results": [
    {
      "error": "planner state read failed"
    }
  ],
  "extra_cycles": 2
}
```

## Expected
```json
{
  "actions": {
    "repair_task_scheduled": true,
    "normal_task_scheduled": false
  },
  "state": {
    "runtime_repair_in_flight": true,
    "paused": true
  },
  "transitions": [
    "paused",
    "paused",
    "paused"
  ],
  "counts": {
    "spawn_calls": { "eq": 1 },
    "runtime_repair_spawn_calls": { "eq": 1 },
    "normal_spawn_calls": { "eq": 0 },
    "created_worktrees": { "eq": 1 }
  },
  "idempotence": {
    "no_new_spawns_on_extra_cycles": true,
    "no_duplicate_runtime_repairs_on_extra_cycles": true
  }
}
```

## Expected Error

```json
{
  "absent": true
}
```
