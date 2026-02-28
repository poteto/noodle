package rtcap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLaunchMetadataRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	launched := now.Add(2 * time.Second)
	meta := LaunchMetadata{
		AttemptID:    "attempt-1",
		SessionID:    "sess-abc",
		Runtime:      "process",
		State:        LaunchStateLaunched,
		StartedAt:    now,
		LaunchedAt:   &launched,
		WorktreeName: "order-1-stage-0",
		Token:        "tok-xyz",
	}

	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got LaunchMetadata
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.AttemptID != meta.AttemptID {
		t.Errorf("AttemptID = %q, want %q", got.AttemptID, meta.AttemptID)
	}
	if got.SessionID != meta.SessionID {
		t.Errorf("SessionID = %q, want %q", got.SessionID, meta.SessionID)
	}
	if got.Runtime != meta.Runtime {
		t.Errorf("Runtime = %q, want %q", got.Runtime, meta.Runtime)
	}
	if got.State != meta.State {
		t.Errorf("State = %q, want %q", got.State, meta.State)
	}
	if !got.StartedAt.Equal(meta.StartedAt) {
		t.Errorf("StartedAt = %v, want %v", got.StartedAt, meta.StartedAt)
	}
	if got.LaunchedAt == nil || !got.LaunchedAt.Equal(*meta.LaunchedAt) {
		t.Errorf("LaunchedAt = %v, want %v", got.LaunchedAt, meta.LaunchedAt)
	}
	if got.WorktreeName != meta.WorktreeName {
		t.Errorf("WorktreeName = %q, want %q", got.WorktreeName, meta.WorktreeName)
	}
	if got.Token != meta.Token {
		t.Errorf("Token = %q, want %q", got.Token, meta.Token)
	}
}

func TestLaunchMetadataSnakeCaseJSON(t *testing.T) {
	meta := LaunchMetadata{
		AttemptID:    "a1",
		WorktreeName: "wt",
		Token:        "tok",
	}
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}

	wantKeys := []string{"attempt_id", "session_id", "runtime", "state", "started_at", "launched_at", "worktree_name", "token"}
	for _, key := range wantKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("JSON key %q not found in marshaled output", key)
		}
	}
}

func TestPersistAndReadLaunchRecord(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().Truncate(time.Second)

	meta := LaunchMetadata{
		AttemptID:    "attempt-42",
		Runtime:      "sprites",
		State:        LaunchStateLaunching,
		StartedAt:    now,
		WorktreeName: "order-7-stage-0",
		Token:        "idempotent-token",
	}

	if err := PersistLaunchRecord(dir, meta); err != nil {
		t.Fatalf("PersistLaunchRecord: %v", err)
	}

	got, err := ReadLaunchRecord(dir, "attempt-42")
	if err != nil {
		t.Fatalf("ReadLaunchRecord: %v", err)
	}

	if got.AttemptID != "attempt-42" {
		t.Errorf("AttemptID = %q, want %q", got.AttemptID, "attempt-42")
	}
	if got.State != LaunchStateLaunching {
		t.Errorf("State = %q, want %q", got.State, LaunchStateLaunching)
	}
	if got.Token != "idempotent-token" {
		t.Errorf("Token = %q, want %q", got.Token, "idempotent-token")
	}
}

func TestTwoPhaseLaunchTransition(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().Truncate(time.Second)

	// Phase 1: persist launching.
	meta := LaunchMetadata{
		AttemptID:    "attempt-99",
		Runtime:      "process",
		State:        LaunchStateLaunching,
		StartedAt:    now,
		WorktreeName: "order-3-stage-1",
		Token:        "token-99",
	}
	if err := PersistLaunchRecord(dir, meta); err != nil {
		t.Fatalf("PersistLaunchRecord: %v", err)
	}

	// Verify it reads back as launching.
	got, err := ReadLaunchRecord(dir, "attempt-99")
	if err != nil {
		t.Fatalf("ReadLaunchRecord: %v", err)
	}
	if got.State != LaunchStateLaunching {
		t.Errorf("State = %q, want %q", got.State, LaunchStateLaunching)
	}
	if got.LaunchedAt != nil {
		t.Errorf("LaunchedAt should be nil before launch, got %v", got.LaunchedAt)
	}

	// Phase 2: mark launched.
	launchedAt := now.Add(3 * time.Second)
	if err := MarkLaunched(dir, "attempt-99", "sess-xyz", launchedAt); err != nil {
		t.Fatalf("MarkLaunched: %v", err)
	}

	// Verify state change.
	got, err = ReadLaunchRecord(dir, "attempt-99")
	if err != nil {
		t.Fatalf("ReadLaunchRecord after launch: %v", err)
	}
	if got.State != LaunchStateLaunched {
		t.Errorf("State = %q, want %q", got.State, LaunchStateLaunched)
	}
	if got.SessionID != "sess-xyz" {
		t.Errorf("SessionID = %q, want %q", got.SessionID, "sess-xyz")
	}
	if got.LaunchedAt == nil || !got.LaunchedAt.Equal(launchedAt) {
		t.Errorf("LaunchedAt = %v, want %v", got.LaunchedAt, launchedAt)
	}
}

func TestMarkLaunchedInvalidState(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().Truncate(time.Second)

	// Write a launched record.
	launched := now.Add(time.Second)
	meta := LaunchMetadata{
		AttemptID:  "attempt-bad",
		State:      LaunchStateLaunched,
		StartedAt:  now,
		LaunchedAt: &launched,
	}
	if err := PersistLaunchRecord(dir, meta); err != nil {
		t.Fatalf("PersistLaunchRecord: %v", err)
	}

	// Trying to mark it launched again should fail.
	err := MarkLaunched(dir, "attempt-bad", "sess-2", now)
	if err == nil {
		t.Fatal("MarkLaunched should fail for already-launched record")
	}
}

func TestMarkFailed(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().Truncate(time.Second)

	meta := LaunchMetadata{
		AttemptID: "attempt-fail",
		State:     LaunchStateLaunching,
		StartedAt: now,
	}
	if err := PersistLaunchRecord(dir, meta); err != nil {
		t.Fatalf("PersistLaunchRecord: %v", err)
	}

	if err := MarkFailed(dir, "attempt-fail"); err != nil {
		t.Fatalf("MarkFailed: %v", err)
	}

	got, err := ReadLaunchRecord(dir, "attempt-fail")
	if err != nil {
		t.Fatalf("ReadLaunchRecord: %v", err)
	}
	if got.State != LaunchStateFailed {
		t.Errorf("State = %q, want %q", got.State, LaunchStateFailed)
	}
}

func TestMarkFailedInvalidState(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().Truncate(time.Second)

	meta := LaunchMetadata{
		AttemptID: "attempt-already-failed",
		State:     LaunchStateFailed,
		StartedAt: now,
	}
	if err := PersistLaunchRecord(dir, meta); err != nil {
		t.Fatalf("PersistLaunchRecord: %v", err)
	}

	err := MarkFailed(dir, "attempt-already-failed")
	if err == nil {
		t.Fatal("MarkFailed should fail for already-failed record")
	}
}

func TestReadLaunchRecordNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadLaunchRecord(dir, "nonexistent")
	if err == nil {
		t.Fatal("ReadLaunchRecord should fail for missing record")
	}
}

func TestReadLaunchRecordCorrupted(t *testing.T) {
	dir := t.TempDir()
	launchesDir := filepath.Join(dir, launchDir)
	if err := os.MkdirAll(launchesDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(launchesDir, "corrupted.json")
	if err := os.WriteFile(path, []byte("{bad json!!!"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := ReadLaunchRecord(dir, "corrupted")
	if err == nil {
		t.Fatal("ReadLaunchRecord should fail for corrupted record")
	}
}

func TestReconcileLaunchingFindsOrphans(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().Truncate(time.Second)

	// Write two launching records.
	for _, id := range []string{"orphan-1", "orphan-2"} {
		meta := LaunchMetadata{
			AttemptID: id,
			State:     LaunchStateLaunching,
			StartedAt: now,
			Runtime:   "process",
		}
		if err := PersistLaunchRecord(dir, meta); err != nil {
			t.Fatalf("PersistLaunchRecord(%s): %v", id, err)
		}
	}

	recovered, err := ReconcileLaunching(dir)
	if err != nil {
		t.Fatalf("ReconcileLaunching: %v", err)
	}
	if len(recovered) != 2 {
		t.Fatalf("ReconcileLaunching returned %d, want 2", len(recovered))
	}
	for _, r := range recovered {
		if !r.IsOrphan {
			t.Errorf("recovered attempt %s should be orphan", r.AttemptID)
		}
		if r.State != LaunchStateLaunching {
			t.Errorf("recovered attempt %s state = %q, want %q", r.AttemptID, r.State, LaunchStateLaunching)
		}
	}
}

func TestReconcileLaunchingIgnoresLaunched(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().Truncate(time.Second)
	launched := now.Add(time.Second)

	// One launching, one launched.
	meta1 := LaunchMetadata{
		AttemptID: "still-launching",
		State:     LaunchStateLaunching,
		StartedAt: now,
	}
	meta2 := LaunchMetadata{
		AttemptID:  "already-launched",
		State:      LaunchStateLaunched,
		StartedAt:  now,
		LaunchedAt: &launched,
		SessionID:  "sess-1",
	}
	if err := PersistLaunchRecord(dir, meta1); err != nil {
		t.Fatalf("PersistLaunchRecord: %v", err)
	}
	if err := PersistLaunchRecord(dir, meta2); err != nil {
		t.Fatalf("PersistLaunchRecord: %v", err)
	}

	recovered, err := ReconcileLaunching(dir)
	if err != nil {
		t.Fatalf("ReconcileLaunching: %v", err)
	}
	if len(recovered) != 1 {
		t.Fatalf("ReconcileLaunching returned %d, want 1", len(recovered))
	}
	if recovered[0].AttemptID != "still-launching" {
		t.Errorf("recovered attempt ID = %q, want %q", recovered[0].AttemptID, "still-launching")
	}
}

func TestReconcileLaunchingIgnoresFailedRecords(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().Truncate(time.Second)

	meta := LaunchMetadata{
		AttemptID: "failed-attempt",
		State:     LaunchStateFailed,
		StartedAt: now,
	}
	if err := PersistLaunchRecord(dir, meta); err != nil {
		t.Fatalf("PersistLaunchRecord: %v", err)
	}

	recovered, err := ReconcileLaunching(dir)
	if err != nil {
		t.Fatalf("ReconcileLaunching: %v", err)
	}
	if len(recovered) != 0 {
		t.Fatalf("ReconcileLaunching returned %d, want 0", len(recovered))
	}
}

func TestReconcileLaunchingMissingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")
	recovered, err := ReconcileLaunching(dir)
	if err != nil {
		t.Fatalf("ReconcileLaunching on missing dir should not error: %v", err)
	}
	if len(recovered) != 0 {
		t.Fatalf("ReconcileLaunching returned %d, want 0", len(recovered))
	}
}

func TestReconcileLaunchingSkipsCorrupted(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().Truncate(time.Second)

	// One valid launching record.
	meta := LaunchMetadata{
		AttemptID: "valid-one",
		State:     LaunchStateLaunching,
		StartedAt: now,
	}
	if err := PersistLaunchRecord(dir, meta); err != nil {
		t.Fatalf("PersistLaunchRecord: %v", err)
	}

	// One corrupted record.
	launchesDir := filepath.Join(dir, launchDir)
	path := filepath.Join(launchesDir, "corrupted.json")
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatalf("write corrupted: %v", err)
	}

	recovered, err := ReconcileLaunching(dir)
	if err != nil {
		t.Fatalf("ReconcileLaunching: %v", err)
	}
	if len(recovered) != 1 {
		t.Fatalf("ReconcileLaunching returned %d, want 1", len(recovered))
	}
	if recovered[0].AttemptID != "valid-one" {
		t.Errorf("recovered = %q, want %q", recovered[0].AttemptID, "valid-one")
	}
}

func TestReconcileLaunchingSkipsNonJSON(t *testing.T) {
	dir := t.TempDir()
	launchesDir := filepath.Join(dir, launchDir)
	if err := os.MkdirAll(launchesDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Write a non-JSON file.
	if err := os.WriteFile(filepath.Join(launchesDir, "readme.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	recovered, err := ReconcileLaunching(dir)
	if err != nil {
		t.Fatalf("ReconcileLaunching: %v", err)
	}
	if len(recovered) != 0 {
		t.Fatalf("ReconcileLaunching returned %d, want 0", len(recovered))
	}
}

func TestReconcileLaunchingSkipsDirectories(t *testing.T) {
	dir := t.TempDir()
	launchesDir := filepath.Join(dir, launchDir)
	if err := os.MkdirAll(filepath.Join(launchesDir, "subdir"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	recovered, err := ReconcileLaunching(dir)
	if err != nil {
		t.Fatalf("ReconcileLaunching: %v", err)
	}
	if len(recovered) != 0 {
		t.Fatalf("ReconcileLaunching returned %d, want 0", len(recovered))
	}
}

func TestMarkLaunchedNotFound(t *testing.T) {
	dir := t.TempDir()
	err := MarkLaunched(dir, "ghost", "sess", time.Now())
	if err == nil {
		t.Fatal("MarkLaunched should fail for missing record")
	}
}
