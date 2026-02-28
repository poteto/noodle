package dispatcher

import (
	"fmt"
	"os"
	"strings"

	"github.com/poteto/noodle/skill"
)

// resolveSkillBundle resolves the skill bundle for a dispatch request: uses
// SystemPrompt verbatim if set, loads a domain+methodology bundle if
// DomainSkill is set, or falls back to loading the named skill bundle.
func resolveSkillBundle(resolver skill.Resolver, req DispatchRequest) (loadedSkill, error) {
	if sp := strings.TrimSpace(req.SystemPrompt); sp != "" {
		return loadedSkill{SystemPrompt: sp}, nil
	}
	if req.DomainSkill != "" {
		return loadExecuteBundle(resolver, req.Provider, req.Skill, req.DomainSkill)
	}
	return loadSkillBundle(resolver, req.Provider, req.Skill)
}

// writePromptFiles writes prompt.txt (the user-facing prompt) and, when the
// composed prompt differs, input.txt (the full prompt sent to the agent).
// It returns the path to the file containing the composed prompt.
func writePromptFiles(sessionDir, promptPath, prompt, composedPrompt string) (inputFile string, err error) {
	if err := os.WriteFile(promptPath, []byte(prompt), 0o644); err != nil {
		return "", fmt.Errorf("write prompt file: %w", err)
	}
	inputFile = promptPath
	if composedPrompt != prompt {
		inputFile = inputPath(sessionDir)
		if err := os.WriteFile(inputFile, []byte(composedPrompt), 0o644); err != nil {
			return "", fmt.Errorf("write input file: %w", err)
		}
	}
	return inputFile, nil
}
