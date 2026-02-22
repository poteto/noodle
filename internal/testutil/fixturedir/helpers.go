package fixturedir

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func AssertValidFixtureRoot(tb testing.TB, root string) {
	tb.Helper()
	issues, err := ValidateFixtureRoot(root)
	if err != nil {
		tb.Fatalf("validate fixture root %s: %v", root, err)
	}
	if len(issues) == 0 {
		return
	}
	messages := make([]string, 0, len(issues))
	for _, issue := range issues {
		messages = append(messages, fmt.Sprintf("%s: %s", issue.Path, issue.Message))
	}
	tb.Fatalf("fixture validation failed:\n%s", strings.Join(messages, "\n"))
}

func MustSection(tb testing.TB, fixtureCase FixtureCase, section string) string {
	tb.Helper()
	raw, ok := fixtureCase.Section(section)
	if !ok {
		tb.Fatalf("fixture %s missing %s section", fixtureCase.Name, section)
	}
	return raw
}

func ParseSectionJSON[T any](tb testing.TB, fixtureCase FixtureCase, section string) T {
	tb.Helper()
	return ParseJSON[T](tb, []byte(MustSection(tb, fixtureCase, section)), section)
}

func ParseStateJSON[T any](tb testing.TB, state FixtureState, relPath string) T {
	tb.Helper()
	return ParseJSON[T](tb, state.MustReadFile(tb, relPath), relPath)
}

func ParseOptionalStateJSON[T any](tb testing.TB, state FixtureState, relPath string) (T, bool) {
	tb.Helper()
	path, ok := state.FilePath(relPath)
	if !ok {
		var zero T
		return zero, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("read fixture file %s: %v", path, err)
	}
	return ParseJSON[T](tb, data, relPath), true
}

func ParseJSON[T any](tb testing.TB, data []byte, label string) T {
	tb.Helper()
	var out T
	if err := json.Unmarshal(data, &out); err != nil {
		tb.Fatalf("parse %s JSON: %v", label, err)
	}
	return out
}

func NonEmptyLines(tb testing.TB, data []byte, label string) []string {
	tb.Helper()
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	lines := make([]string, 0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		tb.Fatalf("scan %s: %v", label, err)
	}
	return lines
}
