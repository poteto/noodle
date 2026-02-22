# Spawner Fixture: Claude Provider Command

## Setup
```json
{
  "request": {
    "name": "cook-a",
    "prompt": "Say ok",
    "provider": "claude",
    "model": "claude-sonnet-4-6",
    "reasoninglevel": "medium",
    "maxturns": 5,
    "budgetcap": 2.5,
    "worktreepath": ".worktrees/phase-06-spawner"
  },
  "prompt_file": "/tmp/prompt.txt",
  "agent_binary": "claude",
  "system_prompt": "skill-system"
}
```

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
