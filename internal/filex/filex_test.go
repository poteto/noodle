package filex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomicCreatesParentAndFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "a", "b", "payload.json")
	if err := WriteFileAtomic(path, []byte(`{"ok":true}`)); err != nil {
		t.Fatalf("write atomic file: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(got) != `{"ok":true}` {
		t.Fatalf("file contents mismatch: got %q", string(got))
	}
}

func TestWriteFileAtomicReplacesExistingContents(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "orders.json")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	if err := WriteFileAtomic(path, []byte("new")); err != nil {
		t.Fatalf("rewrite file: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read rewritten file: %v", err)
	}
	if string(got) != "new" {
		t.Fatalf("file contents mismatch: got %q", string(got))
	}
}
