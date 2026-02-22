package skill

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Resolver resolves skills from ordered search paths.
type Resolver struct {
	SearchPaths []string
}

// SkillPath is the resolved path for a skill and where it was found.
type SkillPath struct {
	Path       string
	SourcePath string
}

// SkillInfo describes a resolved skill for listing.
type SkillInfo struct {
	Name       string
	Path       string
	SourcePath string
	HasSkillMD bool
}

// Resolve resolves a skill by name using first-match-wins precedence.
func (r Resolver) Resolve(name string) (SkillPath, error) {
	for _, raw := range r.SearchPaths {
		sourcePath, ok := resolveSearchPath(raw)
		if !ok {
			continue
		}

		candidate := filepath.Join(sourcePath, name)
		info, err := os.Stat(candidate)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return SkillPath{}, fmt.Errorf("stat %s: %w", candidate, err)
		}
		if !info.IsDir() {
			continue
		}
		if !hasSkillFile(candidate) {
			continue
		}

		absCandidate, err := filepath.Abs(candidate)
		if err != nil {
			return SkillPath{}, fmt.Errorf("resolve absolute path for %s: %w", candidate, err)
		}
		return SkillPath{
			Path:       absCandidate,
			SourcePath: sourcePath,
		}, nil
	}

	return SkillPath{}, fmt.Errorf("skill %q not found", name)
}

// List returns resolved skills from configured paths, with first match winning.
func (r Resolver) List() ([]SkillInfo, error) {
	skills := make([]SkillInfo, 0)
	seen := make(map[string]struct{})

	for _, raw := range r.SearchPaths {
		sourcePath, ok := resolveSearchPath(raw)
		if !ok {
			continue
		}

		entries, err := os.ReadDir(sourcePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", sourcePath, err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if _, exists := seen[name]; exists {
				continue
			}

			dirPath := filepath.Join(sourcePath, name)
			if !hasSkillFile(dirPath) {
				continue
			}
			absDirPath, err := filepath.Abs(dirPath)
			if err != nil {
				return nil, fmt.Errorf("resolve absolute path for %s: %w", dirPath, err)
			}

			skills = append(skills, SkillInfo{
				Name:       name,
				Path:       absDirPath,
				SourcePath: sourcePath,
				HasSkillMD: true,
			})
			seen[name] = struct{}{}
		}
	}

	return skills, nil
}

func resolveSearchPath(raw string) (string, bool) {
	path := strings.TrimSpace(raw)
	if path == "" {
		return "", false
	}
	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", false
		}
		if path == "~" {
			path = homeDir
		} else if strings.HasPrefix(path, "~/") {
			path = filepath.Join(homeDir, strings.TrimPrefix(path, "~/"))
		}
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	return absPath, true
}

func hasSkillFile(path string) bool {
	info, err := os.Stat(filepath.Join(path, "SKILL.md"))
	if err != nil {
		return false
	}
	return !info.IsDir()
}
