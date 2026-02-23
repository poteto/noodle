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

func ParseJSONLines[T any](tb testing.TB, data []byte, label string) []T {
	tb.Helper()

	lines := NonEmptyLines(tb, data, label)
	out := make([]T, 0, len(lines))
	for index, line := range lines {
		lineLabel := fmt.Sprintf("%s line %d", label, index+1)
		out = append(out, ParseJSON[T](tb, []byte(line), lineLabel))
	}
	return out
}

func RequireSingleState(tb testing.TB, fixtureCase FixtureCase) FixtureState {
	tb.Helper()
	if len(fixtureCase.States) != 1 {
		tb.Fatalf(
			"fixture %s must have exactly 1 state, got %d",
			fixtureCase.Name,
			len(fixtureCase.States),
		)
	}
	return fixtureCase.States[0]
}

func ForEachState(tb testing.TB, fixtureCase FixtureCase, fn func(index int, state FixtureState)) {
	tb.Helper()
	for index, state := range fixtureCase.States {
		fn(index, state)
	}
}

func NormalizeFixtureMarkdown(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.TrimRight(content, "\n")
	if content == "" {
		return "\n"
	}
	return content + "\n"
}
