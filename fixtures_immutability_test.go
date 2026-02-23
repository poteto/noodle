package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestFixtureSuitesDoNotMutateFixtureFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping fixture immutability check in short mode")
	}

	roots := []string{
		"parse/testdata",
		"adapter/testdata",
		"stamp/testdata",
		"event/testdata",
		"monitor/testdata",
		"dispatcher/testdata",
		"loop/testdata",
	}
	before := snapshotFixtureFiles(t, roots)

	cmd := exec.Command(
		"go",
		"test",
		"./parse",
		"./adapter",
		"./stamp",
		"./event",
		"./monitor",
		"./dispatcher",
		"./loop",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run fixture suites: %v\n%s", err, strings.TrimSpace(string(out)))
	}

	after := snapshotFixtureFiles(t, roots)
	if diff := diffSnapshots(before, after); diff != "" {
		t.Fatalf("fixture files changed while running tests:\n%s", diff)
	}
}

func snapshotFixtureFiles(t *testing.T, roots []string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	for _, root := range roots {
		root = filepath.Clean(strings.TrimSpace(root))
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			hash := sha256.Sum256(data)
			snapshot[filepath.ToSlash(path)] = hex.EncodeToString(hash[:])
			return nil
		})
		if err != nil {
			t.Fatalf("snapshot fixtures under %s: %v", root, err)
		}
	}
	return snapshot
}

func diffSnapshots(before, after map[string]string) string {
	paths := map[string]struct{}{}
	for path := range before {
		paths[path] = struct{}{}
	}
	for path := range after {
		paths[path] = struct{}{}
	}
	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}
	sort.Strings(ordered)

	lines := make([]string, 0)
	for _, path := range ordered {
		beforeHash, beforeOK := before[path]
		afterHash, afterOK := after[path]
		switch {
		case beforeOK && !afterOK:
			lines = append(lines, fmt.Sprintf("deleted %s", path))
		case !beforeOK && afterOK:
			lines = append(lines, fmt.Sprintf("added %s", path))
		case beforeOK && afterOK && beforeHash != afterHash:
			lines = append(lines, fmt.Sprintf("modified %s", path))
		}
	}
	return strings.Join(lines, "\n")
}
