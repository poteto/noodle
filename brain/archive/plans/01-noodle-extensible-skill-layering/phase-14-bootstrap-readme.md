Back to [[archive/plans/01-noodle-extensible-skill-layering/overview]]

# Phase 14 — Bootstrap + README

## Goal

Write the bootstrap skill and the project README. The bootstrap skill is the entry point for new users — it replaces `noodle init` with an intelligent agent that can adapt to what already exists in the project. The README is the first thing a user sees and must get them from zero to a running Noodle setup in minutes.

Use the `skill-creator` skill when writing the bootstrap skill.

## Implementation Notes

This phase will be implemented by Codex. Keep it simple:

- **No over-engineering.** The bootstrap skill should be a clear, focused SKILL.md — not an exhaustive reference manual. The README should be direct and practical.
- **No backwards compatibility.** This is a greenfield build — there's nothing to be backwards-compatible with.
- **Prefer markdown fixture tests for parser/protocol flows.** Keep input and expected output in the same anonymized `*.fixture.md` file under `testdata/`, and use this pattern wherever practical.
- **Runtime verification in a fake project is required.** After static checks pass, run Noodle in a fresh temporary git repo under `/tmp` and verify the phase behavior there before closing.

## The Bootstrap Skill

The bootstrap skill is the primary way new users adopt Noodle. There is no `noodle init` Go command — the bootstrap skill handles all setup because it can make intelligent decisions about what to create based on what already exists.

### Installation flow

1. User tells their agent: "set up Noodle" and points at the GitHub repo
2. Agent reads the README, which has instructions to install the bootstrap skill
3. Agent installs the bootstrap skill and runs it
4. The bootstrap skill:
   - Creates `~/.noodle/` and `~/.noodle/skills/`
   - Copies default skills from the repo to `~/.noodle/skills/`
   - Builds the Go binary (`go build`) and places it on PATH (or in `~/.noodle/bin/`)
   - Creates `noodle.toml` in the project with sensible defaults
   - Detects the platform-specific paths for `.claude/` and `.codex/` directories and writes them to `agents.claude_dir` and `agents.codex_dir` in config
   - Creates `.noodle/adapters/` with default sync scripts
   - Adds `.noodle/` to the project's `.gitignore`
   - Detects what the project already has and adapts accordingly

### Adaptation logic

The bootstrap skill inspects the project for what it can detect automatically (agent CLI directories, existing config, existing brain/), then **asks the user** for decisions it can't infer.

**Auto-detected (no user input needed):**

- **Has `.claude/`?** — Detect path, write to `agents.claude_dir`
- **Has `.codex/`?** — Detect path, write to `agents.codex_dir`
- **Has existing `noodle.toml`?** — Validate and repair rather than overwrite
- **Has `brain/`?** — Skip brain scaffolding, use existing structure

**Asked via `AskUserQuestion` (interactive mode):**

The skill uses `AskUserQuestion` to ask the user how they manage tasks and plans. Examples:

- "Where do you track tasks?" — options: `brain/todos.md` (default markdown), GitHub Issues, Linear, Jira, Other
- "Do you use a plans/specs system?" — options: `brain/plans/` (default markdown), No plans system, Other

Based on the user's answers, the skill creates the appropriate adapter config and scripts. If the user picks GitHub Issues, the skill writes a `gh`-based sync script. If they pick Linear, it scaffolds a placeholder script with instructions. If they pick the defaults, it creates `brain/todos.md` and `brain/plans/` with example content.

**Non-interactive fallback (Codex, headless, `-p` mode):**

When `AskUserQuestion` is unavailable (Codex sessions, `claude -p`, CI), the skill defaults conservatively: `brain/todos.md` for backlog, no plans adapter. It does not try to infer the task system from directory heuristics (e.g., `.github/` does not imply GitHub Issues is the backlog). After setup, it emits a clear note: "Backlog adapter defaults to brain/todos.md. Run bootstrap interactively or edit noodle.toml to configure a different task system." The user can always reconfigure later — bootstrap just needs to produce a working starting point, not a perfect one.

### What the skill teaches the agent

The bootstrap `SKILL.md` teaches the agent:
- The directory layout (`noodle.toml`, `.noodle/`, `~/.noodle/`)
- How to detect the platform and locate agent CLI directories
- The config schema and sensible defaults for each field
- How to write POSIX-compatible adapter scripts
- How to verify the setup works (`noodle status`, `noodle skills list`)

## README

The README is the project's front door. It should be concise and get a user running quickly.

### Structure

1. **What is Noodle** — One paragraph. AI coding framework, skills as extension point, kitchen brigade model.
2. **Quick Start** — The fastest path from zero to running:
   - Prerequisites (Go, tmux, Claude Code or Codex; Windows requires WSL)
   - Install the bootstrap skill (one command)
   - Run it ("set up Noodle in this project")
   - Start Noodle (`noodle start`)
3. **How it works** — Brief overview of the kitchen brigade: chef (you), sous chef (prioritizes), cooks (do the work), taster (reviews). Link to docs for details.
4. **Configuration** — `noodle.toml` at project root. Show the minimal config and link to full reference.
5. **Skills** — Skills are the only extension point. How to override defaults, where to put project-level skills.
6. **Adapters** — How to connect your backlog/plan system. Show the adapter pattern (skill + scripts).
7. **CLI Reference** — Table of commands with one-line descriptions.
8. **Contributing** — How to build from source, run tests, project structure overview.

### Tone

Direct, practical, no marketing language. Show, don't tell. The README assumes the reader is a developer who wants to understand what Noodle does and start using it, not be sold on it.

- **Overview + phase completeness check (required).** Before closing the phase, review your changes against both the plan overview and this phase document. Verify every detail in Goal, Changes, Data Structures, and Verification is satisfied; track and resolve any mismatch before considering the phase done.
- **Worktree discipline (required).** Execute each phase in its own dedicated git worktree.
- **Commit cadence (required).** Commit as much as possible during the phase: at least one commit per phase, and preferably multiple commits for distinct logical changes.
- **Main-branch finalize step (required).** After all verification and review checks pass, merge the phase worktree to `main` and make sure the final verified state is committed on `main`.

## Changes

- **`bootstrap` skill** — `SKILL.md` + `references/` with config schema reference, adapter script templates, platform detection guidance.
- **`README.md`** — Project README at repo root.

## Data Structures

No Go types. The bootstrap skill is a `SKILL.md` document. The README is markdown.

## Verification

### Static
- Bootstrap skill has a valid `SKILL.md`
- README renders correctly (no broken links, valid markdown)
- README quick start instructions are accurate (commands actually work)

### Runtime
- Fresh project: invoke bootstrap skill → `~/.noodle/` created, binary built, `noodle.toml` created, `.noodle/` populated, `.gitignore` updated, `noodle status` works
- Existing project with `brain/`: bootstrap detects it and adapts config (doesn't overwrite)
- Existing project with `noodle.toml`: bootstrap validates and repairs rather than overwriting
- README quick start: follow the steps from scratch on a clean machine → working Noodle setup
- `noodle skills list` shows all default skills after bootstrap
- Non-interactive bootstrap (no `AskUserQuestion`): falls back to defaults (`brain/todos.md`, no plans adapter), produces working config without user input
