package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Create creates a new worktree and branch, symlinks local settings, and installs deps.
func (a *App) Create(name string) error {
	wtPath := WorktreePath(a.Root, name)

	if a.git("check-ignore", "-q", ".worktrees").Run() != nil {
		f, err := os.OpenFile(
			filepath.Join(a.Root, ".gitignore"),
			os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644,
		)
		if err == nil {
			_, _ = f.WriteString(".worktrees/\n")
			f.Close()
		}
	}

	if _, err := os.Stat(wtPath); err == nil {
		return fmt.Errorf("worktree '%s' already exists at %s", name, wtPath)
	}

	a.info(fmt.Sprintf("Creating worktree at %s...", wtPath))
	if err := a.gitRun("worktree", "add", wtPath, "-b", name); err != nil {
		// Reuse an existing local branch if it already exists but has no worktree path.
		if a.branchExists(name) {
			if retryErr := a.gitRun("worktree", "add", wtPath, name); retryErr == nil {
				goto created
			}
		}
		return fmt.Errorf("failed to create worktree: %w", err)
	}

created:
	settingsPath := filepath.Join(a.Root, ".claude", "settings.local.json")
	if _, err := os.Stat(settingsPath); err == nil {
		wtClaudeDir := filepath.Join(wtPath, ".claude")
		_ = os.MkdirAll(wtClaudeDir, 0o755)
		_ = os.Symlink(settingsPath, filepath.Join(wtClaudeDir, "settings.local.json"))
		a.info("Symlinked .claude/settings.local.json")
	}

	a.installDeps(wtPath)

	a.printf("\nWorktree ready: %s (branch: %s)\n", wtPath, name)
	return nil
}

// Exec runs a command inside the named worktree.
func (a *App) Exec(name string, args []string) error {
	wtPath := WorktreePath(a.Root, name)
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		return fmt.Errorf("worktree '%s' does not exist at %s", name, wtPath)
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = wtPath
	cmd.Stdin = os.Stdin
	cmd.Stdout = a.stdout()
	cmd.Stderr = a.stderr()

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &ExitError{Code: exitErr.ExitCode()}
		}
		return err
	}
	return nil
}
