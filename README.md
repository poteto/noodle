# Noodle

Noodle is an AI coding framework built in Go. It uses skills as the only extension point and a
kitchen brigade model: chef (human), Prioritize (scheduler), cooks (implementation), and Quality
(review).

## Quick Start

Prerequisites:

- Go 1.24+
- `tmux`
- Claude Code or Codex
- Windows users: run inside WSL

1. Clone and enter the repo:

```sh
git clone https://github.com/poteto/noodle.git
cd noodle
```

2. Verify and start:

```sh
~/.noodle/bin/noodle commands
~/.noodle/bin/noodle skills list
~/.noodle/bin/noodle status
~/.noodle/bin/noodle start --once
```

For continuous scheduling, run:

```sh
~/.noodle/bin/noodle start
```

## How It Works

- Chef: human direction and intervention
- Prioritize: prioritizes queue from mise data
- Cook: executes backlog work
- Quality: reviews completed work

Architecture details: [Open-Source Architecture Overview](brain/plans/01-noodle-extensible-skill-layering/overview.md)

## Configuration

Noodle reads `noodle.toml` at project root.

Minimal baseline:

```toml
[routing.defaults]
provider = "claude"
model = "claude-sonnet-4-6"

[prioritize]
skill = "prioritize"

[skills]
paths = [".agents/skills"]

[agents]
claude_dir = ""
codex_dir = ""
```

Schema reference: `config/config.go`

## Skills

Skills are the only extension point.

- Project skills: `.agents/skills/` (default resolver path)
- Resolver order is configured by your `noodle.toml`.

## Adapters

Adapters bridge your backlog/plan system to Noodle using:

1. A skill that teaches semantics (`backlog`, `plans`, or custom)
2. Scripts declared in `noodle.toml` (`sync`, `add`, `done`, `edit`, etc.)

Script templates are project-defined in `noodle.toml`.

## CLI Reference

| Command | Description |
| --- | --- |
| `noodle start` | Run scheduling loop |
| `noodle start --once` | Run one cycle and exit |
| `noodle status` | Show compact runtime status |
| `noodle debug` | Dump canonical runtime debug state |
| `noodle skills list` | List resolved skills with precedence |
| `noodle commands` | List commands |
| `noodle commands --json` | List commands as JSON |
| `noodle worktree <subcommand>` | Worktree operations (`create`, `merge`, `cleanup`, `list`, `prune`, `hook`) |
| `noodle mise` | Build and print current mise brief (internal) |
| `noodle spawn` | Spawn cook session in tmux (internal) |
| `noodle stamp` | Stamp NDJSON logs (internal) |

## Contributing

Build and verify:

```sh
go build ./...
go test ./...
go vet ./...
go run . commands --json
```

Repository layout:

- `.agents/skills/` project skills
- `.noodle/` project-level runtime state and local config
- `brain/` project memory and implementation plans
- `worktree/` git worktree lifecycle helpers
