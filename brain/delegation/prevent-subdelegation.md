# Preventing Manager Subdelegation

## Problem

Managers consistently delegate to subagents (codex workers, Task tool) even when explicitly told not to. Standard instructions like "do not use subagents" are routinely ignored.

## Solution: Direct Mode

The manager skill supports `MODE: direct`. Include this at the top of the prompt and the manager will work directly instead of spawning workers.

The director skill's compose prompts section documents when to use this: verification managers, recovery managers, and small tasks where worker delegation overhead isn't justified.

## Current Status: Not Working

Instruction-based `MODE: direct` failed because (a) the dispatch mechanism's polling failure silently dropped prompt delivery, so managers never saw the instruction, and (b) the `allowed-tools` list includes delegation tools (`Task`, `Skill`, `SendMessage`, `TeamCreate`) — models reach for available tools as default behavior.

**What doesn't work (even when the prompt IS delivered):**
- Prose paragraphs telling the model not to delegate
- Single-mention prohibitions
- Relying on the model to notice a flag at the top of a long prompt
- Implicit tool restrictions (must be explicit)

**Fix (implemented):** The `operator` agent (`.agents/agents/operator.md`) has `Task`, `Skill`, `TeamCreate`, `TeamDelete`, and `SendMessage` removed from `allowed-tools`. The director dispatches it via `noodle dispatch --agent operator`. This is structural — the delegation tools don't exist in the agent's environment, so no instruction-following is required.

This pattern is the canonical example of [[principles/encode-lessons-in-structure]] — three instruction-based failures before a structural fix.
