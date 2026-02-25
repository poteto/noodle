package statusfile

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/poteto/noodle/internal/filex"
)

// Status holds runtime state written by the loop.
type Status struct {
	Active    []string `json:"active,omitempty"`
	LoopState string   `json:"loop_state,omitempty"`
	Autonomy  string   `json:"autonomy,omitempty"`
	MaxCooks  int      `json:"max_cooks,omitempty"`
}

// Read parses status.json at path. Returns zero-value Status if the file
// does not exist.
func Read(path string) (Status, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Status{}, nil
		}
		return Status{}, fmt.Errorf("read status: %w", err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return Status{}, nil
	}
	var status Status
	if err := json.Unmarshal(data, &status); err != nil {
		return Status{}, fmt.Errorf("parse status: %w", err)
	}
	return status, nil
}

// WriteAtomic writes status to path via atomic rename.
func WriteAtomic(path string, status Status) error {
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("encode status: %w", err)
	}
	if err := filex.WriteFileAtomic(path, append(data, '\n')); err != nil {
		return fmt.Errorf("write status file: %w", err)
	}
	return nil
}
