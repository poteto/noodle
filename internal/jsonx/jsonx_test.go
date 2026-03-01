package jsonx_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/poteto/noodle/internal/jsonx"
)

type sample struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestReadJSON_NotFound(t *testing.T) {
	got, err := jsonx.ReadJSON[sample](filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != (sample{}) {
		t.Fatalf("expected zero value, got %+v", got)
	}
}

func TestReadJSON_InvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{not json}"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := jsonx.ReadJSON[sample](path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Fatalf("error should mention unmarshal, got: %v", err)
	}
}

func TestWriteJSON_ReadJSON_Roundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	want := sample{Name: "test", Count: 42}

	if err := jsonx.WriteJSON(path, want); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	got, err := jsonx.ReadJSON[sample](path)
	if err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}
	if got != want {
		t.Fatalf("roundtrip mismatch: got %+v, want %+v", got, want)
	}
}

func TestWriteJSONIndented_ProducesIndentedOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "indented.json")
	if err := jsonx.WriteJSONIndented(path, sample{Name: "a", Count: 1}); err != nil {
		t.Fatalf("WriteJSONIndented: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !strings.Contains(string(data), "\n") {
		t.Fatalf("expected indented output with newlines, got: %s", data)
	}
	if !strings.Contains(string(data), "  ") {
		t.Fatalf("expected two-space indentation, got: %s", data)
	}
}
