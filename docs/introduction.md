# Introduction

This page walks through the ideas behind Noodle. If you want to jump straight to installing, head to [Getting Started](/getting-started).

## A skill is a markdown file

Say you want your coding agent to follow a specific workflow when implementing tasks. You could paste instructions into the chat every time, or you could write it down once:

```yaml
---
name: execute
description: >
  Pick up a task and implement it.
---
Read the task description. Break it into small steps.
Work in a feature branch. Write tests before changing code.
Commit after each step, not all at once.
```

Save this as `.agents/skills/execute/SKILL.md`. Now any agent in your project can use it. You just wrote a skill.

A skill can teach anything you'd explain to a coworker. Here's one for debugging:

```yaml
---
name: debugging
description: >
  Systematic root-cause debugging.
---
1. Reproduce the failure with a minimal case
2. Read the error. Read it again.
3. Form a hypothesis, then find evidence before changing code
4. Fix the root cause, not the symptom
5. Add a test that would have caught it
```

And one for code review:

```yaml
---
name: review
description: >
  Review code changes for correctness and style.
---

Walk the diff in this order: architecture, correctness,
tests, performance. Number each issue. For each one,
suggest two options with tradeoffs. Don't nitpick formatting.
```

These are general skills. An agent uses them when it decides to, or when you tell it to. They're tools in a toolbox.

## What if a skill could run itself?

The skills above wait for someone to invoke them. But some work is repetitive. You review code after every change. You reflect after every session. You check the backlog every morning.

What if you could describe _when_ a skill should run, and have it happen automatically?

```yaml
---
name: review
description: >
  Review code changes for correctness and style.
schedule: >
  After each agent session completes
---
```

That `schedule:` field is just a sentence describing when to run. Here's another:

```yaml
---
name: reflect
description: >
  Capture learnings from the session.
schedule: >
  After an agent session completes and
  the review stage has finished
---
```

An agent could read these `schedule:` fields and figure out when to run things. Sometimes it would get it right. But you'd want something that does it consistently, every time, without you having to remind it.

## Making it automatic

You could do this yourself. Check the backlog, look at what's running, decide what to kick off next. But that's just project management, and you're already describing the rules in the `schedule:` fields. Why not have an agent do it?

What if the scheduler was a skill too:

```yaml
---
name: schedule
description: >
  Decide what work to do next.
schedule: >
  When there's nothing running, after the backlog
  changes, or after a task finishes
---
Read the `schedule:` fields on every skill in the project.
Look at the backlog and what's already in progress.
Pick the highest priority item that isn't blocked and
run the right skill for it.
```

Instead of telling agents which skills to use, you have a skill that figures out how and when to call the others.

## Putting it together

So you've got a `schedule` skill that reads the backlog and picks work, an `execute` skill that does the work, a `review` skill that checks it, and a `reflect` skill that captures what was learned. Each one knows when it should run.

But now you need something to actually run the loop. We still have to manually run the `schedule` skill. What we need is a loop: something that reads the schedule, spawns an agent, lets it work, merges the result, and repeats. You also need to keep agents from stepping on each other: if two are working at the same time, they need separate copies of the repo.

## Noodling the circle

Noodle is the framework that your skills run on. Give it a project with schedulable skills:

```sh
noodle start
```

Noodle figures out which skills can be scheduled, runs the scheduler agent, spawns agents in isolated git worktrees, merges their work back, and loops. You write the skills. Noodle keeps the cycle going.

## What to read next

- [Getting Started](/getting-started) — install Noodle and run your first loop
- [Skills](/concepts/skills) — the full picture on writing and composing skills
- [FAQ](/reference/faq) — common questions about Noodle
