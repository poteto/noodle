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

	roots := discoverFixtureRoots(t)
	before := snapshotFixtureFiles(t, roots)

	goArgs := append([]string{"test"}, fixturePackageTargets(roots)...)
	cmd := exec.Command("go", goArgs...)
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

func discoverFixtureRoots(t *testing.T) []string {
	t.Helper()

	matches, err := filepath.Glob("*" + string(filepath.Separator) + "testdata")
	if err != nil {
		t.Fatalf("discover fixture roots: %v", err)
	}
	roots := make([]string, 0, len(matches))
	for _, match := range matches {
		info, statErr := os.Stat(match)
		if statErr != nil {
			t.Fatalf("stat fixture root %s: %v", match, statErr)
		}
		if !info.IsDir() {
			continue
		}
		roots = append(roots, filepath.ToSlash(filepath.Clean(match)))
	}
	sort.Strings(roots)
	if len(roots) == 0 {
		t.Fatal("no fixture roots found")
	}
	return roots
}

func fixturePackageTargets(roots []string) []string {
	targetsSet := make(map[string]struct{}, len(roots))
	for _, root := range roots {
		pkg := filepath.ToSlash(filepath.Dir(filepath.Clean(root)))
		if strings.TrimSpace(pkg) == "" || pkg == "." {
			continue
		}
		targetsSet["./"+pkg] = struct{}{}
	}
	targets := make([]string, 0, len(targetsSet))
	for target := range targetsSet {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	return targets
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
