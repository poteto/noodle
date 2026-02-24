package skill

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Frontmatter is the parsed YAML header from a SKILL.md file.
type Frontmatter struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description"`
	Model       string      `yaml:"model,omitempty"`
	Noodle      *NoodleMeta `yaml:"noodle,omitempty"`
}

// IsTaskType returns true if this skill has noodle: frontmatter.
func (f Frontmatter) IsTaskType() bool { return f.Noodle != nil }

// NoodleMeta is the noodle-specific scheduling metadata nested under noodle:.
// Schedule is required. Permissions are optional.
type NoodleMeta struct {
	Permissions Permissions `yaml:"permissions"`
	Schedule    string      `yaml:"schedule"`
}

type Permissions struct {
	Merge *bool `yaml:"merge,omitempty"`
}

func (p Permissions) CanMerge() bool {
	if p.Merge == nil {
		return true
	}
	return *p.Merge
}

var frontmatterSep = []byte("---")

// ParseFrontmatter extracts YAML frontmatter from markdown content.
// Returns parsed metadata, the body (content after closing ---), and any error.
// If no frontmatter is found, returns zero Frontmatter and full content as body.
func ParseFrontmatter(content []byte) (Frontmatter, []byte, error) {
	trimmed := bytes.TrimLeft(content, " \t\r\n")
	if !bytes.HasPrefix(trimmed, frontmatterSep) {
		return Frontmatter{}, content, nil
	}

	// Find closing ---
	afterOpen := trimmed[len(frontmatterSep):]
	// Skip the rest of the opening --- line
	if idx := bytes.IndexByte(afterOpen, '\n'); idx >= 0 {
		afterOpen = afterOpen[idx+1:]
	} else {
		// Only --- with nothing after it
		return Frontmatter{}, content, nil
	}

	closeIdx := bytes.Index(afterOpen, frontmatterSep)
	if closeIdx < 0 {
		return Frontmatter{}, content, nil
	}

	yamlBlock := afterOpen[:closeIdx]
	body := afterOpen[closeIdx+len(frontmatterSep):]
	// Skip the rest of the closing --- line
	if idx := bytes.IndexByte(body, '\n'); idx >= 0 {
		body = body[idx+1:]
	} else {
		body = nil
	}

	var fm Frontmatter
	if err := yaml.Unmarshal(yamlBlock, &fm); err != nil {
		return Frontmatter{}, nil, fmt.Errorf("parse frontmatter: %w", err)
	}

	if fm.Noodle != nil && fm.Noodle.Schedule == "" {
		return Frontmatter{}, nil, fmt.Errorf("parse frontmatter: noodle.schedule is required for task types")
	}

	return fm, body, nil
}

// StripFrontmatter removes the YAML frontmatter block, returning only the body.
func StripFrontmatter(content []byte) []byte {
	_, body, err := ParseFrontmatter(content)
	if err != nil {
		return content
	}
	return body
}
