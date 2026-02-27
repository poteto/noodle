package snapshot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/poteto/noodle/internal/testutil/fixturedir"
)

type snapshotFixtureInput struct {
	Now time.Time `json:"now"`
}

func TestSnapshotDirectoryFixtures(t *testing.T) {
	inventory := fixturedir.LoadInventory(t, "testdata")
	fixturedir.AssertValidFixtureRoot(t, "testdata")

	mode := strings.ToLower(strings.TrimSpace(os.Getenv("NOODLE_SNAPSHOT_FIXTURE_MODE")))
	if mode == "" {
		mode = "check"
	}
	if mode != "check" && mode != "record" {
		t.Fatalf("invalid NOODLE_SNAPSHOT_FIXTURE_MODE %q (expected check|record)", mode)
	}

	for _, fixtureCase := range inventory.Cases {
		fixtureCase := fixtureCase
		t.Run(fixtureCase.Name, func(t *testing.T) {
			state := fixturedir.RequireSingleState(t, fixtureCase)

			input, _ := fixturedir.ParseOptionalStateJSON[snapshotFixtureInput](t, state, "input.json")
			now := input.Now
			if now.IsZero() {
				now = time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
			}

			projectDir := t.TempDir()
			runtimeDir := filepath.Join(projectDir, ".noodle")
			if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
				t.Fatalf("mkdir runtime: %v", err)
			}

			fixturedir.ApplyRuntimeSnapshot(t, state, runtimeDir)

			snap, err := LoadSnapshot(runtimeDir, now)
			if mode == "check" {
				fixturedir.AssertError(t, fixtureCase.Name, err, fixtureCase.ExpectedError)
				if err != nil {
					return
				}

				expected := fixturedir.ParseSectionJSON[Snapshot](t, fixtureCase, "Expected Snapshot")
				if !snapshotsEqual(snap, expected) {
					actualJSON := mustJSONIndent(snap)
					expectedJSON := mustJSONIndent(expected)
					t.Fatalf("snapshot mismatch\nactual:\n%s\nexpected:\n%s", actualJSON, expectedJSON)
				}
				return
			}

			// Record mode.
			if err != nil {
				t.Fatalf("LoadSnapshot failed in record mode: %v", err)
			}
			if err := fixturedir.WriteSectionToExpected(fixtureCase.Layout.ExpectedPath, "Expected Snapshot", snap); err != nil {
				t.Fatalf("write expected snapshot: %v", err)
			}
		})
	}
}

func snapshotsEqual(a, b Snapshot) bool {
	return reflect.DeepEqual(a, b)
}

func mustJSONIndent(v any) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(data)
}
