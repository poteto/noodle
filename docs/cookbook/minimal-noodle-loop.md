# Minimal Noodle Loop

The smallest working Noodle project. Two skills, a backlog, and a config file. This is the "hello world." If you can run this, everything else builds on top of it.

## Project structure

```
.noodle.toml
.agents/skills/schedule/SKILL.md
.agents/skills/execute/SKILL.md
brain/todos.md
```

Four files. That's it.

## Configuration

`.noodle.toml` sets the default model and tells Noodle where skills live:

```toml
mode = "auto"

[routing.defaults]
provider = "claude"
model = "claude-sonnet-4-6"

[skills]
paths = [".agents/skills"]
```

`mode = "auto"` means Noodle runs the full noodle loop unattended. The scheduler reads the backlog, creates orders, spawns agents, merges results, and repeats until there's nothing left to do.

## The schedule skill

`.agents/skills/schedule/SKILL.md`:

````yaml
---
name: schedule
description: Reads backlog and produces work orders for the loop.
schedule: "When orders are empty or after backlog changes"
---

# Schedule

Read `brain/todos.md` for backlog items. Write `.noodle/orders-next.json`
with orders for each unchecked item.

Each order has a single execute stage:

​```json
{
  "orders": [
    {
      "id": "todo-1",
      "title": "Add a hello-world CLI command",
      "rationale": "First backlog item, straightforward addition",
      "status": "active",
      "stages": [
        { "skill": "execute", "status": "pending" }
      ]
    }
  ]
}
​```

When no unchecked items remain, write `{"orders": []}`.
````

The scheduler's job is reading. It looks at the backlog, decides what needs doing, and writes structured orders that other skills pick up. It doesn't implement anything itself.

## The execute skill

`.agents/skills/execute/SKILL.md`:

```yaml
---
name: execute
description: Implements a backlog item. Reads the task prompt, makes changes, commits.
schedule: "When backlog items with linked plans are ready for implementation"
---

# Execute

Implement the task described in the prompt. Make the code changes, verify
they work, and commit with a conventional commit message.

## Steps

1. Read the task description from the prompt.
2. Make the required changes.
3. Verify: run tests or checks relevant to the change.
4. Commit with a message in the format: `<type>(<scope>): <description>`.
```

The execute skill is the workhorse. It gets a task, makes changes, verifies them, and commits. Each execution runs in its own git worktree, so multiple agents can work in parallel without stepping on each other.

## The backlog

`brain/todos.md`:

```markdown
# Todos

<!-- next-id: 3 -->

## Backlog

1. [ ] Add a hello-world CLI command that prints "Hello from Noodle"
2. [ ] Write a README explaining what this project does
```

Numbered checkboxes. The scheduler reads these, creates one order per unchecked item, and the noodle loop works through them. As items complete, they get checked off.

## Running it

```sh
noodle start
```

Here's what happens:

1. Noodle reads `.noodle.toml` and discovers the two skills.
2. The scheduler runs first. It reads `brain/todos.md`, finds two unchecked items, and writes orders to `.noodle/orders-next.json`.
3. Noodle picks up the first order and spawns an agent as a child process with its own worktree.
4. The agent runs the execute skill: reads the task, implements it, runs any checks, and commits.
5. The completed worktree merges back to the main branch.
6. The noodle loop re-schedules. The scheduler sees one item is done, creates orders for remaining work.
7. This continues until all backlog items are checked off.

Open `localhost:3000` to watch it in the web UI, or run `noodle status` from the terminal.
