package fixturemd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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
	lines, ok := ReadOptionalSectionLines(tb, fixturePath, section)
	if !ok {
		tb.Fatalf("no lines found in %s section %q", fixturePath, section)
	}
	return lines
}

// ReadOptionalSectionLines reads fenced lines under a markdown section heading.
// It returns ok=false when the section exists but has no fenced lines.
func ReadOptionalSectionLines(tb testing.TB, fixturePath, section string) ([]string, bool) {
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
	return lines, len(lines) > 0
}

// ReadSectionString reads a required markdown section as a newline-joined string.
func ReadSectionString(tb testing.TB, fixturePath, section string) string {
	tb.Helper()
	return strings.Join(ReadSectionLines(tb, fixturePath, section), "\n")
}

// ReadOptionalSectionString reads an optional markdown section as a newline-joined string.
func ReadOptionalSectionString(tb testing.TB, fixturePath, section string) (string, bool) {
	tb.Helper()
	lines, ok := ReadOptionalSectionLines(tb, fixturePath, section)
	if !ok {
		return "", false
	}
	return strings.Join(lines, "\n"), true
}

// ParseSectionJSON parses a required section as JSON into T.
func ParseSectionJSON[T any](tb testing.TB, fixturePath, section string) T {
	tb.Helper()
	raw := ReadSectionString(tb, fixturePath, section)
	var out T
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		tb.Fatalf("parse %s section %q JSON: %v", fixturePath, section, err)
	}
	return out
}

// ParseOptionalSectionJSON parses an optional section as JSON into T.
func ParseOptionalSectionJSON[T any](tb testing.TB, fixturePath, section string) (T, bool) {
	tb.Helper()
	raw, ok := ReadOptionalSectionString(tb, fixturePath, section)
	var zero T
	if !ok {
		return zero, false
	}
	var out T
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		tb.Fatalf("parse %s section %q JSON: %v", fixturePath, section, err)
	}
	return out, true
}

// IsErrorFixture reports whether a fixture expects failure semantics.
// Convention: filename starts with "error".
func IsErrorFixture(fixturePath string) bool {
	name := strings.ToLower(filepath.Base(strings.TrimSpace(fixturePath)))
	return strings.HasPrefix(name, "error")
}

// ErrorExpectation describes expected error behavior for a fixture.
type ErrorExpectation struct {
	Any      bool   `json:"any,omitempty"`
	Contains string `json:"contains,omitempty"`
	Equals   string `json:"equals,omitempty"`
	Absent   bool   `json:"absent,omitempty"`
}

// ExpectedError resolves error expectations from optional "Expected Error"
// section, then falls back to filename convention (error-* => any error).
func ExpectedError(tb testing.TB, fixturePath string) *ErrorExpectation {
	tb.Helper()

	if explicit, ok := ParseOptionalSectionJSON[ErrorExpectation](tb, fixturePath, "Expected Error"); ok {
		normalized := explicit
		if !normalized.Absent &&
			!normalized.Any &&
			strings.TrimSpace(normalized.Contains) == "" &&
			strings.TrimSpace(normalized.Equals) == "" {
			normalized.Any = true
		}
		return &normalized
	}
	if IsErrorFixture(fixturePath) {
		return &ErrorExpectation{Any: true}
	}
	return nil
}

// AssertError verifies the actual error against fixture expectations.
func AssertError(tb testing.TB, context string, err error, expect *ErrorExpectation) {
	tb.Helper()

	if context = strings.TrimSpace(context); context == "" {
		context = "fixture"
	}
	if expect == nil {
		if err != nil {
			tb.Fatalf("%s error: %v", context, err)
		}
		return
	}
	if expect.Absent {
		if err != nil {
			tb.Fatalf("%s unexpected error: %v", context, err)
		}
		return
	}
	if err == nil {
		tb.Fatalf("%s expected error", context)
	}
	if want := strings.TrimSpace(expect.Equals); want != "" && err.Error() != want {
		tb.Fatalf("%s error = %q, want exact %q", context, err.Error(), want)
	}
	if want := strings.TrimSpace(expect.Contains); want != "" && !strings.Contains(err.Error(), want) {
		tb.Fatalf("%s error = %q, want contains %q", context, err.Error(), want)
	}
}

// CountExpectation supports exact and range checks for count assertions.
type CountExpectation struct {
	Eq  *int `json:"eq,omitempty"`
	Min *int `json:"min,omitempty"`
	Max *int `json:"max,omitempty"`
}

// StringExpectation supports semantic string assertions.
type StringExpectation struct {
	Equals   string `json:"equals,omitempty"`
	Contains string `json:"contains,omitempty"`
	Prefix   string `json:"prefix,omitempty"`
}

// AssertCounts checks integer metrics by key.
func AssertCounts(tb testing.TB, label string, got map[string]int, want map[string]CountExpectation) {
	tb.Helper()
	for key, expect := range want {
		actual, ok := got[key]
		if !ok {
			tb.Fatalf("%s missing count key %q", label, key)
		}
		if expect.Eq != nil && actual != *expect.Eq {
			tb.Fatalf("%s[%s] = %d, want eq %d", label, key, actual, *expect.Eq)
		}
		if expect.Min != nil && actual < *expect.Min {
			tb.Fatalf("%s[%s] = %d, want min %d", label, key, actual, *expect.Min)
		}
		if expect.Max != nil && actual > *expect.Max {
			tb.Fatalf("%s[%s] = %d, want max %d", label, key, actual, *expect.Max)
		}
	}
}

// AssertBools checks boolean metrics by key.
func AssertBools(tb testing.TB, label string, got map[string]bool, want map[string]bool) {
	tb.Helper()
	for key, expected := range want {
		actual, ok := got[key]
		if !ok {
			tb.Fatalf("%s missing bool key %q", label, key)
		}
		if actual != expected {
			tb.Fatalf("%s[%s] = %v, want %v", label, key, actual, expected)
		}
	}
}

// AssertStrings checks string metrics by key using semantic expectations.
func AssertStrings(tb testing.TB, label string, got map[string]string, want map[string]StringExpectation) {
	tb.Helper()
	for key, expected := range want {
		actual, ok := got[key]
		if !ok {
			tb.Fatalf("%s missing string key %q", label, key)
		}
		if value := strings.TrimSpace(expected.Equals); value != "" && actual != value {
			tb.Fatalf("%s[%s] = %q, want equals %q", label, key, actual, value)
		}
		if value := strings.TrimSpace(expected.Contains); value != "" && !strings.Contains(actual, value) {
			tb.Fatalf("%s[%s] = %q, want contains %q", label, key, actual, value)
		}
		if value := strings.TrimSpace(expected.Prefix); value != "" && !strings.HasPrefix(actual, value) {
			tb.Fatalf("%s[%s] = %q, want prefix %q", label, key, actual, value)
		}
	}
}

// AssertSequence checks ordered state transitions.
func AssertSequence[T comparable](tb testing.TB, label string, got, want []T) {
	tb.Helper()
	if !reflect.DeepEqual(got, want) {
		tb.Fatalf("%s = %v, want %v", label, got, want)
	}
}

// Keys returns the sorted keys for deterministic error/debug messages.
func Keys[V any](input map[string]V) []string {
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// DebugMap formats a map with sorted keys for easy failure context.
func DebugMap[V any](input map[string]V) string {
	keys := Keys(input)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", key, input[key]))
	}
	return strings.Join(parts, ", ")
}
