# Noodle

Skill-based agent orchestration. Built in Go.

## Install

```sh
brew install poteto/tap/noodle
```

Requires [tmux](https://github.com/tmux/tmux) and at least one of [Claude Code](https://docs.anthropic.com/en/docs/claude-code) or [Codex](https://github.com/openai/codex).

## Set up your project

Point your coding agent at [`INSTALL.md`](INSTALL.md) and it'll configure everything — skills, config, backlog structure — tailored to your project. Then run `noodle start` and the loop takes over.

Or follow the [getting started guide](https://noodle-run.github.io/noodle/getting-started) to set up manually.

## Docs

Full documentation lives at [noodle-run.github.io/noodle](https://noodle-run.github.io/noodle/).

## Contributing

```sh
pnpm build    # build
pnpm check    # lint + test + vet
```
