package adapter

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestMarkdownFixtures(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("testdata", "*.fixture.md"))
	if err != nil {
		t.Fatalf("glob fixtures: %v", err)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		t.Fatal("no adapter fixtures found")
	}

	for _, fixturePath := range paths {
		fixturePath := fixturePath
		t.Run(filepath.Base(fixturePath), func(t *testing.T) {
			input := strings.Join(readFixtureBlockLines(t, fixturePath, "Input"), "\n")
			expectedRaw := strings.Join(readFixtureBlockLines(t, fixturePath, "Expected"), "\n")

			if strings.HasPrefix(filepath.Base(fixturePath), "backlog") {
				actual, err := ParseBacklogItems(input)
				if err != nil {
					t.Fatalf("parse backlog fixture: %v", err)
				}
				var expected []BacklogItem
				if err := json.Unmarshal([]byte(expectedRaw), &expected); err != nil {
					t.Fatalf("parse expected backlog fixture: %v", err)
				}
				if !reflect.DeepEqual(actual, expected) {
					t.Fatalf("fixture mismatch\nactual:   %#v\nexpected: %#v", actual, expected)
				}
				return
			}

			actual, err := ParsePlanItems(input)
			if err != nil {
				t.Fatalf("parse plans fixture: %v", err)
			}
			var expected []PlanItem
			if err := json.Unmarshal([]byte(expectedRaw), &expected); err != nil {
				t.Fatalf("parse expected plans fixture: %v", err)
			}
			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("fixture mismatch\nactual:   %#v\nexpected: %#v", actual, expected)
			}
		})
	}
}

func readFixtureBlockLines(t *testing.T, fixturePath, section string) []string {
	t.Helper()

	file, err := os.Open(fixturePath)
	if err != nil {
		t.Fatalf("open %s: %v", fixturePath, err)
	}
	defer file.Close()

	heading := "## " + section
	insideSection := false
	insideFence := false
	lines := make([]string, 0)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "## ") {
			insideSection = trimmed == heading
			insideFence = false
			continue
		}
		if !insideSection {
			continue
		}
		if strings.HasPrefix(trimmed, "```") {
			insideFence = !insideFence
			continue
		}
		if !insideFence || trimmed == "" {
			continue
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan fixture file: %v", err)
	}
	if len(lines) == 0 {
		t.Fatalf("no lines found in fixture section %q", section)
	}
	return lines
}
