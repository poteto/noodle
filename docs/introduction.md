# Introduction

This page walks through the ideas behind Noodle. If you want to jump straight to installing, head to [Getting Started](/getting-started).

---

## What if a skill could run itself?

You review code after every change. You record learnings and mistakes after every session. You check the backlog every morning.

What if you could describe _when_ a skill should run, and have it happen automatically?

```yaml
---
description: Rigorous review with a subagent team
schedule: After finishing complex tasks
---
```

Well, what if you add some information about when skills should run? Like a `schedule:` field. But what would schedule it? Maybe an agent could read these and figure out when to run things.

We could write a skill that could schedule other skills:

```yaml
---
name: schedule
---
Read the `schedule:` fields on every skill in the project.
...And figure out when to use them?
```

But there's something missing here.

## We need a loop

You need something to actually _schedule_ the skills. With what we have so far, we still have to manually run the `schedule` skill. That's a little better but not entirely different from where we started.

What we need is a loop: something that reads the schedule, spawns an agent, lets it work, merges the result, and repeats. You also need to keep agents from stepping on each other: if two are working at the same time, they need separate copies of the repo.

It would be even better if the scheduler itself was an agent. Then we wouldn't need complicated workflow syntax or APIs.

## Skills that run themselves

Noodle is the framework that makes it possible for skills to run themselves. Give it a project with schedulable skills:

```sh
noodle start
```

Noodle figures out which of your skills can be scheduled, runs the scheduler agent, spawns agents in isolated git worktrees, merges their work back, and loops.

You write the skills. Noodle makes them run themselves.

## What to read next

- [Getting Started](/getting-started) — install Noodle and run your first loop
- [Skills](/concepts/skills) — the full picture on writing and composing skills
- [FAQ](/reference/faq) — common questions about Noodle
