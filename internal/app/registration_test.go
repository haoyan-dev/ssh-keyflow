package app_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/state"
)

func TestNewDestinationContinuesDirectlyThroughRegistrationAndVerification(testingContext *testing.T) {
	home := testingContext.TempDir()
	public := writeFixture(testingContext, home, "key.pub", publicKey)
	private := writeFixture(testingContext, home, "key", "private fixture")
	external := registrationRunner{testingContext: testingContext, home: home}
	input := "5\ni\n" + public + "\n" + private + "\nfixture\nyes\n1\nn\ndirect\nserver.example\nme\n\nlinux\n1\nyes\np\nyes\nno\n0\n"
	output, err := runCLI(testingContext, home, input, external)
	document, loadErr := state.Load(home)
	if err != nil || loadErr != nil || len(document.Registrations) != 1 || document.Registrations[0].Status != "registered" || document.Registrations[0].ConnectionCheck != "verified" || !strings.Contains(output, "Verified linux login me") || !strings.Contains(output, "Generate SSH alias configuration?") {
		testingContext.Fatal("new destination did not continue into registration", err, loadErr, document.Registrations, output)
	}
}

func TestBootstrapKeyUsesOriginalInputAndVerificationFailureRetainsRegistration(testingContext *testing.T) {
	home, external := prepareRegistration(testingContext, true)
	bootstrap := writeFixture(testingContext, home, "bootstrap key", "existing private fixture")
	input := strings.NewReader("1\na\n1\nyes\nk\n" + bootstrap + "\nyes\nno\n0\n")
	var output bytes.Buffer
	external.ssh = func(command app.Command) error {
		if command.Stdin != input {
			testingContext.Fatal("credential input replaced")
		}
		arguments := strings.Join(command.Args, "\n")
		if strings.Contains(command.Args[len(command.Args)-1], "KEYFLOW_VERIFIED") {
			if !strings.Contains(arguments, "-i\n"+filepath.Join(home, "key with spaces")) {
				testingContext.Fatal("wrong verification key")
			}
			return fmt.Errorf("specified key denied")
		}
		if !strings.Contains(arguments, "-i\n"+bootstrap) || !strings.Contains(arguments, "PasswordAuthentication=no") {
			testingContext.Fatal("bootstrap authentication leaked to other methods")
		}
		if strings.Contains(command.Args[len(command.Args)-1], "KEYFLOW_PROBE") {
			fmt.Fprintln(command.Stdout, "KEYFLOW_TARGET:/home/me/.ssh/authorized_keys")
		} else {
			fmt.Fprintln(command.Stdout, "KEYFLOW_REGISTERED")
		}
		return nil
	}
	err := app.Run(context.Background(), app.Options{Home: home, Input: input, Output: &output, Runner: external})
	document, loadErr := state.Load(home)
	if err != nil || loadErr != nil || document.Registrations[0].Status != "registered" || document.Registrations[0].ConnectionCheck != "failed" || !strings.Contains(output.String(), "connection=failed") {
		testingContext.Fatal(err, loadErr, document.Registrations, output.String())
	}
}

func TestSharedAdministratorImpactMustBeApprovedBeforeRemoteWrite(testingContext *testing.T) {
	home, external := prepareRegistration(testingContext, false)
	if err := state.Update(home, func(document *state.State) error {
		document.Hosts[0].OS = "windows"
		document.Hosts[0].WindowsAccount = "administrator"
		return nil
	}); err != nil {
		testingContext.Fatal(err)
	}
	external.ssh = func(command app.Command) error {
		fmt.Fprintln(command.Stdout, `KEYFLOW_TARGET:C:\ProgramData\ssh\administrators_authorized_keys`)
		return nil
	}
	output, err := runCLI(testingContext, home, "1\na\n1\nyes\np\nyes\nno\n0\n", external)
	document, _ := state.Load(home)
	if err != nil || len(document.History) != 0 || !strings.Contains(output, "shared administrator target not approved") {
		testingContext.Fatal(err, output)
	}
}

type registrationRunner struct {
	testingContext *testing.T
	home           string
	ssh            func(app.Command) error
}

func TestRegistrationPreservesBothKnownHostsFilesInSpacedHome(testingContext *testing.T) {
	home, external := prepareRegistration(testingContext, false)
	spaced := filepath.Join(home, "home with spaces")
	if err := os.Rename(filepath.Join(home, ".ssh"), filepath.Join(home, "moved-ssh")); err != nil {
		testingContext.Fatal(err)
	}
	if err := os.MkdirAll(spaced, 0700); err != nil {
		testingContext.Fatal(err)
	}
	if err := os.Rename(filepath.Join(home, "moved-ssh"), filepath.Join(spaced, ".ssh")); err != nil {
		testingContext.Fatal(err)
	}
	primary := writeFixture(testingContext, spaced, ".ssh/known_hosts", "server.example original-host-key\n")
	secondary := writeFixture(testingContext, spaced, ".ssh/known_hosts2", "server.example second-host-key\n")
	external.home = spaced
	external.ssh = func(command app.Command) error {
		expected := "UserKnownHostsFile=\"" + filepath.ToSlash(primary) + "\" \"" + filepath.ToSlash(secondary) + "\""
		if !strings.Contains(strings.Join(command.Args, "\n"), expected) {
			testingContext.Fatalf("known hosts not isolated/preserved: %v", command.Args)
		}
		return fmt.Errorf("REMOTE HOST IDENTIFICATION HAS CHANGED")
	}
	output, err := runCLI(testingContext, spaced, "1\na\n1\nyes\np\n0\n", external)
	if err != nil || !strings.Contains(output, "HOST IDENTIFICATION HAS CHANGED") {
		testingContext.Fatal(err, output)
	}
	for _, path := range []string{primary, secondary} {
		content, readErr := os.ReadFile(path)
		if readErr != nil || !strings.Contains(string(content), "host-key\n") {
			testingContext.Fatal("existing host record changed", readErr, string(content))
		}
	}
	document, _ := state.Load(spaced)
	if len(document.History) != 0 {
		testingContext.Fatal("host-key failure wrote registration intent")
	}
}

func TestSpecifiedKeyVerificationDisablesOtherAuthenticationAndMasters(testingContext *testing.T) {
	home, external := prepareRegistration(testingContext, true)
	verified := false
	external.ssh = func(command app.Command) error {
		arguments := strings.Join(command.Args, "\n")
		for _, required := range []string{"-F\nnone", "ControlMaster=no", "ControlPath=none", "IdentityAgent=none", "CertificateFile=none", "IdentitiesOnly=yes", "IdentityFile=none", "UserKnownHostsFile=\""} {
			if !strings.Contains(arguments, required) {
				testingContext.Fatalf("missing isolation %q: %v", required, command.Args)
			}
		}
		payload := command.Args[len(command.Args)-1]
		switch {
		case strings.Contains(payload, "KEYFLOW_PROBE"):
			fmt.Fprintln(command.Stdout, "KEYFLOW_TARGET:/home/me/.ssh/authorized_keys")
		case strings.Contains(payload, "KEYFLOW_REGISTER"):
			document, err := state.Load(home)
			if err != nil || document.Registrations[0].Stage != "pending" || len(document.History) != 1 {
				testingContext.Fatalf("write without durable intent: %v %+v", err, document)
			}
			fmt.Fprintln(command.Stdout, "KEYFLOW_REGISTERED")
		case strings.Contains(payload, "KEYFLOW_VERIFIED"):
			for _, required := range []string{"-i\n" + filepath.Join(home, "key with spaces"), "PreferredAuthentications=publickey", "PasswordAuthentication=no", "KbdInteractiveAuthentication=no", "GSSAPIAuthentication=no", "HostbasedAuthentication=no", "StrictHostKeyChecking=yes"} {
				if !strings.Contains(arguments, required) {
					testingContext.Fatalf("verification fallback %q: %v", required, command.Args)
				}
			}
			verified = true
			fmt.Fprintln(command.Stdout, "KEYFLOW_VERIFIED")
		default:
			testingContext.Fatalf("unexpected payload %s", payload)
		}
		return nil
	}
	output, err := runCLI(testingContext, home, "1\na\n1\nyes\np\nyes\nno\n0\n", external)
	document, loadErr := state.Load(home)
	if err != nil || loadErr != nil || !verified || document.Registrations[0].ConnectionCheck != "verified" || len(document.History) != 3 {
		testingContext.Fatalf("verification: %v %v %+v %s", err, loadErr, document, output)
	}
}

func TestUnknownRegistrationReconcilesBeforeRetryAndPreservesEvidence(testingContext *testing.T) {
	for _, result := range []string{"present", "absent", "unreadable"} {
		testingContext.Run(result, func(testingContext *testing.T) {
			home, external := prepareRegistration(testingContext, false)
			writes := 0
			reconciled := false
			external.ssh = func(command app.Command) error {
				payload := command.Args[len(command.Args)-1]
				switch {
				case strings.Contains(payload, "KEYFLOW_PROBE"):
					fmt.Fprintln(command.Stdout, "KEYFLOW_TARGET:/home/me/.ssh/authorized_keys")
				case strings.Contains(payload, "KEYFLOW_RECONCILE"):
					reconciled = true
					if result == "unreadable" {
						return fmt.Errorf("permission denied")
					}
					if result == "present" {
						fmt.Fprintln(command.Stdout, "KEYFLOW_PRESENT")
					} else {
						fmt.Fprintln(command.Stdout, "KEYFLOW_ABSENT")
					}
				case strings.Contains(payload, "KEYFLOW_REGISTER"):
					writes++
					if writes == 1 {
						return fmt.Errorf("connection lost after remote write")
					}
					if !reconciled {
						testingContext.Fatal("retry without reconciliation")
					}
					fmt.Fprintln(command.Stdout, "KEYFLOW_REGISTERED")
				default:
					testingContext.Fatal("unexpected remote operation")
				}
				return nil
			}
			runCLI(testingContext, home, "1\na\n1\nyes\np\nyes\n0\n", external)
			before, err := state.Load(home)
			if err != nil || before.Registrations[0].Status != "unknown" || before.Registrations[0].Stage != "unknown" {
				testingContext.Fatalf("uncertain result lost: %v %+v", err, before)
			}
			runCLI(testingContext, home, "1\na\n1\nyes\np\nyes\nyes\nno\n0\n", external)
			after, err := state.Load(home)
			if err != nil || !reconciled || len(after.History) < len(before.History) {
				testingContext.Fatalf("missing reconciliation %v %+v", err, after)
			}
			for index, event := range before.History {
				if after.History[index] != event {
					testingContext.Fatal("history was rewritten")
				}
			}
			expectedWrites := 1
			if result == "absent" {
				expectedWrites = 2
			}
			if writes != expectedWrites {
				testingContext.Fatalf("writes=%d expected=%d", writes, expectedWrites)
			}
			if result != "unreadable" && after.Registrations[0].Status != "registered" {
				testingContext.Fatal("reconciled result not recorded")
			}
		})
	}
}

func TestUnsafeRemoteProbeNeverWrites(testingContext *testing.T) {
	for _, evidence := range []string{"changed-host-key", "OS mismatch", "Nonstandard AuthorizedKeysFile", "KEYFLOW_TARGET:\nforged"} {
		testingContext.Run(evidence, func(testingContext *testing.T) {
			home, external := prepareRegistration(testingContext, false)
			external.ssh = func(command app.Command) error {
				if !strings.Contains(command.Args[len(command.Args)-1], "KEYFLOW_PROBE") {
					testingContext.Fatal("remote write after unsafe probe")
				}
				if evidence == "KEYFLOW_TARGET:\nforged" {
					fmt.Fprintln(command.Stdout, evidence)
					return nil
				}
				return fmt.Errorf("%s", evidence)
			}
			output, err := runCLI(testingContext, home, "1\na\n1\nyes\np\nyes\n0\n", external)
			document, _ := state.Load(home)
			if err != nil || document.Registrations[0].Status != "unknown" || len(document.History) != 0 || !strings.Contains(output, "Manual registration") {
				testingContext.Fatalf("unsafe probe: %v %s", err, output)
			}
		})
	}
}

func (external registrationRunner) LookPath(name string) (string, error) { return name, nil }
func (external registrationRunner) Run(ctx context.Context, command app.Command) error {
	if command.Name == "ssh-keygen" {
		fmt.Fprintln(command.Stdout, publicKey)
		return nil
	}
	if command.Name != "ssh" {
		external.testingContext.Fatalf("unexpected command %s", command.Name)
	}
	if !command.Interactive || command.Stdin == nil {
		external.testingContext.Fatal("SSH must own original terminal input")
	}
	if _, original := command.Stdin.(*strings.Reader); !original {
		external.testingContext.Fatal("fixture terminal input replaced by script")
	}
	if _, err := os.Stat(filepath.Join(external.home, ".ssh/ssh-keyflow/state.lock")); err != nil {
		external.testingContext.Fatal("operation lock not held", err)
	}
	if external.ssh != nil {
		return external.ssh(command)
	}
	payload := command.Args[len(command.Args)-1]
	if strings.Contains(payload, "KEYFLOW_RECONCILE") {
		fmt.Fprintln(command.Stdout, "KEYFLOW_PRESENT")
		return nil
	}
	if strings.Contains(payload, "KEYFLOW_VERIFIED") {
		fmt.Fprintln(command.Stdout, "KEYFLOW_VERIFIED")
		return nil
	}
	if strings.Contains(payload, "KEYFLOW_PROBE") {
		fmt.Fprintln(command.Stdout, "KEYFLOW_TARGET:/home/me/.ssh/authorized_keys")
		return nil
	}
	if strings.Contains(payload, "KEYFLOW_REGISTER") {
		fmt.Fprintln(command.Stdout, "KEYFLOW_REGISTERED")
		return nil
	}
	return fmt.Errorf("unexpected SSH payload")
}

func prepareRegistration(testingContext *testing.T, private bool) (string, registrationRunner) {
	testingContext.Helper()
	home := testingContext.TempDir()
	publicPath := writeFixture(testingContext, home, "key with spaces.pub", publicKey)
	companion := ""
	if private {
		companion = writeFixture(testingContext, home, "key with spaces", "private fixture")
	}
	external := registrationRunner{testingContext: testingContext, home: home}
	input := "5\ni\n" + publicPath + "\n" + companion + "\nfixture\nyes\n1\nn\nserver\nserver.example\nme\n\nlinux\n1\nyes\n0\n"
	output, err := runCLI(testingContext, home, input, external)
	if err != nil || !strings.Contains(output, "Manual registration") {
		testingContext.Fatalf("setup: %v %s", err, output)
	}
	return home, external
}

func TestAliasConfigRequiresExplicitIncludeConsentAndPreservesBackup(testingContext *testing.T) {
	for _, consent := range []string{"yes", "no"} {
		testingContext.Run(consent, func(testingContext *testing.T) {
			home, external := prepareRegistration(testingContext, true)
			external.ssh = func(command app.Command) error {
				payload := command.Args[len(command.Args)-1]
				if strings.Contains(payload, "KEYFLOW_PROBE") {
					fmt.Fprintln(command.Stdout, "KEYFLOW_TARGET:/home/me/.ssh/authorized_keys")
				} else if strings.Contains(payload, "KEYFLOW_REGISTER") {
					fmt.Fprintln(command.Stdout, "KEYFLOW_REGISTERED")
				} else {
					fmt.Fprintln(command.Stdout, "KEYFLOW_VERIFIED")
				}
				return nil
			}
			original := "Host *\r\n  ServerAliveInterval 30\r\nMatch exec should-never-run\r\n  Compression yes\r\n"
			config := writeFixture(testingContext, home, ".ssh/config", original)
			output, err := runCLI(testingContext, home, "1\na\n1\nyes\np\nyes\nyes\n"+consent+"\n0\n", external)
			managed, readErr := os.ReadFile(filepath.Join(home, ".ssh/ssh-keyflow/config"))
			if err != nil || readErr != nil || !strings.Contains(string(managed), "Host server\n") || !strings.Contains(string(managed), "IdentityFile \"") || !strings.HasSuffix(string(managed), "Host *\n") {
				testingContext.Fatalf("generated config: %v %v %s %s", err, readErr, managed, output)
			}
			current, _ := os.ReadFile(config)
			backups, _ := filepath.Glob(config + ".ssh-keyflow-backup-*")
			if consent == "yes" {
				if string(current) != "Include \"ssh-keyflow/config\"\r\n"+original || len(backups) != 1 {
					testingContext.Fatalf("Include/backup unsafe: %s %v %s", current, backups, output)
				}
				backup, _ := os.ReadFile(backups[0])
				if string(backup) != original {
					testingContext.Fatal("backup does not preserve original bytes")
				}
			} else if string(current) != original || len(backups) != 0 || !strings.Contains(output, "Manually add") {
				testingContext.Fatal("declined Include changed unmanaged config", output)
			}
			document, loadErr := state.Load(home)
			if loadErr != nil || len(document.Configs) != 1 || document.Configs[0].SHA256 == "" || document.Configs[0].PendingSHA256 != "" {
				testingContext.Fatalf("missing config ownership: %v %+v", loadErr, document.Configs)
			}
		})
	}
}

func TestUnownedDedicatedConfigIsNeverOverwritten(testingContext *testing.T) {
	home, external := prepareRegistration(testingContext, false)
	path := writeFixture(testingContext, home, ".ssh/ssh-keyflow/config", "Host mine\n HostName keep.example\n")
	output, err := runCLI(testingContext, home, "1\na\n1\nyes\np\nyes\nyes\nyes\n0\n", external)
	current, _ := os.ReadFile(path)
	if err != nil || string(current) != "Host mine\n HostName keep.example\n" || !strings.Contains(output, "unowned") {
		testingContext.Fatalf("unowned config protection: %v %s %s", err, current, output)
	}
}

func TestChangedOwnedConfigStopsRegenerationWithoutChangingInclude(testingContext *testing.T) {
	home, external := prepareRegistration(testingContext, true)
	external.ssh = func(command app.Command) error {
		payload := command.Args[len(command.Args)-1]
		switch {
		case strings.Contains(payload, "KEYFLOW_PROBE"):
			fmt.Fprintln(command.Stdout, "KEYFLOW_TARGET:/home/me/.ssh/authorized_keys")
		case strings.Contains(payload, "KEYFLOW_RECONCILE"):
			fmt.Fprintln(command.Stdout, "KEYFLOW_PRESENT")
		case strings.Contains(payload, "KEYFLOW_REGISTER"):
			fmt.Fprintln(command.Stdout, "KEYFLOW_REGISTERED")
		default:
			fmt.Fprintln(command.Stdout, "KEYFLOW_VERIFIED")
		}
		return nil
	}
	output, err := runCLI(testingContext, home, "1\na\n1\nyes\np\nyes\nyes\nyes\n0\n", external)
	if err != nil {
		testingContext.Fatal(err, output)
	}
	managed := writeFixture(testingContext, home, ".ssh/ssh-keyflow/config", "external edit\n")
	before, _ := os.ReadFile(filepath.Join(home, ".ssh/config"))
	external.ssh = func(command app.Command) error {
		testingContext.Fatal("SSH started despite managed configuration drift")
		return nil
	}
	output, err = runCLI(testingContext, home, "1\na\n1\nyes\np\nyes\nyes\n0\n", external)
	after, _ := os.ReadFile(filepath.Join(home, ".ssh/config"))
	retained, _ := os.ReadFile(managed)
	if err != nil || string(retained) != "external edit\n" || string(before) != string(after) || !strings.Contains(output, "digest mismatch") {
		testingContext.Fatalf("drift protection: %v %s", err, output)
	}
}

func TestExistingScopedIncludeIsNotDuplicated(testingContext *testing.T) {
	for _, layout := range []string{"direct", "nested"} {
		testingContext.Run(layout, func(testingContext *testing.T) {
			home, external := prepareRegistration(testingContext, true)
			output, err := runCLI(testingContext, home, "1\na\n1\nyes\np\nyes\nyes\nno\n0\n", external)
			if err != nil {
				testingContext.Fatal(err, output)
			}
			original := "Host other\n Include ssh-keyflow/config\n"
			if layout == "nested" {
				original = "Host other\n Include nested.conf\n"
				writeFixture(testingContext, home, ".ssh/nested.conf", "Include ssh-keyflow/config\n")
			}
			path := writeFixture(testingContext, home, ".ssh/config", original)
			output, err = runCLI(testingContext, home, "1\na\n1\nyes\np\nyes\nyes\nyes\n0\n", external)
			current, _ := os.ReadFile(path)
			backups, _ := filepath.Glob(path + ".ssh-keyflow-backup-*")
			if err != nil || string(current) != original || len(backups) != 0 || !strings.Contains(output, "no duplicate") {
				testingContext.Fatalf("scoped Include duplicated: %v %s %s", err, current, output)
			}
		})
	}
}

func TestInterruptedIncludeRecoversWithoutReplacingItsBackup(testingContext *testing.T) {
	for _, interruption := range []string{"after-backup", "after-config"} {
		testingContext.Run(interruption, func(testingContext *testing.T) {
			home, external := prepareRegistration(testingContext, true)
			original := "Host old\n HostName old.example\n"
			path := writeFixture(testingContext, home, ".ssh/config", original)
			output, err := runCLI(testingContext, home, "1\na\n1\nyes\np\nyes\nyes\nyes\n0\n", external)
			if err != nil {
				testingContext.Fatal(err, output)
			}
			if interruption == "after-backup" {
				if err := os.WriteFile(path, []byte(original), 0600); err != nil {
					testingContext.Fatal(err)
				}
			}
			if err := state.Update(home, func(document *state.State) error { document.ConfigEdits[0].Complete = false; return nil }); err != nil {
				testingContext.Fatal(err)
			}
			output, err = runCLI(testingContext, home, "1\na\n1\nyes\np\nyes\nyes\nyes\n0\n", external)
			document, _ := state.Load(home)
			backups, _ := filepath.Glob(path + ".ssh-keyflow-backup-*")
			current, _ := os.ReadFile(path)
			if err != nil || len(document.ConfigEdits) != 1 || !document.ConfigEdits[0].Complete || len(backups) != 1 || string(current) != "Include \"ssh-keyflow/config\"\n"+original {
				testingContext.Fatalf("Include recovery: %v %+v %v %s", err, document.ConfigEdits, backups, output)
			}
		})
	}
}

func TestInvalidRegistrationEvidenceStopsStartup(testingContext *testing.T) {
	home, _ := prepareRegistration(testingContext, false)
	filename := filepath.Join(home, ".ssh/ssh-keyflow/state.json")
	original, err := os.ReadFile(filename)
	if err != nil {
		testingContext.Fatal(err)
	}
	for _, extra := range []string{
		`"history":[{"registration_id":"missing","at":"2026-11-04T00:00:00Z","status":"registered","connection_check":"verified","stage":"complete"}],`,
		`"configs":[{"path":"relative","sha256":"not-a-digest"}],`,
	} {
		corrupt := strings.Replace(string(original), "{", "{"+extra, 1)
		if err := os.WriteFile(filename, []byte(corrupt), 0600); err != nil {
			testingContext.Fatal(err)
		}
		output, runErr := runCLI(testingContext, home, "0\n", runner{})
		if runErr == nil {
			testingContext.Fatalf("invalid evidence accepted: %s %s", extra, output)
		}
	}
}

func TestAutomaticRegistrationKeepsPublicOnlyUnverified(testingContext *testing.T) {
	home, external := prepareRegistration(testingContext, false)
	output, err := runCLI(testingContext, home, "1\na\n1\nyes\np\nyes\nno\n0\n", external)
	document, loadErr := state.Load(home)
	if err != nil || loadErr != nil || len(document.Registrations) != 1 || document.Registrations[0].Status != "registered" || document.Registrations[0].ConnectionCheck != "unverified" || !strings.Contains(output, "registered") {
		testingContext.Fatalf("registration: %v %v %+v\n%s", err, loadErr, document, output)
	}
}
