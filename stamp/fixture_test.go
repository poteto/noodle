package stamp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/internal/testutil/fixturedir"
	"github.com/poteto/noodle/parse"
)

func TestProcessorDirectoryFixtures(t *testing.T) {
	fixturedir.AssertValidFixtureRoot(t, "testdata")
	inventory := fixturedir.LoadInventory(t, "testdata")

	for _, fixtureCase := range inventory.Cases {
		fixtureCase := fixtureCase
		t.Run(fixtureCase.Name, func(t *testing.T) {
			errorExpectation := fixtureCase.ExpectedError
			processor := NewProcessor()
			processor.Now = func() time.Time {
				return time.Date(2026, 2, 22, 18, 20, 0, 0, time.UTC)
			}
			state := fixtureCase.States[0]

			input := state.MustReadFile(t, "input.ndjson")
			var stampedOut bytes.Buffer
			var eventsOut bytes.Buffer
			err := processor.Process(context.Background(), bytes.NewReader(input), &stampedOut, &eventsOut)
			fixturedir.AssertError(t, "stamp fixture", err, errorExpectation)
			if errorExpectation != nil {
				return
			}

			expectedStampedRaw := fixturedir.MustSection(t, fixtureCase, "Expected Stamped")
			actualStamped := readJSONObjects(t, stampedOut.Bytes())
			expectedStamped := readJSONObjects(
				t,
				[]byte(strings.TrimSpace(expectedStampedRaw)+"\n"),
			)
			if !reflect.DeepEqual(actualStamped, expectedStamped) {
				t.Fatalf("stamped fixture mismatch\nactual:   %#v\nexpected: %#v", actualStamped, expectedStamped)
			}

			expectedEventsRaw := fixturedir.MustSection(t, fixtureCase, "Expected Events")
			actualEvents := readCanonicalEvents(t, eventsOut.Bytes())
			expectedEvents := readCanonicalEvents(
				t,
				[]byte(strings.TrimSpace(expectedEventsRaw)+"\n"),
			)
			if !reflect.DeepEqual(actualEvents, expectedEvents) {
				t.Fatalf("events fixture mismatch\nactual:   %#v\nexpected: %#v", actualEvents, expectedEvents)
			}
		})
	}
}

func readJSONObjects(t *testing.T, data []byte) []map[string]any {
	t.Helper()

	lines := readNonEmptyLines(t, data)
	out := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Fatalf("parse JSON object %q: %v", line, err)
		}
		out = append(out, obj)
	}
	return out
}

func readCanonicalEvents(t *testing.T, data []byte) []parse.CanonicalEvent {
	t.Helper()

	lines := readNonEmptyLines(t, data)
	out := make([]parse.CanonicalEvent, 0, len(lines))
	for _, line := range lines {
		var event parse.CanonicalEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("parse canonical event %q: %v", line, err)
		}
		out = append(out, event)
	}
	return out
}

func readNonEmptyLines(t *testing.T, data []byte) []string {
	t.Helper()

	scanner := bufio.NewScanner(bytes.NewReader(data))
	lines := make([]string, 0)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan fixture lines: %v", err)
	}
	return lines
}
