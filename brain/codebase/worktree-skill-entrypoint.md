# Worktree Skill Entrypoint

- In this repo, do not direct agents to run raw `git worktree` commands.
- Canonical path is Skill(worktree), which routes through the Go CLI (`noodle worktree`).
- For policy wording, refer to "linked worktree created via Skill(worktree)" rather than "`git worktree add`".
- User preference (2026-02-20): always use a linked worktree, including planning/documentation changes that might otherwise feel small.
- Codex correction (2026-02-23): for substantial implementation requests, create the linked worktree before any code edits and keep commit splits per logical change.

See also [[codebase/worktree-gotchas]], [[codebase/worktree-prune-patch-equivalence]], [[principles/encode-lessons-in-structure]]
