# Introduction

This page walks through the ideas behind Noodle. If you want to jump straight to installing, head to [Getting Started](/getting-started).

---

## What if a skill could run itself?

Skills wait for you to call them. But some work is repetitive. You review code after every change. You reflect after every session. You check the backlog every morning.

What if you could describe _when_ a skill should run, and have it happen automatically?

```yaml
---
description: Rigorous review with a subagent team
schedule: After finishing complex tasks
---
```

Maybe an agent could read these `schedule:` fields and figure out when to run things. Sometimes it would get it right. But you'd want something that does it consistently, every time, without you having to remind it.

You could do this yourself. Check the backlog, look at what's running, decide what to kick off next. But that's just project management, and you're already describing the rules in the `schedule:` fields.

What if the scheduler was a skill too?

```yaml
---
name: schedule
---
Read the `schedule:` fields on every skill in the project.
Look at the backlog and what's already in progress.
Pick the highest priority item that isn't blocked and
run the right skill for it.
```

Instead of telling agents which skills to use, you have a skill you can call that teaches an agent to figure out how and when to call the others.

## We need a loop

So you've got a `schedule` skill that reads the backlog and picks work, an `execute` skill that does the work, and a `review` skill that checks it. Each one now knows when it should run because you added a `schedule:` field.

But now you need something to actually _schedule_ everything. We still have to manually run the `schedule` skill.

What we need is a loop: something that reads the schedule, spawns an agent, lets it work, merges the result, and repeats. You also need to keep agents from stepping on each other: if two are working at the same time, they need separate copies of the repo.

It would be even better if the scheduler itself was an agent. Then we wouldn't need complicated workflow syntax or APIs. If you're bitter lesson pilled, you want to just let agents do it.

## Noodling the circle

Noodle is the framework that makes this possible. Give it a project with schedulable skills:

```sh
noodle start
```

Noodle figures out which of your skills can be scheduled, runs the scheduler agent, spawns agents in isolated git worktrees, merges their work back, and loops.

You write the skills. Noodle makes it all work.

## What to read next

- [Getting Started](/getting-started) — install Noodle and run your first loop
- [Skills](/concepts/skills) — the full picture on writing and composing skills
- [FAQ](/reference/faq) — common questions about Noodle
