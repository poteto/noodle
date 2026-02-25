package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Create creates a new plan directory with overview.md.
// It appends a wikilink to {plansDir}/index.md.
// Returns the absolute path to the created plan directory.
func Create(plansDir string, todoID int, slug string) (string, error) {
	dirName := fmt.Sprintf("%d-%s", todoID, slug)
	planDir := filepath.Join(plansDir, dirName)

	if err := os.MkdirAll(planDir, 0o755); err != nil {
		return "", fmt.Errorf("plan directory not created: %w", err)
	}

	today := time.Now().Format("2006-01-02")
	title := titleFromSlug(slug)

	overview := fmt.Sprintf("---\nid: %d\ncreated: %s\nstatus: ready\n---\n\n# %s\n", todoID, today, title)
	if err := os.WriteFile(filepath.Join(planDir, "overview.md"), []byte(overview), 0o644); err != nil {
		return "", fmt.Errorf("overview.md not written: %w", err)
	}

	if err := appendWikilink(plansDir, dirName); err != nil {
		return "", err
	}

	absDir, err := filepath.Abs(planDir)
	if err != nil {
		return "", fmt.Errorf("plan directory absolute path not resolved: %w", err)
	}
	return absDir, nil
}

// Done sets a plan's status to "done" and either archives or removes it.
// onDone "keep": move to archived_plans/, update indexes and wikilinks.
// onDone "remove": delete the plan directory and remove wikilinks.
func Done(plansDir string, planID int, onDone string) error {
	planDir, err := findPlanDir(plansDir, planID)
	if err != nil {
		return err
	}

	overviewPath := filepath.Join(planDir, "overview.md")
	data, err := os.ReadFile(overviewPath)
	if err != nil {
		return fmt.Errorf("overview.md not readable: %w", err)
	}

	content := string(data)
	updated := strings.Replace(content, "status: ready", "status: done", 1)
	if updated == content {
		updated = strings.Replace(content, "status: active", "status: done", 1)
	}
	if updated == content && !strings.Contains(content, "status: done") {
		return fmt.Errorf("plan %d has no recognized status field", planID)
	}

	if err := os.WriteFile(overviewPath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("overview.md not written: %w", err)
	}

	dirName := filepath.Base(planDir)
	brainDir := filepath.Dir(plansDir)

	if onDone == "remove" {
		if err := os.RemoveAll(planDir); err != nil {
			return fmt.Errorf("plan directory not removed: %w", err)
		}
		if err := removeWikilink(plansDir, dirName); err != nil {
			return err
		}
		return removeTodoLinks(brainDir, dirName)
	}

	// Default "keep": archive to archived_plans/.
	archivedDir := filepath.Join(brainDir, "archived_plans")

	if err := os.MkdirAll(archivedDir, 0o755); err != nil {
		return fmt.Errorf("archived_plans directory not created: %w", err)
	}

	if err := os.Rename(planDir, filepath.Join(archivedDir, dirName)); err != nil {
		return fmt.Errorf("plan directory not moved to archive: %w", err)
	}

	if err := removeWikilink(plansDir, dirName); err != nil {
		return err
	}
	if err := appendArchivedWikilink(archivedDir, dirName); err != nil {
		return err
	}

	oldPrefix := "plans/" + dirName
	newPrefix := "archived_plans/" + dirName
	if err := rewriteInternalLinks(filepath.Join(archivedDir, dirName), oldPrefix, newPrefix); err != nil {
		return err
	}
	return rewriteTodoLinks(brainDir, oldPrefix, newPrefix)
}

// Activate sets a plan's status from "ready" to "active".
func Activate(plansDir string, planID int) error {
	planDir, err := findPlanDir(plansDir, planID)
	if err != nil {
		return err
	}

	overviewPath := filepath.Join(planDir, "overview.md")
	data, err := os.ReadFile(overviewPath)
	if err != nil {
		return fmt.Errorf("overview.md not readable: %w", err)
	}

	content := string(data)
	updated := strings.Replace(content, "status: ready", "status: active", 1)
	if updated == content {
		return fmt.Errorf("plan %d status not ready", planID)
	}

	if err := os.WriteFile(overviewPath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("overview.md not written: %w", err)
	}
	return nil
}

// PhaseAdd creates a new numbered phase file in an existing plan directory.
// Auto-numbers based on existing phase-*.md files.
// Returns the absolute path to the created phase file.
func PhaseAdd(plansDir string, planID int, phaseName string) (string, error) {
	planDir, err := findPlanDir(plansDir, planID)
	if err != nil {
		return "", err
	}

	nextNum, err := nextPhaseNumber(planDir)
	if err != nil {
		return "", err
	}

	phaseSlug := slugify(phaseName)
	filename := fmt.Sprintf("phase-%02d-%s.md", nextNum, phaseSlug)
	dirName := filepath.Base(planDir)

	phaseContent := fmt.Sprintf("Back to [[plans/%s/overview]]\n\n# Phase %d: %s\n", dirName, nextNum, phaseName)
	phasePath := filepath.Join(planDir, filename)
	if err := os.WriteFile(phasePath, []byte(phaseContent), 0o644); err != nil {
		return "", fmt.Errorf("phase file not written: %w", err)
	}

	absPath, err := filepath.Abs(phasePath)
	if err != nil {
		return "", fmt.Errorf("phase file absolute path not resolved: %w", err)
	}
	return absPath, nil
}

// findPlanDir locates a plan directory matching {planID}-* in plansDir.
func findPlanDir(plansDir string, planID int) (string, error) {
	pattern := filepath.Join(plansDir, fmt.Sprintf("%d-*", planID))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("plan directory glob failed: %w", err)
	}

	// Filter to directories only.
	var dirs []string
	for _, m := range matches {
		info, err := os.Stat(m)
		if err == nil && info.IsDir() {
			dirs = append(dirs, m)
		}
	}

	if len(dirs) == 0 {
		return "", fmt.Errorf("plan %d not found", planID)
	}
	if len(dirs) > 1 {
		return "", fmt.Errorf("plan %d matched multiple directories", planID)
	}
	return dirs[0], nil
}

// nextPhaseNumber counts existing phase-*.md files and returns the next number.
func nextPhaseNumber(planDir string) (int, error) {
	pattern := filepath.Join(planDir, "phase-*.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return 0, fmt.Errorf("phase file glob failed: %w", err)
	}
	return len(matches) + 1, nil
}

// appendWikilink appends a plan wikilink to index.md, creating it if needed.
func appendWikilink(plansDir, dirName string) error {
	indexPath := filepath.Join(plansDir, "index.md")
	link := fmt.Sprintf("- [ ] [[plans/%s/overview]]", dirName)

	existing, err := os.ReadFile(indexPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("index.md not readable: %w", err)
	}

	var content string
	if len(existing) == 0 {
		content = "# Plans\n\n" + link + "\n"
	} else {
		s := string(existing)
		if !strings.HasSuffix(s, "\n") {
			s += "\n"
		}
		content = s + link + "\n"
	}

	if err := os.WriteFile(indexPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("index.md not written: %w", err)
	}
	return nil
}

// removeWikilink removes the line containing a wikilink for dirName from plansDir/index.md.
func removeWikilink(plansDir, dirName string) error {
	indexPath := filepath.Join(plansDir, "index.md")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("index.md not readable: %w", err)
	}

	target := "[[plans/" + dirName + "/overview]]"
	lines := strings.Split(string(data), "\n")
	var filtered []string
	for _, line := range lines {
		if !strings.Contains(line, target) {
			filtered = append(filtered, line)
		}
	}

	if err := os.WriteFile(indexPath, []byte(strings.Join(filtered, "\n")), 0o644); err != nil {
		return fmt.Errorf("index.md not written: %w", err)
	}
	return nil
}

// appendArchivedWikilink appends a plan wikilink to archived_plans/index.md, creating it if needed.
func appendArchivedWikilink(archivedDir, dirName string) error {
	indexPath := filepath.Join(archivedDir, "index.md")
	link := fmt.Sprintf("- [x] [[archived_plans/%s/overview]]", dirName)

	existing, err := os.ReadFile(indexPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("archived index.md not readable: %w", err)
	}

	var content string
	if len(existing) == 0 {
		content = "# Archived Plans\n\n" + link + "\n"
	} else {
		s := string(existing)
		if !strings.HasSuffix(s, "\n") {
			s += "\n"
		}
		content = s + link + "\n"
	}

	if err := os.WriteFile(indexPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("archived index.md not written: %w", err)
	}
	return nil
}

// rewriteInternalLinks replaces wikilinks containing oldPrefix with newPrefix
// in all .md files within planDir.
func rewriteInternalLinks(planDir, oldPrefix, newPrefix string) error {
	entries, err := os.ReadDir(planDir)
	if err != nil {
		return fmt.Errorf("plan directory not readable: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := filepath.Join(planDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%s not readable: %w", entry.Name(), err)
		}

		content := string(data)
		updated := strings.ReplaceAll(content, "[["+oldPrefix, "[["+newPrefix)
		if updated != content {
			if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
				return fmt.Errorf("%s not written: %w", entry.Name(), err)
			}
		}
	}
	return nil
}

// removeTodoLinks removes wikilink references to a plan directory from todos.md.
func removeTodoLinks(brainDir, dirName string) error {
	todosPath := filepath.Join(brainDir, "todos.md")
	data, err := os.ReadFile(todosPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("todos.md not readable: %w", err)
	}

	target := "[[plans/" + dirName + "/"
	lines := strings.Split(string(data), "\n")
	var filtered []string
	for _, line := range lines {
		if !strings.Contains(line, target) {
			filtered = append(filtered, line)
		}
	}

	if err := os.WriteFile(todosPath, []byte(strings.Join(filtered, "\n")), 0o644); err != nil {
		return fmt.Errorf("todos.md not written: %w", err)
	}
	return nil
}

// rewriteTodoLinks replaces wikilinks containing oldPrefix with newPrefix in todos.md.
func rewriteTodoLinks(brainDir, oldPrefix, newPrefix string) error {
	todosPath := filepath.Join(brainDir, "todos.md")
	data, err := os.ReadFile(todosPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("todos.md not readable: %w", err)
	}

	content := string(data)
	updated := strings.ReplaceAll(content, "[["+oldPrefix, "[["+newPrefix)
	if updated != content {
		if err := os.WriteFile(todosPath, []byte(updated), 0o644); err != nil {
			return fmt.Errorf("todos.md not written: %w", err)
		}
	}
	return nil
}

// titleFromSlug converts a slug like "test-plan" to "Test Plan".
func titleFromSlug(slug string) string {
	words := strings.Split(slug, "-")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// slugify converts a name like "Phase Name" to "phase-name".
func slugify(name string) string {
	s := strings.ToLower(name)
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return '-'
	}, s)
	// Collapse consecutive hyphens and trim.
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '-' })
	return strings.Join(parts, "-")
}
