---
name: bootstrap
description: Set up Noodle in a project from scratch or repair an incomplete setup. Use when a user asks to install, initialize, or configure Noodle.
---

# Bootstrap

Use this skill when a project needs a working Noodle setup.

Read these references before writing files:
- `references/config-schema.md`
- `references/adapter-script-templates.md`
- `references/platform-detection.md`

## Goals

1. Install user-level Noodle runtime (`~/.noodle/`).
2. Create or repair project-level setup (`noodle.toml`, `.noodle/adapters/`, `.gitignore`).
3. Ensure required skills are available in agent skill directories.
4. Adapt to existing project state instead of overwriting.
5. Verify setup with real commands.

## Workflow

1. Detect runtime mode:
   - Interactive mode: `AskUserQuestion` is available.
   - Non-interactive mode (Codex, CI, `claude -p`): use safe defaults with no user prompts.
2. Inspect current state:
   - Existing `noodle.toml`
   - Existing `brain/`
   - Existing `.claude/` and `.codex/`
   - Existing `.gitignore`
3. Set up global user paths:
   - Create `~/.noodle/`, `~/.noodle/bin/`, and `~/.noodle/skills/`.
   - Copy default skills from repository `skills/` into `~/.noodle/skills/` (do not delete unrelated custom skills).
   - Build Noodle binary into `~/.noodle/bin/noodle`.
4. Handle missing skills:
   - If Noodle reports a missing skill, install that skill from `https://github.com/poteto/noodle`.
   - Use an ephemeral temp clone (no cache): clone to a temp directory, copy required `skills/<name>` directories, then delete the temp clone.
   - Install target policy:
     - If `~/.agents` exists, install to `~/.agents/skills/`.
     - If `~/.claude` exists, install to `~/.claude/skills/`.
     - If both exist, install to both.
     - If neither exists, create `~/.agents/skills/` and install there.
   - Never overwrite unrelated existing custom skills.
5. Create or repair project setup:
   - Ensure `noodle.toml` exists and contains valid defaults from `references/config-schema.md`.
   - Detect agent directories using `references/platform-detection.md` and set:
     - `agents.claude_dir`
     - `agents.codex_dir`
   - Ensure `.noodle/adapters/` scripts exist and are executable, using templates from `references/adapter-script-templates.md`.
   - Ensure `.noodle/` is listed in `.gitignore` once.
6. Adapter decision:
   - Interactive:
     - Ask where tasks are tracked.
     - Ask whether a plans system is used.
     - Configure adapters to match those answers.
   - Non-interactive fallback:
     - Configure backlog as `brain/todos.md`.
     - Do not configure plans adapter.
     - Print this note in the final summary:
       - `Backlog adapter defaults to brain/todos.md. Run bootstrap interactively or edit noodle.toml to configure a different task system.`
7. Verify:
   - `~/.noodle/bin/noodle commands`
   - `~/.noodle/bin/noodle skills list`
   - `~/.noodle/bin/noodle status`
   - If asked for runtime verification: `~/.noodle/bin/noodle start --once`

## Interactive Questions

Only ask questions you cannot infer from files.

1. `Where do you track tasks?`
   - `brain/todos.md` (default markdown)
   - `GitHub Issues`
   - `Linear`
   - `Jira`
2. `Do you use a plans/specs system?`
   - `brain/plans/` (default markdown)
   - `No plans system`
   - `Other`

If the user picks external systems, scaffold scripts with explicit TODO markers and keep the config runnable.

## Existing State Rules

- If `noodle.toml` exists: repair missing or invalid fields; do not replace wholesale.
- If `brain/` exists: do not regenerate the vault.
- If adapter scripts already exist: do not overwrite custom logic without clear need.
- If `.gitignore` exists: append `.noodle/` only if missing.

## Output Summary

At the end, report:

1. What was created or repaired.
2. Which adapter choices were applied.
3. Exact commands for the user to run next.
