package mise

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadQualityVerdicts_MultipleSortedByTimestamp(t *testing.T) {
	dir := t.TempDir()
	qualityDir := filepath.Join(dir, "quality")
	if err := os.MkdirAll(qualityDir, 0o755); err != nil {
		t.Fatalf("create quality directory: %v", err)
	}

	older := QualityVerdict{
		SessionID: "cook-a",
		Accept:    true,
		Timestamp: time.Date(2026, 2, 20, 10, 0, 0, 0, time.UTC),
	}
	newer := QualityVerdict{
		SessionID: "cook-b",
		TargetID:  "42",
		Accept:    false,
		Feedback:  "tests missing",
		Timestamp: time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC),
	}

	writeVerdict(t, qualityDir, "older.json", older)
	writeVerdict(t, qualityDir, "newer.json", newer)

	verdicts, err := ReadQualityVerdicts(dir)
	if err != nil {
		t.Fatalf("read quality verdicts: %v", err)
	}
	if len(verdicts) != 2 {
		t.Fatalf("verdict count = %d, want 2", len(verdicts))
	}
	if verdicts[0].SessionID != "cook-b" {
		t.Fatalf("first verdict session = %q, want cook-b (newest)", verdicts[0].SessionID)
	}
	if verdicts[1].SessionID != "cook-a" {
		t.Fatalf("second verdict session = %q, want cook-a (oldest)", verdicts[1].SessionID)
	}
}

func TestReadQualityVerdicts_NonExistentDirectory(t *testing.T) {
	dir := t.TempDir()

	verdicts, err := ReadQualityVerdicts(dir)
	if err != nil {
		t.Fatalf("unexpected error for missing directory: %v", err)
	}
	if len(verdicts) != 0 {
		t.Fatalf("verdict count = %d, want 0", len(verdicts))
	}
}

func TestReadQualityVerdicts_MalformedJSONSkipped(t *testing.T) {
	dir := t.TempDir()
	qualityDir := filepath.Join(dir, "quality")
	if err := os.MkdirAll(qualityDir, 0o755); err != nil {
		t.Fatalf("create quality directory: %v", err)
	}

	good := QualityVerdict{
		SessionID: "cook-a",
		Accept:    true,
		Timestamp: time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC),
	}
	writeVerdict(t, qualityDir, "good.json", good)

	if err := os.WriteFile(filepath.Join(qualityDir, "bad.json"), []byte("{invalid json"), 0o644); err != nil {
		t.Fatalf("write malformed file: %v", err)
	}

	verdicts, err := ReadQualityVerdicts(dir)
	if err != nil {
		t.Fatalf("unexpected error with malformed file: %v", err)
	}
	if len(verdicts) != 1 {
		t.Fatalf("verdict count = %d, want 1 (malformed file should be skipped)", len(verdicts))
	}
	if verdicts[0].SessionID != "cook-a" {
		t.Fatalf("verdict session = %q, want cook-a", verdicts[0].SessionID)
	}
}

func TestReadQualityVerdicts_CappedAt20(t *testing.T) {
	dir := t.TempDir()
	qualityDir := filepath.Join(dir, "quality")
	if err := os.MkdirAll(qualityDir, 0o755); err != nil {
		t.Fatalf("create quality directory: %v", err)
	}

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 25; i++ {
		v := QualityVerdict{
			SessionID: "cook-" + string(rune('a'+i)),
			Accept:    true,
			Timestamp: base.Add(time.Duration(i) * time.Hour),
		}
		name := "verdict-" + string(rune('a'+i)) + ".json"
		writeVerdict(t, qualityDir, name, v)
	}

	verdicts, err := ReadQualityVerdicts(dir)
	if err != nil {
		t.Fatalf("read quality verdicts: %v", err)
	}
	if len(verdicts) != 20 {
		t.Fatalf("verdict count = %d, want 20", len(verdicts))
	}
	// Newest should be first (i=24 is newest).
	if !verdicts[0].Timestamp.After(verdicts[1].Timestamp) {
		t.Fatal("verdicts not sorted newest first")
	}
}

func TestReadQualityVerdicts_NonJSONFilesIgnored(t *testing.T) {
	dir := t.TempDir()
	qualityDir := filepath.Join(dir, "quality")
	if err := os.MkdirAll(qualityDir, 0o755); err != nil {
		t.Fatalf("create quality directory: %v", err)
	}

	good := QualityVerdict{
		SessionID: "cook-a",
		Accept:    true,
		Timestamp: time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC),
	}
	writeVerdict(t, qualityDir, "good.json", good)

	if err := os.WriteFile(filepath.Join(qualityDir, "notes.txt"), []byte("not a verdict"), 0o644); err != nil {
		t.Fatalf("write txt file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(qualityDir, "data.yaml"), []byte("key: value"), 0o644); err != nil {
		t.Fatalf("write yaml file: %v", err)
	}

	verdicts, err := ReadQualityVerdicts(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(verdicts) != 1 {
		t.Fatalf("verdict count = %d, want 1 (non-json files should be ignored)", len(verdicts))
	}
}

func writeVerdict(t *testing.T, dir, name string, v QualityVerdict) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal verdict: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		t.Fatalf("write verdict file %s: %v", name, err)
	}
}
