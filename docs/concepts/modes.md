# Modes

Noodle has three run modes that control how much autonomy agents have. The mode determines whether work merges automatically, requires review, or needs human approval.

## The Three Modes

### Auto

Agents merge their own work. No human in the loop. The scheduling agent runs continuously, cooks execute and merge without waiting for approval.

Use auto mode for trusted projects with good test coverage, or when you want agents to work unattended. Everything still goes through the merge queue, so changes are serialized and can be rolled back.

### Supervised

Agents work and a quality gate reviews the output. Approved work merges automatically. This is the default mode.

The quality gate can be another agent reviewing the diff, a test suite that must pass, or both. Work that fails review gets flagged but does not block other orders. You see the results and can intervene if needed.

### Manual

Same as supervised, but humans confirm before merging. The agent does the work, the quality gate reviews it, and then the merge waits for your explicit approval.

Use manual mode for high-stakes changes — production deployments, database migrations, security-sensitive code. You review the diff and decide.

## Setting the Mode

Set the mode in `.noodle.toml`:

```toml
mode = "supervised"
```

The mode can also change at runtime through control commands. Mode transitions are tracked — the state records what mode was active, when it changed, and why.

## Trust as a Dial

The mode is a trust dial, not a capability switch. All three modes use the same scheduling loop, the same skills, and the same agents. The only difference is what happens after the work is done.

A mature project with good tests can run in auto. A new project starts in supervised. Critical infrastructure stays in manual. You move the dial as trust builds.

## The Kitchen Brigade

Noodle's orchestration model borrows from the kitchen brigade system used in professional kitchens. Each role has a clear responsibility:

- **Chef** (the scheduling agent) — reads the mise, decides what to cook, writes orders. Owns the menu.
- **Cooks** (execution agents) — pick up orders and do the work. Each cook works in their own worktree, focused on one task.
- **Quality** (review agents) — inspect the output before it merges. Check for correctness, style, test coverage.
- **You** (the human) — set the mode, stock the backlog, adjust skills, and course-correct when needed. Strategy and judgment, not execution.

The mode controls where the human sits in this chain. In auto, you are hands-off. In supervised, you observe. In manual, you approve each merge.
