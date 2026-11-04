package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ssh-keyflow/internal/app"
)

type runner struct{}

func (runner) LookPath(name string) (string, error) { return name, nil }
func (runner) Run(ctx context.Context, command app.Command) error {
	if command.Name != "ssh-keygen" {
		panic("unexpected external command: " + command.Name)
	}
	return nil
}

func TestStartupShowsMenu(testingContext *testing.T) {
	var output bytes.Buffer
	err := app.Run(context.Background(), app.Options{Home: testingContext.TempDir(), Input: strings.NewReader("0\n"), Output: &output, Runner: runner{}})
	if err != nil || !strings.Contains(output.String(), "SSH KeyFlow") || !strings.Contains(output.String(), "2: List") {
		testingContext.Fatalf("startup: %v\n%s", err, output.String())
	}
}

func writeFixture(testingContext *testing.T, home, relative, content string) string {
	testingContext.Helper()
	path := filepath.Join(home, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		testingContext.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		testingContext.Fatal(err)
	}
	return path
}

func TestInvalidStateStopsStartupWithoutOverwriting(testingContext *testing.T) {
	for _, document := range []string{`{`, `null`, `{"version":99,"keys":[],"hosts":[],"registrations":[]}`, `{"version":1,"keys":[],"hosts":[],"registrations":[],"unknown":true}`, `{"version":1,"keys":[],"hosts":[],"registrations":[]} {}`} {
		testingContext.Run(document, func(testingContext *testing.T) {
			home := testingContext.TempDir()
			path := writeFixture(testingContext, home, ".ssh/ssh-keyflow/state.json", document)
			var output bytes.Buffer
			err := app.Run(context.Background(), app.Options{Home: home, Input: strings.NewReader("0\n"), Output: &output, Runner: runner{}})
			after, readErr := os.ReadFile(path)
			if err == nil || readErr != nil || string(after) != document {
				testingContext.Fatalf("invalid state accepted/overwritten: %v %v %s", err, readErr, after)
			}
		})
	}
}

func TestStateRelationshipsAndDuplicateFieldsAreRejected(testingContext *testing.T) {
	for _, document := range []string{
		`{"version":1,"version":1,"keys":[],"hosts":[],"registrations":[]}`,
		`{"version":1,"Version":1,"keys":[],"hosts":[],"registrations":[]}`,
		`{"version":1,"keys":[{}],"hosts":[],"registrations":[]}`,
		`{"version":1,"keys":[],"hosts":[{"id":"host","alias":"test","hostname":"example","user":"me","port":0,"os":"linux"}],"registrations":[]}`,
		`{"version":1,"keys":[],"hosts":[],"registrations":[{"id":"reg","key_id":"missing","host_id":"missing","status":"registered","connection_check":"unverified","operation":"register","stage":"complete"}]}`,
	} {
		home := testingContext.TempDir()
		writeFixture(testingContext, home, ".ssh/ssh-keyflow/state.json", document)
		var output bytes.Buffer
		if err := app.Run(context.Background(), app.Options{Home: home, Input: strings.NewReader("0\n"), Output: &output, Runner: runner{}}); err == nil {
			testingContext.Fatalf("accepted invalid relationships: %s", document)
		}
	}
}

func TestListDiscoversRecursiveIncludesWithoutExecutingOrChangingThem(testingContext *testing.T) {
	home := testingContext.TempDir()
	config := "Include conf/*.conf missing.conf\nHost original\n IdentityFile ~/.ssh/custom\nMatch exec touch-danger\n Include nested.conf\n"
	path := writeFixture(testingContext, home, ".ssh/config", config)
	writeFixture(testingContext, home, ".ssh/conf/first.conf", "Host nested\nInclude config\n")
	writeFixture(testingContext, home, ".ssh/nested.conf", "Host conditional\n")
	writeFixture(testingContext, home, ".ssh/id_existing", "-----BEGIN OPENSSH PRIVATE KEY-----\nfixture")
	writeFixture(testingContext, home, ".ssh/ssh-keyflow/keys/another.pub", "ssh-ed25519 fixture")
	var output bytes.Buffer
	err := app.Run(context.Background(), app.Options{Home: home, Input: strings.NewReader("2\n0\n"), Output: &output, Runner: runner{}})
	for _, expected := range []string{"unmanaged", "original", "nested", "conditional", "id_existing", "another.pub", "cycle", "missing.conf", "Match exec"} {
		if err != nil || !strings.Contains(output.String(), expected) {
			testingContext.Fatalf("missing %q: %v\n%s", expected, err, output.String())
		}
	}
	after, _ := os.ReadFile(path)
	if string(after) != config {
		testingContext.Fatal("discovery changed unmanaged config")
	}
}

const publicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB test@example"

const otherPublicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIC another@example"

func runCLI(testingContext *testing.T, home, input string, external app.Runner) (string, error) {
	testingContext.Helper()
	var output bytes.Buffer
	err := app.Run(context.Background(), app.Options{Home: home, Input: strings.NewReader(input), Output: &output, Runner: external})
	return output.String(), err
}

func importInput(path, name string) string { return "5\ni\n" + path + "\n\n" + name + "\nyes\n0\n" }

func TestSuccessfulUpdateKeepsPreviousValidStateBackup(testingContext *testing.T) {
	home := testingContext.TempDir()
	first := writeFixture(testingContext, home, ".ssh/first.pub", publicKey)
	second := writeFixture(testingContext, home, ".ssh/second.pub", otherPublicKey)
	if output, err := runCLI(testingContext, home, importInput(first, "first"), runner{}); err != nil || !strings.Contains(output, "Imported") {
		testingContext.Fatalf("first import: %v %s", err, output)
	}
	before, err := os.ReadFile(filepath.Join(home, ".ssh/ssh-keyflow/state.json"))
	if err != nil {
		testingContext.Fatal(err)
	}
	if output, err := runCLI(testingContext, home, importInput(second, "second"), runner{}); err != nil || !strings.Contains(output, "Imported") {
		testingContext.Fatalf("second import: %v %s", err, output)
	}
	backup, err := os.ReadFile(filepath.Join(home, ".ssh/ssh-keyflow/state.json.bak"))
	if err != nil || !bytes.Equal(before, backup) {
		testingContext.Fatalf("previous valid backup missing: %v", err)
	}
}

func TestLockedOrUnwritableUpdateNeverReportsSuccess(testingContext *testing.T) {
	for _, obstruction := range []string{"lock", "backup"} {
		testingContext.Run(obstruction, func(testingContext *testing.T) {
			home := testingContext.TempDir()
			first := writeFixture(testingContext, home, ".ssh/first.pub", publicKey)
			second := writeFixture(testingContext, home, ".ssh/second.pub", otherPublicKey)
			runCLI(testingContext, home, importInput(first, "first"), runner{})
			path := filepath.Join(home, ".ssh/ssh-keyflow/state.json")
			before, _ := os.ReadFile(path)
			if obstruction == "lock" {
				writeFixture(testingContext, home, ".ssh/ssh-keyflow/state.lock", "pid=unknown\n")
			} else if err := os.Mkdir(path+".bak", 0700); err != nil {
				testingContext.Fatal(err)
			}
			output, err := runCLI(testingContext, home, importInput(second, "second"), runner{})
			after, _ := os.ReadFile(path)
			if err != nil || !strings.Contains(output, "Import stopped") || strings.Contains(output, "Imported") || !bytes.Equal(before, after) {
				testingContext.Fatalf("failed update misreported: %v\n%s", err, output)
			}
		})
	}
}

type pairRunner struct{ derived string }

func (pairRunner) LookPath(name string) (string, error) { return name, nil }
func (fake pairRunner) Run(ctx context.Context, command app.Command) error {
	if command.Name != "ssh-keygen" {
		return fmt.Errorf("unexpected external operation")
	}
	if len(command.Args) > 0 && command.Args[0] == "-y" {
		_, err := io.WriteString(command.Stdout, fake.derived)
		return err
	}
	return nil
}

func TestImportValidatesPairForEitherSelectedHalf(testingContext *testing.T) {
	for _, selectedHalf := range []string{"public", "private"} {
		testingContext.Run(selectedHalf, func(testingContext *testing.T) {
			home := testingContext.TempDir()
			public := writeFixture(testingContext, home, ".ssh/key.pub", publicKey)
			private := writeFixture(testingContext, home, ".ssh/key", "-----BEGIN OPENSSH PRIVATE KEY-----\nfixture\n")
			selected, companion := public, private
			if selectedHalf == "private" {
				selected, companion = private, ""
			}
			output, err := runCLI(testingContext, home, "5\ni\n"+selected+"\n"+companion+"\nPaired\nyes\n2\n0\n", pairRunner{derived: publicKey})
			if err != nil || !strings.Contains(output, "Imported Paired") || !strings.Contains(output, "connection=unverified") {
				testingContext.Fatalf("pair: %v\n%s", err, output)
			}
		})
	}
}

func TestManagedMismatchIsVisibleAndBlocksFurtherImports(testingContext *testing.T) {
	home := testingContext.TempDir()
	first := writeFixture(testingContext, home, ".ssh/first.pub", publicKey)
	second := writeFixture(testingContext, home, ".ssh/second.pub", otherPublicKey)
	if output, err := runCLI(testingContext, home, importInput(first, "first"), runner{}); err != nil || !strings.Contains(output, "Imported") {
		testingContext.Fatalf("setup: %v %s", err, output)
	}
	writeFixture(testingContext, home, ".ssh/first.pub", otherPublicKey)
	output, err := runCLI(testingContext, home, "2\n"+importInput(second, "second"), runner{})
	if err != nil || !strings.Contains(output, "requires review") || strings.Contains(output, "Imported second") {
		testingContext.Fatalf("managed mismatch: %v\n%s", err, output)
	}
}

type replacingRunner struct{ path string }

func (replacingRunner) LookPath(name string) (string, error) { return name, nil }
func (fake replacingRunner) Run(ctx context.Context, command app.Command) error {
	return os.WriteFile(fake.path, []byte(otherPublicKey), 0600)
}

func TestImportRechecksSourceAfterConfirmation(testingContext *testing.T) {
	home := testingContext.TempDir()
	path := writeFixture(testingContext, home, ".ssh/source.pub", publicKey)
	output, err := runCLI(testingContext, home, importInput(path, "changed"), replacingRunner{path: path})
	if err != nil || !strings.Contains(output, "requires review") || strings.Contains(output, "Imported") {
		testingContext.Fatalf("source replacement accepted: %v\n%s", err, output)
	}
}

type replacingPrivateRunner struct{ path string }

func (replacingPrivateRunner) LookPath(name string) (string, error) { return name, nil }
func (fake replacingPrivateRunner) Run(ctx context.Context, command app.Command) error {
	if command.Args[0] == "-y" {
		if err := os.WriteFile(fake.path, []byte("replacement private key"), 0600); err != nil {
			return err
		}
		_, err := io.WriteString(command.Stdout, publicKey)
		return err
	}
	return nil
}

func TestPrivateReplacementDuringValidationIsRejected(testingContext *testing.T) {
	home := testingContext.TempDir()
	public := writeFixture(testingContext, home, ".ssh/key.pub", publicKey)
	private := writeFixture(testingContext, home, ".ssh/key", "-----BEGIN OPENSSH PRIVATE KEY-----\nfixture\n")
	output, err := runCLI(testingContext, home, "5\ni\n"+public+"\n"+private+"\nChanged\nyes\n0\n", replacingPrivateRunner{path: private})
	if err != nil || !strings.Contains(output, "requires review") || strings.Contains(output, "Imported") {
		testingContext.Fatalf("private replacement accepted: %v\n%s", err, output)
	}
}

func TestImportedKeyHasValidatedProvenance(testingContext *testing.T) {
	home := testingContext.TempDir()
	path := writeFixture(testingContext, home, ".ssh/source.pub", publicKey)
	output, err := runCLI(testingContext, home, importInput(path, "recorded"), runner{})
	if err != nil || !strings.Contains(output, "Imported") {
		testingContext.Fatalf("setup: %v %s", err, output)
	}
	statePath := filepath.Join(home, ".ssh/ssh-keyflow/state.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		testingContext.Fatal(err)
	}
	var document struct {
		Keys []struct {
			Origin     string `json:"origin"`
			ImportedAt string `json:"imported_at"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		testingContext.Fatal(err)
	}
	if len(document.Keys) != 1 || document.Keys[0].Origin != "imported" {
		testingContext.Fatalf("missing import provenance: %s", data)
	}
	if _, err := time.Parse(time.RFC3339Nano, document.Keys[0].ImportedAt); err != nil {
		testingContext.Fatal("missing valid import time:", err)
	}
	for _, invalid := range []string{"invalid-origin", "invalid-time"} {
		var fields map[string]any
		if err := json.Unmarshal(data, &fields); err != nil {
			testingContext.Fatal(err)
		}
		key := fields["keys"].([]any)[0].(map[string]any)
		if invalid == "invalid-origin" {
			key["origin"] = "discovered"
		} else {
			key["imported_at"] = "yesterday"
		}
		changed, _ := json.Marshal(fields)
		writeFixture(testingContext, home, ".ssh/ssh-keyflow/state.json", string(changed))
		if _, err := runCLI(testingContext, home, "0\n", runner{}); err == nil {
			testingContext.Fatalf("accepted %s", invalid)
		}
	}
}

type callbackRunner struct{ execute func(app.Command) error }

func (callbackRunner) LookPath(name string) (string, error) { return name, nil }
func (fake callbackRunner) Run(ctx context.Context, command app.Command) error {
	return fake.execute(command)
}

func TestMutationLockCoversExternalValidationAndConfirmation(testingContext *testing.T) {
	home := testingContext.TempDir()
	first := writeFixture(testingContext, home, ".ssh/first.pub", publicKey)
	second := writeFixture(testingContext, home, ".ssh/second.pub", otherPublicKey)
	nestedOutput := ""
	external := callbackRunner{execute: func(command app.Command) error {
		var err error
		nestedOutput, err = runCLI(testingContext, home, importInput(second, "second"), runner{})
		return err
	}}
	output, err := runCLI(testingContext, home, importInput(first, "first"), external)
	if err != nil || !strings.Contains(output, "Imported first") || !strings.Contains(nestedOutput, "locked") || strings.Contains(nestedOutput, "Imported second") {
		testingContext.Fatalf("operation lock: %v\nouter: %s\nsecond: %s", err, output, nestedOutput)
	}
}

func TestInvalidPrivateDigestMetadataStopsStartup(testingContext *testing.T) {
	home := testingContext.TempDir()
	public := writeFixture(testingContext, home, ".ssh/key.pub", publicKey)
	private := writeFixture(testingContext, home, ".ssh/key", "-----BEGIN OPENSSH PRIVATE KEY-----\nfixture\n")
	output, err := runCLI(testingContext, home, "5\ni\n"+public+"\n"+private+"\nPaired\nyes\n0\n", pairRunner{derived: publicKey})
	if err != nil || !strings.Contains(output, "Imported") {
		testingContext.Fatalf("setup: %v %s", err, output)
	}
	data, _ := os.ReadFile(filepath.Join(home, ".ssh/ssh-keyflow/state.json"))
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		testingContext.Fatal(err)
	}
	document["keys"].([]any)[0].(map[string]any)["private_sha256"] = strings.Repeat("z", 64)
	changed, _ := json.Marshal(document)
	writeFixture(testingContext, home, ".ssh/ssh-keyflow/state.json", string(changed))
	if _, err := runCLI(testingContext, home, "0\n", runner{}); err == nil {
		testingContext.Fatal("accepted non-hex private digest metadata")
	}
}

func TestMismatchedPairNeverBecomesManaged(testingContext *testing.T) {
	home := testingContext.TempDir()
	public := writeFixture(testingContext, home, ".ssh/key.pub", publicKey)
	private := writeFixture(testingContext, home, ".ssh/key", "-----BEGIN OPENSSH PRIVATE KEY-----\nfixture\n")
	output, err := runCLI(testingContext, home, "5\ni\n"+public+"\n"+private+"\nMismatched\nyes\n2\n0\n", pairRunner{derived: otherPublicKey})
	if err != nil || !strings.Contains(output, "fingerprint mismatch") || strings.Contains(output, "managed key Mismatched") || strings.Contains(output, "Imported") {
		testingContext.Fatalf("mismatch: %v\n%s", err, output)
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh/ssh-keyflow/state.json")); !os.IsNotExist(err) {
		testingContext.Fatal("mismatched pair changed state")
	}
}

func TestImportRequiresExplicitConfirmationAndDoesNotDuplicateKeys(testingContext *testing.T) {
	home := testingContext.TempDir()
	path := writeFixture(testingContext, home, ".ssh/key.pub", publicKey)
	output, err := runCLI(testingContext, home, "5\ni\n"+path+"\n\nCancelled\nno\n0\n", runner{})
	if err != nil || !strings.Contains(output, "Cancelled") || strings.Contains(output, "Imported") {
		testingContext.Fatalf("cancel: %v %s", err, output)
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh/ssh-keyflow/state.json")); !os.IsNotExist(err) {
		testingContext.Fatal("cancel wrote state")
	}
	output, err = runCLI(testingContext, home, importInput(path, "first"), runner{})
	if err != nil || !strings.Contains(output, "Imported first") {
		testingContext.Fatalf("setup: %v %s", err, output)
	}
	before, _ := os.ReadFile(filepath.Join(home, ".ssh/ssh-keyflow/state.json"))
	output, err = runCLI(testingContext, home, importInput(path, "duplicate"), runner{})
	after, _ := os.ReadFile(filepath.Join(home, ".ssh/ssh-keyflow/state.json"))
	if err != nil || !strings.Contains(output, "already managed") || !bytes.Equal(before, after) {
		testingContext.Fatalf("duplicate: %v %s", err, output)
	}
}

func TestChildOwnsCredentialInputWithoutMenuReadAhead(testingContext *testing.T) {
	home := testingContext.TempDir()
	public := writeFixture(testingContext, home, ".ssh/key.pub", publicKey)
	private := writeFixture(testingContext, home, ".ssh/key", "-----BEGIN OPENSSH PRIVATE KEY-----\nfixture\n")
	credential := "child-only-secret"
	external := callbackRunner{execute: func(command app.Command) error {
		if strings.Contains(strings.Join(command.Args, " "), credential) {
			testingContext.Fatal("credential in arguments")
		}
		if command.Args[0] != "-y" {
			return nil
		}
		if !command.Interactive {
			testingContext.Fatal("private key derivation must require terminal input")
		}
		buffer := make([]byte, len(credential)+1)
		if _, err := io.ReadFull(command.Stdin, buffer); err != nil || string(buffer) != credential+"\n" {
			testingContext.Fatalf("menu consumed child input: %v %q", err, buffer)
		}
		_, err := io.WriteString(command.Stdout, publicKey)
		return err
	}}
	output, err := runCLI(testingContext, home, "5\ni\n"+public+"\n"+private+"\nTerminal\n"+credential+"\nyes\n0\n", external)
	if err != nil || !strings.Contains(output, "Imported Terminal") || strings.Contains(output, credential) {
		testingContext.Fatalf("credential handoff: %v %s", err, output)
	}
	data, _ := os.ReadFile(filepath.Join(home, ".ssh/ssh-keyflow/state.json"))
	if strings.Contains(string(data), credential) {
		testingContext.Fatal("credential persisted")
	}
}

func TestStartupReportsUnwritableStorage(testingContext *testing.T) {
	home := testingContext.TempDir()
	writeFixture(testingContext, home, ".ssh/ssh-keyflow", "not a directory")
	if _, err := runCLI(testingContext, home, "0\n", runner{}); err == nil {
		testingContext.Fatal("startup accepted unusable storage")
	}
}

func TestUnsupportedMenuActionsReportNoChanges(testingContext *testing.T) {
	home := testingContext.TempDir()
	output, err := runCLI(testingContext, home, "3\n4\n6\n0\n", runner{})
	if err != nil || strings.Count(output, "Not implemented in this build") != 3 {
		testingContext.Fatalf("unimplemented choices: %v %s", err, output)
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh/ssh-keyflow/state.json")); !os.IsNotExist(err) {
		testingContext.Fatal("unimplemented choice changed state")
	}
}

func TestStateValidatesIDsRelationshipsAndLifecycleFields(testingContext *testing.T) {
	home := testingContext.TempDir()
	path := writeFixture(testingContext, home, ".ssh/key.pub", publicKey)
	if output, err := runCLI(testingContext, home, importInput(path, "Stable"), runner{}); err != nil || !strings.Contains(output, "Imported") {
		testingContext.Fatalf("setup: %v %s", err, output)
	}
	statePath := filepath.Join(home, ".ssh/ssh-keyflow/state.json")
	data, _ := os.ReadFile(statePath)
	var baseline map[string]any
	if err := json.Unmarshal(data, &baseline); err != nil {
		testingContext.Fatal(err)
	}
	keyID := baseline["keys"].([]any)[0].(map[string]any)["id"]
	baseline["hosts"] = []any{map[string]any{"id": "host-1", "alias": "vps", "hostname": "example.test", "user": "owner", "port": 22, "os": "linux"}}
	baseline["registrations"] = []any{map[string]any{"id": "registration-1", "key_id": keyID, "host_id": "host-1", "status": "registered", "connection_check": "unverified", "operation": "register", "stage": "complete"}}
	valid, _ := json.Marshal(baseline)
	writeFixture(testingContext, home, ".ssh/ssh-keyflow/state.json", string(valid))
	output, err := runCLI(testingContext, home, "2\n0\n", runner{})
	if err != nil || !strings.Contains(output, "["+keyID.(string)+"]") || !strings.Contains(output, "status=registered connection=unverified operation=register stage=complete") {
		testingContext.Fatalf("valid lifecycle/identity: %v %s", err, output)
	}
	mutations := map[string]func(map[string]any){
		"duplicate key ID": func(document map[string]any) {
			keys := document["keys"].([]any)
			document["keys"] = append(keys, keys[0])
		},
		"duplicate host alias": func(document map[string]any) {
			hosts := document["hosts"].([]any)
			document["hosts"] = append(hosts, map[string]any{"id": "host-2", "alias": "vps", "hostname": "other.test", "user": "owner", "port": 22, "os": "linux"})
		},
		"dangling key": func(document map[string]any) {
			document["registrations"].([]any)[0].(map[string]any)["key_id"] = "missing"
		},
		"dangling host": func(document map[string]any) {
			document["registrations"].([]any)[0].(map[string]any)["host_id"] = "missing"
		},
		"duplicate relationship": func(document map[string]any) {
			registration := document["registrations"].([]any)[0].(map[string]any)
			duplicate := map[string]any{}
			for name, value := range registration {
				duplicate[name] = value
			}
			duplicate["id"] = "registration-2"
			document["registrations"] = append(document["registrations"].([]any), duplicate)
		},
		"unsupported status": func(document map[string]any) {
			document["registrations"].([]any)[0].(map[string]any)["status"] = "active"
		},
		"unsupported connection result": func(document map[string]any) {
			document["keys"].([]any)[0].(map[string]any)["connection_check"] = "okay"
		},
		"unsupported stage": func(document map[string]any) {
			document["registrations"].([]any)[0].(map[string]any)["stage"] = "finished"
		},
		"unknown nested field": func(document map[string]any) {
			document["keys"].([]any)[0].(map[string]any)["password"] = "must-not-be-accepted"
		},
	}
	for name, mutate := range mutations {
		testingContext.Run(name, func(testingContext *testing.T) {
			var document map[string]any
			if err := json.Unmarshal(valid, &document); err != nil {
				testingContext.Fatal(err)
			}
			mutate(document)
			changed, _ := json.Marshal(document)
			writeFixture(testingContext, home, ".ssh/ssh-keyflow/state.json", string(changed))
			if _, err := runCLI(testingContext, home, "0\n", runner{}); err == nil {
				testingContext.Fatal("accepted invalid state")
			}
			after, _ := os.ReadFile(statePath)
			if !bytes.Equal(changed, after) {
				testingContext.Fatal("startup overwrote invalid state")
			}
		})
	}
}

func TestPublicOnlyImportPersistsAndListsStableManagedIdentity(testingContext *testing.T) {
	home := testingContext.TempDir()
	path := writeFixture(testingContext, home, ".ssh/鍵 with spaces.pub", publicKey)
	var output bytes.Buffer
	input := "5\ni\n" + path + "\n\nLaptop key\nyes\n2\n0\n"
	err := app.Run(context.Background(), app.Options{Home: home, Input: strings.NewReader(input), Output: &output, Runner: runner{}})
	if err != nil || !strings.Contains(output.String(), "Imported") || !strings.Contains(output.String(), "managed key Laptop key [") || !strings.Contains(output.String(), "connection=unverified") {
		testingContext.Fatalf("import: %v\n%s", err, output.String())
	}
	var next bytes.Buffer
	err = app.Run(context.Background(), app.Options{Home: home, Input: strings.NewReader("2\n0\n"), Output: &next, Runner: runner{}})
	if err != nil || !strings.Contains(next.String(), "managed key Laptop key [") {
		testingContext.Fatalf("reload: %v %s", err, next.String())
	}
	after, _ := os.ReadFile(path)
	if string(after) != publicKey {
		testingContext.Fatal("import changed source")
	}
}

type missingRunner struct{ ready bool }

func (fake *missingRunner) LookPath(name string) (string, error) {
	if name == "ssh-keygen" && !fake.ready {
		fake.ready = true
		return "", fmt.Errorf("missing")
	}
	return name, nil
}
func (*missingRunner) Run(ctx context.Context, command app.Command) error { return nil }

func TestMissingDependencyCanBeRechecked(testingContext *testing.T) {
	var output bytes.Buffer
	err := app.Run(context.Background(), app.Options{Home: testingContext.TempDir(), Input: strings.NewReader("r\n0\n"), Output: &output, Runner: &missingRunner{}})
	for _, expected := range []string{"ssh-keygen", "Add-WindowsCapability", "apt install openssh-client", "xcode-select --install", "Recheck", "2: List"} {
		if err != nil || !strings.Contains(output.String(), expected) {
			testingContext.Fatalf("missing %q: %v\n%s", expected, err, output.String())
		}
	}
}
