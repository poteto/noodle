package loop

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/poteto/noodle/skill"
)

const bootstrapSessionPrefix = "bootstrap-"

// buildBootstrapPrompt resolves the bootstrap skill from the skill search
// paths, strips frontmatter, and substitutes {{history_dirs}} based on
// provider. Returns an error if the skill is not found.
func buildBootstrapPrompt(provider string, searchPaths []string) (string, error) {
	resolver := skill.Resolver{SearchPaths: searchPaths}
	sp, err := resolver.Resolve("bootstrap")
	if err != nil {
		return "", fmt.Errorf("bootstrap skill not found — create .agents/skills/bootstrap/SKILL.md or run noodle init")
	}
	data, err := os.ReadFile(filepath.Join(sp.Path, "SKILL.md"))
	if err != nil {
		return "", fmt.Errorf("read bootstrap skill: %w", err)
	}
	body := skill.StripFrontmatter(data)

	provider = strings.ToLower(strings.TrimSpace(provider))
	var historyDirs string
	switch provider {
	case "codex":
		historyDirs = "`.codex/` (and `.claude/` if it exists)"
	case "claude":
		historyDirs = "`.claude/` (and `.codex/` if it exists)"
	default:
		historyDirs = "`.claude/` and/or `.codex/` if they exist"
	}

	return strings.ReplaceAll(string(body), "{{history_dirs}}", historyDirs), nil
}

func isBootstrapSession(name string) bool {
	return strings.HasPrefix(name, bootstrapSessionPrefix)
}
