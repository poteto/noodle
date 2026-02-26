# Encode Lessons in Structure

**Principle:** Encode recurring fixes in mechanisms (tools, code, metadata, automation) rather than textual instructions. Every error, human correction, and unexpected outcome is a learning signal — capture it, route it, and close the loop.

## Why

Textual instructions ("do not use subagents", "always run fmt") are routinely ignored. They require the reader to notice, remember, and comply. Structural mechanisms — lint rules, frontmatter flags, runtime reads, automation scripts — enforce the rule without cooperation.

## Pattern

When you catch yourself writing the same instruction a second time:

1. Ask: can this be a lint rule, a metadata flag, a runtime check, or a script?
2. If yes, encode it. Delete the instruction.
3. If no (genuinely requires judgment), make the instruction more prominent and add an example of the failure mode.

**Corollary — don't paper over symptoms.** If the fix is structural, ONLY use the structural fix. The instruction IS the symptom — if you're writing "don't do X" in a prompt, ask whether you can make X impossible instead.

## Feedback Loop

- **Capture every correction.** When the human intervenes or tests fail, decide if it's a one-off or a pattern. If it can recur, record the fix.
- **Route to the right layer.** A one-off → brain note. A recurring fix → skill or lint rule. A systemic issue → principle.
- **Close the loop.** Don't only record — apply now or create a concrete todo.

## Anti-Patterns

- **Acknowledging without recording.** "I'll keep that in mind" does not persist across sessions.
- **Recording without routing.** A brain note about a lint rule that should exist is wasted unless the lint rule gets implemented.
- **Fixing without generalizing.** Fixing one instance while leaving the recurring pattern intact.

## Relationship to Other Principles

[[principles/never-block-on-the-human]] says agents should proceed autonomously. This defines what happens when autonomy goes wrong: the system learns and hardens.

See also [[principles/never-block-on-the-human]]
