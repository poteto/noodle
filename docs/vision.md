# Vision

Skills drive everything in Noodle. Every behavior an agent can perform — scheduling work, executing tasks, reviewing code, learning from mistakes — is a skill. There are no plugins, no hooks, no configuration DSL. You extend Noodle by writing skills, and skills are just markdown files.

This sounds almost too simple. It is simple. That's the point.

## Skills: one extension point

Most frameworks give you plugins, hooks, middleware, configuration DSLs, and a lifecycle API that takes a week to internalize. Noodle gives you skills.

A skill is a markdown file that teaches an agent how to do something. The body is a prompt. The frontmatter is metadata. That's it.

```yaml
---
name: deploy
description: Deploy after successful execution on main
schedule: "After a successful execute completes on main branch"
---

Check that CI passed on the main branch. Run the deploy script.
Tag the release.
```

If you've used React, think of skills the way you think of components. Components are React's single abstraction — you build everything out of them, you compose them together, and the framework handles the rest. Skills work the same way in Noodle. Every behavior your agents have comes from a skill. Every workflow you build is a composition of skills. You don't need to learn a second concept.

The `schedule` field in the frontmatter is what turns a skill from something you invoke manually into something the system runs on its own. Write a natural-language description of when it should fire, and the scheduling agent reads it and decides when the conditions are met. No cron syntax, no event bus, no pub/sub wiring. Just a sentence.

## The self-improving loop

Here's where it gets interesting. Two skills give you a minimal autonomous system:

- **schedule** reads the backlog and decides what to work on next.
- **execute** picks up a task and does the work.

```yaml
---
name: schedule
description: Orders scheduler. Reads .noodle/mise.json, writes .noodle/orders-next.json.
schedule: "When orders are empty, after backlog changes, or when session history suggests re-evaluation"
---
```

```yaml
---
name: execute
description: Implementation methodology for executing tasks.
schedule: "When backlog items with linked plans are ready for implementation"
---
```

That loop runs on its own. But it doesn't learn. Add two more skills and it does:

- **reflect** runs after each session. The agent writes down what happened — what worked, what broke, what it would do differently. Those notes go into the brain (more on that in a second).
- **meditate** runs periodically. It reads everything the agents have learned and distills higher-level principles. Then it updates your skills with those principles, so every future agent session starts with better instructions.

```yaml
---
name: reflect
description: Reflect on the conversation and update the brain.
schedule: "After a cook session completes"
---
```

```yaml
---
name: meditate
description: Audit and evolve the brain vault — prune outdated content, discover cross-cutting principles.
schedule: "Periodically after several reflect cycles have accumulated"
---
```

Each skill you add makes the system more capable. Reflect teaches it from experience. Meditate turns experience into principles. Those principles flow back into your skills, which produce better work, which generates better reflections. The loop compounds.

## The brain

Noodle has a brain. It's a directory of markdown files — an Obsidian vault if you use Obsidian, plain markdown if you don't — that agents read before they start working and write to after they finish.

The brain holds your project's working knowledge. Principles you care about. Patterns that work. Architectural decisions and why they were made. Mistakes that have already been made once.

When an agent reflects after a session, it might notice that a certain test pattern keeps causing flaky failures and write a note. The next agent reads that note before it starts and avoids the same trap. Over time the brain accumulates real, project-specific knowledge — not generic best practices from a training corpus, but lessons earned from working on your actual codebase.

Your principles live here too. If you care about subtracting before you add, or redesigning from first principles instead of patching, the meditate skill finds those themes in your brain and encodes them into your skills. Every future decision an agent makes gets grounded by what your project actually values.

## Run anywhere

Noodle can orchestrate agents locally or in the cloud. Local agents run as child processes on your machine. Remote agents run in VMs or containers. The scheduling agent picks the runtime for each task and can match the model to the work — use a powerful model for architecture decisions, a fast one for straightforward implementation.

Run twenty agents in parallel on cloud VMs while your laptop sits idle. Route expensive tasks to powerful remote machines and keep cheap ones local. The agents work on branches, push their changes, and Noodle merges them back.

## What this means for you

You don't learn Noodle by reading API docs. You learn it by reading a few skills and writing your own.

Each skill you write captures how you want work done. Each reflection captures what your agents learned. Each meditation sharpens the whole system. The framework gets better at your project specifically, not at projects in general.

Start with schedule and execute. Add reflect when you want your agents to learn. Add meditate when you want them to think. Build from there.
