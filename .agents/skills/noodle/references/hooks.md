# Hook Reference

## Brain injection hook

Injects brain vault content into the agent's context at session start. Add to `.claude/settings.json`:

```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "matcher": "SessionStart",
        "hooks": [
          {
            "type": "command",
            "command": "noodle worktree hook"
          }
        ]
      }
    ]
  }
}
```
