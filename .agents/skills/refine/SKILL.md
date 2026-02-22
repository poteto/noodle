---
name: refine
description: Clarify and sharpen todo items in brain/todos.md by asking the user targeted questions about unclear entries, then rewriting them as precise, actionable prompts. Use when the user says "refine", "refine todos", "clarify todos", "sharpen my todos", or wants to make their backlog items clearer.
---

# Refine

Read `brain/todos.md` and refine unclear items into actionable prompts through targeted questioning.

## Workflow

**Use Tasks to track progress.** Create a task for each step below (TaskCreate), mark each in_progress when starting and completed when done (TaskUpdate). Check TaskList after each step.

1. **Read** `brain/todos.md` and review items under `## Open`.
2. **Triage** each item: is it clear enough to act on without further context? A clear item specifies *what* is wrong, *where* it happens, and *what the fix should be*. Skip items that already meet this bar.
3. **Ask** clarifying questions about unclear items using `AskUserQuestion`. Group related questions — aim for one round of questions, not a back-and-forth. Good clarifying questions target:
   - **What** — what exactly is the problem or desired behavior?
   - **Where** — which part of the app, which component, which flow?
   - **How** — how should it work instead? What does "fixed" look like?
4. **Rewrite** unclear items in-place using `Edit`, incorporating the user's answers. Each rewritten item should be a self-contained prompt that another agent could act on without additional context.
5. **Remove** any items the user says are already done or no longer relevant.
6. **Report** a summary of what changed.
