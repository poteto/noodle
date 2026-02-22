# Loop Fixture: Runtime Repair Exits Before Completion

## Setup
```json
{
  "queue_items": [],
  "mise_results": [
    {
      "warnings": [
        "backlog sync script missing; returning empty backlog"
      ]
    },
    {
      "warnings": [
        "backlog sync script missing; returning empty backlog"
      ]
    }
  ],
  "complete_runtime_repair_session_with_status": "exited"
}
```

## Expected
```json
{
  "spawn_calls": 1,
  "first_spawn_name_prefix": "repair-runtime-",
  "created_worktrees": 1,
  "runtime_repair_in_flight": false,
  "second_cycle_error_contains": "exited before completion"
}
```
