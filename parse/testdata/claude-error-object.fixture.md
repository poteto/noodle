# Claude Error Object Fixture

## Input

```ndjson
{"type":"result","subtype":"success","is_error":true,"error":{"message":"Provider internal error","type":"api_error"},"_ts":"2026-02-22T19:00:00Z"}
```

## Expected Events

```ndjson
{"type":"error","message":"Provider internal error","timestamp":"2026-02-22T19:00:00Z"}
```
