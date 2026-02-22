# Loop Fixture: Runtime Repair Spawn Failure Is Fatal

## Setup
```json
{
  "queue_items": [],
  "mise_results": [
    {
      "error": "plans sync failed"
    }
  ],
  "spawner_error": "agent unavailable"
}
```

## Expected
```json
{
  "spawn_calls": 1,
  "first_spawn_name_prefix": "repair-runtime-",
  "created_worktrees": 1,
  "runtime_repair_in_flight": false,
  "first_cycle_error_contains": "runtime repair unavailable"
}
```
