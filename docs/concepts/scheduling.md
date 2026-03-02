# Scheduling

The noodle loop decides what work to do, when to do it, and which agent handles it. The scheduler is itself a [skill](/concepts/skills). It reads project state, evaluates skills with `schedule` fields, and writes work orders.

## The Noodle Loop

Each cycle follows this sequence:

1. **Brief.** Noodle gathers current project state into a mise (short for _mise en place_, a French culinary term which means "putting in place"). This snapshot includes the backlog, active agents, recent history, available capacity, and registered schedulable skills.
2. **Schedule.** The scheduling agent reads the mise and decides what to work on next. It writes decisions as orders to `.noodle/orders-next.json`.
3. **Validate.** Noodle reads the new orders and validates them against the skill registry, then promotes the file to `.noodle/orders.json`.
4. **Dispatch.** For each pending stage, Noodle spawns an agent session to do the work.
5. **Execute.** The agent executes in an isolated worktree, reads the skill prompt, and does the work.
6. **Merge.** When an agent finishes making changes, its changes merge back to the main branch. Merges are serialized to avoid conflicts.

The noodle loop runs continuously and reacts to file changes: when `orders-next.json` or `control.ndjson` changes on disk, Noodle runs a cycle immediately.

## Mise en Place

Before each scheduling decision, Noodle builds a mise (a snapshot of everything the scheduler needs). Written to `.noodle/mise.json`, it contains:

| Section          | Contents                                                                    |
| ---------------- | --------------------------------------------------------------------------- |
| `backlog`        | Items from your backlog adapter (GitHub issues, Linear tickets, local file) |
| `active_summary` | Running agents grouped by skill, status, and runtime                        |
| `resources`      | Total agent capacity and available slots                                    |
| `recent_history` | Outcomes of recently completed sessions                                     |
| `task_types`     | All skills with `schedule` fields and their triggers                        |
| `routing`        | Default provider/model configuration and available runtimes                  |
| `warnings`       | Issues the scheduler should know about (stale config, failed registries)    |

Because the mise is plain JSON, you can inspect it directly to see exactly what the scheduler sees.

## Orders

An order is a unit of work with one or more stages. The scheduling agent writes orders to `.noodle/orders-next.json`.

```json
{
  "id": "order-abc",
  "title": "Fix login timeout bug",
  "rationale": "High-priority backlog item, affects production",
  "stages": [
    {
      "task_key": "execute",
      "prompt": "Fix the login timeout bug described in issue #42",
      "provider": "claude",
      "model": "claude-opus-4-6",
      "status": "pending"
    }
  ],
  "status": "active"
}
```

| Field       | Description                        |
| ----------- | ---------------------------------- |
| `id`        | Unique identifier                  |
| `title`     | Human-readable summary             |
| `rationale` | Why the scheduler chose this work  |
| `plan`      | Optional list of high-level steps  |
| `stages`    | Pipeline of stages to execute      |
| `status`    | `active`, `completed`, or `failed` |

## Stages

A stage is a single step within an order.

| Field      | Description                                     |
| ---------- | ----------------------------------------------- |
| `task_key` | Which schedulable skill to run                  |
| `prompt`   | Instructions for the agent                      |
| `skill`    | Skill to invoke (alternative to `task_key`)     |
| `provider` | Agent provider for this stage                   |
| `model`    | Model to use                                    |
| `runtime`  | Where to run: `process` or `sprites`             |
| `group`    | Parallel execution group number                 |
| `status`   | Stage lifecycle status                          |

### Stage Lifecycle

```
pending → active → merging → completed
                           → failed
                           → cancelled
```

- **pending:** waiting to be dispatched
- **active:** an agent is running this stage
- **merging:** agent finished, worktree is being merged
- **completed:** work merged successfully
- **failed:** agent or merge failed
- **cancelled:** cancelled manually or superseded

### Composition and Concurrency

Stages run sequentially by default. The first stage finishes before the second starts. This is how you build pipelines: one stage's output feeds the next.

```json
{
  "stages": [
    {
      "task_key": "execute",
      "provider": "codex",
      "model": "gpt-5.3-codex",
      "runtime": "sprites",
      "status": "pending"
    },
    {
      "task_key": "quality",
      "provider": "claude",
      "model": "claude-opus-4-6",
      "runtime": "process",
      "status": "pending"
    },
    {
      "task_key": "reflect",
      "provider": "claude",
      "model": "claude-opus-4-6",
      "runtime": "process",
      "status": "pending"
    }
  ]
}
```

Each stage can use a different provider, model, and runtime. In this example, a fast coding model does the implementation on a cloud VM, then a reasoning model reviews the output locally. The scheduler decides these assignments when writing orders.

Groups are how the scheduling agent expresses a dependency graph. Stages with the same `group` number run in parallel. All stages in a group must complete before the next group starts.

```json
{
  "stages": [
    {
      "task_key": "execute",
      "group": 0,
      "status": "pending"
    },
    {
      "task_key": "execute",
      "group": 0,
      "status": "pending"
    },
    {
      "task_key": "quality",
      "group": 1,
      "status": "pending"
    }
  ]
}
```

For example, here 2 execute stages run concurrently (group 0), then quality runs after both finish (group 1). Useful when a plan has independent phases that can be split across agents.

### Routing

Each stage specifies a provider, model, and runtime. The scheduler can route different stages to different configurations:

- Run analysis on a powerful model, then run tests on a cheaper one
- Execute locally for fast iteration, or on cloud VMs for parallel throughput
- Match the model to the task: a coding model for implementation, a reasoning model for architecture

Default routing comes from `.noodle.toml`:

```toml
[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"
```

The scheduling agent can override defaults per stage when writing orders. See [Runtimes](/concepts/runtimes) for where stages execute.

## Agent Concurrency

Multiple agents run in parallel, each in its own git worktree. The `max_concurrency` setting controls how many run at once:

```toml
[concurrency]
max_concurrency = 4
```

Each agent works on an isolated branch. When it finishes, its changes enter a merge queue. Merges are serialized, one at a time, to keep the main branch clean. If a merge fails, the stage is marked failed and can be retried.
