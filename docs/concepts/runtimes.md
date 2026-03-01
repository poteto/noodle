# Runtimes

A runtime is where a cook session executes. Noodle ships with three: local processes, cloud sandboxes, and Cursor background agents. The scheduling agent can pick a runtime per stage, or you can set a project-wide default.

## Process

The default runtime. Runs agent CLIs as child processes on the host machine.

No setup required — it works out of the box. Best for local development, quick iteration, and small teams where a single machine has enough capacity.

```toml
[runtime]
default = "process"

[runtime.process]
max_concurrent = 4
```

`max_concurrent` defaults to 4. This caps how many local agent processes run simultaneously, independent of the global `max_cooks` ceiling.

## Sprites

Cloud sandboxes via [sprites.dev](https://sprites.dev). Each cook runs in an isolated cloud VM with its own filesystem and git worktree.

Use sprites when you need parallel throughput beyond what a single machine can handle. Good for CI/CD pipelines, large backlogs, and scaling out.

```toml
[runtime.sprites]
token_env = "SPRITES_TOKEN"
sprite_name = "my-project"
max_concurrent = 20
```

| Field           | Description                                                |
| --------------- | ---------------------------------------------------------- |
| `token_env`     | Environment variable holding your Sprites API token        |
| `base_url`      | Override the default Sprites API endpoint                   |
| `sprite_name`   | Name of the sprite to use for sessions                     |
| `git_token_env` | Environment variable for git auth (defaults to `GITHUB_TOKEN`) |
| `max_concurrent`| Per-runtime concurrency cap (default 50)                   |

## Cursor

[Cursor](https://cursor.com) background agents. Dispatches work to Cursor's cloud-hosted agent sessions.

Use cursor if your team already uses Cursor and wants to route Noodle work through it.

```toml
[runtime.cursor]
api_key_env = "CURSOR_API_KEY"
repository = "owner/repo"
max_concurrent = 10
```

| Field           | Description                                                |
| --------------- | ---------------------------------------------------------- |
| `api_key_env`   | Environment variable holding your Cursor API key           |
| `base_url`      | Override the default Cursor API endpoint                    |
| `repository`    | GitHub repository in `owner/repo` format                   |
| `max_concurrent`| Per-runtime concurrency cap (default 10)                   |

## Choosing a Runtime

| Runtime   | Best for                                          | Default `max_concurrent` |
| --------- | ------------------------------------------------- | -----------------------: |
| process   | Local dev, quick iteration, small teams           |                        4 |
| sprites   | Parallel throughput, CI/CD, scaling out            |                       50 |
| cursor    | Teams already using Cursor                        |                       10 |

## Configuration

Set the project-wide default runtime:

```toml
[runtime]
default = "process"
```

The scheduling agent can override this per stage by setting the `runtime` field in an order's stage. A common pattern: run fast tasks locally, fan out heavy implementation to sprites.

```json
{
  "stages": [
    {
      "task_key": "execute",
      "prompt": "Implement the new auth flow",
      "runtime": "sprites",
      "status": "pending"
    }
  ]
}
```

Process is always available. Sprites and cursor require valid credentials in their configured environment variables — Noodle checks for these at startup and only advertises runtimes that have working tokens.
