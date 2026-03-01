package worktree

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// App holds the resolved git root and provides worktree operations.
type App struct {
	Root              string
	CmdPrefix         string        // command prefix for user-facing messages (e.g., "noodle worktree")
	MergeLockTimeout  time.Duration // 0 uses defaultMergeLockTimeout
	IntegrationBranch string        // override for the integration branch; auto-discovered when empty
	Quiet             bool          // suppress stdout/stderr chatter for programmatic callers
}

// cmdName returns the full command name for user-facing messages.
// Falls back to "worktree" if CmdPrefix is empty.
func (a *App) cmdName() string {
	if a.CmdPrefix != "" {
		return a.CmdPrefix
	}
	return "worktree"
}

// integrationBranch returns the branch that worktrees merge into.
// Priority: explicit IntegrationBranch field > git remote HEAD > "main".
func (a *App) integrationBranch() string {
	if a.IntegrationBranch != "" {
		return a.IntegrationBranch
	}
	return discoverBaseBranch(a.Root)
}

// ExitError represents a process exit with a specific code.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("process exited with code %d", e.Code)
}

// NewApp discovers the git root from the current working directory.
func NewApp() (*App, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return nil, fmt.Errorf("not inside a git repository")
	}
	return &App{Root: strings.TrimSpace(string(out))}, nil
}

// WorktreePath returns the on-disk path for a named worktree.
func WorktreePath(root, name string) string {
	return filepath.Join(root, ".worktrees", name)
}

// IsCWDInsideWorktree reports whether cwd is at or below wtPath.
func IsCWDInsideWorktree(cwd, wtPath string) bool {
	return cwd == wtPath || strings.HasPrefix(cwd, wtPath+string(filepath.Separator))
}

// CheckCWDSafe returns an error if cwd is inside the named worktree.
func CheckCWDSafe(cwd, root, name string) error {
	wtPath := WorktreePath(root, name)
	if IsCWDInsideWorktree(cwd, wtPath) {
		return fmt.Errorf(
			"shell CWD is inside the worktree (%s).\n  Run first:  cd %s\n  Then retry",
			cwd, root,
		)
	}
	return nil
}

// DetectPkgManager returns the package manager name based on lock files in dir.
func DetectPkgManager(dir string) string {
	checks := []struct{ file, pm string }{
		{"pnpm-lock.yaml", "pnpm"},
		{"bun.lockb", "bun"},
		{"bun.lock", "bun"},
		{"yarn.lock", "yarn"},
		{"package-lock.json", "npm"},
		{"Cargo.toml", "cargo"},
	}
	for _, c := range checks {
		if _, err := os.Stat(filepath.Join(dir, c.file)); err == nil {
			return c.pm
		}
	}
	return ""
}

// InstallArgs returns the binary and arguments for installing deps.
func InstallArgs(pm string) (bin string, args []string) {
	switch pm {
	case "pnpm":
		return "pnpm", []string{"install", "--frozen-lockfile"}
	case "bun":
		return "bun", []string{"install", "--frozen-lockfile"}
	case "yarn":
		return "yarn", []string{"install", "--frozen-lockfile"}
	case "npm":
		return "npm", []string{"ci"}
	case "cargo":
		return "cargo", []string{"build"}
	default:
		return "", nil
	}
}

// git creates an exec.Cmd that runs git from the App root.
func (a *App) git(args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Dir = a.Root
	return cmd
}

// gitOutput runs a git command and returns trimmed stdout.
func (a *App) gitOutput(args ...string) (string, error) {
	return gitOutput(a.Root, args...)
}

// gitRun runs a git command with stdout/stderr connected to the terminal.
func (a *App) gitRun(args ...string) error {
	cmd := a.git(args...)
	cmd.Stdout = a.stdout()
	cmd.Stderr = a.stderr()
	return cmd.Run()
}

// assertRootClean returns an error if the root checkout has uncommitted changes
// (staged or unstaged). This prevents merges from incorporating unintended state.
func (a *App) assertRootClean() error {
	unstaged := a.git("diff", "--quiet").Run() != nil
	staged := a.git("diff", "--cached", "--quiet").Run() != nil
	if unstaged || staged {
		return fmt.Errorf("root checkout has uncommitted changes — commit or stash before merging")
	}
	return nil
}

func (a *App) assertCWDSafe(name string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return CheckCWDSafe(cwd, a.Root, name)
}

func (a *App) countUnmergedCommits(name string) int {
	unmerged, _, err := a.cherryStatus(name)
	if err != nil {
		return 0
	}
	return unmerged
}

func (a *App) HasUnmergedCommits(name string) (bool, error) {
	unmerged, _, err := a.cherryStatus(name)
	if err != nil {
		return false, err
	}
	return unmerged > 0, nil
}

func (a *App) branchExists(branch string) bool {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return false
	}
	return a.git("show-ref", "--verify", "--quiet", "refs/heads/"+branch).Run() == nil
}

func (a *App) hasMergeConflicts(repoPath string) bool {
	out, err := a.gitOutput("-C", repoPath, "diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

func (a *App) resolveBranchName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	if a.branchExists(name) {
		return name
	}

	// Cook-created worktrees use "noodle/<worktree-name>" branches.
	prefixed := "noodle/" + name
	if a.branchExists(prefixed) {
		return prefixed
	}

	wtPath := a.resolveWorktreePath(name)
	if IsRealWorktree(wtPath) {
		if branch, err := a.gitOutput("-C", wtPath, "branch", "--show-current"); err == nil {
			branch = strings.TrimSpace(branch)
			if branch != "" {
				return branch
			}
		}
	}

	return name
}

// WorktreeEntry is a non-main worktree discovered from git metadata.
type WorktreeEntry struct {
	Name   string // display name (last path component or relative path under .worktrees/)
	Path   string // absolute on-disk path
	Branch string // branch name (empty for detached HEAD)
	Source string // "noodle", "claude", "cursor", or "other"
}

// allWorktreeEntries returns all non-main worktrees from git metadata,
// including those under .worktrees/, .claude/worktrees/, and ~/.cursor/worktrees/.
func (a *App) allWorktreeEntries() ([]WorktreeEntry, error) {
	out, err := a.gitOutput("worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	mainRoot := canonicalPath(a.Root)
	noodleRoot := canonicalPath(filepath.Join(a.Root, ".worktrees"))
	claudeRoot := canonicalPath(filepath.Join(a.Root, ".claude", "worktrees"))

	home, _ := os.UserHomeDir()
	repoName := filepath.Base(a.Root)
	cursorRoot := ""
	if home != "" {
		cursorRoot = canonicalPath(filepath.Join(home, ".cursor", "worktrees", repoName))
	}

	var entries []WorktreeEntry
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, "worktree ") {
			continue
		}
		wtPath := canonicalPath(strings.TrimSpace(strings.TrimPrefix(line, "worktree ")))
		if wtPath == mainRoot {
			continue
		}

		entry := WorktreeEntry{Path: wtPath}

		// Classify by location.
		if rel, err := filepath.Rel(noodleRoot, wtPath); err == nil && !strings.HasPrefix(rel, "..") {
			entry.Name = filepath.ToSlash(rel)
			entry.Source = "noodle"
		} else if rel, err := filepath.Rel(claudeRoot, wtPath); err == nil && !strings.HasPrefix(rel, "..") {
			entry.Name = filepath.ToSlash(rel)
			entry.Source = "claude"
		} else if cursorRoot != "" {
			if rel, err := filepath.Rel(cursorRoot, wtPath); err == nil && !strings.HasPrefix(rel, "..") {
				entry.Name = filepath.ToSlash(rel)
				entry.Source = "cursor"
			} else {
				entry.Name = filepath.Base(wtPath)
				entry.Source = "other"
			}
		} else {
			entry.Name = filepath.Base(wtPath)
			entry.Source = "other"
		}

		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Source != entries[j].Source {
			return entries[i].Source < entries[j].Source
		}
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

// managedWorktreeNames returns branch/worktree names managed under .worktrees/
// as reported by git worktree metadata. Names are normalized with "/" separators
// so slash-named branches (for example "feature/ui-polish") round-trip safely.
func (a *App) managedWorktreeNames() ([]string, error) {
	entries, err := a.allWorktreeEntries()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.Source == "noodle" {
			names = append(names, e.Name)
		}
	}
	sort.Strings(names)
	return names, nil
}

// resolveWorktreePath returns the on-disk path for a worktree by name.
// Checks .worktrees/<name> first, then searches all git worktree entries.
func (a *App) resolveWorktreePath(name string) string {
	// Fast path: standard noodle worktree.
	standard := WorktreePath(a.Root, name)
	if _, err := os.Stat(standard); err == nil {
		return standard
	}

	// Search all worktrees for a name match.
	entries, err := a.allWorktreeEntries()
	if err != nil {
		return standard
	}
	for _, e := range entries {
		if e.Name == name {
			return e.Path
		}
	}
	return standard
}

func canonicalPath(path string) string {
	p := filepath.Clean(path)
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		p = resolved
	}
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	return filepath.Clean(p)
}

// cherryStatus returns counts from `git cherry <base> <branch>` where:
// "+" lines are commits not yet represented on base,
// "-" lines are patch-equivalent commits already represented on base.
func (a *App) cherryStatus(name string) (unmerged int, equivalent int, err error) {
	branch := a.resolveBranchName(name)
	out, err := a.gitOutput("cherry", a.integrationBranch(), branch)
	if err != nil {
		return 0, 0, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return 0, 0, nil
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "+"):
			unmerged++
		case strings.HasPrefix(line, "-"):
			equivalent++
		}
	}
	return unmerged, equivalent, nil
}

func (a *App) isWorktreeClean(path string) (bool, error) {
	out, err := a.gitOutput("-C", path, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "", nil
}

// IsRealWorktree reports whether dir looks like a valid git worktree
// (has a .git file or directory).
func IsRealWorktree(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

func (a *App) installDeps(dir string) {
	pm := DetectPkgManager(dir)
	bin, args := InstallArgs(pm)
	if bin == "" {
		a.info("No lock file found — skipping dep install")
		return
	}
	a.info(fmt.Sprintf("Installing deps with %s...", pm))
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Stdout = a.stdout()
	cmd.Stderr = a.stderr()
	_ = cmd.Run()
}

func (a *App) stdout() io.Writer {
	if a.Quiet {
		return io.Discard
	}
	return os.Stdout
}

func (a *App) stderr() io.Writer {
	if a.Quiet {
		return io.Discard
	}
	return os.Stderr
}

func (a *App) info(msg string) {
	if a.Quiet {
		return
	}
	fmt.Fprintf(a.stdout(), "  %s\n", msg)
}

func (a *App) warnf(format string, args ...any) {
	if a.Quiet {
		return
	}
	fmt.Fprintf(a.stderr(), format, args...)
}

func (a *App) printf(format string, args ...any) {
	if a.Quiet {
		return
	}
	fmt.Fprintf(a.stdout(), format, args...)
}
