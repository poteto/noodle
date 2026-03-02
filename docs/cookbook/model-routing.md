# Model Routing

Not every task needs the same model. Formatting a file is a waste of Opus. Designing an architecture is a bad fit for a small, fast model. Noodle's routing system lets you match models to tasks.

## Defaults

Set your default model in `.noodle.toml`:

```toml
[routing.defaults]
provider = "claude"
model = "claude-opus-4-6"
```

Every stage inherits this unless the stage explicitly sets `provider` and `model` in `orders.json`.
This is the only runtime routing override path today.

## Stage-level override

The scheduler can set `provider` and `model` directly per stage:

```json
{
  "orders": [
    {
      "id": "todo-5",
      "title": "Refactor utils to use new date library",
      "stages": [
        {
          "skill": "execute",
          "status": "pending",
          "provider": "codex",
          "model": "gpt-5.3-codex"
        },
        {
          "skill": "test",
          "status": "pending",
          "provider": "claude",
          "model": "claude-sonnet-4-6"
        },
        {
          "skill": "review",
          "status": "pending",
          "provider": "claude",
          "model": "claude-opus-4-6",
          "prompt": "Review the refactor for correctness and edge cases"
        }
      ]
    }
  ]
}
```

This gives the scheduler full control, and allows you to define your own criteria for routing tasks to the right model.
