Back to [[plans/01-noodle-extensible-skill-layering/overview]]

# Phase 4: Verify Standalone Noodle Repo

## Goal
Confirm that `~/code/noodle/` is clean, self-contained, and contains no leaked parent-project artifacts. Initialize git with a clean history. The old code in `old_noodle/` builds and tests pass.

## Changes
- Verify `old_noodle/` builds: update its `go.mod` module path if needed so it compiles from the new location.
- Grep the entire repo for stale parent-project references (old module paths, parent-project-specific paths) and fix any remaining ones.
- Ensure `old_noodle/` tests pass (these will be reference material during the rewrite).
- **Audit for parent-project leaks:** Verify no parent-project-specific source code, product plans, private keys, or proprietary content remains. Only Noodle Go code (in `old_noodle/`), general-purpose dev skills, brain principles, and project config should be present.
- **Now `git init` and create the initial commit.** Since all parent-project files were deleted in phases 2-3 before any commit, the parent project's private code never exists in git history. Add a `.gitignore`.
- Add a `LICENSE` file (choose appropriate open-source license).
- Add a minimal `README.md` with project description.

## Data Structures
- No new types.

## Verification

### Static
- `go -C ~/code/noodle/old_noodle build ./...` — compiles
- `go -C ~/code/noodle/old_noodle test ./...` — all pass
- No grep hits for parent-project references in source files (excluding brain notes where the parent project is mentioned as context for Noodle's origin)
- No frontend files (`*.tsx`, `*.ts`, `*.css`, `*.html`) outside of `old_noodle/`
- `git -C ~/code/noodle log --oneline` — single clean initial commit
- `git -C ~/code/noodle log -p` — verify the initial commit diff contains no parent-project product code

### Runtime
- `go -C ~/code/noodle/old_noodle run . --help` — CLI renders help text
- The parent project's repo is completely untouched
