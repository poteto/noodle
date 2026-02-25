package skill

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrNotFound is returned when a skill cannot be found in any search path.
var ErrNotFound = errors.New("skill not found")

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

	return SkillPath{}, fmt.Errorf("skill %q: %w", name, ErrNotFound)
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

// SkillMeta is the full resolved metadata for a skill, including parsed frontmatter.
type SkillMeta struct {
	Name        string
	Path        string
	SourcePath  string
	Frontmatter Frontmatter
}

// ResolveWithMeta resolves a skill by name and parses its frontmatter.
func (r Resolver) ResolveWithMeta(name string) (SkillMeta, error) {
	sp, err := r.Resolve(name)
	if err != nil {
		return SkillMeta{}, err
	}
	fm, err := readSkillFrontmatter(sp.Path)
	if err != nil {
		return SkillMeta{}, fmt.Errorf("skill %q: %w", name, err)
	}
	return SkillMeta{
		Name:        name,
		Path:        sp.Path,
		SourcePath:  sp.SourcePath,
		Frontmatter: fm,
	}, nil
}

// ListWithMeta returns all skills with parsed frontmatter.
func (r Resolver) ListWithMeta() ([]SkillMeta, error) {
	infos, err := r.List()
	if err != nil {
		return nil, err
	}
	metas := make([]SkillMeta, 0, len(infos))
	for _, info := range infos {
		fm, err := readSkillFrontmatter(info.Path)
		if err != nil {
			return nil, fmt.Errorf("skill %q: %w", info.Name, err)
		}
		metas = append(metas, SkillMeta{
			Name:        info.Name,
			Path:        info.Path,
			SourcePath:  info.SourcePath,
			Frontmatter: fm,
		})
	}
	return metas, nil
}

// DiscoverTaskTypes returns only skills with noodle: frontmatter.
func (r Resolver) DiscoverTaskTypes() ([]SkillMeta, error) {
	all, err := r.ListWithMeta()
	if err != nil {
		return nil, err
	}
	types := make([]SkillMeta, 0)
	for _, m := range all {
		if m.Frontmatter.IsTaskType() {
			types = append(types, m)
		}
	}
	return types, nil
}

func readSkillFrontmatter(skillPath string) (Frontmatter, error) {
	data, err := os.ReadFile(filepath.Join(skillPath, "SKILL.md"))
	if err != nil {
		return Frontmatter{}, fmt.Errorf("read SKILL.md: %w", err)
	}
	fm, _, err := ParseFrontmatter(data)
	if err != nil {
		return Frontmatter{}, err
	}
	return fm, nil
}

func hasSkillFile(path string) bool {
	info, err := os.Stat(filepath.Join(path, "SKILL.md"))
	if err != nil {
		return false
	}
	return !info.IsDir()
}
