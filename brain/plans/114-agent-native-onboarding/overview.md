---
id: 114
created: 2026-03-03
status: active
---

# Agent-Native Onboarding

## Context

Getting started with Noodle is hard. You need at least 2 skills (schedule + execute) to have a working loop, the noodle skill requires manual curl commands, and there's no in-product communication of what Noodle can do. New users hit a wall before they see value.

## Scope

**In scope:**
- `INSTALL.md` — agent-readable bootstrap document in the Noodle repo
- Remove `brain/` from `EnsureProjectStructure` — brain is brainmaxxing, not core Noodle
- Bootstrap backlog — seed `todos.md` with self-setup tasks the loop executes on first run
- CLI welcome message — post-first-run "what happened + next steps"
- `/onboarding` UI route — for brand-new installs
- Improved empty states across the UI (dashboard, feed, sidebar)
- README update linking to INSTALL.md

**Out of scope:**
- Separate human-readable getting-started guide (docs site covers this)
- `noodle init` CLI command (the agent IS the installer)
- Skill registry or package manager
- Starter skill pack / pre-simplified skill copies (the agent adapts from reference implementations)
- Adapters (GitHub Issues, Linear) — separate concern
- `brain/` directory creation (that's brainmaxxing's responsibility)

## Constraints

- INSTALL.md is agent-first: optimized for coding agents (Claude, Codex) to read and execute. Humans can read it but it's not the primary audience.
- Fully idempotent: running the install a second time on a configured project is a safe no-op.
- Cross-platform: the agent handles platform differences natively — no shell-specific commands needed.
- The agent adapts, not copies. INSTALL.md links to Noodle's own skills as reference implementations and annotates what matters for each. The agent reads them, understands the concepts, and writes skills tailored to the user's project, language, and workflow.
- No new CLI commands — the agent drives the install by reading INSTALL.md and using its own tools.
- No new directories or pre-simplified skill copies. The Noodle repo's existing `.agents/skills/` are the reference implementations.

## Alternatives Considered

1. **CLI wizard (`noodle init --guided`)** — interactive prompts for provider, model, project type. Rejected: adds a new command, doesn't leverage the agent that's already present, requires maintaining wizard logic.
2. **Template repo (`noodle init --from template`)** — clone a starter repo. Rejected: heavy, couples setup to a specific project structure, doesn't adapt to existing projects.
3. **Starter skill pack (`starter-pack/`)** — pre-simplified skill copies for new users. Rejected: creates a second skill surface that drifts from the real skills, pre-chews what the agent should understand and adapt itself, and doesn't account for the user's specific project/language/workflow.
4. **Agent-readable INSTALL.md with reference implementations (chosen)** — the user's coding agent reads INSTALL.md, studies Noodle's real skills as references, and writes tailored skills for the user's project. The agent decides and adapts. Zero new files to maintain, zero drift risk.

## Applicable Skills

- `unslop` — for INSTALL.md prose (phase 1). Agent-readable doesn't mean robotic — it should sound human.
- `frontend-design` — for the /onboarding route and empty states
- `go-best-practices` — for CLI welcome message changes in Go
- `react-best-practices` — for UI components

## Phases

1. [[plans/114-agent-native-onboarding/phase-01-install-md]]
2. [[plans/114-agent-native-onboarding/phase-02-cli-welcome]]
3. [[plans/114-agent-native-onboarding/phase-03-onboarding-route]]
4. [[plans/114-agent-native-onboarding/phase-04-empty-states]]
5. [[plans/114-agent-native-onboarding/phase-05-readme-update]]

## Verification

- `pnpm build`
- `pnpm check`
- `go test ./startup/... ./cmd/...`
- `pnpm --filter noodle-ui test`
- Manual: point a fresh Claude Code session at INSTALL.md and verify it bootstraps a working loop
- Manual: run `noodle start` on a fresh project, verify welcome message and UI onboarding
