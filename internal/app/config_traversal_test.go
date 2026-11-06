package app_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoveryContinuesButOnboardingRefusesUnresolvedConfiguration(testingContext *testing.T) {
	for _, scenario := range []struct {
		name       string
		directive  string
		diagnostic string
	}{
		{"cycle", "Include config", "Include cycle"},
		{"token", "Include %d/keys.conf", "Include unresolved"},
		{"missing", "Include missing.conf", "Include unreadable or unmatched"},
		{"parse", "Include \"unterminated", "configuration parse note"},
	} {
		testingContext.Run(scenario.name, func(testingContext *testing.T) {
			home := fixtureTempDir(testingContext)
			original := scenario.directive + "\nHost after-problem\n"
			config := writeFixture(testingContext, home, ".ssh/config", original)
			output, err := runCLI(testingContext, home, "2\n0\n", runner{})
			if err != nil || !strings.Contains(output, scenario.diagnostic) || !strings.Contains(output, "unmanaged Host after-problem") {
				testingContext.Fatal("discovery did not continue", err, output)
			}
			output, err = runCLI(testingContext, home, "1\nn\nfresh\nexample.test\nme\n\nlinux\n0\n", runner{})
			if err != nil || !strings.Contains(output, "requires review") || strings.Contains(output, "Managed key number") {
				testingContext.Fatal("onboarding ignored incomplete alias discovery", err, output)
			}
			after, readErr := os.ReadFile(config)
			if readErr != nil || string(after) != original {
				testingContext.Fatal("configuration changed", readErr)
			}
			if _, err := os.Stat(filepath.Join(home, ".ssh/ssh-keyflow/state.json")); !os.IsNotExist(err) {
				testingContext.Fatal("refused onboarding persisted state", err)
			}
		})
	}
}

func TestRepeatedQuotedIncludesAreNotCyclesAndStillProtectAliases(testingContext *testing.T) {
	home := fixtureTempDir(testingContext)
	writeFixture(testingContext, home, ".ssh/config", "Include=\"conf with spaces/host.conf\"\nInclude \"conf with spaces/host.conf\"\n")
	writeFixture(testingContext, home, ".ssh/conf with spaces/host.conf", "Include nested.conf\n")
	writeFixture(testingContext, home, ".ssh/nested.conf", "Host reserved\n")
	output, err := runCLI(testingContext, home, "2\n0\n", runner{})
	if err != nil || strings.Contains(output, "cycle") || strings.Count(output, "unmanaged Host reserved") != 2 {
		testingContext.Fatal("repeated include was treated as a cycle or resolved against its parent", err, output)
	}
	output, err = runCLI(testingContext, home, "1\nn\nreserved\nexample.test\nme\n\nlinux\n0\n", runner{})
	if err != nil || !strings.Contains(output, "matches existing Host reserved") {
		testingContext.Fatal("nested alias was not protected", err, output)
	}
}
