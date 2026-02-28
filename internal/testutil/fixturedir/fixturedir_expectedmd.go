package fixturedir

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func parseExpectedDocument(data []byte) (FixtureMetadata, map[string]string, map[string]bool, error) {
	frontmatter, body, err := splitFrontmatter(string(data))
	if err != nil {
		return FixtureMetadata{}, nil, nil, err
	}
	metadata, err := parseMetadata(frontmatter)
	if err != nil {
		return FixtureMetadata{}, nil, nil, err
	}
	sections, sectionPresence := parseSections(body)
	return metadata, sections, sectionPresence, nil
}

func splitFrontmatter(content string) (string, string, error) {
	lines := strings.Split(content, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return "", "", fmt.Errorf("expected.md must start with frontmatter delimited by ---")
	}

	closingIndex := -1
	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == "---" {
			closingIndex = index
			break
		}
	}
	if closingIndex < 0 {
		return "", "", fmt.Errorf("expected.md frontmatter is missing closing --- delimiter")
	}
	frontmatter := strings.Join(lines[1:closingIndex], "\n")
	body := strings.Join(lines[closingIndex+1:], "\n")
	return frontmatter, body, nil
}

func parseMetadata(frontmatter string) (FixtureMetadata, error) {
	metadata := FixtureMetadata{}
	seen := map[string]bool{}
	allowed := map[string]struct{}{
		"schema_version":   {},
		"expected_failure": {},
		"bug":              {},
		"source_hash":      {},
	}

	lines := strings.Split(frontmatter, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			return FixtureMetadata{}, fmt.Errorf("invalid frontmatter line %q", trimmed)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if _, ok := allowed[key]; !ok {
			return FixtureMetadata{}, fmt.Errorf("unsupported frontmatter key %q", key)
		}
		if seen[key] {
			return FixtureMetadata{}, fmt.Errorf("duplicate frontmatter key %q", key)
		}
		seen[key] = true

		switch key {
		case "schema_version":
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return FixtureMetadata{}, fmt.Errorf("schema_version must be an integer")
			}
			metadata.SchemaVersion = parsed
		case "expected_failure":
			parsed, err := strconv.ParseBool(strings.ToLower(value))
			if err != nil {
				return FixtureMetadata{}, fmt.Errorf("expected_failure must be true or false")
			}
			metadata.ExpectedFailure = parsed
		case "bug":
			parsed, err := strconv.ParseBool(strings.ToLower(value))
			if err != nil {
				return FixtureMetadata{}, fmt.Errorf("bug must be true or false")
			}
			metadata.Bug = parsed
		case "source_hash":
			metadata.SourceHash = trimQuotes(value)
		}
	}

	for _, key := range []string{"schema_version", "expected_failure", "bug"} {
		if !seen[key] {
			return FixtureMetadata{}, fmt.Errorf("missing required frontmatter key %q", key)
		}
	}
	if !seen["source_hash"] {
		return FixtureMetadata{}, fmt.Errorf("missing required frontmatter key %q", "source_hash")
	}
	if strings.TrimSpace(metadata.SourceHash) == "" {
		return FixtureMetadata{}, fmt.Errorf("source_hash must be non-empty")
	}
	if metadata.Bug && !metadata.ExpectedFailure {
		return FixtureMetadata{}, fmt.Errorf("bug=true requires expected_failure=true")
	}
	if metadata.SchemaVersion != FixtureSchemaVersion {
		return FixtureMetadata{}, fmt.Errorf(
			"unsupported schema_version %d (expected %d)",
			metadata.SchemaVersion,
			FixtureSchemaVersion,
		)
	}
	return metadata, nil
}

func parseSections(body string) (map[string]string, map[string]bool) {
	sections := map[string]string{}
	presence := map[string]bool{}
	currentHeading := ""
	insideFence := false
	seenFence := false
	lines := make([]string, 0)

	flush := func() {
		if currentHeading == "" || !seenFence {
			return
		}
		sections[currentHeading] = strings.TrimSpace(strings.Join(lines, "\n"))
		presence[currentHeading] = true
	}

	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			flush()
			currentHeading = strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
			insideFence = false
			seenFence = false
			lines = lines[:0]
			continue
		}
		if currentHeading == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "```") {
			insideFence = !insideFence
			seenFence = true
			continue
		}
		if insideFence {
			lines = append(lines, line)
		}
	}
	flush()
	return sections, presence
}

func parseExpectedError(raw string, present bool) (*ErrorExpectation, error) {
	if !present {
		return nil, nil
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := decodeStrictJSON[ErrorExpectation](raw)
	if err != nil {
		return nil, err
	}
	if !parsed.Absent && !parsed.Any &&
		strings.TrimSpace(parsed.Contains) == "" &&
		strings.TrimSpace(parsed.Equals) == "" {
		parsed.Any = true
	}
	return &parsed, nil
}

func expectationDemandsFailure(expect *ErrorExpectation) bool {
	if expect == nil {
		return false
	}
	if expect.Absent {
		return false
	}
	return true
}

func decodeStrictJSON[T any](raw string) (T, error) {
	var out T
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&out); err != nil {
		return out, err
	}
	if decoder.More() {
		return out, fmt.Errorf("unexpected trailing JSON tokens")
	}
	return out, nil
}

func trimQuotes(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
			(strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`)) {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func formatIssues(issues []FixtureValidationIssue) string {
	if len(issues) == 0 {
		return ""
	}
	parts := make([]string, 0, len(issues))
	for _, issue := range issues {
		parts = append(parts, fmt.Sprintf("%s (%s)", issue.Message, issue.Path))
	}
	return strings.Join(parts, "; ")
}

func assertExpectedMarkdownSynced(fixturePath, expectedPath string) error {
	fixturePath = strings.TrimSpace(fixturePath)
	expectedPath = strings.TrimSpace(expectedPath)
	if fixturePath == "" || expectedPath == "" {
		return fmt.Errorf("expected fixture paths are missing")
	}
	expectedData, err := os.ReadFile(expectedPath)
	if err != nil {
		return fmt.Errorf("read expected markdown: %w", err)
	}
	metadata, _, _, err := parseExpectedDocument(expectedData)
	if err != nil {
		return fmt.Errorf("parse expected markdown: %w", err)
	}
	sourceHash, err := computeFixtureInputHash(fixturePath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(metadata.SourceHash), strings.TrimSpace(sourceHash)) {
		return fmt.Errorf("expected.md is out of date; run `noodle fixtures sync`")
	}
	return nil
}
