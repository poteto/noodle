package monitor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/poteto/noodle/internal/testutil/fixturedir"
)

func TestClaimsDirectoryFixtures(t *testing.T) {
	fixturedir.AssertValidFixtureRoot(t, "testdata")
	inventory := fixturedir.LoadInventory(t, "testdata")
	for _, fixtureCase := range inventory.Cases {
		fixtureCase := fixtureCase
		t.Run(fixtureCase.Name, func(t *testing.T) {
			runtimeDir := t.TempDir()
			sessionID := "cook-a"
			sessionPath := filepath.Join(runtimeDir, "sessions", sessionID)
			if err := os.MkdirAll(sessionPath, 0o755); err != nil {
				t.Fatalf("mkdir session path: %v", err)
			}
			state := fixtureCase.States[0]

			input := strings.Join(fixturedir.NonEmptyLines(t, state.MustReadFile(t, "input.ndjson"), "input.ndjson"), "\n") + "\n"
			if err := os.WriteFile(filepath.Join(sessionPath, "canonical.ndjson"), []byte(input), 0o644); err != nil {
				t.Fatalf("write canonical fixture: %v", err)
			}

			reader := NewCanonicalClaimsReader(runtimeDir)
			actual, err := reader.ReadSession(sessionID)
			if err != nil {
				t.Fatalf("read fixture claims: %v", err)
			}

			expectedRaw := fixturedir.MustSection(t, fixtureCase, "Expected Claims")
			var expected SessionClaims
			if err := json.Unmarshal([]byte(expectedRaw), &expected); err != nil {
				t.Fatalf("parse expected claims: %v", err)
			}

			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("claims mismatch\nactual:   %#v\nexpected: %#v", actual, expected)
			}
		})
	}
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
