# Config Schema Reference

## Routing tags

Override the default model for specific task categories:

```toml
[routing.tags.frontend]
provider = "claude"
model = "claude-opus-4-6"

[routing.tags.backend]
provider = "codex"
model = "gpt-5.3-codex"
```

## Config validation

`noodle start` validates config on every run. Diagnostics are classified as:
- **Fatal** -- blocks startup (missing tmux, invalid routing defaults)
- **Repairable** -- warns but allows startup (missing adapter scripts)

On interactive terminals, `noodle start` offers to spawn a repair session for repairable issues.
