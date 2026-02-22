package event

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestTicketMaterializerMarkdownFixtures(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("testdata", "*.fixture.md"))
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no fixture files found")
	}
	sort.Strings(paths)

	for _, path := range paths {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			runTicketFixture(t, path)
		})
	}
}

func runTicketFixture(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	content := string(data)

	sessionsBlock := extractFencedBlockForSection(content, "Sessions")
	optionsBlock := extractFencedBlockForSection(content, "Options")
	expectedBlock := extractFencedBlockForSection(content, "Expected Tickets")
	if sessionsBlock == "" || optionsBlock == "" || expectedBlock == "" {
		t.Fatalf("fixture %s is missing required sections", path)
	}

	var sessions map[string][]Event
	if err := json.Unmarshal([]byte(sessionsBlock), &sessions); err != nil {
		t.Fatalf("decode sessions block: %v", err)
	}

	var options struct {
		Now            time.Time `json:"now"`
		Timeout        string    `json:"timeout"`
		ActiveSessions []string  `json:"active_sessions"`
	}
	if err := json.Unmarshal([]byte(optionsBlock), &options); err != nil {
		t.Fatalf("decode options block: %v", err)
	}
	timeout, err := time.ParseDuration(options.Timeout)
	if err != nil {
		t.Fatalf("parse timeout %q: %v", options.Timeout, err)
	}

	var expected []Ticket
	if err := json.Unmarshal([]byte(expectedBlock), &expected); err != nil {
		t.Fatalf("decode expected tickets: %v", err)
	}

	runtimeDir := filepath.Join(t.TempDir(), ".noodle")
	for sessionID, events := range sessions {
		writer, err := NewEventWriter(runtimeDir, sessionID)
		if err != nil {
			t.Fatalf("new writer %s: %v", sessionID, err)
		}
		for _, event := range events {
			if event.SessionID == "" {
				event.SessionID = sessionID
			}
			if err := writer.Append(context.Background(), event); err != nil {
				t.Fatalf("append fixture event for %s: %v", sessionID, err)
			}
		}
	}

	materializer := NewTicketMaterializer(runtimeDir)
	materializer.now = func() time.Time { return options.Now }
	materializer.staleTimeout = timeout

	actual, err := materializer.Materialize(context.Background(), options.ActiveSessions)
	if err != nil {
		t.Fatalf("materialize tickets: %v", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("fixture mismatch\nactual:   %#v\nexpected: %#v", actual, expected)
	}
}

func extractFencedBlockForSection(content, section string) string {
	needle := "## " + section
	idx := strings.Index(content, needle)
	if idx < 0 {
		return ""
	}
	return extractFencedBlock(content[idx:])
}

func extractFencedBlock(content string) string {
	start := strings.Index(content, "```")
	if start < 0 {
		return ""
	}
	rest := content[start+3:]
	if newline := strings.Index(rest, "\n"); newline >= 0 {
		rest = rest[newline+1:]
	}
	end := strings.Index(rest, "```")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:end])
}
