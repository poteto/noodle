# Claude Print Flag Gotchas

- In `claude -p` mode, `--include-partial-messages` only works with `--output-format stream-json`.
- If `--output-format stream-json` is not used, omit `--include-partial-messages`.
- `--tools ""` works to disable tools, but `--tools` can consume positional args.
- When passing a prompt argument with `--tools`, use `--` before the prompt:
  - `claude -p --dangerously-skip-permissions --tools "" -- "<prompt>"`
- When Claude needs access outside the current repo (for example a local reference clone), pass explicit allowlist directories with `--add-dir`:
  - `claude -p --dangerously-skip-permissions --add-dir <reference-dir> --tools "" -- "<prompt>"`
- Global session logs are written under `~/.claude/projects/<project-slug>/*.jsonl`.
- For long non-interactive reviews, monitor liveness from `~/.claude/projects/*.jsonl` mtime while the Claude process is alive. Treat as stalled only if logs stop changing for >180s and the process is still running.
