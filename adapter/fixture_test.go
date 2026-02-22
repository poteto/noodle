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
			errorExpectation := fixturemd.ExpectedError(t, fixturePath)
			fixtureName := strings.TrimPrefix(strings.ToLower(filepath.Base(fixturePath)), "error-")
			expectedRaw := ""
			if errorExpectation == nil {
				expectedRaw = strings.Join(fixturemd.ReadSectionLines(t, fixturePath, "Expected"), "\n")
			}

			if strings.HasPrefix(fixtureName, "backlog") {
				actual, err := ParseBacklogItems(input)
				fixturemd.AssertError(t, "parse backlog fixture", err, errorExpectation)
				if errorExpectation != nil {
					return
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
			fixturemd.AssertError(t, "parse plans fixture", err, errorExpectation)
			if errorExpectation != nil {
				return
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
