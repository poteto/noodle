package rtcap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/poteto/noodle/internal/filex"
)

// LaunchRecordState is the two-phase launch lifecycle state.
type LaunchRecordState string

const (
	LaunchStateLaunching LaunchRecordState = "launching"
	LaunchStateLaunched  LaunchRecordState = "launched"
	LaunchStateFailed    LaunchRecordState = "failed"
)

// LaunchMetadata is the persisted record for a two-phase launch.
type LaunchMetadata struct {
	AttemptID    string            `json:"attempt_id"`
	SessionID    string            `json:"session_id"`
	Runtime      string            `json:"runtime"`
	State        LaunchRecordState `json:"state"`
	StartedAt    time.Time         `json:"started_at"`
	LaunchedAt   *time.Time        `json:"launched_at"`
	WorktreeName string            `json:"worktree_name"`
	Token        string            `json:"token"`
}

// RecoveredAttempt describes a launch record found during startup recovery.
type RecoveredAttempt struct {
	LaunchMetadata
	IsOrphan bool `json:"is_orphan"`
}

const launchDir = "launches"

// launchRecordPath returns the file path for a launch record.
func launchRecordPath(dir, attemptID string) string {
	return filepath.Join(dir, launchDir, attemptID+".json")
}

// PersistLaunchRecord writes a launch record atomically.
func PersistLaunchRecord(dir string, meta LaunchMetadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("encode launch record %s: %w", meta.AttemptID, err)
	}
	path := launchRecordPath(dir, meta.AttemptID)
	if err := filex.WriteFileAtomic(path, append(data, '\n')); err != nil {
		return fmt.Errorf("persist launch record %s: %w", meta.AttemptID, err)
	}
	return nil
}

// ReadLaunchRecord reads a launch record from disk.
func ReadLaunchRecord(dir, attemptID string) (LaunchMetadata, error) {
	path := launchRecordPath(dir, attemptID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return LaunchMetadata{}, fmt.Errorf("launch record %s not found", attemptID)
		}
		return LaunchMetadata{}, fmt.Errorf("read launch record %s: %w", attemptID, err)
	}
	var meta LaunchMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return LaunchMetadata{}, fmt.Errorf("corrupted launch record %s: %w", attemptID, err)
	}
	return meta, nil
}

// MarkLaunched transitions a launch record from launching to launched.
func MarkLaunched(dir, attemptID string, sessionID string, launchedAt time.Time) error {
	meta, err := ReadLaunchRecord(dir, attemptID)
	if err != nil {
		return err
	}
	if meta.State != LaunchStateLaunching {
		return fmt.Errorf("launch record %s in state %q, not launching", attemptID, meta.State)
	}
	meta.State = LaunchStateLaunched
	meta.SessionID = sessionID
	meta.LaunchedAt = &launchedAt
	return PersistLaunchRecord(dir, meta)
}

// MarkFailed transitions a launch record from launching to failed.
func MarkFailed(dir, attemptID string) error {
	meta, err := ReadLaunchRecord(dir, attemptID)
	if err != nil {
		return err
	}
	if meta.State != LaunchStateLaunching {
		return fmt.Errorf("launch record %s in state %q, not launching", attemptID, meta.State)
	}
	meta.State = LaunchStateFailed
	return PersistLaunchRecord(dir, meta)
}

// ReconcileLaunching scans the launch directory for records in "launching"
// state and returns them as RecoveredAttempt entries. All launching records
// found on startup are considered orphans since the process that wrote them
// is no longer running.
func ReconcileLaunching(dir string) ([]RecoveredAttempt, error) {
	launchesDir := filepath.Join(dir, launchDir)
	entries, err := os.ReadDir(launchesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read launch directory: %w", err)
	}

	var recovered []RecoveredAttempt
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".json" {
			continue
		}
		attemptID := name[:len(name)-len(".json")]

		meta, err := ReadLaunchRecord(dir, attemptID)
		if err != nil {
			// Skip corrupted or unreadable records.
			continue
		}
		if meta.State != LaunchStateLaunching {
			continue
		}

		recovered = append(recovered, RecoveredAttempt{
			LaunchMetadata: meta,
			IsOrphan:       true,
		})
	}

	return recovered, nil
}
