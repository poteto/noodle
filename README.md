# Noodle

AI coding framework. Agents do the work, humans steer. Built in Go.

## Install

```sh
brew install poteto/tap/noodle
```

Requires [tmux](https://github.com/tmux/tmux) and at least one of [Claude Code](https://docs.anthropic.com/en/docs/claude-code) or [Codex](https://github.com/openai/codex).

## First run

```sh
noodle start
```

On first run, Noodle scaffolds a `brain/` directory, `.noodle/` runtime state, and a `.noodle.toml` config file. From there, the scheduling agent reads your project and starts assigning work.

See the [getting started guide](https://noodle-run.github.io/noodle/getting-started) for a full walkthrough.

## Docs

Full documentation lives at [noodle-run.github.io/noodle](https://noodle-run.github.io/noodle/).

## Contributing

```sh
pnpm build    # build
pnpm check    # lint + test + vet
```
