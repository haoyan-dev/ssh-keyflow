package app_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/state"
)

const publicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB fixture@example"
const otherPublicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIC other@example"
const timestamp = "2026-11-08T00:00:00Z"

type localRunner struct {
	t        *testing.T
	home     string
	input    io.Reader
	generate func(string) error
	derived  string
	before   func(app.Command)
}

func (r localRunner) LookPath(name string) (string, error) {
	if name != "ssh-keygen" {
		r.t.Fatalf("unexpected dependency %s", name)
	}
	return name, nil
}
func (r localRunner) Run(_ context.Context, command app.Command) error {
	if command.Name != "ssh-keygen" {
		r.t.Fatalf("remote process attempted: %s", command.Name)
	}
	if _, err := os.Stat(filepath.Join(state.Directory(r.home), "state.lock")); err != nil {
		r.t.Fatal("native validation/generation not locked", err)
	}
	if r.before != nil {
		r.before(command)
	}
	if command.Args[0] == "-l" {
		return nil
	}
	if !command.Interactive || (r.input != nil && command.Stdin != r.input) {
		r.t.Fatal("native terminal handoff changed")
	}
	if command.Args[0] == "-y" {
		text := r.derived
		if text == "" {
			text = publicKey
		}
		_, err := fmt.Fprintln(command.Stdout, text)
		return err
	}
	if !reflect.DeepEqual(command.Args[:3], []string{"-t", "ed25519", "-f"}) {
		r.t.Fatalf("unexpected keygen %v", command.Args)
	}
	doc := load(r.t, r.home)
	if len(doc.Creations) == 0 || doc.Creations[len(doc.Creations)-1].Stage != "pending" {
		r.t.Fatal("no durable pending intent before key generation")
	}
	path := command.Args[3]
	if r.generate != nil {
		return r.generate(path)
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		r.t.Fatal("attempt to overwrite private file")
	}
	write(r.t, path, "private fixture")
	write(r.t, path+".pub", publicKey)
	return nil
}

func tempHome(t *testing.T) string {
	t.Helper()
	path, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return path
}
func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}
func read(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func load(t *testing.T, home string) state.State {
	t.Helper()
	doc, err := state.Load(home)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}
func saveFixture(t *testing.T, home string, doc state.State) []byte {
	t.Helper()
	if err := doc.Validate(); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(state.Directory(home), "state.json"), string(data))
	return data
}
func run(t *testing.T, home string, request app.Request, customize ...func(*app.Options)) (string, error) {
	t.Helper()
	var out bytes.Buffer
	input := strings.NewReader("native fixture input")
	options := app.Options{Home: home, Input: input, Output: &out, Request: &request, Runner: localRunner{t: t, home: home, input: input}, Confirm: func(string) (bool, error) { return true, nil }}
	for _, change := range customize {
		change(&options)
	}
	err := app.Run(context.Background(), options)
	return out.String(), err
}
func imported(t *testing.T, home, name, public string, private bool) state.Key {
	t.Helper()
	path := filepath.Join(home, name+".pub")
	write(t, path, public)
	request := app.Request{Operation: "import", Name: name, PublicPath: path}
	if private {
		request.PrivatePath = strings.TrimSuffix(path, ".pub")
		write(t, request.PrivatePath, "private fixture")
	}
	if out, err := run(t, home, request); err != nil {
		t.Fatalf("import: %v %s", err, out)
	}
	doc := load(t, home)
	return doc.Keys[len(doc.Keys)-1]
}
func destination(alias, system, account string) state.Host {
	return state.Host{Alias: alias, Hostname: "server.example", User: "alice", Port: 2222, OS: system, WindowsAccount: account}
}

func TestReadOnlyAndRemovedOperationsNeverInitializeStorage(t *testing.T) {
	for _, operation := range []string{"list", "register", "unregister", "delete"} {
		home := tempHome(t)
		_, err := run(t, home, app.Request{Operation: operation})
		if (operation == "list") != (err == nil) {
			t.Fatalf("%s: %v", operation, err)
		}
		if _, err := os.Stat(filepath.Join(home, ".ssh")); !os.IsNotExist(err) {
			t.Fatal("storage initialized")
		}
	}
	home := tempHome(t)
	if _, err := app.PublicKey(home, "missing"); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatal(err)
	}
	if _, err := app.Guide(home, "missing", "missing"); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatal(err)
	}
	if _, err := app.Snapshot(app.Options{Home: home}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh")); !os.IsNotExist(err) {
		t.Fatal("snapshot initialized storage")
	}
}

func TestImportPairValidationAndConfirmationPreserveSources(t *testing.T) {
	for _, mode := range []string{"public-only", "pair", "mismatch", "declined", "missing-confirmation", "source-changed", "private-changed"} {
		t.Run(mode, func(t *testing.T) {
			home := tempHome(t)
			pub := filepath.Join(home, "source.pub")
			priv := filepath.Join(home, "source")
			write(t, pub, publicKey)
			write(t, priv, "private fixture")
			request := app.Request{Operation: "import", Name: "source", PublicPath: pub}
			if mode == "pair" || mode == "mismatch" || mode == "private-changed" {
				request.PrivatePath = priv
			}
			_, err := run(t, home, request, func(o *app.Options) {
				r := localRunner{t: t, home: home, input: o.Input}
				if mode == "mismatch" {
					r.derived = otherPublicKey
				}
				if mode == "private-changed" {
					r.before = func(c app.Command) {
						if c.Args[0] == "-y" {
							write(t, priv, "replaced private")
						}
					}
				}
				o.Runner = r
				if mode == "declined" {
					o.Confirm = func(string) (bool, error) { return false, nil }
				}
				if mode == "missing-confirmation" {
					o.Confirm = nil
				}
				if mode == "source-changed" {
					o.Confirm = func(string) (bool, error) { write(t, pub, otherPublicKey); return true, nil }
				}
			})
			success := mode == "public-only" || mode == "pair"
			if success != (err == nil) {
				t.Fatalf("result: %v", err)
			}
			doc := load(t, home)
			if success {
				if len(doc.Keys) != 1 || doc.Keys[0].Origin != "imported" || doc.Keys[0].ConnectionCheck != "unverified" {
					t.Fatal("invalid import metadata", doc)
				}
				if string(read(t, pub)) != publicKey || string(read(t, priv)) != "private fixture" {
					t.Fatal("source changed")
				}
			} else if len(doc.Keys) != 0 {
				t.Fatal("failed import saved a key")
			}
		})
	}
}

func TestInterruptedCreationResumesSameFilesWithoutOverwrite(t *testing.T) {
	for _, partial := range []string{"neither", "private", "pair", "public"} {
		t.Run(partial, func(t *testing.T) {
			home := tempHome(t)
			original := ""
			_, err := run(t, home, app.Request{Operation: "create", Name: "日本語 / ../laptop"}, func(o *app.Options) {
				o.Runner = localRunner{t: t, home: home, input: o.Input, generate: func(path string) error {
					original = path
					if partial == "private" || partial == "pair" {
						write(t, path, "private fixture")
					}
					if partial == "public" || partial == "pair" {
						write(t, path+".pub", publicKey)
					}
					return fmt.Errorf("interrupted fixture")
				}}
			})
			if err == nil {
				t.Fatal("interruption hidden")
			}
			doc := load(t, home)
			creation := doc.Creations[0]
			if len(doc.Keys) != 0 || creation.PrivatePath != original || filepath.Dir(original) != filepath.Join(state.Directory(home), "keys") {
				t.Fatal("creation intent lost")
			}
			request := app.Request{Operation: "resume", Resume: creation.ID}
			_, err = run(t, home, request, func(o *app.Options) {
				o.Runner = localRunner{t: t, home: home, input: o.Input, generate: func(path string) error {
					if partial != "neither" || path != original {
						t.Fatal("recovery attempted overwrite/new path")
					}
					write(t, path, "private fixture")
					write(t, path+".pub", publicKey)
					return nil
				}}
			})
			if partial == "public" {
				if err == nil {
					t.Fatal("public-only recovery generated another key")
				}
				if _, err := os.Stat(original); !os.IsNotExist(err) {
					t.Fatal("public-only recovery wrote private key")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			doc = load(t, home)
			if len(doc.Keys) != 1 || doc.Keys[0].PrivatePath != original || doc.Creations[0].Stage != "complete" || doc.Keys[0].ConnectionCheck != "unverified" {
				t.Fatal(doc)
			}
			if request.Operation != "resume" || request.Resume != creation.ID {
				t.Fatal("request mutated")
			}
			if _, err := run(t, home, request); !errors.Is(err, app.ErrInvalidInput) {
				t.Fatal("completed creation resumed", err)
			}
		})
	}
}

func TestCreationProtectsFilesAppearingDuringConfirmation(t *testing.T) {
	home := tempHome(t)
	_, err := run(t, home, app.Request{Operation: "create", Name: "safe"}, func(o *app.Options) {
		o.Confirm = func(string) (bool, error) {
			dir := filepath.Join(state.Directory(home), "keys")
			// The preview contains the exact automatic path; create its public half.
			text := o.Output.(*bytes.Buffer).String()
			for _, line := range strings.Split(text, "\n") {
				if strings.HasPrefix(line, "Public: ") {
					write(t, strings.TrimPrefix(line, "Public: "), "occupied")
				}
			}
			if _, e := os.Stat(dir); e != nil {
				t.Fatal(e)
			}
			return true, nil
		}
	})
	if err == nil || !strings.Contains(err.Error(), "appeared") {
		t.Fatal(err)
	}
	if len(load(t, home).Creations) != 0 {
		t.Fatal("occupied paths claimed")
	}
}

func TestGuidanceForEveryOSIsReadOnlyAndDoesNotImplySuccess(t *testing.T) {
	for _, target := range []struct{ os, account, path string }{
		{"linux", "", "~/.ssh/authorized_keys"}, {"macos", "", "~/.ssh/authorized_keys"},
		{"windows", "standard", `%USERPROFILE%\.ssh\authorized_keys`},
		{"windows", "administrator", `%ProgramData%\ssh\administrators_authorized_keys`},
	} {
		t.Run(target.os+target.account, func(t *testing.T) {
			home := tempHome(t)
			key := imported(t, home, "key", publicKey, true)
			h := destination("server", target.os, target.account)
			if _, err := run(t, home, app.Request{Operation: "add", Key: key.ID, Destination: h}); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(state.Directory(home), "state.json")
			before := read(t, path)
			guide, err := app.Guide(home, "SERVER", key.ID)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{target.path, "not checked", "Manual removal", "Host server", "IdentityFile", "connect (POSIX shell / PowerShell)", "local config step", "\nssh server\n"} {
				if !strings.Contains(guide, want) {
					t.Fatalf("missing %q: %s", want, guide)
				}
			}
			for _, unwanted := range []string{"BEGIN REGISTRATION SCRIPT", "apt install", "Start-Service sshd", "Start-Service -Name sshd", "Set-Service -Name sshd", "New-NetFirewallRule"} {
				if strings.Contains(guide, unwanted) {
					t.Fatalf("automation retained: %s", guide)
				}
			}
			if !bytes.Equal(before, read(t, path)) {
				t.Fatal("guide changed state")
			}
			doc := load(t, home)
			if len(doc.Registrations) != 0 || len(doc.History) != 0 || len(doc.Onboardings) != 0 || len(doc.Associations) != 1 {
				t.Fatal("association fabricated remote state", doc)
			}
			public, err := app.PublicKey(home, key.ID)
			if err != nil || public != strings.Join(strings.Fields(publicKey)[:2], " ") {
				t.Fatal(public, err)
			}
		})
	}
}

func TestPublicOnlyGuidanceHasNoUsablePrivateIdentity(t *testing.T) {
	home := tempHome(t)
	key := imported(t, home, "public-only", publicKey, false)
	if _, err := run(t, home, app.Request{Operation: "add", Key: key.ID, Destination: destination("server", "linux", "")}); err != nil {
		t.Fatal(err)
	}
	text, err := app.Guide(home, "server", key.ID)
	if err != nil || !strings.Contains(text, "Public-only") || strings.Contains(text, "IdentityFile \"") || strings.Contains(text, "\nssh ") {
		t.Fatal(text, err)
	}
}

func TestAssociationsValidateTargetsAndRecheckAfterConfirmation(t *testing.T) {
	home := tempHome(t)
	key := imported(t, home, "key", publicKey, false)
	host := destination("server", "linux", "")
	request := app.Request{Operation: "add", Key: key.ID, Destination: host}
	_, err := run(t, home, request, func(o *app.Options) {
		o.Confirm = func(string) (bool, error) { write(t, key.PublicPath, otherPublicKey); return true, nil }
	})
	if err == nil || len(load(t, home).Hosts) != 0 {
		t.Fatal("changed key associated")
	}
	write(t, key.PublicPath, publicKey)
	if _, err := run(t, home, request); err != nil {
		t.Fatal(err)
	}
	before := read(t, filepath.Join(state.Directory(home), "state.json"))
	if _, err := run(t, home, request); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatal("duplicate host accepted", err)
	}
	if _, err := run(t, home, app.Request{Operation: "associate", Host: "server", Key: key.ID}); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatal("duplicate association accepted", err)
	}
	if !bytes.Equal(before, read(t, filepath.Join(state.Directory(home), "state.json"))) {
		t.Fatal("rejected association changed state")
	}
	second := imported(t, home, "second", otherPublicKey, false)
	if _, err := run(t, home, app.Request{Operation: "associate", Host: "server", Key: second.ID}); err != nil {
		t.Fatal(err)
	}
	if len(load(t, home).Associations) != 2 {
		t.Fatal("second association missing")
	}
}

func TestLegacyStateMigratesWithoutChangingHistoryOrSSHFiles(t *testing.T) {
	home := tempHome(t)
	key := imported(t, home, "key", publicKey, false)
	doc := load(t, home)
	doc.Version = 1
	doc.Hosts = []state.Host{{ID: "host-old", Alias: "old", Hostname: "old.example", User: "me", Port: 22, OS: "linux", ConfigKeyID: key.ID}}
	doc.Registrations = []state.Registration{{ID: "reg-old", KeyID: key.ID, HostID: "host-old", Status: "unknown", ConnectionCheck: "unknown", Operation: "register", Stage: "unknown"}}
	doc.Onboardings = []state.Onboarding{{ID: "onboarding-old", HostID: "host-old", KeyID: key.ID, Stage: "manual", CreatedAt: timestamp}}
	doc.History = []state.RegistrationEvent{{RegistrationID: "reg-old", At: timestamp, Status: "unknown", ConnectionCheck: "unknown", Operation: "register", Stage: "unknown"}}
	config := filepath.Join(state.Directory(home), "config")
	main := filepath.Join(home, ".ssh", "config")
	backup := main + ".backup"
	for _, path := range []string{config, main, backup} {
		write(t, path, "existing SSH configuration bytes\r\n")
	}
	digest := sha256.Sum256(read(t, config))
	hash := hex.EncodeToString(digest[:])
	doc.Configs = []state.Config{{Path: config, SHA256: hash}}
	doc.ConfigEdits = []state.ConfigEdit{{Path: main, BeforeSHA256: hash, AfterSHA256: hash, BackupPath: backup, Complete: false}}
	before := saveFixture(t, home, doc)
	if _, err := app.Snapshot(app.Options{Home: home}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, read(t, filepath.Join(state.Directory(home), "state.json"))) {
		t.Fatal("read migrated v1")
	}
	if _, err := run(t, home, app.Request{Operation: "associate", Host: "old", Key: key.ID}); err != nil {
		t.Fatal(err)
	}
	after := load(t, home)
	if after.Version != 2 || len(after.Associations) != 1 {
		t.Fatal("migration missing")
	}
	after.Version = 1
	after.Associations = nil
	if !reflect.DeepEqual(doc, after) {
		t.Fatalf("legacy records changed\nbefore=%+v\nafter=%+v", doc, after)
	}
	if !bytes.Equal(before, read(t, filepath.Join(state.Directory(home), "state.json.bak"))) {
		t.Fatal("previous state backup not exact")
	}
	for _, path := range []string{config, main, backup} {
		if string(read(t, path)) != "existing SSH configuration bytes\r\n" {
			t.Fatal("configuration changed", path)
		}
	}
	for _, id := range []string{"reg-old", "onboarding-old"} {
		if _, err := run(t, home, app.Request{Operation: "resume", Resume: id}); !errors.Is(err, app.ErrInvalidInput) {
			t.Fatal("legacy workflow resumed", err)
		}
	}
}

func TestLegacyPendingDeletionIsQuarantinedWithoutBlockingOtherKeys(t *testing.T) {
	home := tempHome(t)
	old := imported(t, home, "old", publicKey, false)
	doc := load(t, home)
	doc.Version = 1
	digest := sha256.Sum256(read(t, old.PublicPath))
	doc.Deletions = []state.LocalDeletion{{ID: "deletion-old", KeyID: old.ID, CreatedAt: timestamp, Stage: "pending", PublicSHA256: hex.EncodeToString(digest[:]), PrivateRemoved: true, PublicRemoved: true}}
	saveFixture(t, home, doc)
	if err := os.Remove(old.PublicPath); err != nil {
		t.Fatal(err)
	}
	view, err := app.Snapshot(app.Options{Home: home})
	if err != nil || !strings.Contains(view.Review, "manual review") {
		t.Fatal(view, err)
	}
	if _, err := app.PublicKey(home, old.ID); err == nil {
		t.Fatal("quarantined key used")
	}
	if _, err := run(t, home, app.Request{Operation: "remove-key", Key: old.ID}); err == nil || !strings.Contains(err.Error(), "manual review") {
		t.Fatal("quarantined key removed", err)
	}
	second := imported(t, home, "healthy", otherPublicKey, false)
	if _, err := run(t, home, app.Request{Operation: "add", Key: second.ID, Destination: destination("server", "linux", "")}); err != nil {
		t.Fatal(err)
	}
	after := load(t, home)
	if !reflect.DeepEqual(doc.Deletions, after.Deletions) || len(after.Keys) != 2 {
		t.Fatal("legacy deletion completed or key lost")
	}
	if _, err := run(t, home, app.Request{Operation: "resume", Resume: "deletion-old"}); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatal(err)
	}
}

func TestLocksAndBackupFailurePreserveStateAndNeverReportSaved(t *testing.T) {
	for _, obstacle := range []string{"lock", "backup"} {
		t.Run(obstacle, func(t *testing.T) {
			home := tempHome(t)
			key := imported(t, home, "key", publicKey, false)
			path := filepath.Join(state.Directory(home), "state.json")
			before := read(t, path)
			if obstacle == "lock" {
				write(t, filepath.Join(state.Directory(home), "state.lock"), "unknown lock")
			} else {
				if err := os.Mkdir(path+".bak", 0700); err != nil {
					t.Fatal(err)
				}
			}
			var reports []app.Report
			out, err := run(t, home, app.Request{Operation: "add", Key: key.ID, Destination: destination("server", "linux", "")}, func(o *app.Options) { o.Report = func(r app.Report) { reports = append(reports, r) } })
			if err == nil || strings.Contains(out, "Association saved") || !bytes.Equal(before, read(t, path)) {
				t.Fatal(err, out)
			}
			for _, report := range reports {
				if report.Saved {
					t.Fatal("failed save reported committed")
				}
			}
		})
	}
}

func TestInvalidStateIsRejectedWithoutOverwrite(t *testing.T) {
	for _, invalid := range []string{
		`{`, `null`, `{"version":99,"keys":[],"hosts":[],"registrations":[]}`,
		`{"version":1,"version":1,"keys":[],"hosts":[],"registrations":[]}`,
		`{"version":1,"Version":1,"keys":[],"hosts":[],"registrations":[]}`,
		`{"version":2,"keys":[],"hosts":[],"registrations":[],"secret":"no"}`,
		`{"version":2,"keys":[],"hosts":[],"registrations":[],"associations":[{"host_id":"missing","key_id":"missing"}]}`,
		`{"version":2,"keys":[],"hosts":[],"registrations":[]} {}`,
	} {
		home := tempHome(t)
		path := filepath.Join(state.Directory(home), "state.json")
		write(t, path, invalid)
		if _, err := app.Snapshot(app.Options{Home: home}); err == nil {
			t.Fatal("invalid state accepted", invalid)
		}
		if _, err := run(t, home, app.Request{Operation: "create", Name: "key"}); err == nil {
			t.Fatal("invalid state updated")
		}
		if string(read(t, path)) != invalid {
			t.Fatal("invalid state overwritten")
		}
	}
}

func TestIdentityChangesStopCopyAndFurtherImport(t *testing.T) {
	home := tempHome(t)
	key := imported(t, home, "key", publicKey, true)
	write(t, key.PrivatePath, "changed private")
	if _, err := app.PublicKey(home, key.ID); err == nil {
		t.Fatal("private replacement ignored")
	}
	path := filepath.Join(home, "second.pub")
	write(t, path, otherPublicKey)
	if _, err := run(t, home, app.Request{Operation: "import", Name: "second", PublicPath: path}); err == nil {
		t.Fatal("integrity failure ignored")
	}
}

func TestReadConfirmationDoesNotConsumeNativeInput(t *testing.T) {
	input := strings.NewReader("yes\npassphrase\n")
	approved, err := app.ReadConfirmation(input)
	remaining, _ := io.ReadAll(input)
	if err != nil || !approved || string(remaining) != "passphrase\n" {
		t.Fatal(approved, err, string(remaining))
	}
	var output bytes.Buffer
	err = (app.SystemRunner{}).Run(context.Background(), app.Command{Name: "ssh-keygen", Interactive: true, Stdin: strings.NewReader("secret"), Stdout: &output, Stderr: &output})
	if err == nil || !strings.Contains(err.Error(), "real terminal") || output.Len() != 0 {
		t.Fatal("nonterminal credentials accepted", err)
	}
}

func TestResumeRejectsChangedRecordedPrivateFileWithoutGenerating(t *testing.T) {
	home := tempHome(t)
	_, err := run(t, home, app.Request{Operation: "create", Name: "recover"}, func(o *app.Options) {
		o.Runner = localRunner{t: t, home: home, input: o.Input, generate: func(path string) error {
			write(t, path, "original private")
			return fmt.Errorf("interrupted")
		}}
	})
	if err == nil {
		t.Fatal("interruption hidden")
	}
	doc := load(t, home)
	creation := doc.Creations[0]
	write(t, creation.PrivatePath, "replacement private")
	before := read(t, filepath.Join(state.Directory(home), "state.json"))
	_, err = run(t, home, app.Request{Operation: "resume", Resume: creation.ID})
	if err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatal(err)
	}
	if !bytes.Equal(before, read(t, filepath.Join(state.Directory(home), "state.json"))) {
		t.Fatal("changed creation saved")
	}
	if string(read(t, creation.PrivatePath)) != "replacement private" {
		t.Fatal("replacement overwritten")
	}
}

func TestAmbiguousKeyAndInvalidAssociationsAreRejected(t *testing.T) {
	home := tempHome(t)
	first := imported(t, home, "first", publicKey, false)
	second := imported(t, home, "second", otherPublicKey, false)
	doc := load(t, home)
	doc.Keys[0].Name, doc.Keys[1].Name = "shared", "shared"
	doc.Hosts = []state.Host{{ID: "host", Alias: "server", Hostname: "server.example", User: "alice", Port: 22, OS: "linux"}}
	saveFixture(t, home, doc)
	if _, err := app.PublicKey(home, "shared"); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatal("ambiguous name accepted", err)
	}
	if _, err := app.PublicKey(home, first.ID); err != nil {
		t.Fatal(err)
	}
	for _, associations := range [][]state.Association{
		{{HostID: "missing", KeyID: first.ID}},
		{{HostID: "host", KeyID: "missing"}},
		{{HostID: "host", KeyID: second.ID}, {HostID: "host", KeyID: second.ID}},
	} {
		doc.Associations = associations
		if doc.Validate() == nil {
			t.Fatal("invalid association accepted", associations)
		}
	}
	doc.Version = 1
	doc.Associations = []state.Association{{HostID: "host", KeyID: first.ID}}
	if doc.Validate() == nil {
		t.Fatal("v1 association accepted")
	}
}

func TestGuideQuotesLiteralPercentAndApostropheInIdentityPath(t *testing.T) {
	home := tempHome(t)
	key := imported(t, home, "key%20's with spaces", publicKey, true)
	if _, err := run(t, home, app.Request{Operation: "add", Key: key.ID, Destination: destination("server", "linux", "")}); err != nil {
		t.Fatal(err)
	}
	text, err := app.Guide(home, "server", key.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"key%%20's with spaces", "local config step", "\nssh server\n"} {
		if !strings.Contains(text, want) {
			t.Fatalf("lost literal identity path %q: %s", want, text)
		}
	}
}

func TestNativeOpenSSHParsesCopiedSnippetWithoutConnection(t *testing.T) {
	ssh, err := exec.LookPath("ssh")
	if err != nil {
		t.Skip("native OpenSSH unavailable")
	}
	home := tempHome(t)
	key := imported(t, home, "key with spaces", publicKey, true)
	if _, err := run(t, home, app.Request{Operation: "add", Key: key.ID, Destination: destination("server", "linux", "")}); err != nil {
		t.Fatal(err)
	}
	text, err := app.Guide(home, "server", key.ID)
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(text, "\nHost server\n")
	end := strings.Index(text[start:], "\n    IdentitiesOnly yes") + len("\n    IdentitiesOnly yes")
	config := filepath.Join(home, "copied config")
	write(t, config, text[start+1:start+end]+"\n")
	output, err := exec.Command(ssh, "-G", "-F", config, "server").CombinedOutput()
	if err != nil {
		t.Fatalf("native parser: %v %s", err, output)
	}
	for _, want := range []string{"hostname server.example", "user alice", "port 2222", "identityfile " + filepath.ToSlash(key.PrivatePath), "identitiesonly yes"} {
		if !strings.Contains(string(output), want) {
			t.Fatalf("missing %q: %s", want, output)
		}
	}
}

func TestGuidePreservesPlacementButOmitsExpandableIdentityPath(t *testing.T) {
	home := tempHome(t)
	key := imported(t, home, "${KEYFLOW_MISSING_ENV_VAR}", publicKey, true)
	if _, err := run(t, home, app.Request{Operation: "add", Key: key.ID, Destination: destination("server", "linux", "")}); err != nil {
		t.Fatal(err)
	}
	before := read(t, filepath.Join(state.Directory(home), "state.json"))
	text, err := app.Guide(home, "server", key.ID)
	if err != nil || !strings.Contains(text, publicKey[:strings.LastIndex(publicKey, " ")]) || !strings.Contains(text, "~/.ssh/authorized_keys") || !strings.Contains(text, "cannot be represented literally") {
		t.Fatal(err, text)
	}
	if strings.Contains(text, "\nHost server\n") || strings.Contains(text, "\nssh ") {
		t.Fatal("expandable identity path offered as a literal", text)
	}
	if !bytes.Equal(before, read(t, filepath.Join(state.Directory(home), "state.json"))) {
		t.Fatal("read-only guide changed state")
	}
}

func TestNativeOpenSSHExpandsEscapedPercentWithoutNetwork(t *testing.T) {
	ssh, err := exec.LookPath("ssh")
	if err != nil {
		t.Skip("native OpenSSH unavailable")
	}
	home := tempHome(t)
	key := imported(t, home, "key%20's with spaces", publicKey, true)
	if _, err := run(t, home, app.Request{Operation: "add", Key: key.ID, Destination: destination("server", "linux", "")}); err != nil {
		t.Fatal(err)
	}
	text, err := app.Guide(home, "server", key.ID)
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(text, "\nHost server\n")
	end := strings.Index(text[start:], "\n    IdentitiesOnly yes") + len("\n    IdentitiesOnly yes")
	config := filepath.Join(home, "copied config")
	write(t, config, text[start+1:start+end]+"\n")
	// A local ssh -V proxy exits without opening a network connection.
	output, err := exec.Command(ssh, "-vvv", "-F", config, "-o", "ProxyCommand=ssh -V", "-o", "BatchMode=yes", "server").CombinedOutput()
	if err == nil || !strings.Contains(string(output), "identity file "+filepath.ToSlash(key.PrivatePath)+" type") || strings.Contains(string(output), "invalid environment variable expansion") {
		t.Fatalf("native identity expansion: %v %s", err, output)
	}
}

func TestRetainedStateValidationRejectsCorruptMetadataWithoutOverwrite(t *testing.T) {
	for name, corrupt := range map[string]func(*state.State){
		"nonhex-private-digest": func(d *state.State) { d.Keys[0].PrivateSHA256 = strings.Repeat("z", 64) },
		"invalid-origin":        func(d *state.State) { d.Keys[0].Origin = "guessed" },
		"invalid-timestamp":     func(d *state.State) { d.Keys[0].ImportedAt = "yesterday" },
		"mixed-provenance":      func(d *state.State) { d.Keys[0].CreatedAt = timestamp },
		"same-normalized-path": func(d *state.State) {
			d.Keys[0].PrivatePath = filepath.Dir(d.Keys[0].PublicPath) + string(filepath.Separator) + "." + string(filepath.Separator) + filepath.Base(d.Keys[0].PublicPath)
		},
		"duplicate-id": func(d *state.State) { d.Hosts[0].ID = d.Keys[0].ID },
		"orphan-registration": func(d *state.State) {
			d.Registrations = []state.Registration{{ID: "registration", KeyID: "missing", HostID: "host", Status: "unknown", ConnectionCheck: "unverified", Operation: "register", Stage: "unknown"}}
		},
		"invalid-registration-stage": func(d *state.State) {
			d.Registrations = []state.Registration{{ID: "registration", KeyID: d.Keys[0].ID, HostID: "host", Status: "unknown", ConnectionCheck: "unverified", Operation: "register", Stage: "invented"}}
		},
		"orphan-history": func(d *state.State) {
			d.History = []state.RegistrationEvent{{RegistrationID: "missing", At: timestamp, Status: "unknown", ConnectionCheck: "unverified", Operation: "register", Stage: "unknown"}}
		},
		"invalid-creation-stage": func(d *state.State) {
			d.Creations = []state.Creation{{ID: "creation", KeyID: "new", Name: "new", PrivatePath: filepath.Join(filepath.Dir(d.Keys[0].PublicPath), "new"), PublicPath: filepath.Join(filepath.Dir(d.Keys[0].PublicPath), "new.pub"), CreatedAt: timestamp, Stage: "invented"}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			home := tempHome(t)
			imported(t, home, "key", publicKey, true)
			document := load(t, home)
			document.Hosts = []state.Host{{ID: "host", Alias: "server", Hostname: "server.example", User: "alice", Port: 22, OS: "linux"}}
			corrupt(&document)
			data, err := json.Marshal(document)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(state.Directory(home), "state.json")
			write(t, path, string(data))
			if _, err := app.Snapshot(app.Options{Home: home}); err == nil {
				t.Fatal("corrupt state loaded")
			}
			if _, err := run(t, home, app.Request{Operation: "create", Name: "new"}); err == nil {
				t.Fatal("corrupt state updated")
			}
			if !bytes.Equal(data, read(t, path)) {
				t.Fatal("corrupt state overwritten")
			}
		})
	}
}
