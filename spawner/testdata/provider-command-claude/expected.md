---
schema_version: 1
expected_failure: false
bug: false
regression: provider-command-claude
source_hash: d0b8d61613a88f6af69bade1e731a53b28ffe10392df89829718cb9d5ae898d8
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
