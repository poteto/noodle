# Suspect State Before Code

**Principle:** When a system works in isolation but fails after a restart, investigate persistent state before tracing code paths.

## Why

Code doesn't change between runs. State does. When the symptom is "nothing happens after restart," the cause is almost always stale or invalid persistent state being loaded into a context where it no longer makes sense. Tracing code paths for a state bug wastes enormous time because the code is correct — it's the data that's wrong.

## Pattern

- **Weight timeline evidence first.** When the user gives a concrete action sequence ("defer, quit, restart, stall"), treat that sequence as primary debug evidence before exploring unrelated startup paths.
- **Use "reset unwedged it" as a strong signal.** If clearing a state file restores behavior, prioritize persisted-state validation/sanitization as the first fix path.
- **Inventory persistent artifacts first.** Before reading any code, list everything that survives a restart: config files, caches, lock files, serialized state, log files. These are your suspects.
- **Check for staleness.** Does the loaded state assume a context (queue composition, active sessions, epoch) that no longer holds? Stale references to items that moved, ranks that changed, or sessions that died are the most common culprit.
- **Test by clearing.** The fastest hypothesis test is often: clear the suspect state file and restart. If the system works, you found it.
- **Don't read the state file as history.** When you open a state file, ask "could this data itself be the problem?" not "what sequence of actions produced this data?"

## Anti-Pattern: Bottom-Up Code Tracing

Reading function after function looking for a logic error when all tests pass is a signal you're looking in the wrong place. If the code is correct in tests but broken in production, the difference is the data it's operating on — not the code itself.

## Relationship to Other Principles

Extends [[principles/fix-root-causes]]: the root cause of a restart bug is usually in the persisted state, not the code that reads it.

Extends [[principles/observe-directly]]: observe the actual state being loaded, not just the code that loads it.
