# Claude Subprocess Dispatch Patterns

Agent subprocesses (cook sessions) are dispatched via `noodle dispatch`, which encapsulates tmux session management, prompt delivery, and NDJSON log capture.

## Key Gotchas

- **Write prompts to files, not inline.** Embedding prompts in shell commands causes quoting issues and permission prompts. Use the Write tool to write the prompt to a file, then pass it as an argument to `noodle dispatch`.
- **No variable assignments in Bash commands.** Variable assignments before commands (e.g., `LOG="/tmp/foo" && grep ...`) trigger permission prompts. Inline literal paths directly.

See also [[principles/boundary-discipline]], [[principles/guard-the-context-window]], [[codebase/claude-print-flag-gotchas]]
