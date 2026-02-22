# Plans Sync Error Fixture

## Input
```ndjson
{"id":"plan-1","title":"Shipping plan","status":"active","phases":[]}
```

## Expected Error

```json
{
  "contains": "missing required field phases"
}
```
