package main

import (
	"os"
	"strings"
	"testing"
)

func TestVersionFlagPrintsCanonicalVersionString(t *testing.T) {
	wantBytes, err := os.ReadFile("VERSION")
	if err != nil {
		t.Fatalf("read VERSION: %v", err)
	}
	want := strings.TrimSpace(string(wantBytes))
	if want == "" {
		t.Fatal("VERSION file is empty")
	}

	root := NewRootCmd()
	root.SetArgs([]string{"--version"})

	out := captureStdout(t, func() {
		if err := root.Execute(); err != nil {
			t.Fatalf("--version: %v", err)
		}
	})
	got := strings.TrimSpace(out)
	if got == "" {
		t.Fatal("expected non-empty version output")
	}
	if got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
}
