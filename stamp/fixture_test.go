package stamp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/parse"
)

func TestProcessorMarkdownFixtures(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("testdata", "*.fixture.md"))
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		t.Fatal("no stamp fixtures found")
	}

	for _, fixturePath := range paths {
		fixturePath := fixturePath
		t.Run(filepath.Base(fixturePath), func(t *testing.T) {
			processor := NewProcessor()
			processor.Now = func() time.Time {
				return time.Date(2026, 2, 22, 18, 20, 0, 0, time.UTC)
			}

			input := []byte(strings.Join(readFixtureBlockLines(t, fixturePath, "Input"), "\n") + "\n")
			var stampedOut bytes.Buffer
			var eventsOut bytes.Buffer
			if err := processor.Process(context.Background(), bytes.NewReader(input), &stampedOut, &eventsOut); err != nil {
				t.Fatalf("process fixture input: %v", err)
			}

			actualStamped := readJSONObjects(t, stampedOut.Bytes())
			expectedStamped := readJSONObjects(
				t,
				[]byte(strings.Join(readFixtureBlockLines(t, fixturePath, "Expected Stamped"), "\n")+"\n"),
			)
			if !reflect.DeepEqual(actualStamped, expectedStamped) {
				t.Fatalf("stamped fixture mismatch\nactual:   %#v\nexpected: %#v", actualStamped, expectedStamped)
			}

			actualEvents := readCanonicalEvents(t, eventsOut.Bytes())
			expectedEvents := readCanonicalEvents(
				t,
				[]byte(strings.Join(readFixtureBlockLines(t, fixturePath, "Expected Events"), "\n")+"\n"),
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
