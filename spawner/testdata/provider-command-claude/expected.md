---
schema_version: 1
expected_failure: false
bug: provider-command-claude
---

## Expected

```json
{
  "contains": [
    "'claude'",
    "'--output-format' 'stream-json'",
    "'--model' 'claude-sonnet-4-6'",
    "'--max-turns' '5'",
    "'--max-budget-usd' '2.50'",
    "'--append-system-prompt' 'skill-system'",
    "< '/tmp/prompt.txt'",
    "2>&1"
  ],
  "omits": []
}
```

