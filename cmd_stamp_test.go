package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunStampRequiresOutputPath(t *testing.T) {
	err := runStamp(context.Background(), "", "")
	if err == nil {
		t.Fatal("expected missing output error")
	}
	if !strings.Contains(err.Error(), "output path is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunStampDoesNotTruncateOutputWhenEventsOpenFails(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "stamped.ndjson")
	if err := os.WriteFile(outputPath, []byte("keep-me"), 0o644); err != nil {
		t.Fatalf("seed output file: %v", err)
	}

	eventsPath := filepath.Join(tempDir, "events-dir")
	if err := os.MkdirAll(eventsPath, 0o755); err != nil {
		t.Fatalf("create events directory: %v", err)
	}

	err := runStamp(context.Background(), outputPath, eventsPath)
	if err == nil {
		t.Fatal("expected events path open error")
	}

	data, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		t.Fatalf("read output file after failure: %v", readErr)
	}
	if string(data) != "keep-me" {
		t.Fatalf("output file was unexpectedly modified: %q", string(data))
	}
}
