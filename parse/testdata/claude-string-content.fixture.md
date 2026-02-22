# Claude String Content Fixture

## Input

```ndjson
{"type":"assistant","message":{"role":"assistant","content":"Work complete and tests passing"},"_ts":"2026-02-22T19:10:00Z"}
```

## Expected Events

```ndjson
{"type":"action","message":"text:Work complete and tests passing","timestamp":"2026-02-22T19:10:00Z"}
```
