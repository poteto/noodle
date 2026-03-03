Back to [[plans/114-agent-native-onboarding/overview]]

# Phase 1: INSTALL.md

## Goal

Write the agent-readable bootstrap document. The agent reads it, understands what Noodle is and how the skill system works, then sets up a working loop tailored to the user's project.

## Changes

**New file `INSTALL.md`** at repo root.

The document has two parts: **context** (what Noodle is, how the loop works, what skills are) and **steps** (what the agent does to set up the project). Each step has an idempotency check.

### Part 1: Context

Concise explanation the agent needs to make good decisions:
- What Noodle is (skill-based agent orchestration, kitchen brigade model)
- How the loop works (schedule → execute → quality → reflect → merge)
- What skills are (SKILL.md with frontmatter, the `schedule` field makes a skill a task type)
- What a working loop requires (at minimum: a schedule skill and one task-type skill like execute)

### Part 2: Steps

1. **Install binary** — detect platform, install via brew (macOS) or curl (Linux/Windows). Check: `noodle --version` succeeds → skip.
2. **Install noodle skill** — fetch the noodle skill files from GitHub raw URLs into `.agents/skills/noodle/`. This skill teaches the agent how to use the CLI and write skills going forward. Check: `.agents/skills/noodle/SKILL.md` AND `references/` files all exist → skip. If only SKILL.md exists but references are missing, re-fetch the missing files (handles interrupted installs).
3. **Configure** — if `.noodle.toml` doesn't exist, ask the user which provider they use (Claude or Codex) and generate config. If it exists, skip.
4. **Add .noodle/ to .gitignore** — grep first, append only if absent.
5. **Seed the backlog** — if `todos.md` doesn't exist, create it with bootstrap tasks. If `todos.md` exists but `.agents/skills/schedule/` doesn't, the agent should still guide the user through skill creation (skip the file write, proceed to step 6). The seeded tasks are ordered by priority — skills first, exploration second:
   - `[ ] Write a schedule skill and execute skill for this project` — the agent reads the user's past conversation logs (`.claude/`, `.codex/`) to understand the project, then writes skills adapted to their workflow, language, and tools.
   - `[ ] Browse github.com/poteto/noodle/.agents/skills/ and find skills that might be useful to adapt to this codebase`
   The seeded backlog is the onboarding — the loop bootstraps itself.
6. **Ask about brainmaxxing** — ask the user if they want brainmaxxing installed. If yes, follow the brainmaxxing install instructions from https://github.com/poteto/brainmaxxing. This adds the `brain/` vault, reflect/meditate/ruminate skills, and persistent memory across sessions.
7. **First run** — tell the user to run `noodle start`.

### Skill authoring guidance

INSTALL.md links to Noodle's own skills as reference implementations with annotations about what's important to define for each:

**Schedule skill** — link to Noodle's schedule skill. Annotate:
- The `schedule` frontmatter field — when should the scheduler re-evaluate?
- Reading the backlog source (usually `todos.md`, but could be any adapter)
- Writing valid `orders-next.json` — link to the orders schema
- Routing stages to the right provider/model based on the user's config

**Execute skill** — link to Noodle's execute skill. Annotate:
- The `schedule` frontmatter field — when is work ready?
- Verification strategy for the specific codebase (what tests to run, what build command)
- Commit conventions

This guidance lives in INSTALL.md so the bootstrap task ("write skills for this project") has everything it needs. The agent reads the references, understands the pattern, and writes skills that fit. Quality and reflect skills come via brainmaxxing if the user opts in.

### Tone and structure

Direct, imperative. Each step is a clear instruction with a skip condition. The context section teaches concepts; the steps section is mechanical except where the agent needs to make decisions (writing skills, asking about provider).

**Agent interaction:** Where steps say "ask the user," use natural language — not specific tool names like `AskUserQuestion`. Different agents have different interaction primitives. The agent will use whatever it has.

**Key principle:** The agent adapts, not copies. Reference implementations are for understanding, not for verbatim reproduction. The agent should ask itself: "what does this user's project need?" and write skills that fit.

## Data Structures

- No code types — this is a markdown document.

## Design Notes

**Transport:** INSTALL.md provides raw GitHub URLs for reference skills (e.g., `https://raw.githubusercontent.com/poteto/noodle/main/.agents/skills/schedule/SKILL.md`). The agent fetches and reads these to understand the pattern, then writes its own version. The noodle skill (step 2) is the only file copied verbatim — it's a reference manual, not project-specific.

**Version pinning:** Raw GitHub URLs should reference a tagged release (e.g., `.../v1.2.0/...`) matching the installed binary version, not `main`. This prevents version skew where the binary expects one skill shape and the reference shows another. If the installed version can't be determined, fall back to `main`.

**Idempotency:** Check-before-write for file copies (noodle skill) and config. Skills the agent writes are project-specific — if they already exist, the agent skips (the user may have customized them). `.gitignore` uses grep-before-append. `.noodle.toml` skipped if exists.

**Why not a starter-pack directory?** Pre-simplified skill copies create a second surface that drifts from the real skills, don't account for the user's project specifics, and pre-chew what the agent should understand itself. The agent reading real skills and adapting is both simpler (zero new files to maintain) and better (tailored output).

## Applicable Skills

- `unslop` — run on the final INSTALL.md prose, but with a constraint: precision over warmth. The document must be unambiguous for agents (exact paths, explicit skip conditions, no hedging) while still reading like a human wrote it. Unslop the filler and AI patterns; preserve the structural clarity.

## Routing

- Provider: `claude`, Model: `claude-opus-4-6` (writing quality agent-readable instructions requires judgment)

## Verification

### Static
- INSTALL.md exists at repo root
- All referenced GitHub raw URLs resolve to real files
- Reference skill links point to current skill paths

### Runtime
- Point a fresh Claude Code session at INSTALL.md in a new project. Verify it:
  - Installs the binary (or recognizes it's already installed)
  - Writes project-adapted skills into `.agents/skills/`
  - Generates `.noodle.toml` with the user's chosen provider
  - Adds `.noodle/` to `.gitignore`
- Run the same agent on the same project again. Verify it skips all steps (idempotent).
- Verify the generated schedule skill produces valid `orders-next.json` when run.
