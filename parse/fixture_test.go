package parse

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/poteto/noodle/internal/testutil/fixturemd"
)

func TestMarkdownFixtures(t *testing.T) {
	t.Helper()

	paths := fixturemd.Paths(t, "testdata")

	for _, fixturePath := range paths {
		fixturePath := fixturePath
		t.Run(filepath.Base(fixturePath), func(t *testing.T) {
			adapter := adapterForFixture(t, fixturePath)
			inputLines := fixturemd.ReadSectionLines(t, fixturePath, "Input")
			expectError := fixturemd.IsErrorFixture(fixturePath)

			expectedEvents := make([]CanonicalEvent, 0)
			if !expectError {
				expectedEvents = parseCanonicalEvents(
					t,
					fixturemd.ReadSectionLines(t, fixturePath, "Expected Events"),
				)
			}

			actualEvents := make([]CanonicalEvent, 0)
			var parseErr error
			for _, line := range inputLines {
				events, err := adapter.Parse([]byte(line))
				if err != nil {
					parseErr = err
					break
				}
				actualEvents = append(actualEvents, events...)
			}

			if expectError {
				if parseErr == nil {
					t.Fatalf("expected parse error for fixture %s", filepath.Base(fixturePath))
				}
				return
			}
			if parseErr != nil {
				t.Fatalf("parse fixture failed: %v", parseErr)
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
