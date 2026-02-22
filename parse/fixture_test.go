package parse

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestMarkdownFixtures(t *testing.T) {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join("testdata", "*.fixture.md"))
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		t.Fatal("no parse fixtures found")
	}

	for _, fixturePath := range paths {
		fixturePath := fixturePath
		t.Run(filepath.Base(fixturePath), func(t *testing.T) {
			adapter := adapterForFixture(t, fixturePath)
			inputLines := readFixtureBlockLines(t, fixturePath, "Input")
			expectedEvents := parseCanonicalEvents(t, readFixtureBlockLines(t, fixturePath, "Expected Events"))

			actualEvents := make([]CanonicalEvent, 0)
			for _, line := range inputLines {
				events, err := adapter.Parse([]byte(line))
				if err != nil {
					t.Fatalf("parse fixture line %q: %v", line, err)
				}
				actualEvents = append(actualEvents, events...)
			}

			if !reflect.DeepEqual(actualEvents, expectedEvents) {
				t.Fatalf("fixture mismatch\nactual:   %#v\nexpected: %#v", actualEvents, expectedEvents)
			}
		})
	}
}

func adapterForFixture(t *testing.T, fixturePath string) LogAdapter {
	t.Helper()

	name := filepath.Base(fixturePath)
	switch {
	case strings.HasPrefix(name, "claude"):
		return ClaudeAdapter{}
	case strings.HasPrefix(name, "codex"):
		return CodexAdapter{}
	default:
		t.Fatalf("fixture %s must start with claude or codex", fixturePath)
		return nil
	}
}

func parseCanonicalEvents(t *testing.T, lines []string) []CanonicalEvent {
	t.Helper()

	events := make([]CanonicalEvent, 0, len(lines))
	for _, line := range lines {
		var event CanonicalEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("parse expected event %q: %v", line, err)
		}
		events = append(events, event)
	}
	return events
}

func readFixtureBlockLines(t *testing.T, fixturePath, section string) []string {
	t.Helper()

	file, err := os.Open(fixturePath)
	if err != nil {
		t.Fatalf("open %s: %v", fixturePath, err)
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
		t.Fatalf("scan %s: %v", fixturePath, err)
	}
	if len(lines) == 0 {
		t.Fatalf("no fixture lines found in %s section %q", fixturePath, section)
	}
	return lines
}
