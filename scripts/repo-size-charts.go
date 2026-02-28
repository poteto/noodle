package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type fileEntry struct {
	Path  string
	Lines int64
}

type dirEntry struct {
	Path  string
	Lines int64
}

func main() {
	topDirs := flag.Int("top-dirs", 12, "number of directories to show")
	topFiles := flag.Int("top-files", 15, "number of files to show")
	barWidth := flag.Int("bar-width", 42, "ASCII bar width")
	allFiles := flag.Bool("all-files", false, "include all files on disk (default: git-tracked only)")
	repo := flag.String("repo", "", "repository root (default: auto-detect with git)")
	flag.Parse()

	if *topDirs < 1 || *topFiles < 1 || *barWidth < 1 {
		fail("top-dirs, top-files, and bar-width must be >= 1")
	}

	repoRoot, err := resolveRepoRoot(*repo)
	if err != nil {
		fail(err.Error())
	}

	paths, err := collectPaths(repoRoot, *allFiles)
	if err != nil {
		fail(err.Error())
	}
	if len(paths) == 0 {
		fail("no files found")
	}

	files := make([]fileEntry, 0, len(paths))
	dirs := map[string]int64{}

	for _, path := range paths {
		fullPath := filepath.Join(repoRoot, filepath.FromSlash(path))
		info, err := os.Lstat(fullPath)
		if err != nil {
			fail(fmt.Sprintf("stat failed for %q: %v", path, err))
		}
		if !info.Mode().IsRegular() {
			continue
		}

		lines, err := countLines(fullPath)
		if err != nil {
			fail(fmt.Sprintf("line count failed for %q: %v", path, err))
		}
		files = append(files, fileEntry{Path: path, Lines: lines})

		key := topLevel(path)
		dirs[key] += lines
	}

	if len(files) == 0 {
		fail("no regular files found")
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].Lines == files[j].Lines {
			return files[i].Path < files[j].Path
		}
		return files[i].Lines > files[j].Lines
	})

	dirList := make([]dirEntry, 0, len(dirs))
	for k, v := range dirs {
		dirList = append(dirList, dirEntry{Path: k, Lines: v})
	}
	sort.Slice(dirList, func(i, j int) bool {
		if dirList[i].Lines == dirList[j].Lines {
			return dirList[i].Path < dirList[j].Path
		}
		return dirList[i].Lines > dirList[j].Lines
	})

	scope := "git-tracked files"
	if *allFiles {
		scope = "all on-disk files (excluding .git/.worktrees)"
	}

	fmt.Printf("Repository: %s\n", repoRoot)
	fmt.Printf("Scope: %s\n", scope)
	fmt.Printf("Files counted: %d\n\n", len(files))

	printDirChart("Largest top-level directories by LOC", dirList, *topDirs, *barWidth)
	fmt.Println()
	printFileChart("Largest files by LOC", files, *topFiles, *barWidth)
}

func resolveRepoRoot(input string) (string, error) {
	if input != "" {
		return filepath.Abs(input)
	}

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("could not detect repository root; pass -repo")
	}
	return strings.TrimSpace(string(out)), nil
}

func collectPaths(repoRoot string, allFiles bool) ([]string, error) {
	if !allFiles {
		cmd := exec.Command("git", "-C", repoRoot, "ls-files", "-z")
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("git ls-files failed: %w", err)
		}

		chunks := bytes.Split(out, []byte{0})
		paths := make([]string, 0, len(chunks))
		for _, chunk := range chunks {
			if len(chunk) == 0 {
				continue
			}
			paths = append(paths, string(chunk))
		}
		return paths, nil
	}

	paths := []string{}
	err := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == ".worktrees" {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}

func topLevel(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) <= 1 {
		return "."
	}
	return parts[0]
}

func printDirChart(title string, dirs []dirEntry, limit, width int) {
	fmt.Println(title)
	fmt.Println(strings.Repeat("-", len(title)))
	if len(dirs) == 0 {
		fmt.Println("(none)")
		return
	}

	if limit > len(dirs) {
		limit = len(dirs)
	}

	maxLines := dirs[0].Lines
	nameWidth := 1
	for i := 0; i < limit; i++ {
		if len(dirs[i].Path) > nameWidth {
			nameWidth = len(dirs[i].Path)
		}
	}

	for i := 0; i < limit; i++ {
		e := dirs[i]
		bar := scaledBar(e.Lines, maxLines, width)
		fmt.Printf("%2d. %-*s  %9d LOC  %s\n", i+1, nameWidth, e.Path, e.Lines, bar)
	}
}

func printFileChart(title string, files []fileEntry, limit, width int) {
	fmt.Println(title)
	fmt.Println(strings.Repeat("-", len(title)))
	if len(files) == 0 {
		fmt.Println("(none)")
		return
	}

	if limit > len(files) {
		limit = len(files)
	}

	maxLines := files[0].Lines
	nameWidth := 1
	for i := 0; i < limit; i++ {
		if len(files[i].Path) > nameWidth {
			nameWidth = len(files[i].Path)
		}
	}

	for i := 0; i < limit; i++ {
		e := files[i]
		bar := scaledBar(e.Lines, maxLines, width)
		fmt.Printf("%2d. %-*s  %9d LOC  %s\n", i+1, nameWidth, e.Path, e.Lines, bar)
	}
}

func scaledBar(value, max int64, width int) string {
	if max <= 0 {
		return ""
	}
	length := int((value * int64(width)) / max)
	if length == 0 && value > 0 {
		length = 1
	}
	return strings.Repeat("#", length)
}

func countLines(path string) (int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return int64(bytes.Count(data, []byte{'\n'})), nil
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "error:", msg)
	os.Exit(1)
}
