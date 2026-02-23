Back to [[plans/23-task-type-skill-suite/overview]]

# Phase 11: TUI Interactive Planning — Chef Chats with Sous-Chef

## Goal

Add an interactive planning session to the TUI where the Chef (human) chats with the sous-chef (LLM) to create and refine plans, instead of one-shot steering. This replaces the current model where the chef writes a prompt and the cook session produces a plan autonomously.

## Current State

- Planning is one-shot: the chef writes a todo item, the prioritize skill schedules a plan cook session, the cook writes a plan and commits it
- The chef can only intervene after the fact (review the plan, approve or request changes)
- No interactive back-and-forth during plan creation

## Design Intent

The chef should be able to:
1. Start a planning session from the TUI (select a todo item or describe new work)
2. Chat with the sous-chef about scope, approach, constraints, tradeoffs
3. See the plan take shape incrementally — phases added, reordered, refined
4. Approve the plan when satisfied, which commits it and schedules execution

This is the "autonomy dial" applied to planning: the chef can be hands-off (one-shot cook session) or hands-on (interactive TUI chat).

## Changes

### Chat app UX
- The planning session should look like a **chat app** (Discord/Slack style)
- Agent text responses are the primary content — displayed as chat messages
- **Tool calls are collapsed** — show a compact summary (e.g. "Read file: config.go") that can be expanded, but don't show raw tool XML or full output by default
- File writes and edits show inline diffs when expanded
- The focus is on the conversation flow, not the machinery

### AskUserQuestion forwarding
- When the agent calls `AskUserQuestion` (or the Codex equivalent), the TUI renders it as an **interactive form inline in the chat**
- The user responds directly in the chat — options as buttons, free text as an input field
- The response is sent back to the agent as a chat message
- This is critical for the plan skill's Step 2 (scope clarification) to work interactively

### TUI planning mode
- Add a "Plan" action in the TUI that starts an interactive planning session
- The session uses the plan skill in interactive mode (AskUserQuestion available)
- Plan files are written incrementally as the conversation progresses

### Integration with cook loop
- When the chef approves, the plan is committed and the prioritize skill can schedule its phases
- Each phase already has a Routing section (from Phase 10) so the scheduler knows what model to use

## Data Structures

- TUI planning session state: conversation history, current plan directory, phase list
- Plan output: same format as Phase 9 (brain/plans/ with overview.md + phase files)

## Verification

- TUI shows "Plan" action for unplanned todo items
- Starting a planning session opens a chat interface with Discord/Slack-like UX
- Tool calls are collapsed by default, expandable on click
- AskUserQuestion renders as interactive form inline (options as buttons, free text as input)
- Chat produces plan files in brain/plans/
- Approving the plan commits it
- Committed plan appears in next mise brief and can be scheduled
