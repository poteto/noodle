# Key Terms

Quick reference for terms used throughout the Noodle docs.

## Adapter

A bridge between an external system (GitHub Issues, Linear, Jira) and Noodle's backlog. Adapters are commands configured in `.noodle.toml` that sync, add, complete, and edit items. Can be any executable: shell scripts, Python, Node, Go binaries. See [Adapters](/concepts/adapters).

## Brain

A directory of markdown files (`brain/`) committed to your repo. Agents read brain files for context before starting work. Compatible with [Obsidian](https://obsidian.md) vaults. Optional, installed via [brainmaxxing](https://github.com/poteto/brainmaxxing). See [Self-Learning](/concepts/brain).

## General skill

A skill without a `schedule` field. General skills are invoked directly by agents during execution rather than triggered automatically by the scheduler. Examples: `commit`, `review`, `debugging`.

## Merge queue

The queue of completed work waiting to be merged back into the main branch. In auto mode, completed work merges immediately. In supervised or manual mode, work waits for human approval.

## Mise

Short for _mise en place_. The project snapshot the scheduler reads before scheduling: current state of the backlog, active orders, recent session history, and project context. Stored at `.noodle/mise.json`.

## Orders

Work assignments written by the scheduler. Each order has one or more stages that execute sequentially. Stored at `.noodle/orders-next.json`. See [Scheduling](/concepts/scheduling#orders).

## Routing

The system that decides which agent provider and model handle each task. The scheduler assigns routing per stage when writing orders. Default provider and model are configured in `.noodle.toml`. See [Configuration](/reference/configuration#routing).

## Schedule field

The `schedule:` field in a skill's YAML frontmatter. A natural-language description of when the skill should fire. The scheduler reads it and decides when conditions are met. Skills with a schedule field run autonomously in the noodle loop. Skills without one are [general skills](#general-skill).

## Scheduler

The agent that reads the [mise](#mise), evaluates the backlog, and writes [orders](#orders). The scheduler never touches code. It only decides what to work on and who does it.

## Skill

A markdown file (`SKILL.md`) that teaches an agent how to do something. The body is a prompt. The frontmatter is metadata. Skills are all you need to extend Noodle. See [Skills](/concepts/skills).

## Stages

Steps within an order that execute sequentially. A typical pipeline: execute, then quality, then reflect. See [Scheduling > Stages](/concepts/scheduling#stages).

## Tags

Labels attached to skills or backlog items. The scheduler can use tags to inform routing decisions when writing orders.

## Worktree

A git worktree. Each agent works in its own worktree so concurrent agents never conflict. Worktrees are created automatically and cleaned up after merge.
