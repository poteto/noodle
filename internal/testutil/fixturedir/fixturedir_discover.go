package fixturedir

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

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
		if syncErr := assertExpectedMarkdownSynced(layout.RootPath, layout.ExpectedPath); syncErr != nil {
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
	if err := assertExpectedMarkdownSynced(layout.RootPath, layout.ExpectedPath); err != nil {
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
				stateConfigPath := filepath.Join(path, ".noodle.toml")
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
		case ".noodle.toml":
			layout.BaseConfigPath = path
		default:
			issues = append(issues, FixtureValidationIssue{
				Path:     filepath.ToSlash(path),
				Severity: "error",
				Message:  "unexpected file; allowed files are expected.md and optional .noodle.toml",
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

	paths, gitScoped, err := listGitVisibleFiles(statePath)
	if err != nil {
		return nil, nil, err
	}
	if gitScoped {
		for _, path := range paths {
			relPath, relErr := filepath.Rel(statePath, path)
			if relErr != nil {
				return nil, nil, fmt.Errorf("compute relative path: %w", relErr)
			}
			normalized := filepath.ToSlash(filepath.Clean(relPath))
			files[normalized] = path
			ordered = append(ordered, normalized)
		}
		sort.Strings(ordered)
		return files, ordered, nil
	}

	err = filepath.WalkDir(statePath, func(path string, d os.DirEntry, walkErr error) error {
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
