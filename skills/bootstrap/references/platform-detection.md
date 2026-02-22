# Platform Detection Guidance

Bootstrap should detect agent CLI directories and write them into:

- `agents.claude_dir`
- `agents.codex_dir`

## Rules

1. Prefer existing project-local directories if present:
   - `<project>/.claude`
   - `<project>/.codex`
2. Otherwise use user-level defaults:
   - `$HOME/.claude`
   - `$HOME/.codex`
3. Only write a path when the directory exists.
4. Keep empty string when nothing is found. Runtime falls back to `PATH`.

## Platform Notes

### macOS / Linux

- Home is `$HOME`.
- Typical paths are `$HOME/.claude` and `$HOME/.codex`.

### Windows (WSL mode required)

- Resolve home with `$HOME` inside WSL.
- Use Linux paths in config (for example `/home/<user>/.claude`).
- Do not write native Windows paths like `C:\Users\...` into `noodle.toml`.

## Detection Command Pattern

Use this flow when probing:

```sh
if [ -d ".claude" ]; then
  claude_dir="$(pwd)/.claude"
elif [ -d "$HOME/.claude" ]; then
  claude_dir="$HOME/.claude"
else
  claude_dir=""
fi
```

Use the same pattern for `.codex`.

