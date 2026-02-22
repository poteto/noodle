Back to [[plans/01-noodle-extensible-skill-layering/overview]]

# Phase 1: Clone the Original Project into Noodle Repo

## Goal
Create `~/code/noodle/` by duplicating the original project, giving us all the dev tooling (Claude settings, skills, hooks, brain) as a starting point.

## Changes
- Copy the entire original project to `~/code/noodle/` (file copy, not git clone — no history carried over).
- **Do NOT `git init` yet.** No commits until all parent-project files are pruned (phase 4). This ensures the parent project's private code never appears in Noodle's git history.
- Delete the `.git/` directory from the copy (leftover from the original project's repo).
- Move `noodle/` to `old_noodle/` — this is the reference codebase for the rewrite, not the active source.

## Data Structures
- No new types. Pure file operations.

## Verification

### Static
- `~/code/noodle/` exists with all files from the original project
- No `.git/` directory exists yet
- `old_noodle/` contains the original Go source
- The parent project is completely untouched
