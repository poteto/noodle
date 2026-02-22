# Codex Parse Error Fixture

## Input

```ndjson
Reading prompt from stdin...
{"type":"thread.started","thread_id":"abc"}
```

## Expected Error

```json
{
  "contains": "invalid character"
}
```
