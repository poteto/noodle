package worktree

import "fmt"

// MergeConflictError marks deterministic git merge/rebase conflicts so callers
// can avoid retry loops and surface manual resolution paths.
type MergeConflictError struct {
	Branch string
	Err    error
}

func (e *MergeConflictError) Error() string {
	branch := e.Branch
	if branch == "" {
		branch = "unknown"
	}
	return fmt.Sprintf("merge conflict on branch %s", branch)
}

func (e *MergeConflictError) Unwrap() error {
	return e.Err
}
