# Loop Fixture: Adopted Running Repair Does Not Duplicate After Restart

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
  "running_runtime_repair_session_id": "repair-runtime-20260222-203500-1-abc123",
  "extra_cycles": 1
}
```

## Expected
```json
{
  "actions": {
    "repair_task_scheduled": false,
    "normal_task_scheduled": false
  },
  "state": {
    "runtime_repair_in_flight": true,
    "paused": true
  },
  "transitions": [
    "paused",
    "paused"
  ],
  "counts": {
    "spawn_calls": { "eq": 0 },
    "runtime_repair_spawn_calls": { "eq": 0 },
    "normal_spawn_calls": { "eq": 0 },
    "created_worktrees": { "eq": 0 }
  },
  "absence": {
    "repair_task_scheduled": true,
    "normal_task_scheduled": true
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
