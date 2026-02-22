# Claude Numeric Timestamp Fixture

## Input

```ndjson
{"type":"system","subType":"init","timestamp":1730000000,"_ts":"2026-02-22T19:05:00Z"}
```

## Expected Events

```ndjson
{"type":"init","message":"session started","timestamp":"2026-02-22T19:05:00Z"}
```
