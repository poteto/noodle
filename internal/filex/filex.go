package filex

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// ReadFile reads an atomic state file without preventing its replacement.
func ReadFile(path string) ([]byte, error) {
	f, err := openShared(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// ExpandHome expands a leading ~ or ~/ prefix to the user's home directory.
// Returns path unchanged if there is no ~ prefix or if the home directory
// cannot be determined.
func ExpandHome(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || !strings.HasPrefix(path, "~") {
		return path
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return homeDir
	}
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		return filepath.Join(homeDir, strings.TrimPrefix(strings.TrimPrefix(path, "~/"), "~\\"))
	}
	return path
}

// WriteFileAtomic writes data to path via a sibling temp file and rename.
func WriteFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp.*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := renameWithSharingRetry(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

// Windows readers can temporarily omit FILE_SHARE_DELETE. Preserve atomic
// replacement and the old destination; never remove it to force a rename.
func renameWithSharingRetry(source, destination string) error {
	for attempt := 0; ; attempt++ {
		err := renameAtomic(source, destination)
		transient := errors.Is(err, syscall.Errno(5)) || errors.Is(err, syscall.Errno(32)) || errors.Is(err, syscall.Errno(33))
		if err == nil || runtime.GOOS != "windows" || !transient || attempt >= 20 {
			return err
		}
		time.Sleep(25 * time.Millisecond)
	}
}
