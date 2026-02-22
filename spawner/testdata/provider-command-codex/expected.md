---
schema_version: 1
expected_failure: false
bug: false
source_hash: 3a1381cbd39342abbaf179758c446d86fd06a914aaebfa80be39404b5baf4833
---

## Expected

```json
{
  "contains": [
    "'codex' 'exec'",
    "'--skip-git-repo-check'",
    "'--full-auto'",
    "'--sandbox' 'workspace-write'",
    "'--json'",
    "'--model' 'gpt-5.3-codex'",
    "< '/tmp/prompt.txt'"
  ],
  "omits": [
    "2>&1"
  ]
}
```
