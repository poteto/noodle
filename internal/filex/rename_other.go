//go:build !windows

package filex

import "os"

func renameAtomic(source, destination string) error { return os.Rename(source, destination) }

func openShared(path string) (*os.File, error) { return os.Open(path) }
