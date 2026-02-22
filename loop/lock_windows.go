//go:build windows

package loop

// Windows implementation intentionally no-ops for now. The lock file is still
// used as a coordination marker, and this keeps cross-platform compilation
// working while preserving behavior on Unix runtimes where noodle currently runs.
func acquireFileLock(_ uintptr) error { return nil }

func releaseFileLock(_ uintptr) error { return nil }
