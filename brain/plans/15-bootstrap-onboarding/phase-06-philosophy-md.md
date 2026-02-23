Back to [[plans/15-bootstrap-onboarding/overview]]

# Phase 6: PHILOSOPHY.md

## Goal

Document the deeper "why" behind Noodle's design. INSTALL.md says *what* Noodle is; PHILOSOPHY.md explains *why* it works this way. This is for agents and humans who want to understand the system deeply — not required for setup, but essential for getting the most out of Noodle.

## Changes

- **`PHILOSOPHY.md`** (new, repo root) — covers:
  1. **The brain** — an Obsidian vault that persists knowledge across sessions. Not a chat log — structured, topic-per-file memory that agents read before acting and write to after learning. The brain is committed to the repo so it evolves with the project.
  2. **Self-learning loop** — how reflect (post-session), meditate (periodic audit), and ruminate (conversation mining) feed discoveries back into the brain. The system gets smarter over time without human curation.
  3. **Why agent-pilled** — humans are the scarcest resource. Every step that blocks on human attention is waste. Agents should stay unblocked by making reasonable decisions, proceeding, and letting the human course-correct after the fact. The human's job is strategy and judgment — choosing what to build, reviewing what was built — not execution.
  4. **Autonomy as a dial** — trust is a spectrum. Full autonomy for trusted projects, review mode for most work, approval mode for high-stakes changes. The dial moves based on trust, not capability.
  5. **Everything is a file** — queue.json, mise.json, verdicts, control.ndjson. No hidden state, no databases. Files are debuggable, diffable, composable. Any tool can read them.
  6. **Skills as the single extension point** — no plugins, no hooks API, no configuration DSL. Skills are just SKILL.md files that teach agents how to do things. Users extend Noodle by writing skills.

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | Articulating design philosophy requires clear thinking and good writing |

## Verification

### Static
- `PHILOSOPHY.md` exists at repo root
- `INSTALL.md` links to it
- No contradictions with the noodle skill or existing brain principles

### Runtime
- Read PHILOSOPHY.md cold. Does it make the design feel inevitable rather than arbitrary? Does it explain the "why" without repeating the "how" from INSTALL.md?
