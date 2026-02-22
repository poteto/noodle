# Spawner Fixture: Codex Provider Command

## Setup
```json
{
  "request": {
    "name": "cook-b",
    "prompt": "Say ok",
    "provider": "codex",
    "model": "gpt-5.3-codex",
    "worktreepath": ".worktrees/phase-06-spawner"
  },
  "prompt_file": "/tmp/prompt.txt",
  "agent_binary": "codex",
  "system_prompt": ""
}
```

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
