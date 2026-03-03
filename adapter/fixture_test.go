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
			expectedRaw := ""
			if errorExpectation == nil {
				expectedRaw = fixturedir.MustSection(t, fixtureCase, "Expected")
			}
			expectedWarningsRaw, hasExpectedWarnings := fixtureCase.Section("Expected Warnings")

			actual, warnings, err := ParseBacklogItems(input)
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

			expectedWarnings := []string{}
			if hasExpectedWarnings && strings.TrimSpace(expectedWarningsRaw) != "" {
				if err := json.Unmarshal([]byte(expectedWarningsRaw), &expectedWarnings); err != nil {
					t.Fatalf("parse expected warnings fixture: %v", err)
				}
			}
			if !reflect.DeepEqual(warnings, expectedWarnings) {
				t.Fatalf("warnings mismatch\nactual:   %#v\nexpected: %#v", warnings, expectedWarnings)
			}
		})
	}
}
