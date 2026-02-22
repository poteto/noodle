# Encode Lessons in Structure

**Principle:** The system must observe itself, learn from every failure, and encode recurring fixes in mechanisms (tools, code, metadata, automation) rather than textual instructions. Every error, human correction, and unexpected outcome is a learning signal — capture it, route it, and close the loop.

## Why

A system that only improves when a human catches mistakes does not scale. Human attention is finite. If agents repeat the same errors across sessions, the human becomes a permanent debugging layer instead of a supervisor.

Textual instructions ("do not use subagents", "always run fmt") are routinely ignored by agents and humans alike. They require the reader to notice, remember, and comply. Structural mechanisms — lint rules, frontmatter flags, runtime reads, automation scripts — enforce the rule without cooperation.

## Self-Healing Loop

- **Capture every correction.** When the human intervenes — redirects an approach, fixes a bug, overrides a decision — log it as a todo, brain note, or skill update.
- **Capture every error.** When tests, builds, verification, or assumptions fail, decide whether it is a one-off or a recurring pattern. If it can recur, record the fix.
- **Route learnings to the right layer.** A one-off fix belongs in a brain note. A recurring fix belongs in a skill or lint rule. A systemic issue belongs in a principle. Match fix durability to problem durability.
- **Close the feedback loop.** Do not only record learnings; apply them now or create a concrete todo for the next round.
- **Observe patterns across sessions.** Brain notes, auto-memory, and todos are cross-session memory. Read them before starting and write back what this session learned.

## Evidence

- `prevent-subdelegation`: Three failed attempts at instruction-based solutions before `MODE: direct` baked enforcement into the skill itself.
- Lint rules: `go vet`, `staticcheck`, and custom linters enforce rules structurally. No one needs to remember.
- `gofmt`: "Never fix formatting manually" — the formatter is the mechanism.
- Skills read brain at runtime: Instead of hardcoding principle text (which drifts), a `Read` call fetches the authoritative source.
- Worktree script: "Replace manual workflow with deterministic script" — moved from rules agents follow to a script that can't be gotten wrong.

## Encoding Pattern

When you catch yourself writing the same instruction a second time:

1. Ask: can this be a lint rule, a metadata flag, a runtime check, or a script?
2. If yes, encode it. Delete the instruction.
3. If no (genuinely requires judgment), make the instruction more prominent and add an example of the failure mode.

**Corollary — don't paper over symptoms.** If the fix is structural, ONLY use the structural fix. Adding instructions on top of a mechanism is redundant noise:

- Lint enforces `no-non-null-assertion` → don't also tell workers "never use `!`"
- Concurrent merges cause data loss → don't add "merge sequentially" to two skill files; build a lockfile in the worktree script

The instruction IS the symptom. If you're writing "don't do X" in a prompt, ask whether you can make X impossible instead.

## Anti-Patterns

- **Acknowledging without recording.** "I'll keep that in mind" does not persist across sessions.
- **Recording without routing.** A brain note about a lint rule that should exist is wasted unless the lint rule gets implemented.
- **Fixing without generalizing.** Fixing one instance while leaving the recurring pattern intact.

## Relationship to Other Principles

This is not about *where* validation belongs ([[principles/boundary-discipline]]) or *testing behavior* ([[principles/verify-runtime]]). It's about the meta-question of how to make rules stick — the delivery mechanism for lessons learned.

[[principles/never-block-on-the-human]] says agents should proceed autonomously. This principle defines what happens when autonomy goes wrong: the system learns and hardens instead of waiting for intervention.

See also [[principles/never-block-on-the-human]], [[delegation/prevent-subdelegation]]
