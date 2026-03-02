# Modes

Noodle has three run modes that control how much autonomy agents have. The mode determines whether work merges automatically, requires review, or needs human approval.

## The Three Modes

### Auto

Full autonomy. The scheduler picks work, agents execute, and completed work merges. No human in the noodle loop.

Use auto mode for projects with good test coverage, or when you want agents to work unattended. Everything still goes through the merge queue, so changes are serialized and can be rolled back.

### Supervised

The scheduler picks work and agents execute automatically, but merges need your approval. You review the output and decide what lands.

Use supervised mode when you trust agents to do the work but want to see the results before they hit the main branch.

### Manual

You control everything. No automatic scheduling, no automatic dispatch, no auto-merge. You trigger each step.

Use manual mode for high-stakes changes: production deployments, database migrations, security-sensitive code.

## Setting the Mode

Set the mode in `.noodle.toml`:

```toml
mode = "auto"
```

Valid values: `auto`, `supervised`, `manual`.

The mode can also change at runtime through control commands. Transitions are tracked. The state records what mode was active, when it changed, and why.

## What each mode gates

|            | Schedule | Dispatch | Merge |
| ---------- | -------- | -------- | ----- |
| `auto`       | auto     | auto     | auto  |
| `supervised` | auto     | auto     | human   |
| `manual`     | human      | human      | human   |

## Trust as a Dial

The mode is a trust dial, not a capability switch. All three modes use the same noodle loop, the same skills, the same agents. The only difference is who triggers each step.

A mature project with good tests can run in auto. A new project starts in supervised. Critical infrastructure stays in manual. Move the dial as trust builds.
