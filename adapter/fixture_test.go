package adapter

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/poteto/noodle/internal/testutil/fixturemd"
)

func TestMarkdownFixtures(t *testing.T) {
	paths := fixturemd.Paths(t, "testdata")

	for _, fixturePath := range paths {
		fixturePath := fixturePath
		t.Run(filepath.Base(fixturePath), func(t *testing.T) {
			input := strings.Join(fixturemd.ReadSectionLines(t, fixturePath, "Input"), "\n")
			expectError := fixturemd.IsErrorFixture(fixturePath)
			expectedRaw := ""
			if !expectError {
				expectedRaw = strings.Join(fixturemd.ReadSectionLines(t, fixturePath, "Expected"), "\n")
			}

			if strings.HasPrefix(filepath.Base(fixturePath), "backlog") {
				actual, err := ParseBacklogItems(input)
				if expectError {
					if err == nil {
						t.Fatalf("expected backlog parse error for fixture %s", filepath.Base(fixturePath))
					}
					return
				}
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
			if expectError {
				if err == nil {
					t.Fatalf("expected plans parse error for fixture %s", filepath.Base(fixturePath))
				}
				return
			}
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
