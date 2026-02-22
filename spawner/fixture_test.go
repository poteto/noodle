package spawner

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/poteto/noodle/internal/testutil/fixturemd"
)

type providerCommandFixtureSetup struct {
	Request      SpawnRequest `json:"request"`
	PromptFile   string       `json:"prompt_file"`
	AgentBinary  string       `json:"agent_binary"`
	SystemPrompt string       `json:"system_prompt"`
}

type providerCommandFixtureExpected struct {
	Contains []string `json:"contains"`
	Omits    []string `json:"omits"`
}

func TestBuildProviderCommandMarkdownFixtures(t *testing.T) {
	paths := fixturemd.Paths(t, "testdata")
	for _, fixturePath := range paths {
		if !strings.HasPrefix(filepath.Base(fixturePath), "provider-command-") {
			continue
		}
		fixturePath := fixturePath
		t.Run(filepath.Base(fixturePath), func(t *testing.T) {
			setup := parseProviderCommandFixtureSetup(t, fixturePath)
			expected := parseProviderCommandFixtureExpected(t, fixturePath)

			command := buildProviderCommand(
				setup.Request,
				setup.PromptFile,
				setup.AgentBinary,
				setup.SystemPrompt,
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

func parseProviderCommandFixtureSetup(t *testing.T, fixturePath string) providerCommandFixtureSetup {
	t.Helper()
	raw := strings.Join(fixturemd.ReadSectionLines(t, fixturePath, "Setup"), "\n")
	var setup providerCommandFixtureSetup
	if err := json.Unmarshal([]byte(raw), &setup); err != nil {
		t.Fatalf("parse setup fixture %s: %v", fixturePath, err)
	}
	return setup
}

func parseProviderCommandFixtureExpected(t *testing.T, fixturePath string) providerCommandFixtureExpected {
	t.Helper()
	raw := strings.Join(fixturemd.ReadSectionLines(t, fixturePath, "Expected"), "\n")
	var expected providerCommandFixtureExpected
	if err := json.Unmarshal([]byte(raw), &expected); err != nil {
		t.Fatalf("parse expected fixture %s: %v", fixturePath, err)
	}
	return expected
}
