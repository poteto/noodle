package fixturedir

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
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

	expectedPaths := make([]string, 0)
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
		if strings.EqualFold(strings.TrimSpace(d.Name()), "expected.md") {
			expectedPaths = append(expectedPaths, path)
		}
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("walk fixture root: %w", walkErr)
	}
	sort.Strings(expectedPaths)

	changed := make([]string, 0)
	stale := make([]string, 0)
	for _, expectedPath := range expectedPaths {
		fixtureRoot := filepath.Dir(expectedPath)
		expectedData, err := os.ReadFile(expectedPath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", expectedPath, err)
		}

		sourceHash, err := computeFixtureInputHash(fixtureRoot)
		if err != nil {
			return nil, err
		}
		renderedExpected, err := renderExpectedMarkdownWithSourceHash(string(expectedData), sourceHash)
		if err != nil {
			return nil, fmt.Errorf("render %s: %w", expectedPath, err)
		}
		if renderedExpected == normalizeFixtureMarkdown(string(expectedData)) {
			continue
		}
		if checkOnly {
			stale = append(stale, expectedPath)
			continue
		}
		if err := os.WriteFile(expectedPath, []byte(renderedExpected), 0o644); err != nil {
			return nil, fmt.Errorf("write %s: %w", expectedPath, err)
		}
		changed = append(changed, expectedPath)
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

func renderExpectedMarkdownWithSourceHash(expected string, sourceHash string) (string, error) {
	normalizedExpected := normalizeFixtureMarkdown(expected)
	metadata, _, _, err := parseExpectedDocument([]byte(normalizedExpected))
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(sourceHash) == "" {
		return "", fmt.Errorf("source_hash cannot be empty")
	}
	_, body, err := splitFrontmatter(normalizedExpected)
	if err != nil {
		return "", err
	}
	body = strings.TrimLeft(strings.ReplaceAll(body, "\r\n", "\n"), "\n")

	var out strings.Builder
	out.WriteString("---\n")
	out.WriteString(fmt.Sprintf("schema_version: %d\n", metadata.SchemaVersion))
	out.WriteString(fmt.Sprintf("expected_failure: %t\n", metadata.ExpectedFailure))
	out.WriteString(fmt.Sprintf("bug: %t\n", metadata.Bug))
	out.WriteString(fmt.Sprintf("source_hash: %s\n", sourceHash))
	out.WriteString("---\n")
	if strings.TrimSpace(body) != "" {
		out.WriteString("\n")
		out.WriteString(body)
	}
	return normalizeFixtureMarkdown(out.String()), nil
}

func computeFixtureInputHash(fixtureRoot string) (string, error) {
	fixtureRoot = filepath.Clean(strings.TrimSpace(fixtureRoot))
	if fixtureRoot == "" {
		return "", fmt.Errorf("fixture root is required")
	}

	paths, gitScoped, err := listGitVisibleFiles(fixtureRoot)
	if err != nil {
		return "", err
	}
	if !gitScoped {
		paths, err = listAllFixtureInputs(fixtureRoot)
		if err != nil {
			return "", err
		}
	}
	sort.Strings(paths)

	hash := sha256.New()
	for _, path := range paths {
		if strings.EqualFold(strings.TrimSpace(filepath.Base(path)), "expected.md") {
			continue
		}
		relPath, err := filepath.Rel(fixtureRoot, path)
		if err != nil {
			return "", fmt.Errorf("relative path for %s: %w", path, err)
		}
		normalized := filepath.ToSlash(filepath.Clean(relPath))
		if _, err := io.WriteString(hash, normalized); err != nil {
			return "", fmt.Errorf("hash path %s: %w", normalized, err)
		}
		if _, err := hash.Write([]byte{0}); err != nil {
			return "", fmt.Errorf("hash path separator: %w", err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read fixture input %s: %w", path, err)
		}
		if _, err := hash.Write(data); err != nil {
			return "", fmt.Errorf("hash file %s: %w", normalized, err)
		}
		if _, err := hash.Write([]byte{0}); err != nil {
			return "", fmt.Errorf("hash file separator: %w", err)
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func listAllFixtureInputs(fixtureRoot string) ([]string, error) {
	paths := make([]string, 0)
	err := filepath.WalkDir(fixtureRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		name := strings.TrimSpace(d.Name())
		if strings.EqualFold(name, "expected.md") {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk fixture inputs %s: %w", fixtureRoot, err)
	}
	return paths, nil
}
