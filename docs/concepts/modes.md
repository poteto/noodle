# Modes

Noodle has three run modes that control how much autonomy agents have. The mode determines whether work merges automatically, requires review, or needs human approval.

## The Three Modes

### Auto

The default. Agents merge their own work. No human in the noodle loop. The scheduling agent runs continuously, agents execute and merge without waiting for approval.

Use auto mode for projects with good test coverage, or when you want agents to work unattended. Everything still goes through the merge queue, so changes are serialized and can be rolled back.

### Supervised

Agents work and a quality gate reviews the output. Approved work merges automatically.

The quality gate can be another agent reviewing the diff, a test suite that must pass, or both. Work that fails review gets flagged but does not block other orders.

### Manual

Same as supervised, but humans confirm before merging. The agent does the work, the quality gate reviews it, and the merge waits for your explicit approval.

Use manual mode for high-stakes changes: production deployments, database migrations, security-sensitive code.

## Setting the Mode

Set the mode in `.noodle.toml`:

```toml
mode = "auto"
```

Valid values: `auto`, `supervised`, `manual`.

The mode can also change at runtime through control commands. Transitions are tracked. The state records what mode was active, when it changed, and why.

## Trust as a Dial

The mode is a trust dial, not a capability switch. All three modes use the same noodle loop, the same skills, the same agents. The only difference is what happens after the work is done.

A mature project with good tests can run in auto. A new project starts in supervised. Critical infrastructure stays in manual. Move the dial as trust builds.

## The Kitchen Brigade

Noodle borrows from the kitchen brigade system used in professional kitchens:

- **Scheduler** (scheduling agent): reads the mise, decides what to work on, writes orders
- **Agents** (execution agents): pick up orders and do the work, each in their own worktree
- **Quality** (review agents): inspect output before merge
- **You:** set the mode, stock the backlog, adjust skills, course-correct

The mode controls where you sit in this chain. In auto, you are hands-off. In supervised, you observe. In manual, you approve each merge.
