# Runtimes

A runtime is where an agent session executes. Noodle ships with two: local processes and [Sprites](https://sprites.dev) cloud sandboxes. The scheduling agent can pick a runtime per stage, or you can set a project-wide default. Custom runtime plugins are on the roadmap, so you'll be able to define your own.

## Process

The default runtime. Runs agent CLIs as child processes on the host machine.

No setup required. It works out of the box. Best for local development, quick iteration, and small teams where a single machine has enough capacity.

```toml
[runtime]
default = "process"

[runtime.process]
max_concurrent = 4
```

`max_concurrent` defaults to 4. This caps how many local agent processes run simultaneously, independent of the global `max_cooks` ceiling.

## Sprites

Sprites ships as a built-in runtime today but will be split into its own runtime plugin in the future. Cloud sandboxes via [sprites.dev](https://sprites.dev). Each agent runs in an isolated cloud VM with its own filesystem and git worktree.

Use sprites when you want more parallelism than your local machine can handle, or when you want to free up your machine for something else while agents work in the cloud.

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

## Choosing a Runtime

| Runtime   | Best for                                | Default `max_concurrent` |
| --------- | --------------------------------------- | -----------------------: |
| process   | Local dev, quick iteration, small teams |                        4 |
| sprites   | Parallel throughput, CI/CD, scaling out |                       50 |

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

Process is always available. Sprites requires a valid token in the configured environment variable. Noodle checks for this at startup and only advertises runtimes that have working credentials.

### Teaching your scheduler to use runtimes

The scheduler reads `routing.available_runtimes` from the mise to know what's available. You teach it how to pick runtimes by writing instructions in your `schedule` skill. Here's an excerpt from a real schedule skill:

```markdown
## Runtime Routing

Read `routing.available_runtimes` from mise before writing orders.

- If only `process` is available, set stage `"runtime": "process"`.
- If `sprites` is available, prefer `"runtime": "sprites"` for
  long-running `execute` work.
- Keep `review`, `reflect`, and `meditate` on `"runtime": "process"`
  unless explicitly justified.
- Always include `"runtime"` on scheduled stages so dispatch routing
  is explicit.
```

Because the scheduler is just a skill, you have full control over how it assigns runtimes. You can tell it to always use sprites for implementation, keep reviews local, or split work across runtimes based on whatever criteria make sense for your project.

## Custom Runtimes

Custom runtime plugins are coming. You'll be able to add your own VM or execution environment by implementing Noodle's runtime interface. If you run your own infrastructure and the built-in runtimes don't fit, this will let you plug it in directly.
