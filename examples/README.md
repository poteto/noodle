# Examples

Working example projects demonstrating Noodle setups.

## Prerequisites

- [tmux](https://github.com/tmux/tmux)
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) or [Codex CLI](https://github.com/openai/codex)
- Noodle installed (see [INSTALL.md](../INSTALL.md))

## Projects

### [minimal/](minimal/)

The smallest possible Noodle setup. One schedule skill, one execute skill, a two-item backlog. Demonstrates the core loop: schedule, execute, commit.

Start here if you want to understand how the pieces fit together.

### [multi-skill/](multi-skill/)

Extends the minimal example with custom task types. Adds `test` and `deploy` skills with `schedule` frontmatter so the scheduler can incorporate them into order pipelines. Demonstrates multi-stage orders and routing policies.

## Running an Example

```sh
cd examples/minimal   # or examples/multi-skill
noodle start
```

Noodle reads `.noodle.toml`, discovers skills in `.agents/skills/`, and begins the schedule-execute loop.
