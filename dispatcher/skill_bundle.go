package dispatcher

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/poteto/noodle/skill"
)

type loadedSkill struct {
	SystemPrompt string
	Warnings     []string
}

func loadSkillBundle(
	resolver skill.Resolver,
	provider string,
	skillName string,
) (loadedSkill, error) {
	skillName = strings.TrimSpace(skillName)
	if skillName == "" {
		return loadedSkill{}, nil
	}

	resolved, err := resolver.Resolve(skillName)
	if err != nil {
		return loadedSkill{}, err
	}

	skillPath := resolved.Path
	skillMarkdown, err := os.ReadFile(filepath.Join(skillPath, "SKILL.md"))
	if err != nil {
		return loadedSkill{}, fmt.Errorf("read SKILL.md: %w", err)
	}

	sections := []string{
		fmt.Sprintf("# Skill: %s", skillName),
		"## SKILL.md",
		string(skill.StripFrontmatter(skillMarkdown)),
	}

	referenceRoot := filepath.Join(skillPath, "references")
	referenceFiles, err := collectReferenceFiles(referenceRoot)
	if err != nil {
		return loadedSkill{}, err
	}

	isCodex := strings.EqualFold(strings.TrimSpace(provider), "codex")
	budgetUsed := 0
	omitted := make([]string, 0)

	for _, referenceFile := range referenceFiles {
		content, err := os.ReadFile(referenceFile.AbsPath)
		if err != nil {
			return loadedSkill{}, fmt.Errorf("read reference file %s: %w", referenceFile.RelativePath, err)
		}
		if isCodex && budgetUsed+len(content) > codexSkillRefsLimitBytes {
			omitted = append(omitted, referenceFile.RelativePath)
			continue
		}
		budgetUsed += len(content)
		sections = append(sections,
			fmt.Sprintf("## references/%s", referenceFile.RelativePath),
			string(content),
		)
	}

	warnings := make([]string, 0)
	if len(omitted) > 0 {
		sort.Strings(omitted)
		warnings = append(warnings,
			fmt.Sprintf("Codex reference truncation: omitted %d files (%s)",
				len(omitted),
				strings.Join(omitted, ", "),
			),
		)
		sections = append(sections,
			fmt.Sprintf(
				"## Reference Truncation\nOmitted files due 50KB Codex reference limit: %s",
				strings.Join(omitted, ", "),
			),
		)
	}

	return loadedSkill{
		SystemPrompt: strings.Join(sections, "\n\n"),
		Warnings:     warnings,
	}, nil
}

// loadExecuteBundle loads execute methodology + adapter-configured domain skill.
func loadExecuteBundle(
	resolver skill.Resolver,
	provider string,
	methodologySkill string,
	domainSkill string,
) (loadedSkill, error) {
	methodology, err := loadSkillBundle(resolver, provider, methodologySkill)
	if err != nil {
		return loadedSkill{}, fmt.Errorf("load methodology skill %s: %w", methodologySkill, err)
	}
	if domainSkill == "" || domainSkill == methodologySkill {
		return methodology, nil
	}
	domain, err := loadSkillBundle(resolver, provider, domainSkill)
	if err != nil {
		methodology.Warnings = append(methodology.Warnings,
			fmt.Sprintf("domain skill %q not found: %v", domainSkill, err))
		return methodology, nil
	}
	return loadedSkill{
		SystemPrompt: methodology.SystemPrompt + "\n\n" + domain.SystemPrompt,
		Warnings:     append(methodology.Warnings, domain.Warnings...),
	}, nil
}

type referenceFile struct {
	RelativePath string
	AbsPath      string
}

func collectReferenceFiles(referenceRoot string) ([]referenceFile, error) {
	info, err := os.Stat(referenceRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat references directory: %w", err)
	}
	if !info.IsDir() {
		return nil, nil
	}

	files := make([]referenceFile, 0)
	err = filepath.WalkDir(referenceRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(referenceRoot, path)
		if err != nil {
			return err
		}
		files = append(files, referenceFile{
			RelativePath: filepath.ToSlash(rel),
			AbsPath:      path,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk references directory: %w", err)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].RelativePath < files[j].RelativePath
	})
	return files, nil
}
