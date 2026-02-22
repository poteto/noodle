# Stamp Processor Error Fixture

## Input

```ndjson
{"type":"assistant","message":"ok"
```

## Expected Error

```json
{
  "contains": "parse JSON object"
}
```
