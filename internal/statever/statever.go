// Package statever provides schema version tracking for .noodle state files.
//
// A state marker file (.noodle/state.json) records the schema version and
// generation timestamp. On startup the binary reads the marker and refuses
// to proceed if the on-disk version is newer than what it supports.
package statever

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/poteto/noodle/internal/filex"
)

// SchemaVersion is an integer that tracks the on-disk state format.
// It starts at 1 and increments with each incompatible state change.
type SchemaVersion int

// Current is the schema version produced by this binary.
const Current SchemaVersion = 1

// StateMarker is persisted to .noodle/state.json. It records the schema
// version and the time the marker was last written.
type StateMarker struct {
	SchemaVersion SchemaVersion `json:"schema_version"`
	GeneratedAt   time.Time     `json:"generated_at"`
}

// Read reads a state marker from path. If the file does not exist, it
// returns a zero-value marker and no error (first run).
func Read(path string) (StateMarker, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return StateMarker{}, nil
		}
		return StateMarker{}, fmt.Errorf("read state marker: %w", err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return StateMarker{}, nil
	}
	var m StateMarker
	if err := json.Unmarshal(data, &m); err != nil {
		return StateMarker{}, fmt.Errorf("corrupted state marker at %s: %w", path, err)
	}
	return m, nil
}

// Write writes a state marker to path using an atomic temp-file-and-rename.
func Write(path string, m StateMarker) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state marker: %w", err)
	}
	if err := filex.WriteFileAtomic(path, append(data, '\n')); err != nil {
		return fmt.Errorf("write state marker: %w", err)
	}
	return nil
}

// CheckCompatibility reads the marker at path and returns an error if the
// on-disk version is newer than what this binary supports. A missing file
// is treated as a fresh state directory (no error).
func CheckCompatibility(path string) error {
	m, err := Read(path)
	if err != nil {
		return err
	}
	// Zero marker means the file was absent or empty — first run.
	if m.SchemaVersion == 0 {
		return nil
	}
	if m.SchemaVersion > Current {
		return &VersionTooNewError{
			OnDisk:    m.SchemaVersion,
			Supported: Current,
		}
	}
	return nil
}

// VersionTooNewError is returned when the on-disk state was written by a
// newer binary.
type VersionTooNewError struct {
	OnDisk    SchemaVersion
	Supported SchemaVersion
}

func (e *VersionTooNewError) Error() string {
	return fmt.Sprintf(
		"state version %d not supported by this binary (supports up to %d)",
		e.OnDisk, e.Supported,
	)
}
