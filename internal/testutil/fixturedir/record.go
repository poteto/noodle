package fixturedir

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// WriteSectionToExpected reads expected.md, replaces (or appends) the named
// ## Section with a JSON code block containing data, and preserves frontmatter
// and other sections. This is the shared version of the logic previously
// hand-rolled as writeRuntimeDumpSection in loop/fixture_test.go.
func WriteSectionToExpected(expectedPath string, sectionName string, data any) error {
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", expectedPath, err)
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", sectionName, err)
	}

	updated, err := spliceSection(string(content), sectionName, string(dataJSON))
	if err != nil {
		return err
	}
	return os.WriteFile(expectedPath, []byte(NormalizeFixtureMarkdown(updated)), 0o644)
}

// spliceSection replaces (or appends) a named ## Section in a markdown document.
// The section is written as a JSON code block. Frontmatter and other sections
// are preserved.
func spliceSection(content, sectionName, jsonBody string) (string, error) {
	frontmatter, body, err := splitFrontmatter(content)
	if err != nil {
		return "", err
	}

	// Build the replacement block.
	replacement := fmt.Sprintf("## %s\n\n```json\n%s\n```", sectionName, jsonBody)

	// Try to replace existing section.
	pattern := regexp.MustCompile(
		`(?ms)^##\s+` + regexp.QuoteMeta(sectionName) + `\s*\n.*?(?:\n##\s|\z)`,
	)
	loc := pattern.FindStringIndex(body)
	if loc != nil {
		// Check if the match ends at another section heading.
		matched := body[loc[0]:loc[1]]
		if strings.HasSuffix(matched, "\n##") || strings.HasSuffix(matched, "\n## ") {
			// Preserve the next section heading.
			trimLen := 0
			for i := len(matched) - 1; i >= 0; i-- {
				if matched[i] == '\n' {
					trimLen = len(matched) - i
					break
				}
			}
			loc[1] -= trimLen
		}
		var out strings.Builder
		out.WriteString("---\n")
		out.WriteString(frontmatter)
		out.WriteString("\n---\n")
		out.WriteString(body[:loc[0]])
		out.WriteString(replacement)
		out.WriteString("\n")
		out.WriteString(body[loc[1]:])
		return out.String(), nil
	}

	// Section not found — append.
	var out strings.Builder
	out.WriteString("---\n")
	out.WriteString(frontmatter)
	out.WriteString("\n---\n")
	trimmedBody := strings.TrimRight(body, "\n ")
	if trimmedBody != "" {
		out.WriteString(trimmedBody)
		out.WriteString("\n\n")
	} else {
		out.WriteString("\n")
	}
	out.WriteString(replacement)
	out.WriteString("\n")
	return out.String(), nil
}
