# Multi-Stage Pipeline

The minimal loop runs a single skill per order. Real projects need more: implement, then test, then deploy. Multi-stage orders chain skills together so each stage runs only after the previous one passes.

## What changes from the minimal setup

You add more skills and tell the scheduler to create orders with multiple stages. The config also sets routing defaults and concurrency.

## Configuration

`.noodle.toml`:

```toml
mode = "auto"

[routing.defaults]
provider = "claude"
model = "claude-sonnet-4-6"

[skills]
paths = [".agents/skills"]

[concurrency]
max_concurrency = 2
```

Two things are new here. Routing defaults define the baseline provider/model for generated stages, and `max_concurrency = 2` lets two agents work simultaneously in separate worktrees.

## Additional skills

The schedule and execute skills are the same as the [minimal loop](minimal-noodle-loop). You add two more.

### Test

`.agents/skills/test/SKILL.md`:

```yaml
---
name: test
description: Runs the project test suite and reports results.
schedule: "After execute stages complete, to verify changes"
---

# Test

Run the project test suite against the changes made in the previous stage.

## Steps

1. Read the test configuration for the project.
2. Run the full test suite.
3. Report results: pass or fail with details.
4. If tests fail, describe what broke and why.
```

### Deploy

`.agents/skills/deploy/SKILL.md`:

```yaml
---
name: deploy
description: Deploys the project to the target environment.
schedule: "After test stages pass, to ship verified changes"
---

# Deploy

Deploy the verified changes to the target environment.

## Steps

1. Confirm all test stages passed.
2. Run the deployment process.
3. Verify the deployment succeeded.
4. Report the deployment status.
```

Both skills have `schedule` fields, which registers them as schedulable skills the scheduler can include in orders.

## Multi-stage orders

The scheduler now creates orders with three stages instead of one:

```json
{
  "orders": [
    {
      "id": "todo-1",
      "title": "Add /healthz endpoint",
      "rationale": "Standard health check for the API",
      "status": "active",
      "stages": [
        { "skill": "execute", "status": "pending" },
        { "skill": "test", "status": "pending" },
        { "skill": "deploy", "status": "pending" }
      ]
    }
  ]
}
```

Stages run in sequence within an order. The test skill runs only after execute finishes. Deploy runs only after tests pass. If any stage fails, the remaining stages don't run.

## Project principles

`brain/principles.md`:

```markdown
- Test before deploy. Never ship code that has not passed the test suite.
- Small changes. Each commit should be one logical change.
- Conventional commits.
```

The brain isn't just for backlog items. Principles give the agent context about how your project works. The execute skill reads them to understand coding standards. The test skill reads them to know what "passing" means for your project.

## How the pipeline runs

```sh
noodle start
```

1. The scheduler reads the backlog and creates multi-stage orders.
2. An agent picks up an order and runs the execute stage, implements the change, and commits.
3. The same worktree passes to the test stage. An agent runs the test suite against the changes.
4. If tests pass, the deploy stage runs. If they fail, the order stops.
5. With `max_concurrency = 2`, a second order can start while the first is mid-pipeline.

The pipeline is sequential within an order but concurrent across orders. Two different backlog items can progress through their pipelines at the same time.
