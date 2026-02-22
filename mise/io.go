package mise

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func writeBriefAtomic(path string, brief Brief) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create mise directory: %w", err)
	}

	data, err := json.Marshal(brief)
	if err != nil {
		return fmt.Errorf("encode mise json: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "mise-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary mise file: %w", err)
	}
	tmpPath := tmp.Name()
	keepTemp := true
	defer func() {
		if keepTemp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary mise file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary mise file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace mise file: %w", err)
	}
	keepTemp = false
	return nil
}
