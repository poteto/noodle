# Claude Print Flag Gotchas

- `--tools ""` can consume positional args. Use `--` before the prompt: `claude -p --tools "" -- "<prompt>"`
- `--include-partial-messages` only works with `--output-format stream-json`.
- For long non-interactive reviews, monitor liveness from `~/.claude/projects/*.jsonl` mtime. Treat as stalled only if logs stop changing for >180s and the process is still running.

See also [[codebase/claude-subprocess-spawn-patterns]], [[codebase/claude-code-ndjson-protocol]]
