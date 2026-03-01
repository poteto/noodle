// Package jsonx provides generic JSON file read/write helpers.
package jsonx

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/poteto/noodle/internal/filex"
)

// ReadJSON reads a JSON file at path and unmarshals it into T.
// Returns the zero value and nil if the file does not exist.
func ReadJSON[T any](path string) (T, error) {
	var zero T
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return zero, nil
		}
		return zero, fmt.Errorf("read %s: %w", path, err)
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return zero, fmt.Errorf("unmarshal %s: %w", path, err)
	}
	return v, nil
}

// WriteJSON marshals v as compact JSON and writes it atomically to path.
func WriteJSON(path string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	if err := filex.WriteFileAtomic(path, data); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// WriteJSONIndented marshals v as indented JSON and writes it atomically to path.
func WriteJSONIndented(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	if err := filex.WriteFileAtomic(path, data); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
