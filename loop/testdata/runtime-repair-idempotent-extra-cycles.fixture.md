# Loop Fixture: Runtime Repair Is Idempotent Across Extra Cycles

## Setup
```json
{
  "queue_items": [],
  "mise_results": [
    {
      "warnings": [
        "backlog sync script missing; returning empty backlog"
      ]
    }
  ],
  "extra_cycles": 2
}
```

## Expected
```json
{
  "counts": {
    "runtime_repair_spawn_calls": { "eq": 1 },
    "spawn_calls": { "eq": 1 },
    "normal_spawn_calls": { "eq": 0 }
  },
  "actions": {
    "repair_task_scheduled": true,
    "normal_task_scheduled": false
  },
  "absence": {
    "normal_task_scheduled": true
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
  "idempotence": {
    "no_new_spawns_on_extra_cycles": true,
    "no_duplicate_runtime_repairs_on_extra_cycles": true
  },
  "routing": {
    "runtime_repair_skill": { "equals": "debugging" },
    "runtime_repair_name": { "prefix": "repair-runtime-" },
    "runtime_repair_prompt": { "contains": "Scope: mise.sync" }
  }
}
```

## Expected Error

```json
{
  "absent": true
}
```
