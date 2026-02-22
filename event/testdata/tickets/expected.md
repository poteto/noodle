---
schema_version: 1
expected_failure: false
bug: tickets
---

## Expected Tickets

```json
[
  {
    "target": "src/legacy/file.go",
    "target_type": "file",
    "cook_id": "cook-c",
    "claimed_at": "2026-02-22T20:00:00Z",
    "last_progress": "2026-02-22T20:00:00Z",
    "status": "stale"
  },
  {
    "target": "phase-03",
    "target_type": "plan_phase",
    "cook_id": "cook-b",
    "claimed_at": "2026-02-22T20:10:00Z",
    "last_progress": "2026-02-22T20:10:00Z",
    "status": "stale",
    "blocked_by": "42",
    "reason": "waiting on dependency"
  },
  {
    "target": "42",
    "target_type": "backlog_item",
    "cook_id": "cook-a",
    "files": ["src/auth/token.go"],
    "claimed_at": "2026-02-22T21:00:00Z",
    "last_progress": "2026-02-22T21:10:00Z",
    "status": "active"
  }
]
```

