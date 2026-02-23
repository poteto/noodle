package plan

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// PlanMeta holds YAML frontmatter from a plan's overview.md.
type PlanMeta struct {
	ID       int    `yaml:"id" json:"id"`
	Created  string `yaml:"created" json:"created"`
	Status   string `yaml:"status" json:"status"` // draft | active | done
	Provider string `yaml:"provider" json:"provider,omitempty"`
	Model    string `yaml:"model" json:"model,omitempty"`
}

// Plan represents a single plan directory with its phases.
type Plan struct {
	Meta      PlanMeta `json:"meta"`
	Title     string   `json:"title"`
	Directory string   `json:"directory"` // absolute path
	Slug      string   `json:"slug"`      // directory basename (e.g., "23-task-type-skill-suite")
	Phases    []Phase  `json:"phases"`
}

// Phase represents a single phase file within a plan.
type Phase struct {
	Name     string `json:"name"`     // from heading or filename
	Filename string `json:"filename"` // e.g., "phase-01-scaffold.md"
	Status   string `json:"status"`   // pending | active | done
}

var wikilinkPattern = regexp.MustCompile(`\[\[plans/([^/\]]+)/overview\]\]`)

// ReadAll discovers and reads plans from plansDir. Plans with status "done"
// are excluded from the result. Returns an empty slice (not nil) when no
// plans are found or the index is missing.
func ReadAll(plansDir string) ([]Plan, error) {
	indexPath := filepath.Join(plansDir, "index.md")
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Plan{}, nil
		}
		return nil, err
	}

	slugs := extractSlugs(string(indexData))
	if len(slugs) == 0 {
		return []Plan{}, nil
	}

	var plans []Plan
	for _, slug := range slugs {
		planDir := filepath.Join(plansDir, slug)
		p, ok := readPlan(planDir, slug)
		if !ok {
			continue
		}
		if p.Meta.Status == "done" {
			continue
		}
		plans = append(plans, p)
	}

	if plans == nil {
		return []Plan{}, nil
	}
	return plans, nil
}

// extractSlugs finds all [[plans/SLUG/overview]] wikilinks in the index content.
func extractSlugs(content string) []string {
	matches := wikilinkPattern.FindAllStringSubmatch(content, -1)
	slugs := make([]string, 0, len(matches))
	for _, m := range matches {
		slugs = append(slugs, m[1])
	}
	return slugs
}

// readPlan reads a single plan directory. Returns false if the overview
// is missing or malformed.
func readPlan(planDir, slug string) (Plan, bool) {
	overviewPath := filepath.Join(planDir, "overview.md")
	data, err := os.ReadFile(overviewPath)
	if err != nil {
		return Plan{}, false
	}

	meta, body, ok := parseFrontmatter(string(data))
	if !ok {
		return Plan{}, false
	}

	title := extractHeading(body)

	absDir, err := filepath.Abs(planDir)
	if err != nil {
		return Plan{}, false
	}

	phases := discoverPhases(planDir, body)

	return Plan{
		Meta:      meta,
		Title:     title,
		Directory: absDir,
		Slug:      slug,
		Phases:    phases,
	}, true
}

// parseFrontmatter splits YAML frontmatter (between --- delimiters) from
// the markdown body. Returns false when frontmatter is missing or unparseable.
func parseFrontmatter(content string) (PlanMeta, string, bool) {
	if !strings.HasPrefix(content, "---\n") {
		return PlanMeta{}, "", false
	}

	end := strings.Index(content[4:], "\n---")
	if end < 0 {
		return PlanMeta{}, "", false
	}

	fmRaw := content[4 : 4+end]
	body := content[4+end+4:] // skip past closing "---\n" (or "---" at EOF)

	var meta PlanMeta
	if err := yaml.Unmarshal([]byte(fmRaw), &meta); err != nil {
		return PlanMeta{}, "", false
	}

	return meta, body, true
}

// extractHeading returns the text of the first # heading in the body,
// or an empty string if none is found.
func extractHeading(body string) string {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(trimmed[2:])
		}
	}
	return ""
}

// discoverPhases globs phase-*.md files, sorts them by filename, and
// infers status from the overview checklist.
func discoverPhases(planDir, overviewBody string) []Phase {
	pattern := filepath.Join(planDir, "phase-*.md")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return nil
	}

	sort.Strings(matches)

	checklist := parseChecklist(overviewBody)

	var phases []Phase
	for _, path := range matches {
		filename := filepath.Base(path)
		name := phaseNameFromFile(path, filename)
		status := inferPhaseStatus(name, checklist)
		phases = append(phases, Phase{
			Name:     name,
			Filename: filename,
			Status:   status,
		})
	}

	return phases
}

// checklistEntry records whether a checklist item is checked.
type checklistEntry struct {
	name    string
	checked bool
}

// parseChecklist extracts checklist items from the overview body.
// It looks for lines matching `- [x] ...` or `- [ ] ...` that contain
// wikilinks to phase files, extracting phase names from the link text
// or the text after the checkbox.
func parseChecklist(body string) []checklistEntry {
	var entries []checklistEntry
	phaseLink := regexp.MustCompile(`\[\[(?:[^\]]*/)?(phase-[^\]]+)\]\]`)

	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)

		var checked bool
		var rest string
		if strings.HasPrefix(trimmed, "- [x] ") || strings.HasPrefix(trimmed, "- [X] ") {
			checked = true
			rest = trimmed[6:]
		} else if strings.HasPrefix(trimmed, "- [ ] ") {
			checked = false
			rest = trimmed[6:]
		} else {
			continue
		}

		// Try to extract a phase link to get the display name.
		if m := phaseLink.FindStringSubmatch(rest); m != nil {
			// Use the slug portion (e.g., "phase-01-dynamic-task-registry")
			// and convert to a human-readable name.
			name := cleanFilename(m[1])
			entries = append(entries, checklistEntry{name: name, checked: checked})
			continue
		}

		// Plain text checklist item: use the text directly.
		name := strings.TrimSpace(rest)
		if name != "" {
			entries = append(entries, checklistEntry{name: name, checked: checked})
		}
	}

	return entries
}

// inferPhaseStatus determines the status of a phase based on the overview
// checklist. The first unchecked item is "active", all checked items before
// it are "done", and all items after it are "pending". If no checklist is
// present, all phases default to "pending".
func inferPhaseStatus(phaseName string, checklist []checklistEntry) string {
	if len(checklist) == 0 {
		return "pending"
	}

	// Find the first unchecked entry index.
	firstUnchecked := -1
	for i, entry := range checklist {
		if !entry.checked {
			firstUnchecked = i
			break
		}
	}

	// Try matching by normalized name containment.
	normalizedPhase := normalizeForMatch(phaseName)
	for i, entry := range checklist {
		normalizedEntry := normalizeForMatch(entry.name)
		if normalizedEntry == normalizedPhase || strings.Contains(normalizedEntry, normalizedPhase) || strings.Contains(normalizedPhase, normalizedEntry) {
			if entry.checked {
				return "done"
			}
			if firstUnchecked >= 0 && i == firstUnchecked {
				return "active"
			}
			return "pending"
		}
	}

	// Phase not found in checklist: default to pending.
	return "pending"
}

// normalizeForMatch lowercases, strips non-alphanumeric characters, and
// collapses whitespace for fuzzy name matching.
func normalizeForMatch(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == ' ' {
			return r
		}
		return ' '
	}, s)
	parts := strings.Fields(s)
	return strings.Join(parts, " ")
}

// phaseNameFromFile extracts a phase name from the first heading in the
// file, falling back to a cleaned version of the filename.
func phaseNameFromFile(path, filename string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return cleanFilename(strings.TrimSuffix(filename, ".md"))
	}

	heading := extractHeading(string(data))
	if heading != "" {
		return heading
	}

	return cleanFilename(strings.TrimSuffix(filename, ".md"))
}

// cleanFilename converts a filename like "phase-01-scaffold" to
// "Phase 01 Scaffold".
func cleanFilename(name string) string {
	name = strings.ReplaceAll(name, "-", " ")
	words := strings.Fields(name)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
