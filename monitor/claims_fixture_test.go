package monitor

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

func TestClaimsMarkdownFixtures(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("testdata", "*.fixture.md"))
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		t.Fatal("no monitor fixtures found")
	}

	for _, fixturePath := range paths {
		fixturePath := fixturePath
		t.Run(filepath.Base(fixturePath), func(t *testing.T) {
			runtimeDir := t.TempDir()
			sessionID := "cook-a"
			sessionPath := filepath.Join(runtimeDir, "sessions", sessionID)
			if err := os.MkdirAll(sessionPath, 0o755); err != nil {
				t.Fatalf("mkdir session path: %v", err)
			}

			input := strings.Join(readFixtureBlockLines(t, fixturePath, "Input"), "\n") + "\n"
			if err := os.WriteFile(filepath.Join(sessionPath, "canonical.ndjson"), []byte(input), 0o644); err != nil {
				t.Fatalf("write canonical fixture: %v", err)
			}

			reader := NewCanonicalClaimsReader(runtimeDir)
			actual, err := reader.ReadSession(sessionID)
			if err != nil {
				t.Fatalf("read fixture claims: %v", err)
			}

			var expected SessionClaims
			if err := json.Unmarshal(
				[]byte(strings.Join(readFixtureBlockLines(t, fixturePath, "Expected Claims"), "\n")),
				&expected,
			); err != nil {
				t.Fatalf("parse expected claims: %v", err)
			}

			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("claims mismatch\nactual:   %#v\nexpected: %#v", actual, expected)
			}
		})
	}
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
		t.Fatalf("scan fixture file: %v", err)
	}
	if len(lines) == 0 {
		t.Fatalf("no lines found in fixture section %q", section)
	}
	return lines
}

func TestReadSessionMissingFile(t *testing.T) {
	reader := NewCanonicalClaimsReader(t.TempDir())
	claims, err := reader.ReadSession("cook-a")
	if err != nil {
		t.Fatalf("read missing session: %v", err)
	}
	if claims.HasEvents {
		t.Fatal("claims should report no events for missing file")
	}
	if claims.SessionID != "cook-a" {
		t.Fatalf("session ID = %q", claims.SessionID)
	}
	if !claims.LastEventAt.IsZero() || !claims.FirstEventAt.IsZero() {
		t.Fatal("event timestamps should be zero for missing file")
	}
}

func TestReadSessionUsesSpawnMetadataForModel(t *testing.T) {
	runtimeDir := t.TempDir()
	sessionID := "cook-a"
	sessionPath := filepath.Join(runtimeDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionPath, 0o755); err != nil {
		t.Fatalf("mkdir session path: %v", err)
	}

	canonical := `{"provider":"claude","type":"init","message":"started","timestamp":"2026-02-22T15:00:00Z"}`
	if err := os.WriteFile(filepath.Join(sessionPath, "canonical.ndjson"), []byte(canonical+"\n"), 0o644); err != nil {
		t.Fatalf("write canonical events: %v", err)
	}
	spawn := `{"provider":"claude","model":"claude-opus-4-6"}`
	if err := os.WriteFile(filepath.Join(sessionPath, "spawn.json"), []byte(spawn), 0o644); err != nil {
		t.Fatalf("write spawn metadata: %v", err)
	}

	reader := NewCanonicalClaimsReader(runtimeDir)
	claims, err := reader.ReadSession(sessionID)
	if err != nil {
		t.Fatalf("read session claims: %v", err)
	}
	if claims.Provider != "claude" {
		t.Fatalf("provider = %q", claims.Provider)
	}
	if claims.Model != "claude-opus-4-6" {
		t.Fatalf("model = %q", claims.Model)
	}
}

func TestReadSessionIgnoresMalformedSpawnMetadata(t *testing.T) {
	runtimeDir := t.TempDir()
	sessionID := "cook-a"
	sessionPath := filepath.Join(runtimeDir, "sessions", sessionID)
	if err := os.MkdirAll(sessionPath, 0o755); err != nil {
		t.Fatalf("mkdir session path: %v", err)
	}

	canonical := `{"provider":"codex","type":"action","message":"run tests","timestamp":"2026-02-22T15:00:00Z"}`
	if err := os.WriteFile(filepath.Join(sessionPath, "canonical.ndjson"), []byte(canonical+"\n"), 0o644); err != nil {
		t.Fatalf("write canonical events: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionPath, "spawn.json"), []byte("{bad-json"), 0o644); err != nil {
		t.Fatalf("write malformed spawn metadata: %v", err)
	}

	reader := NewCanonicalClaimsReader(runtimeDir)
	claims, err := reader.ReadSession(sessionID)
	if err != nil {
		t.Fatalf("read session claims: %v", err)
	}
	if claims.Provider != "codex" {
		t.Fatalf("provider = %q", claims.Provider)
	}
	if claims.Model != "" {
		t.Fatalf("model = %q", claims.Model)
	}
}
