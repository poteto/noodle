# Backlog Sync Error Fixture

## Input
```ndjson
{"title":"Fix login bug","status":"open"}
```

## Expected Error

```json
{
  "contains": "missing required field id"
}
```
