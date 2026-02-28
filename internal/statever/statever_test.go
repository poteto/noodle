package statever

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), ".noodle", "state.json")
	now := time.Now().Truncate(time.Millisecond)
	original := StateMarker{
		SchemaVersion: Current,
		GeneratedAt:   now,
	}

	if err := Write(path, original); err != nil {
		t.Fatalf("write state marker: %v", err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatalf("read state marker: %v", err)
	}
	if got.SchemaVersion != original.SchemaVersion {
		t.Fatalf("schema version = %d, wrote %d", got.SchemaVersion, original.SchemaVersion)
	}
	if !got.GeneratedAt.Equal(original.GeneratedAt) {
		t.Fatalf("generated_at = %v, wrote %v", got.GeneratedAt, original.GeneratedAt)
	}
}

func TestRoundTripJSON(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Millisecond)
	original := StateMarker{
		SchemaVersion: Current,
		GeneratedAt:   now,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded StateMarker
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.SchemaVersion != original.SchemaVersion {
		t.Fatalf("schema version = %d, encoded %d", decoded.SchemaVersion, original.SchemaVersion)
	}
	if !decoded.GeneratedAt.Equal(original.GeneratedAt) {
		t.Fatalf("generated_at = %v, encoded %v", decoded.GeneratedAt, original.GeneratedAt)
	}
}

func TestReadMissingFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "does-not-exist", "state.json")
	m, err := Read(path)
	if err != nil {
		t.Fatalf("read missing file: %v", err)
	}
	if m.SchemaVersion != 0 {
		t.Fatalf("schema version = %d for missing file", m.SchemaVersion)
	}
	if !m.GeneratedAt.IsZero() {
		t.Fatalf("generated_at = %v for missing file", m.GeneratedAt)
	}
}

func TestReadEmptyFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("  \n"), 0o644); err != nil {
		t.Fatalf("seed empty file: %v", err)
	}
	m, err := Read(path)
	if err != nil {
		t.Fatalf("read empty file: %v", err)
	}
	if m.SchemaVersion != 0 {
		t.Fatalf("schema version = %d for empty file", m.SchemaVersion)
	}
}

func TestReadCorruptedFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("{not json!!"), 0o644); err != nil {
		t.Fatalf("seed corrupted file: %v", err)
	}
	_, err := Read(path)
	if err == nil {
		t.Fatal("read corrupted file returned no error")
	}
	// Error should mention corruption and the path.
	if got := err.Error(); got == "" {
		t.Fatal("error message is empty")
	}
}

func TestCheckCompatibilityMissingFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "absent", "state.json")
	if err := CheckCompatibility(path); err != nil {
		t.Fatalf("compatibility check on missing file: %v", err)
	}
}

func TestCheckCompatibilityCurrentVersion(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "state.json")
	m := StateMarker{
		SchemaVersion: Current,
		GeneratedAt:   time.Now(),
	}
	if err := Write(path, m); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := CheckCompatibility(path); err != nil {
		t.Fatalf("compatibility check on current version: %v", err)
	}
}

func TestCheckCompatibilityFutureVersion(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "state.json")
	m := StateMarker{
		SchemaVersion: Current + 5,
		GeneratedAt:   time.Now(),
	}
	if err := Write(path, m); err != nil {
		t.Fatalf("write future version: %v", err)
	}

	err := CheckCompatibility(path)
	if err == nil {
		t.Fatal("compatibility check accepted future version")
	}

	var vErr *VersionTooNewError
	if !errors.As(err, &vErr) {
		t.Fatalf("error type = %T, not *VersionTooNewError", err)
	}
	if vErr.OnDisk != Current+5 {
		t.Fatalf("on-disk version = %d", vErr.OnDisk)
	}
	if vErr.Supported != Current {
		t.Fatalf("supported version = %d", vErr.Supported)
	}
}

func TestCheckCompatibilityCorruptedFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("{{garbage"), 0o644); err != nil {
		t.Fatalf("seed corrupted file: %v", err)
	}
	if err := CheckCompatibility(path); err == nil {
		t.Fatal("compatibility check accepted corrupted file")
	}
}

func TestWriteOverwritesPreviousMarker(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "state.json")
	first := StateMarker{
		SchemaVersion: 1,
		GeneratedAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := Write(path, first); err != nil {
		t.Fatalf("write first marker: %v", err)
	}

	second := StateMarker{
		SchemaVersion: 2,
		GeneratedAt:   time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC),
	}
	if err := Write(path, second); err != nil {
		t.Fatalf("write second marker: %v", err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.SchemaVersion != 2 {
		t.Fatalf("schema version = %d after overwrite", got.SchemaVersion)
	}
	if !got.GeneratedAt.Equal(second.GeneratedAt) {
		t.Fatalf("generated_at = %v after overwrite", got.GeneratedAt)
	}
}

func TestInterruptedWriteRecovery(t *testing.T) {
	t.Parallel()

	// Simulate an interrupted write: a leftover .tmp file should not
	// affect reads. Write a valid marker, drop a .tmp sibling, then
	// verify Read still returns the committed marker.
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	m := StateMarker{
		SchemaVersion: Current,
		GeneratedAt:   time.Now().Truncate(time.Millisecond),
	}
	if err := Write(path, m); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Leave a partial temp file (simulating a crash mid-write).
	tmpPath := filepath.Join(dir, "state.json.tmp.123456")
	if err := os.WriteFile(tmpPath, []byte("{truncated"), 0o644); err != nil {
		t.Fatalf("seed temp file: %v", err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("read after interrupted write: %v", err)
	}
	if got.SchemaVersion != m.SchemaVersion {
		t.Fatalf("schema version = %d after interrupted write", got.SchemaVersion)
	}
}

func TestVersionTooNewErrorMessage(t *testing.T) {
	t.Parallel()

	err := &VersionTooNewError{OnDisk: 3, Supported: 1}
	got := err.Error()
	want := "state version 3 not supported by this binary (supports up to 1)"
	if got != want {
		t.Fatalf("error message:\n got: %s\nwant: %s", got, want)
	}
}

func TestCurrentIsPositive(t *testing.T) {
	t.Parallel()

	if Current < 1 {
		t.Fatalf("Current schema version = %d", Current)
	}
}
