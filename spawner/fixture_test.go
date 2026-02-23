package spawner

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/poteto/noodle/internal/testutil/fixturedir"
)

type providerCommandFixtureSetup struct {
	Request      SpawnRequest `json:"request"`
	PromptFile   string       `json:"prompt_file"`
	AgentBinary  string       `json:"agent_binary"`
	SystemPrompt string       `json:"system_prompt"`
	StderrFile   string       `json:"stderr_file"`
}

type providerCommandFixtureExpected struct {
	Contains []string `json:"contains"`
	Omits    []string `json:"omits"`
}

func TestBuildProviderCommandDirectoryFixtures(t *testing.T) {
	fixturedir.AssertValidFixtureRoot(t, "testdata")
	inventory := fixturedir.LoadInventory(t, "testdata")
	for _, fixtureCase := range inventory.Cases {
		if !strings.HasPrefix(fixtureCase.Name, "provider-command-") {
			continue
		}
		fixtureCase := fixtureCase
		t.Run(fixtureCase.Name, func(t *testing.T) {
			setup := parseProviderCommandFixtureSetup(t, fixtureCase)
			expected := parseProviderCommandFixtureExpected(t, fixtureCase)

			command := buildProviderCommand(
				setup.Request,
				setup.PromptFile,
				setup.AgentBinary,
				setup.SystemPrompt,
				setup.StderrFile,
			)

			for _, want := range expected.Contains {
				if !strings.Contains(command, want) {
					t.Fatalf("command missing %q:\n%s", want, command)
				}
			}
			for _, omit := range expected.Omits {
				if strings.Contains(command, omit) {
					t.Fatalf("command must not contain %q:\n%s", omit, command)
				}
			}
		})
	}
}

func parseProviderCommandFixtureSetup(t *testing.T, fixtureCase fixturedir.FixtureCase) providerCommandFixtureSetup {
	t.Helper()
	raw := string(fixtureCase.States[0].MustReadFile(t, "setup.json"))
	var setup providerCommandFixtureSetup
	if err := json.Unmarshal([]byte(raw), &setup); err != nil {
		t.Fatalf("parse setup fixture %s: %v", fixtureCase.Name, err)
	}
	return setup
}

func parseProviderCommandFixtureExpected(t *testing.T, fixtureCase fixturedir.FixtureCase) providerCommandFixtureExpected {
	t.Helper()
	raw := fixturedir.MustSection(t, fixtureCase, "Expected")
	var expected providerCommandFixtureExpected
	if err := json.Unmarshal([]byte(raw), &expected); err != nil {
		t.Fatalf("parse expected fixture %s: %v", fixtureCase.Name, err)
	}
	return expected
}
