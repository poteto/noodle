Back to [[plans/15-bootstrap-onboarding/overview]]

# Phase 5: INSTALL.md

## Goal

`INSTALL.md` is the entry point for any agent setting up Noodle. It covers binary installation, gives a high-level vision of what Noodle is and how it thinks, then points to the noodle skill for configuration details.

## Changes

- **`INSTALL.md`** (new, repo root) — structured for agents to follow sequentially:
  1. **Prerequisites** — tmux, Claude Code or Codex CLI
  2. **Install binary** — `brew install poteto/tap/noodle` for macOS, `go install` as fallback
  3. **What is Noodle** — high-level vision (2-3 paragraphs):
     - Everything is a file: queue.json, mise.json, verdicts, control.ndjson — no hidden state
     - Skills are the single extension point — they drive everything, they're how you customize Noodle
     - Agent-pilled: agents do the work, humans review and plan. Humans add value through strategy and judgment, not execution. Let the agent do almost everything.
     - Kitchen brigade: Chef (human), Sous Chef (scheduler), Cook (implementer), Taster (reviewer)
  4. **Set up your project** — install the noodle skill to `.agents/skills/noodle/`, read it for configuration guidance. The skill covers config, adapters, hooks, and everything else.
  5. **Start** — `noodle start`. First run scaffolds missing directories and config. The agent handles the rest.
  6. **Learn more** — point to `PHILOSOPHY.md` for the deeper rationale (brain, self-learning, autonomy)

- **`README.md`** — update Getting Started to point at `INSTALL.md` with the one-liner: `brew install poteto/tap/noodle && noodle start`

## Routing

| Provider | Model | Rationale |
|----------|-------|-----------|
| `claude` | `claude-opus-4-6` | Writing clear, agent-readable instructions with the right level of vision requires judgment |

## Verification

### Static
- `INSTALL.md` exists at repo root
- No broken links
- Mentions the noodle skill by name and path
- Does not duplicate config details that belong in the skill

### Runtime
- Give `INSTALL.md` to a fresh Claude Code session with no Noodle context. Can it follow the instructions to install and set up Noodle in a test project? This is the real test.
