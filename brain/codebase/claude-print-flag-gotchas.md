# Claude Print Flag Gotchas

- In `claude -p` mode, `--include-partial-messages` only works with `--output-format stream-json`.
- If `--output-format stream-json` is not used, omit `--include-partial-messages`.
- `--tools ""` works to disable tools, but `--tools` can consume positional args.
- When passing a prompt argument with `--tools`, use `--` before the prompt:
  - `claude -p --dangerously-skip-permissions --tools "" -- "<prompt>"`
- Global session logs are written under `~/.claude/projects/<project-slug>/*.jsonl`.
