# Glossary

Quick reference for terms used throughout the Noodle docs.

## Adapter

A bridge between an external system (GitHub Issues, Linear, Jira) and Noodle's backlog. Adapters are shell scripts configured in `.noodle.toml` that sync, add, complete, and edit items. See [Adapters](/concepts/adapters).

## Brain

A directory of markdown files (`brain/`) committed to your repo. Agents read brain files for context before starting work. Compatible with [Obsidian](https://obsidian.md) vaults. The brain is optional -- Noodle works without it. See [Brain](/concepts/brain).

## Chef

The scheduling agent. Reads the [mise](#mise), evaluates the backlog, and writes [orders](#orders). The chef never touches code -- it only decides what to work on and who does it.

## Cook

An execution agent. Picks up an [order](#orders) and does the work in an isolated [worktree](#worktree). Each cook runs as a separate LLM session, so concurrent cooks each consume API credits.

## General skill

A skill without a `schedule` field. General skills are invoked directly by agents during execution rather than triggered automatically by the chef. Examples: `commit`, `review`, `debugging`.

## Merge queue

The queue of completed cook work waiting to be merged back into the main branch. Work enters the merge queue after passing the quality gate (in supervised/manual mode) or immediately (in auto mode).

## Mise

Short for *mise en place*. The project snapshot the chef reads before scheduling -- current state of the backlog, active orders, recent session history, and project context. Stored at `.noodle/mise.json`.

## Orders

Work assignments written by the chef. Each order specifies a task, the skill to run, and routing metadata (model, tags). Stored at `.noodle/orders-next.json`. Cooks pick up orders and execute them.

## Routing

The system that decides which LLM provider and model handle each task. Routing has global defaults and per-tag overrides configured in `.noodle.toml`. See [Configuration](/reference/configuration#routing).

## Schedule field

The `schedule` field in a skill's YAML frontmatter. A natural-language description of when the skill should fire. The chef reads it and decides when conditions are met. Skills with a schedule field are [task types](#task-type); skills without one are [general skills](#general-skill).

## Skill

A markdown file (`SKILL.md`) that teaches an agent how to do something. The body is a prompt. The frontmatter is metadata. Skills are Noodle's only extension point. See [Skills](/concepts/skills).

## Stages

Phases of the cook loop: schedule, execute, merge. Work flows through stages sequentially. The web UI shows stage status for each active order.

## Tags

Labels attached to skills or backlog items that control [routing](#routing). A tag like `mechanical` or `review` can override the default model, sending cheap work to a fast model and complex work to a capable one.

## Task type

A skill with a `schedule` field. Task types run autonomously in the loop -- the chef evaluates their schedule conditions and fires them when appropriate. Compare with [general skill](#general-skill).

## Worktree

A git worktree. Each cook works in its own worktree so concurrent agents never conflict. Worktrees are created automatically and cleaned up after merge.
