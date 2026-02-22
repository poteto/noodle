# Loop Fixture: Missing Sync + Existing Queue Work

## Setup
```json
{
  "queue_items": [
    {
      "id": "42",
      "provider": "claude",
      "model": "claude-sonnet-4-6"
    }
  ],
  "mise_results": [
    {
      "warnings": [
        "backlog sync script missing; returning empty backlog"
      ]
    }
  ]
}
```

## Expected
```json
{
  "actions": {
    "repair_task_scheduled": false,
    "normal_task_scheduled": true
  },
  "state": {
    "runtime_repair_in_flight": false,
    "running": true
  },
  "transitions": [
    "running"
  ],
  "counts": {
    "spawn_calls": { "eq": 1 },
    "runtime_repair_spawn_calls": { "eq": 0 },
    "normal_spawn_calls": { "eq": 1 },
    "created_worktrees": { "eq": 1 }
  },
  "routing": {
    "first_spawn_name": { "equals": "42" },
    "first_spawn_provider": { "equals": "claude" },
    "first_spawn_model": { "equals": "claude-sonnet-4-6" }
  }
}
```

## Expected Error

```json
{
  "absent": true
}
```
