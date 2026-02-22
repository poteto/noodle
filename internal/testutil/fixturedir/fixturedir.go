package fixturedir

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const FixtureSchemaVersion = 1

var stateDirPattern = regexp.MustCompile(`^state-(\d{2})$`)

type FixtureLayout struct {
	RootPath           string
	ExpectedPath       string
	ExpectedSourcePath string
	BaseConfigPath     string
	States             []FixtureStateDir
}

type FixtureStateDir struct {
	ID         string
	Index      int
	Path       string
	ConfigPath string
}

type FixtureConfigScope struct {
	BaseConfigPath    string
	StateOverridePath string
}

type FixtureMetadata struct {
	ExpectedFailure bool
	Bug             bool
	Regression      string
	SchemaVersion   int
	SourceHash      string
}

type FixtureCase struct {
	Name          string
	Path          string
	Layout        FixtureLayout
	Metadata      FixtureMetadata
	ExpectedError *ErrorExpectation
	Sections      map[string]string
	States        []FixtureState
}

type FixtureState struct {
	ID          string
	Path        string
	ConfigScope FixtureConfigScope
	Files       map[string]string
	FileOrder   []string
}

type FixtureInventory struct {
	Root  string
	Cases []FixtureCase
}

type FixtureValidationIssue struct {
	Path     string
	Severity string
	Message  string
}

func LoadInventory(tb testing.TB, root string) FixtureInventory {
	tb.Helper()
	inventory, err := Discover(root)
	if err != nil {
		tb.Fatalf("load fixture inventory %s: %v", root, err)
	}
	return inventory
}

func Discover(root string) (FixtureInventory, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	entries, err := os.ReadDir(root)
	if err != nil {
		return FixtureInventory{}, fmt.Errorf("read fixture root: %w", err)
	}

	fixtureDirs := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" || strings.HasPrefix(name, ".") {
			continue
		}
		fixtureDirs = append(fixtureDirs, name)
	}
	sort.Strings(fixtureDirs)
	if len(fixtureDirs) == 0 {
		return FixtureInventory{}, fmt.Errorf("no fixture directories found in %s", root)
	}

	inventory := FixtureInventory{Root: root, Cases: make([]FixtureCase, 0, len(fixtureDirs))}
	for _, name := range fixtureDirs {
		fixturePath := filepath.Join(root, name)
		fixtureCase, err := loadFixtureCase(name, fixturePath)
		if err != nil {
			return FixtureInventory{}, err
		}
		inventory.Cases = append(inventory.Cases, fixtureCase)
	}
	return inventory, nil
}

func ValidateFixtureRoot(root string) ([]FixtureValidationIssue, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read fixture root: %w", err)
	}

	issues := make([]FixtureValidationIssue, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" || strings.HasPrefix(name, ".") {
			continue
		}
		fixturePath := filepath.Join(root, name)
		layout, layoutIssues := resolveLayout(fixturePath)
		issues = append(issues, layoutIssues...)
		if len(layoutIssues) > 0 {
			continue
		}

		content, readErr := os.ReadFile(layout.ExpectedPath)
		if readErr != nil {
			issues = append(issues, FixtureValidationIssue{
				Path:     filepath.ToSlash(layout.ExpectedPath),
				Severity: "error",
				Message:  fmt.Sprintf("read expected.md: %v", readErr),
			})
			continue
		}
		if _, _, _, parseErr := parseExpectedDocument(content); parseErr != nil {
			issues = append(issues, FixtureValidationIssue{
				Path:     filepath.ToSlash(layout.ExpectedPath),
				Severity: "error",
				Message:  parseErr.Error(),
			})
		}
		if syncErr := assertExpectedMarkdownSynced(layout.ExpectedSourcePath, layout.ExpectedPath); syncErr != nil {
			issues = append(issues, FixtureValidationIssue{
				Path:     filepath.ToSlash(layout.ExpectedPath),
				Severity: "error",
				Message:  syncErr.Error(),
			})
		}
	}

	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Path == issues[j].Path {
			return issues[i].Message < issues[j].Message
		}
		return issues[i].Path < issues[j].Path
	})
	return issues, nil
}

func loadFixtureCase(name, fixturePath string) (FixtureCase, error) {
	layout, issues := resolveLayout(fixturePath)
	if len(issues) > 0 {
		return FixtureCase{}, fmt.Errorf("invalid fixture %s: %s", fixturePath, formatIssues(issues))
	}
	if err := assertExpectedMarkdownSynced(layout.ExpectedSourcePath, layout.ExpectedPath); err != nil {
		return FixtureCase{}, fmt.Errorf("invalid fixture %s: %w", fixturePath, err)
	}

	expectedData, err := os.ReadFile(layout.ExpectedPath)
	if err != nil {
		return FixtureCase{}, fmt.Errorf("read expected.md %s: %w", layout.ExpectedPath, err)
	}
	metadata, sections, presentSections, err := parseExpectedDocument(expectedData)
	if err != nil {
		return FixtureCase{}, fmt.Errorf("parse expected.md %s: %w", layout.ExpectedPath, err)
	}

	expectedError, err := parseExpectedError(sections["Expected Error"], presentSections["Expected Error"])
	if err != nil {
		return FixtureCase{}, fmt.Errorf("parse expected error for %s: %w", fixturePath, err)
	}

	if metadata.ExpectedFailure && expectedError == nil {
		expectedError = &ErrorExpectation{Any: true}
	}
	if !metadata.ExpectedFailure && expectationDemandsFailure(expectedError) {
		return FixtureCase{}, fmt.Errorf(
			"fixture %s sets expected_failure=false but expected error requires failure",
			fixturePath,
		)
	}
	if metadata.ExpectedFailure && expectedError != nil && expectedError.Absent {
		return FixtureCase{}, fmt.Errorf(
			"fixture %s sets expected_failure=true but expected error requires absence",
			fixturePath,
		)
	}

	states := make([]FixtureState, 0, len(layout.States))
	for _, stateDir := range layout.States {
		files, ordered, loadErr := loadStateFiles(stateDir.Path)
		if loadErr != nil {
			return FixtureCase{}, loadErr
		}
		states = append(states, FixtureState{
			ID:   stateDir.ID,
			Path: stateDir.Path,
			ConfigScope: FixtureConfigScope{
				BaseConfigPath:    layout.BaseConfigPath,
				StateOverridePath: stateDir.ConfigPath,
			},
			Files:     files,
			FileOrder: ordered,
		})
	}

	return FixtureCase{
		Name:          name,
		Path:          fixturePath,
		Layout:        layout,
		Metadata:      metadata,
		ExpectedError: expectedError,
		Sections:      sections,
		States:        states,
	}, nil
}

func resolveLayout(fixturePath string) (FixtureLayout, []FixtureValidationIssue) {
	layout := FixtureLayout{RootPath: fixturePath}
	issues := make([]FixtureValidationIssue, 0)

	entries, err := os.ReadDir(fixturePath)
	if err != nil {
		issues = append(issues, FixtureValidationIssue{
			Path:     filepath.ToSlash(fixturePath),
			Severity: "error",
			Message:  fmt.Sprintf("read fixture directory: %v", err),
		})
		return layout, issues
	}

	states := make([]FixtureStateDir, 0)
	for _, entry := range entries {
		name := strings.TrimSpace(entry.Name())
		if name == "" {
			continue
		}
		path := filepath.Join(fixturePath, name)
		if entry.IsDir() {
			matches := stateDirPattern.FindStringSubmatch(name)
			if len(matches) == 2 {
				index, convErr := strconv.Atoi(matches[1])
				if convErr != nil {
					issues = append(issues, FixtureValidationIssue{
						Path:     filepath.ToSlash(path),
						Severity: "error",
						Message:  fmt.Sprintf("parse state index: %v", convErr),
					})
					continue
				}
				stateConfigPath := filepath.Join(path, "noodle.toml")
				if _, statErr := os.Stat(stateConfigPath); statErr != nil {
					stateConfigPath = ""
				}
				states = append(states, FixtureStateDir{
					ID:         name,
					Index:      index,
					Path:       path,
					ConfigPath: stateConfigPath,
				})
				continue
			}
			issues = append(issues, FixtureValidationIssue{
				Path:     filepath.ToSlash(path),
				Severity: "error",
				Message:  "unexpected directory; only state-XX directories are allowed",
			})
			continue
		}

		switch name {
		case "expected.md":
			layout.ExpectedPath = path
		case "expected.src.md":
			layout.ExpectedSourcePath = path
		case "noodle.toml":
			layout.BaseConfigPath = path
		default:
			issues = append(issues, FixtureValidationIssue{
				Path:     filepath.ToSlash(path),
				Severity: "error",
				Message:  "unexpected file; allowed files are expected.md, expected.src.md, and optional noodle.toml",
			})
		}
	}

	if strings.TrimSpace(layout.ExpectedPath) == "" {
		issues = append(issues, FixtureValidationIssue{
			Path:     filepath.ToSlash(fixturePath),
			Severity: "error",
			Message:  "missing required expected.md",
		})
	}
	if strings.TrimSpace(layout.ExpectedSourcePath) == "" {
		issues = append(issues, FixtureValidationIssue{
			Path:     filepath.ToSlash(fixturePath),
			Severity: "error",
			Message:  "missing required expected.src.md",
		})
	}

	if len(states) == 0 {
		issues = append(issues, FixtureValidationIssue{
			Path:     filepath.ToSlash(fixturePath),
			Severity: "error",
			Message:  "missing required state directories (state-01, state-02, ...)",
		})
		return layout, issues
	}

	sort.Slice(states, func(i, j int) bool {
		if states[i].Index == states[j].Index {
			return states[i].ID < states[j].ID
		}
		return states[i].Index < states[j].Index
	})

	for index, state := range states {
		expected := index + 1
		if state.Index != expected {
			issues = append(issues, FixtureValidationIssue{
				Path:     filepath.ToSlash(state.Path),
				Severity: "error",
				Message:  fmt.Sprintf("state ordering gap: expected state-%02d", expected),
			})
		}
	}
	layout.States = states
	return layout, issues
}

func loadStateFiles(statePath string) (map[string]string, []string, error) {
	files := map[string]string{}
	ordered := make([]string, 0)

	err := filepath.WalkDir(statePath, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(statePath, path)
		if err != nil {
			return fmt.Errorf("compute relative path: %w", err)
		}
		normalized := filepath.ToSlash(filepath.Clean(relPath))
		files[normalized] = path
		ordered = append(ordered, normalized)
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("walk state directory %s: %w", statePath, err)
	}

	sort.Strings(ordered)
	return files, ordered, nil
}

func parseExpectedDocument(data []byte) (FixtureMetadata, map[string]string, map[string]bool, error) {
	frontmatter, body, err := splitFrontmatter(string(data))
	if err != nil {
		return FixtureMetadata{}, nil, nil, err
	}
	metadata, err := parseMetadata(frontmatter)
	if err != nil {
		return FixtureMetadata{}, nil, nil, err
	}
	sections, sectionPresence := parseSections(body)
	return metadata, sections, sectionPresence, nil
}

func splitFrontmatter(content string) (string, string, error) {
	lines := strings.Split(content, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return "", "", fmt.Errorf("expected.md must start with frontmatter delimited by ---")
	}

	closingIndex := -1
	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == "---" {
			closingIndex = index
			break
		}
	}
	if closingIndex < 0 {
		return "", "", fmt.Errorf("expected.md frontmatter is missing closing --- delimiter")
	}
	frontmatter := strings.Join(lines[1:closingIndex], "\n")
	body := strings.Join(lines[closingIndex+1:], "\n")
	return frontmatter, body, nil
}

func parseMetadata(frontmatter string) (FixtureMetadata, error) {
	metadata := FixtureMetadata{}
	seen := map[string]bool{}
	allowed := map[string]struct{}{
		"schema_version":   {},
		"expected_failure": {},
		"bug":              {},
		"regression":       {},
		"source_hash":      {},
	}

	lines := strings.Split(frontmatter, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			return FixtureMetadata{}, fmt.Errorf("invalid frontmatter line %q", trimmed)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if _, ok := allowed[key]; !ok {
			return FixtureMetadata{}, fmt.Errorf("unsupported frontmatter key %q", key)
		}
		if seen[key] {
			return FixtureMetadata{}, fmt.Errorf("duplicate frontmatter key %q", key)
		}
		seen[key] = true

		switch key {
		case "schema_version":
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return FixtureMetadata{}, fmt.Errorf("schema_version must be an integer")
			}
			metadata.SchemaVersion = parsed
		case "expected_failure":
			parsed, err := strconv.ParseBool(strings.ToLower(value))
			if err != nil {
				return FixtureMetadata{}, fmt.Errorf("expected_failure must be true or false")
			}
			metadata.ExpectedFailure = parsed
		case "bug":
			parsed, err := strconv.ParseBool(strings.ToLower(value))
			if err != nil {
				return FixtureMetadata{}, fmt.Errorf("bug must be true or false")
			}
			metadata.Bug = parsed
		case "regression":
			metadata.Regression = trimQuotes(value)
		case "source_hash":
			metadata.SourceHash = trimQuotes(value)
		}
	}

	for _, key := range []string{"schema_version", "expected_failure", "bug", "regression"} {
		if !seen[key] {
			return FixtureMetadata{}, fmt.Errorf("missing required frontmatter key %q", key)
		}
	}
	if strings.TrimSpace(metadata.Regression) == "" {
		return FixtureMetadata{}, fmt.Errorf("regression must be non-empty")
	}
	if metadata.Bug && !metadata.ExpectedFailure {
		return FixtureMetadata{}, fmt.Errorf("bug=true requires expected_failure=true")
	}
	if metadata.SchemaVersion != FixtureSchemaVersion {
		return FixtureMetadata{}, fmt.Errorf(
			"unsupported schema_version %d (expected %d)",
			metadata.SchemaVersion,
			FixtureSchemaVersion,
		)
	}
	return metadata, nil
}

func parseSections(body string) (map[string]string, map[string]bool) {
	sections := map[string]string{}
	presence := map[string]bool{}
	currentHeading := ""
	insideFence := false
	seenFence := false
	lines := make([]string, 0)

	flush := func() {
		if currentHeading == "" || !seenFence {
			return
		}
		sections[currentHeading] = strings.TrimSpace(strings.Join(lines, "\n"))
		presence[currentHeading] = true
	}

	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			flush()
			currentHeading = strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
			insideFence = false
			seenFence = false
			lines = lines[:0]
			continue
		}
		if currentHeading == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "```") {
			insideFence = !insideFence
			seenFence = true
			continue
		}
		if insideFence {
			lines = append(lines, line)
		}
	}
	flush()
	return sections, presence
}

func parseExpectedError(raw string, present bool) (*ErrorExpectation, error) {
	if !present {
		return nil, nil
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := decodeStrictJSON[ErrorExpectation](raw)
	if err != nil {
		return nil, err
	}
	if !parsed.Absent && !parsed.Any &&
		strings.TrimSpace(parsed.Contains) == "" &&
		strings.TrimSpace(parsed.Equals) == "" {
		parsed.Any = true
	}
	return &parsed, nil
}

func expectationDemandsFailure(expect *ErrorExpectation) bool {
	if expect == nil {
		return false
	}
	if expect.Absent {
		return false
	}
	return true
}

func decodeStrictJSON[T any](raw string) (T, error) {
	var out T
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&out); err != nil {
		return out, err
	}
	if decoder.More() {
		return out, fmt.Errorf("unexpected trailing JSON tokens")
	}
	return out, nil
}

func trimQuotes(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
			(strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`)) {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func formatIssues(issues []FixtureValidationIssue) string {
	if len(issues) == 0 {
		return ""
	}
	parts := make([]string, 0, len(issues))
	for _, issue := range issues {
		parts = append(parts, fmt.Sprintf("%s (%s)", issue.Message, issue.Path))
	}
	return strings.Join(parts, "; ")
}

func assertExpectedMarkdownSynced(sourcePath, expectedPath string) error {
	sourcePath = strings.TrimSpace(sourcePath)
	expectedPath = strings.TrimSpace(expectedPath)
	if sourcePath == "" || expectedPath == "" {
		return fmt.Errorf("expected fixture paths are missing")
	}
	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read expected source: %w", err)
	}
	renderedExpected, err := renderExpectedMarkdownFromSource(string(sourceData))
	if err != nil {
		return fmt.Errorf("render expected source: %w", err)
	}
	expectedData, err := os.ReadFile(expectedPath)
	if err != nil {
		return fmt.Errorf("read expected markdown: %w", err)
	}
	if renderedExpected != normalizeFixtureMarkdown(string(expectedData)) {
		return fmt.Errorf("expected.md is out of date; run `noodle fixtures sync`")
	}
	return nil
}

func (inventory FixtureInventory) Names() []string {
	names := make([]string, 0, len(inventory.Cases))
	for _, fixtureCase := range inventory.Cases {
		names = append(names, fixtureCase.Name)
	}
	sort.Strings(names)
	return names
}

func (fixtureCase FixtureCase) Section(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	for key, value := range fixtureCase.Sections {
		if strings.EqualFold(strings.TrimSpace(key), name) {
			return value, true
		}
	}
	return "", false
}

func (fixtureCase FixtureCase) State(stateID string) (FixtureState, bool) {
	stateID = strings.TrimSpace(stateID)
	for _, state := range fixtureCase.States {
		if strings.EqualFold(state.ID, stateID) {
			return state, true
		}
	}
	return FixtureState{}, false
}

func (state FixtureState) FilePath(relPath string) (string, bool) {
	relPath = filepath.ToSlash(filepath.Clean(strings.TrimSpace(relPath)))
	relPath = strings.TrimPrefix(relPath, "./")
	path, ok := state.Files[relPath]
	return path, ok
}

func (state FixtureState) MustReadFile(tb testing.TB, relPath string) []byte {
	tb.Helper()
	path, ok := state.FilePath(relPath)
	if !ok {
		tb.Fatalf("state %s missing file %s (available: %v)", state.ID, relPath, state.FileOrder)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("read state file %s: %v", path, err)
	}
	return data
}

func (state FixtureState) MustReadText(tb testing.TB, relPath string) string {
	tb.Helper()
	return string(state.MustReadFile(tb, relPath))
}
