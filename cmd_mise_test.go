package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/poteto/noodle/config"
)

func TestRunMiseOutputsJSON(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, ".noodle"), 0o755); err != nil {
		t.Fatalf("mkdir runtime dir: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir project dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})

	app := &App{Config: config.DefaultConfig(), Validation: config.ValidationResult{}}
	app.Config.Adapters = map[string]config.AdapterConfig{}

	output := captureStdout(t, func() {
		if err := runMise(context.Background(), app); err != nil {
			t.Fatalf("runMise: %v", err)
		}
	})

	var brief map[string]any
	if err := json.Unmarshal([]byte(output), &brief); err != nil {
		t.Fatalf("parse mise output json: %v", err)
	}
	if _, ok := brief["generated_at"]; !ok {
		t.Fatalf("output missing generated_at: %s", output)
	}
}
