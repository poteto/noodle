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
			errorExpectation := fixturemd.ExpectedError(t, fixturePath)

			expectedEvents := make([]CanonicalEvent, 0)
			if errorExpectation == nil {
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

			fixturemd.AssertError(t, "parse fixture", parseErr, errorExpectation)
			if errorExpectation != nil {
				return
			}
			if !reflect.DeepEqual(actualEvents, expectedEvents) {
				t.Fatalf("fixture mismatch\nactual:   %#v\nexpected: %#v", actualEvents, expectedEvents)
			}
		})
	}
}

func adapterForFixture(t *testing.T, fixturePath string) LogAdapter {
	t.Helper()

	name := strings.ToLower(filepath.Base(fixturePath))
	name = strings.TrimPrefix(name, "error-")
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
