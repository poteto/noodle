package fixturedir

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func SyncExpectedMarkdown(root string, checkOnly bool) ([]string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		root = "."
	}
	root = filepath.Clean(root)

	sourcePaths := make([]string, 0)
	walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := strings.TrimSpace(d.Name())
			if name == ".git" || name == "bin" || name == ".worktrees" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(strings.TrimSpace(d.Name()), "expected.src.md") {
			sourcePaths = append(sourcePaths, path)
		}
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("walk fixture root: %w", walkErr)
	}
	sort.Strings(sourcePaths)

	changed := make([]string, 0)
	stale := make([]string, 0)
	for _, sourcePath := range sourcePaths {
		destinationPath := filepath.Join(filepath.Dir(sourcePath), "expected.md")

		sourceData, err := os.ReadFile(sourcePath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", sourcePath, err)
		}
		renderedExpected, err := renderExpectedMarkdownFromSource(string(sourceData))
		if err != nil {
			return nil, fmt.Errorf("render %s: %w", sourcePath, err)
		}

		destinationData, err := os.ReadFile(destinationPath)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("read %s: %w", destinationPath, err)
		}
		normalizedDestination := normalizeFixtureMarkdown(string(destinationData))
		if renderedExpected == normalizedDestination {
			continue
		}
		if checkOnly {
			stale = append(stale, destinationPath)
			continue
		}
		if err := os.WriteFile(destinationPath, []byte(renderedExpected), 0o644); err != nil {
			return nil, fmt.Errorf("write %s: %w", destinationPath, err)
		}
		changed = append(changed, destinationPath)
	}

	if checkOnly && len(stale) > 0 {
		return stale, fmt.Errorf("%d fixture expected.md file(s) are out of date", len(stale))
	}
	return changed, nil
}

func normalizeFixtureMarkdown(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.TrimRight(content, "\n")
	if content == "" {
		return "\n"
	}
	return content + "\n"
}

func renderExpectedMarkdownFromSource(source string) (string, error) {
	normalizedSource := normalizeFixtureMarkdown(source)
	sourceMetadata, _, _, err := parseExpectedDocument([]byte(normalizedSource))
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(sourceMetadata.SourceHash) != "" {
		return "", fmt.Errorf("expected.src.md must not declare source_hash")
	}
	_, body, err := splitFrontmatter(normalizedSource)
	if err != nil {
		return "", err
	}
	body = strings.TrimLeft(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	hash := sha256.Sum256([]byte(normalizedSource))

	var out strings.Builder
	out.WriteString("---\n")
	out.WriteString(fmt.Sprintf("schema_version: %d\n", sourceMetadata.SchemaVersion))
	out.WriteString(fmt.Sprintf("expected_failure: %t\n", sourceMetadata.ExpectedFailure))
	out.WriteString(fmt.Sprintf("bug: %t\n", sourceMetadata.Bug))
	out.WriteString(fmt.Sprintf("regression: %s\n", sourceMetadata.Regression))
	out.WriteString(fmt.Sprintf("source_hash: %s\n", hex.EncodeToString(hash[:]))) // hash of expected.src.md contents
	out.WriteString("---\n")
	if strings.TrimSpace(body) != "" {
		out.WriteString("\n")
		out.WriteString(body)
	}
	return normalizeFixtureMarkdown(out.String()), nil
}
