package app_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ssh-keyflow/internal/app"
	sshops "ssh-keyflow/internal/ssh"
	"ssh-keyflow/internal/state"
)

func TestUnregisterRequiresFreshAlternatePasswordAndDurableIntent(testingContext *testing.T) {
	home, external := prepareRegistration(testingContext, true)
	before, err := state.Load(home)
	if err != nil {
		testingContext.Fatal(err)
	}
	removed := false
	external.ssh = func(command app.Command) error {
		arguments := strings.Join(command.Args, "\n")
		for _, required := range []string{"-F\nnone", "ControlMaster=no", "ControlPath=none", "IdentityAgent=none", "CertificateFile=none", "IdentityFile=none", "PubkeyAuthentication=no", "GSSAPIAuthentication=no", "HostbasedAuthentication=no"} {
			if !strings.Contains(arguments, required) {
				testingContext.Fatalf("alternate authentication missing %s: %v", required, command.Args)
			}
		}
		payload := command.Args[len(command.Args)-1]
		switch {
		case strings.Contains(payload, "KEYFLOW_ALTERNATE"):
			fmt.Fprintln(command.Stdout, "KEYFLOW_TARGET:/home/me/.ssh/authorized_keys")
		case strings.Contains(payload, "KEYFLOW_REVOCATION_READ"):
			fmt.Fprintln(command.Stdout, "KEYFLOW_PRESENT")
		case strings.Contains(payload, "KEYFLOW_UNREGISTER"):
			document, loadErr := state.Load(home)
			if loadErr != nil || document.Registrations[0].Operation != "unregister" || document.Registrations[0].Stage != "pending" || document.Registrations[0].Status != "unknown" || len(document.History) < 2 {
				testingContext.Fatalf("missing durable intent: %v %+v", loadErr, document)
			}
			removed = true
			fmt.Fprintln(command.Stdout, "KEYFLOW_REMOVED")
		default:
			testingContext.Fatal("unexpected SSH operation", payload)
		}
		return nil
	}
	output, runErr := runCLI(testingContext, home, "4\n1\np\nyes\nyes\n0\n", external)
	document, loadErr := state.Load(home)
	if runErr != nil || loadErr != nil || !removed || document.Registrations[0].Status != "removed" || document.Registrations[0].ID != before.Registrations[0].ID || document.Keys[0].ID != before.Keys[0].ID {
		testingContext.Fatalf("unregister failed: %v %v %+v %s", runErr, loadErr, document.Registrations, output)
	}
	if !strings.Contains(output, before.Keys[0].Fingerprint) || !strings.Contains(output, "me@server.example:22") || !strings.Contains(output, "fresh alternate") {
		testingContext.Fatal("final destination/fingerprint or alternate-login evidence missing", output)
	}
	for _, path := range []string{document.Keys[0].PublicPath, document.Keys[0].PrivatePath} {
		if _, err := os.Stat(path); err != nil {
			testingContext.Fatal("unregister deleted local key", err)
		}
	}
}

func TestUnregisterUnknownOutcomeReconcilesReadOnlyBeforeAnyRetry(testingContext *testing.T) {
	home, external := prepareRegistration(testingContext, true)
	writes, reads := 0, 0
	external.ssh = func(command app.Command) error {
		payload := command.Args[len(command.Args)-1]
		switch {
		case strings.Contains(payload, "KEYFLOW_ALTERNATE"):
			fmt.Fprintln(command.Stdout, "KEYFLOW_TARGET:/home/me/.ssh/authorized_keys")
		case strings.Contains(payload, "KEYFLOW_REVOCATION_READ"):
			reads++
			if writes == 0 {
				fmt.Fprintln(command.Stdout, "KEYFLOW_PRESENT")
			} else {
				fmt.Fprintln(command.Stdout, "KEYFLOW_ABSENT")
			}
		case strings.Contains(payload, "KEYFLOW_UNREGISTER"):
			writes++
			return fmt.Errorf("connection lost after write")
		default:
			testingContext.Fatal("unexpected operation", payload)
		}
		return nil
	}
	output, err := runCLI(testingContext, home, "4\n1\np\nyes\nyes\n0\n", external)
	unknown, _ := state.Load(home)
	plan, _ := sshops.StandardRevocation("linux", "", []byte(publicKey))
	if err != nil || unknown.Registrations[0].Status != "unknown" || unknown.Registrations[0].Stage != "unknown" || !strings.Contains(output, plan.Script) {
		testingContext.Fatal("uncertain result not saved/manual script differs", err, output)
	}
	output, err = runCLI(testingContext, home, "4\n1\np\nyes\n0\n", external)
	reconciled, loadErr := state.Load(home)
	if err != nil || loadErr != nil || writes != 1 || reads != 2 || reconciled.Registrations[0].Status != "removed" || reconciled.Registrations[0].ID != unknown.Registrations[0].ID || !strings.Contains(output, "Already absent") {
		testingContext.Fatal("retry wrote before reconciliation", err, loadErr, writes, reads, output)
	}
	for _, event := range reconciled.History {
		if event.Operation != "unregister" {
			testingContext.Fatal("operation evidence missing", event)
		}
	}
}

func TestUnregisterPreservesFilesWhenUnreachableOrFinalApprovalDeclined(testingContext *testing.T) {
	for _, scenario := range []string{"unreachable", "final declined", "manual"} {
		testingContext.Run(scenario, func(testingContext *testing.T) {
			home, external := prepareRegistration(testingContext, true)
			calls := 0
			external.ssh = func(command app.Command) error {
				calls++
				payload := command.Args[len(command.Args)-1]
				if scenario == "unreachable" {
					return fmt.Errorf("host unreachable")
				}
				if strings.Contains(payload, "KEYFLOW_ALTERNATE") {
					fmt.Fprintln(command.Stdout, "KEYFLOW_TARGET:/home/me/.ssh/authorized_keys")
				} else if strings.Contains(payload, "KEYFLOW_REVOCATION_READ") {
					fmt.Fprintln(command.Stdout, "KEYFLOW_PRESENT")
				} else {
					testingContext.Fatal("unapproved write")
				}
				return nil
			}
			input := "4\n1\np\n0\n"
			if scenario == "manual" {
				input = "4\n1\n0\n0\n"
			}
			if scenario == "final declined" {
				input = "4\n1\np\nyes\nno\n0\n"
			}
			output, err := runCLI(testingContext, home, input, external)
			document, loadErr := state.Load(home)
			if err != nil || loadErr != nil || document.Registrations[0].Status == "removed" || (scenario == "manual" && calls != 0) {
				testingContext.Fatal(err, loadErr, output)
			}
			for _, path := range []string{document.Keys[0].PublicPath, document.Keys[0].PrivatePath} {
				if _, err := os.Stat(path); err != nil {
					testingContext.Fatal("local key lost", err)
				}
			}
			if scenario != "final declined" && !strings.Contains(output, "Manual revocation") {
				testingContext.Fatal("missing manual fallback", output)
			}
		})
	}
}

func TestAlternatePrivateKeyCannotReuseRevokedFingerprintAtAnotherPath(testingContext *testing.T) {
	home, external := prepareRegistration(testingContext, true)
	duplicate := writeFixture(testingContext, home, "another private path", "same key fixture")
	external.ssh = func(command app.Command) error {
		testingContext.Fatal("target fingerprint reached SSH despite another path")
		return nil
	}
	output, err := runCLI(testingContext, home, "4\n1\nk\n"+duplicate+"\n0\n", external)
	if err != nil || !strings.Contains(output, "exclude target fingerprint") || !strings.Contains(output, "Manual revocation") {
		testingContext.Fatal(err, output)
	}
}

func TestLocalDeletionBlocksUnconfirmedRegistrations(testingContext *testing.T) {
	home, external := prepareRegistration(testingContext, true)
	document, _ := state.Load(home)
	output, err := runCLI(testingContext, home, "6\n1\n0\n", external)
	for _, path := range []string{document.Keys[0].PublicPath, document.Keys[0].PrivatePath} {
		if _, statErr := os.Stat(path); statErr != nil {
			testingContext.Fatal("unconfirmed registration lost local file", statErr)
		}
	}
	if err != nil || !strings.Contains(output, "recorded registration") {
		testingContext.Fatal(err, output)
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh", "ssh-keyflow", "state.lock")); !os.IsNotExist(err) {
		testingContext.Fatal("operation lock leaked", err)
	}
}

func TestLocalDeletionRetainsIdentityAndRegistrationHistory(testingContext *testing.T) {
	home, external := prepareRegistration(testingContext, true)
	if err := state.Update(home, func(document *state.State) error {
		document.Registrations[0].Status, document.Registrations[0].Stage, document.Registrations[0].Operation = "removed", "complete", "unregister"
		return nil
	}); err != nil {
		testingContext.Fatal(err)
	}
	before, _ := state.Load(home)
	unmanaged := writeFixture(testingContext, home, ".ssh/unmanaged.pub", otherPublicKey)
	output, err := runCLI(testingContext, home, "6\n1\nyes\n2\n4\n1\n0\n", external)
	after, loadErr := state.Load(home)
	if err != nil || loadErr != nil || len(after.Keys) != 0 || len(after.Registrations) != 1 || after.Registrations[0].ID != before.Registrations[0].ID || len(after.History) != len(before.History) {
		testingContext.Fatalf("deletion lost stable history: %v %v %+v %s", err, loadErr, after, output)
	}
	for _, path := range []string{before.Keys[0].PublicPath, before.Keys[0].PrivatePath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			testingContext.Fatal("managed local file retained", path, err)
		}
	}
	stored, _ := os.ReadFile(filepath.Join(state.Directory(home), "state.json"))
	if !strings.Contains(string(stored), `"deleted_keys"`) || !strings.Contains(string(stored), before.Keys[0].ID) || !strings.Contains(output, "Only recorded destinations") || !strings.Contains(output, "deleted key") || !strings.Contains(output, "Already removed") {
		testingContext.Fatal("missing tombstone/scope/idempotence", string(stored), output)
	}
	if actual, err := os.ReadFile(unmanaged); err != nil || string(actual) != otherPublicKey {
		testingContext.Fatal("unmanaged file changed", err)
	}
}
