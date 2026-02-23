---
id: 15
created: 2026-02-23
status: draft
---

# Bootstrap & Onboarding

## Context

Noodle has no install story. The README assumes a cloned repo and a pre-built binary. Two disconnected skill registries exist (Noodle's resolver vs Claude Code's) with no bridge. The bootstrap skill planned in Plan 1 Phase 14 was never implemented.

The goal: a user shares their GitHub link with an agent, and the agent figures out how to install and configure Noodle for their project. No manual steps beyond "set up Noodle."

## Design

Two actors handle onboarding — each does what it's good at:

1. **`noodle start`** — handles mechanical setup. First run detects missing directories and config, scaffolds them. No separate `init` command.
2. **The user's agent** — handles everything requiring judgment: installing skills, choosing adapters, setting up hooks. Guided by INSTALL.md and the noodle skill.

This follows the agent-pilled philosophy: agents do agent work, the binary does mechanical work.

### Document hierarchy

- **`INSTALL.md`** — the agent's entry point. How to install the binary, the vision of Noodle, points to the noodle skill for configuration.
- **`PHILOSOPHY.md`** — the deeper "why." Brain, self-learning, agent-pilled rationale.
- **Noodle skill** — auto-generated operational reference. Config schema, CLI, adapter guides, hook scripts. CI test fails if it drifts from source code.

## Scope

**In scope:**
- Homebrew tap for macOS binary distribution (Linux/Windows deferred)
- First-run setup in `noodle start` (directories, minimal config)
- Auto-generated noodle skill with CI snapshot guard
- `INSTALL.md` and `PHILOSOPHY.md` at repo root
- Integration test for the full onboarding flow

**Out of scope:**
- Linux/Windows package managers (follow-up)
- `go install` support (todo #7, separate)
- `noodle init` command (eliminated — `noodle start` handles it)
- Adapter implementations beyond documentation (agent sets these up)

## Constraints

- Cross-platform: POSIX shell only, no bash 4+ features
- The noodle skill must be auto-generated from source, not hand-written
- First-run setup must be idempotent — `noodle start` is always safe to run
- Skills live in `.agents/skills/` — single location for both Noodle resolver and Claude Code

## Alternatives Considered

1. **`noodle init` CLI command:** Dedicated setup command with AskUserQuestion prompts. Rejected — splits setup across two commands, duplicates self-heal logic, and does agent work (adapter choice, skill installation) in Go code instead of letting the agent do it.
2. **Bootstrap as a skill (Plan 1 Phase 14 design):** Agent reads README, installs bootstrap skill, skill drives setup. Rejected — circular: requires knowing about skills to install the skill that teaches about skills.
3. **`noodle start` + agent-driven setup (chosen):** Binary handles mechanics, agent handles judgment. Simple, idempotent, no circularity, agent-pilled.

## Applicable Skills

- `noodle` — will be auto-generated in Phase 4
- `skill-creator` — use when designing the skill template (Phase 4) and structuring INSTALL.md (Phase 5)
- `go-best-practices` — use for Go implementation (Phases 3, 4)
- `unslop` — use when writing INSTALL.md and PHILOSOPHY.md (Phases 5, 6)

## Phases

1. [[plans/15-bootstrap-onboarding/phase-01-scaffold]] — Delete stale Plan 1 Phase 14 artifact
2. [[plans/15-bootstrap-onboarding/phase-02-homebrew-tap-and-binary-distribution]] — Homebrew tap + GoReleaser
3. [[plans/15-bootstrap-onboarding/phase-03-first-run-setup]] — First-run scaffolding in `noodle start`
4. [[plans/15-bootstrap-onboarding/phase-04-auto-generated-noodle-skill]] — Auto-generated skill + CI guard
5. [[plans/15-bootstrap-onboarding/phase-05-install-md]] — Agent entry point + vision overview
6. [[plans/15-bootstrap-onboarding/phase-06-philosophy-md]] — Brain, self-learning, agent-pilled rationale
7. [[plans/15-bootstrap-onboarding/phase-07-integration-test]] — End-to-end onboarding test

## Verification

```bash
go test ./... && go vet ./...
sh scripts/lint-arch.sh
```

End-to-end: `noodle start --once` in a clean temp directory succeeds after the agent follows INSTALL.md.
