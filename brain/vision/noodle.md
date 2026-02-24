# noodle: the self-improving agent framework powered by skills

Agents read files. Agents write files. It's the one thing every agent is already good at.

noodle takes a big bet on this. It's an agent orchestration framework where the only API is the one agents know best: files. Every piece of system state is a file, so your agent already knows how to see, use, and control noodle. No SDK to install, no protocol to speak, no integration layer to maintain, and it's very hackable.

A skill is just a markdown file that tells an agent how to do something. noodle is a framework built around that idea. It uses skills to orchestrate agents. That means noodle is extensible and composable. You can use noodle to schedule your skills, and your skills can call other skills. Skills are to noodle, what components are to React. 

Agents decide what to work on, review each other's output, and learn from each session. noodle holds it all together. You step in when you feel like it.

## Everything Is A File

All of noodle's state lives in your project's `.noodle/` directory. If it's not on disk, it doesn't exist. Your agent can natively read `.noodle/queue.json` and reschedule it according to your workflow and scheduling needs.

A side effect of this is composition. Tasks don't call other tasks. Your agents pass state around by writing files and noodle picks those changes up next cycle. An agent running the `schedule` skill sees new state and schedules follow-up work. Agents are the orchestrator in noodle. There's no task composition API because there doesn't need to be one.

## Orchestration Through Skills

Skills become schedulable by adding frontmatter:

```yaml
---
name: deploy
description: Deploy after successful execution on main
noodle:
  blocking: false
  schedule: "After a successful execute completes on main branch"
---
```

The `noodle:` block registers it as a task. The skill resolver scans your skills, parses frontmatter, and builds the registry. If you want a skill to be scheduled, just ask your agent to add the noodle field and tell it when you want to run.

The Hello World minimal autonomous system in noodle is two skills working together:

- **schedule.** The agent reads the backlog and decides what to work on next.
- **execute.** The agent picks up a task off the queue and does the work.

```yaml
---
name: schedule
description: Read the backlog and decide what to work on next
noodle:
  blocking: true
  schedule: "Start of every cycle"
---

Read .noodle/mise.json. It contains the backlog, active agents, recent
session history, and available capacity.

Write .noodle/queue.json with the items you want to schedule next based on <my workflow>.
```

```yaml
---
name: execute
description: Pick up a queued item and do the work
noodle:
  schedule: "When there are items in the queue"
---

Read the plan. Do the work <the way I like it>. Commit to the worktree.
```

From there you add more skills to make it smarter. You can copy the skills noodle uses, but you can also add your own. Here are 2 of my favorites:

- **reflect.** After each session, the agent writes what it learned to the brain. The next agent reads it and avoids the same mistakes.
- **meditate.** An agent looks at all of the learnings in the brain and extracts higher-level principles from those lessons. Those lessons then get encoded into your skills, making them continuously improving.

Each skill you add makes the system more capable. Your workflow becomes a series of skills, and your agent describes to noodle when each one should run. The scheduling agent figures out the rest.

## Wait, It Gets Better

noodle has a brain. It's an Obsidian vault (just markdown files) that agents read before they start and write to after they finish. Principles, patterns, past mistakes, what worked and what didn't.

When an agent `reflect`s after a session, it updates the brain. Maybe it notices a certain test pattern keeps failing and writes a note. The next agent picks that up and avoids the same trap. Over time the brain accumulates your project's actual working knowledge. Lessons that agents are actively learning from and contributing to.

Your principles live in the brain too. If you care about redesigning from first principles, or subtracting before you add, `meditate` extracts them by looking through your brain. It even updates all your skills with these principles so every future decision by an agent is grounded by them.

## Everything Is Really A File

Because all coordination happens through files, noodle can orchestrate agents locally or in the cloud. Local agents run in tmux. Remote agents run in VMs or containers. Tell your scheduling agent to pick the runtime for each task, and even match the model to a task.

Run 20 agents in parallel on cloud VMs while your laptop sits idle. Route expensive tasks to powerful remote machines and keep cheap ones local. The agents work on branches, push their changes, and noodle merges them back.
