---
name: reflect
description: Reflect on the conversation and update the brain (Obsidian vault at brain/). Use when wrapping up a conversation, after making mistakes, after being corrected, or when significant new knowledge about the codebase has been gained. Also use when the user says "reflect", "update the brain", "remember this", or "save what you learned".
---

# Reflect

Review the conversation and persist learnings — to `brain/`, to skill files, or as workflow improvements.

## Process

**Use Tasks to track progress.** Create a task for each step below (TaskCreate), mark each in_progress when starting and completed when done (TaskUpdate). Check TaskList after each step.

1. **Read `brain/index.md`** to understand what notes already exist
2. **Scan the conversation** for:
   - Mistakes made and corrections received
   - User preferences and workflow patterns
   - Codebase knowledge gained (architecture, gotchas, patterns)
   - Tool/library quirks discovered
   - Decisions made and their rationale
   - Friction in skill execution, orchestration, or delegation
   - Repeated manual steps that could be automated or encoded
3. **Skip** anything trivial or already captured in existing brain files
4. **Route each learning** to the right destination (see Routing below)
5. **Update `brain/index.md`** if any brain files were added or removed

## Routing

Not everything belongs in the brain. Route each learning to where it will have the most impact.

### Structural enforcement check

Before routing a learning to `brain/`, ask: can this be a lint rule (`.oxlintrc.json`, clippy), a script, a metadata flag, or a runtime check? If yes, encode it structurally and skip the brain note. If no (genuinely requires judgment), proceed to brain/. See `brain/principles/encode-lessons-in-structure.md`.

### Brain files (`brain/`)

Codebase knowledge, delegation principles, gotchas — anything that informs future sessions. This is the default destination. Use the `brain` skill for writing conventions and the durability test.

### Skill improvements (`.agents/skills/<skill>/`)

If a learning is about how a specific skill should work — its process, its prompts, its edge cases — use the `skill-creator` skill to make the update. It provides guidelines for writing effective skill content and ensures changes follow established patterns.

Examples:
- A Codex prompting pattern that avoids sandbox failures → update `.agents/skills/codex/SKILL.md`
- A better worker prompt template → update `.agents/skills/manager/SKILL.md`
- A missing troubleshooting entry for the director → update `.agents/skills/director/SKILL.md`

### Orchestration workflow improvements

If the session revealed systemic issues in how work is orchestrated (director → manager → worker pipeline), consider whether the fix is:
- A **brain principle** (e.g., a new delegation heuristic) → `brain/delegation/`
- A **skill mechanics change** (e.g., director monitoring loop, manager review step) → the relevant skill file
- A **new skill opportunity** (e.g., a recurring multi-step workflow that could be encoded as a skill) → note it in the summary for the user to decide

### Backlog items (`brain/todos.md`)

If a finding requires follow-up work that can't be done during reflection — such as a noodle bug, a skill that needs a non-trivial rewrite, or a tooling gap that needs investigation — file it as a todo using the `todo` skill. This captures actionable work that would otherwise be forgotten between sessions.

## Summary

After routing all learnings, present a summary to the user:

```
## Brain Updates
- [list files created/updated with one-line descriptions]

## Skill Updates
- [list skill files modified with one-line descriptions]

## Todos Filed
- [any backlog items added for follow-up work]

## Proposed Improvements
- [any workflow, orchestration, or new-skill suggestions not yet applied]
```
