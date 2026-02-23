package mise

import (
	"encoding/json"
	"fmt"

	"github.com/poteto/noodle/internal/filex"
)

func writeBriefAtomic(path string, brief Brief) error {
	data, err := json.Marshal(brief)
	if err != nil {
		return fmt.Errorf("encode mise json: %w", err)
	}
	if err := filex.WriteFileAtomic(path, data); err != nil {
		return fmt.Errorf("replace mise file: %w", err)
	}
	return nil
}
