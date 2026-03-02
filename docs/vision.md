# Vision

Other orchestration tools have you wiring together plugins and lifecycle hooks and impose their opinions on you. Noodle has only one concept: skills. Your agents write them, and Noodle handles the rest: scheduling work, assigning it to agents, merging the output, and looping in a Ralph Wiggum loop.

```yaml
---
name: review
description: Reviews completed work for correctness and principle compliance.
schedule: After each cook session completes
---
Review completed cook work. Check for correctness, scope discipline,
and adherence to project principles. Flag issues before merge.
```

If you've used React, think of skills like components. Components are React's single abstraction. You build everything out of them, compose them, and the framework handles the rest.

Skills work the same way in Noodle. Every behavior your agents have comes from a skill. Every workflow is a composition of skills.

The `schedule` field in the frontmatter is what turns a skill from something you invoke manually into something that Noodle runs on its own. Write a plain-English description of when it should fire, and the scheduling agent decides when the conditions are met.

Then create a `schedule` skill, and the scheduling agent does the supervision for you. Because it's just a skill, you have extreme flexibiity: pick the right model or even runtime (local/remote) for each task, or describe what kind of tasks you want to prioritize first.

## Why not just use an agent directly?

You can. A single agent session with a good prompt will get work done. But you decide what to work on. You review the output. You context-switch between tasks. You run one thing at a time.

Noodle automates that using the skills you've already made. A scheduling agent reads your backlog and writes work orders. Multiple agents execute in parallel, each in an isolated worktree so they never conflict. Completed work merges back and the scheduler re-evaluates. You go from driving one session to supervising a factory.

Your backlog can live anywhere. A single markdown file like `todos.md` works out of the box, but [adapters](/concepts/adapters) let Noodle pull work from GitHub Issues, Linear, Jira, or whatever you already use. Most orchestration tools force you into their task format. Noodle meets you where your tickets already are.

## The Hello World loop

Two skills give you a working system:

- **schedule** reads the backlog and decides what to work on next.
- **execute** picks up a task and does the work.

```yaml
---
name: schedule
description: >
  Scheduler for your orchestration workflow.
schedule: >
  When orders are empty, after backlog changes,
  or when session history suggests re-evaluation
---
```

```yaml
---
name: execute
description: >
  Your workflow for executing tasks.
schedule: >
  When backlog items with linked plans
  are ready for implementation
---
```

The scheduler produces orders and Noodle dispatches them to agents. Each agent runs the execute skill in its own worktree, makes changes, and commits. Completed work enters the merge queue. The scheduler re-evaluates. The loop continues until the backlog is empty or you stop it.

## Run anywhere

Noodle can orchestrate agents locally or in the cloud. Local agents run as child processes on your machine. Remote agents run in VMs or containers. The scheduling agent picks the runtime for each task and can match the model to the work. Use a powerful model for architecture decisions, a fast one for straightforward implementation.

Run twenty agents in parallel on cloud VMs while your laptop sits idle. Route expensive tasks to powerful remote machines and keep cheap ones local. The agents work on branches, push their changes, and Noodle merges them back.

## Who is this for

Noodle is for developers who already use AI coding agents and want to scale them up. You've used Claude Code, Codex, or Cursor. You've seen what one agent session can do. Now you want to run multiple sessions in parallel, schedule work automatically, and have agents learn from each other.

If you haven't tried AI-assisted coding yet, start there first. Use an agent on a real task. Once you have a feel for what works, come back and Noodle will help you run more of it.

## What this means for you

You don't learn Noodle by reading API docs. You learn it by reading a few skills and writing your own.

Each skill you write captures how you want work done. The schedule field tells the scheduler when to fire it. The body tells the agent how to do it. That's the whole model. Start with schedule and execute. Add a review skill when you want a quality gate. Add more skills as your workflow demands them.

## Going further

Noodle works without persistent memory, but if you want agents that learn across sessions, check out [brainmaxxing](https://github.com/poteto/brainmaxxing). It's an optional skill pack that adds reflect, meditate, and ruminate -- three skills that capture session learnings, distill principles, and mine past conversations for knowledge that was never written down. The brain (a directory of markdown files) accumulates your project's working knowledge over time.

Brainmaxxing is a separate install. Noodle's core loop does not depend on it.
