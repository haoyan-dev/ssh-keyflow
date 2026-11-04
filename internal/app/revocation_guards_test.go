package app_test

import (
	"fmt"
	"os"
	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/state"
	"strings"
	"testing"
)

func TestAlternateIdentityDriftStopsBeforeReadOrMutation(testingContext *testing.T) {
	home, external := prepareRegistration(testingContext, true)
	alternate := writeFixture(testingContext, home, "alternate", "original private fixture")
	calls := 0
	external.ssh = func(command app.Command) error {
		calls++
		if err := os.WriteFile(alternate, []byte("changed private fixture"), 0600); err != nil {
			testingContext.Fatal(err)
		}
		fmt.Fprintln(command.Stdout, "KEYFLOW_TARGET:/home/me/.ssh/authorized_keys")
		return nil
	}
	output, err := runCLI(testingContext, home, "4\n1\nk\n"+alternate+"\nyes\n0\n", alternateKeyRunner{registrationRunner: external, derived: otherPublicKey})
	document, loadErr := state.Load(home)
	if err != nil || loadErr != nil || calls != 1 || document.Registrations[0].Stage != "unknown" || !strings.Contains(output, "no remote mutation attempted") {
		testingContext.Fatal("changed alternate identity reused", err, loadErr, calls, output)
	}
}

func TestSharedAdministratorRevocationRequiresSeparateImpactApproval(testingContext *testing.T) {
	home, external := prepareRegistration(testingContext, true)
	if err := state.Update(home, func(document *state.State) error {
		document.Hosts[0].OS, document.Hosts[0].WindowsAccount = "windows", "administrator"
		return nil
	}); err != nil {
		testingContext.Fatal(err)
	}
	calls := 0
	external.ssh = func(command app.Command) error {
		calls++
		encoded := strings.Fields(command.Args[len(command.Args)-1])
		script := decodePowerShell(testingContext, encoded[len(encoded)-1])
		if strings.Contains(script, "KEYFLOW_ALTERNATE") {
			fmt.Fprintln(command.Stdout, `KEYFLOW_TARGET:C:\fixture\administrators_authorized_keys`)
		} else if strings.Contains(script, "KEYFLOW_REVOCATION_READ") {
			fmt.Fprintln(command.Stdout, "KEYFLOW_PRESENT")
		} else {
			testingContext.Fatal("shared target mutated without impact approval")
		}
		return nil
	}
	output, err := runCLI(testingContext, home, "4\n1\np\nyes\nno\n0\n", external)
	document, loadErr := state.Load(home)
	if err != nil || loadErr != nil || calls != 2 || document.Registrations[0].Stage != "planned" || !strings.Contains(output, "affects all administrators") {
		testingContext.Fatal("separate impact approval missing", err, loadErr, calls, output)
	}
}
