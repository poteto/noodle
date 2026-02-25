# Noodle

Noodle is an AI coding framework built in Go. It uses skills as the only extension point and a
kitchen brigade model: chef (human), Schedule (scheduler), cooks (implementation), and Quality
(review).

## Quick Start

Prerequisites:

- `tmux`
- Claude Code or Codex
- Windows users: run inside WSL

Install via Homebrew (macOS):

```sh
brew install poteto/tap/noodle
```

Then start:

```sh
noodle start
```

For agents setting up Noodle in a project, see [INSTALL.md](INSTALL.md).

## How It Works

- Chef: human direction and intervention
- Schedule: schedules queue from mise data
- Cook: executes backlog work
- Quality: reviews completed work

Architecture details: [Open-Source Architecture Overview](brain/archived_plans/01-noodle-extensible-skill-layering/overview.md)

## Configuration

Noodle reads `.noodle.toml` at project root.

Minimal baseline:

```toml
[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"

[schedule]
skill = "schedule"

[skills]
paths = [".agents/skills"]

[agents.claude]
path = ""
args = []

[agents.codex]
path = ""
args = []
```

Schema reference: `config/config.go`

## Skills

Skills are the only extension point.

- Project skills: `.agents/skills/` (default resolver path)
- Resolver order is configured by your `.noodle.toml`.

## Adapters

Adapters bridge your backlog/plan system to Noodle using:

1. A skill that teaches semantics (`backlog`, `plans`, or custom)
2. Scripts declared in `.noodle.toml` (`sync`, `add`, `done`, `edit`, etc.)

Script templates are project-defined in `.noodle.toml`.

## CLI Reference

| Command | Description |
| --- | --- |
| `noodle start` | Run scheduling loop |
| `noodle start --once` | Run one cycle and exit |
| `noodle status` | Show compact runtime status |
| `noodle debug` | Dump canonical runtime debug state |
| `noodle skills list` | List resolved skills with precedence |
| `noodle worktree <subcommand>` | Worktree operations (`create`, `merge`, `cleanup`, `list`, `prune`, `hook`) |
| `noodle plan <subcommand>` | Plan management (`create`, `done`, `phase-add`, `list`) |
| `noodle schema [target]` | Print generated runtime schema docs (`list`, `mise`, `queue`) |
| `noodle mise` | Build and print current mise brief (internal) |
| `noodle dispatch` | Dispatch a cook session in tmux (internal) |
| `noodle stamp` | Stamp NDJSON logs (internal) |
| `noodle --help` | List all available commands |

## Contributing

Build and verify:

```sh
go build ./...
go test ./...
go vet ./...
go run . --help
```

Repository layout:

- `.agents/skills/` project skills
- `.noodle/` project-level runtime state and local config
- `brain/` project memory and implementation plans
- `worktree/` git worktree lifecycle helpers
