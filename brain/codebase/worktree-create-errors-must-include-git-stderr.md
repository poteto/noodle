# Worktree Create Errors Must Include Git Stderr

- `worktree.App` commonly runs with `Quiet=true` inside loop dependencies.
- Plain `cmd.Run()` failures surface as `exit status <code>` and drop git's actual failure reason.
- `worktree.Create` must use a command path that captures `CombinedOutput` and wraps stderr text into the returned error.
- This keeps `cycle.spawn` failures actionable in noodle logs (for example: branch already exists, branch checked out elsewhere, invalid ref name).

See also [[codebase/worktree-gotchas]], [[principles/fix-root-causes]]
