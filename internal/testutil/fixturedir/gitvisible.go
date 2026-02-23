package fixturedir

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// listGitVisibleFiles returns files under root that are visible to git
// (tracked + untracked non-ignored). If root is not in a git worktree,
// gitScoped=false and callers should fall back to directory walking.
func listGitVisibleFiles(root string) ([]string, bool, error) {
	cmd := exec.Command("git", "-C", root, "ls-files", "--cached", "--others", "--exclude-standard", "--", ".")
	output, err := cmd.Output()
	if err != nil {
		return nil, false, nil
	}
	lines := strings.Split(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n")
	paths := make([]string, 0, len(lines))
	for _, line := range lines {
		relPath := strings.TrimSpace(line)
		if relPath == "" {
			continue
		}
		absPath := filepath.Join(root, filepath.FromSlash(relPath))
		info, statErr := os.Stat(absPath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				continue
			}
			return nil, true, fmt.Errorf("stat fixture input %s: %w", absPath, statErr)
		}
		if info.IsDir() {
			continue
		}
		paths = append(paths, absPath)
	}
	return paths, true, nil
}
