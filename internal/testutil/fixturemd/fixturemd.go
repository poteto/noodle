package fixturemd

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Paths returns sorted markdown fixture paths from dir.
func Paths(tb testing.TB, dir string) []string {
	tb.Helper()

	paths, err := filepath.Glob(filepath.Join(dir, "*.fixture.md"))
	if err != nil {
		tb.Fatalf("glob fixtures in %s: %v", dir, err)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		tb.Fatalf("no fixtures found in %s", dir)
	}
	return paths
}

// ReadSectionLines reads fenced lines under a markdown section heading.
func ReadSectionLines(tb testing.TB, fixturePath, section string) []string {
	tb.Helper()

	file, err := os.Open(fixturePath)
	if err != nil {
		tb.Fatalf("open %s: %v", fixturePath, err)
	}
	defer file.Close()

	heading := "## " + section
	insideSection := false
	insideFence := false
	lines := make([]string, 0)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "## ") {
			insideSection = trimmed == heading
			insideFence = false
			continue
		}
		if !insideSection {
			continue
		}
		if strings.HasPrefix(trimmed, "```") {
			insideFence = !insideFence
			continue
		}
		if !insideFence || trimmed == "" {
			continue
		}

		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		tb.Fatalf("scan %s: %v", fixturePath, err)
	}
	if len(lines) == 0 {
		tb.Fatalf("no lines found in %s section %q", fixturePath, section)
	}
	return lines
}

// IsErrorFixture reports whether a fixture expects failure semantics.
// Convention: filename starts with "error".
func IsErrorFixture(fixturePath string) bool {
	name := strings.ToLower(filepath.Base(strings.TrimSpace(fixturePath)))
	return strings.HasPrefix(name, "error")
}
