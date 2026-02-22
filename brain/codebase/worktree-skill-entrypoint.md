# Worktree Skill Entrypoint

- In this repo, do not direct agents to run raw `git worktree` commands.
- Canonical path is Skill(worktree), which routes through the Go CLI at `old_noodle/` during extraction.
- For policy wording, refer to "linked worktree created via Skill(worktree)" rather than "`git worktree add`".
- User preference (2026-02-20): always use a linked worktree, including planning/documentation changes that might otherwise feel small.

See also [[codebase/worktree-gotchas]], [[principles/encode-lessons-in-structure]]
