package adapter

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/poteto/noodle/internal/testutil/fixturedir"
)

func TestDirectoryFixtures(t *testing.T) {
	fixturedir.AssertValidFixtureRoot(t, "testdata")
	inventory := fixturedir.LoadInventory(t, "testdata")

	for _, fixtureCase := range inventory.Cases {
		fixtureCase := fixtureCase
		t.Run(fixtureCase.Name, func(t *testing.T) {
			state := fixtureCase.States[0]
			input := strings.Join(fixturedir.NonEmptyLines(t, state.MustReadFile(t, "input.ndjson"), "input.ndjson"), "\n")
			errorExpectation := fixtureCase.ExpectedError
			fixtureName := strings.TrimPrefix(strings.ToLower(fixtureCase.Name), "error-")
			expectedRaw := ""
			if errorExpectation == nil {
				expectedRaw = fixturedir.MustSection(t, fixtureCase, "Expected")
			}

			if strings.HasPrefix(fixtureName, "backlog") {
				actual, err := ParseBacklogItems(input)
				fixturedir.AssertError(t, "parse backlog fixture", err, errorExpectation)
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
			fixturedir.AssertError(t, "parse plans fixture", err, errorExpectation)
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
