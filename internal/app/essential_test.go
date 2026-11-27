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
		r.t.Fatalf("unexpected process: %s", command.Name)
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

// imported adopts a key pair whose private half derives to public.
func imported(t *testing.T, home, name, public string) state.Key {
	t.Helper()
	path := filepath.Join(home, name+".pub")
	write(t, path, public)
	request := app.Request{Operation: "import", Name: name, PublicPath: path, PrivatePath: strings.TrimSuffix(path, ".pub")}
	write(t, request.PrivatePath, "private fixture "+name)
	if out, err := run(t, home, request, func(o *app.Options) {
		o.Runner = localRunner{t: t, home: home, input: o.Input, derived: public}
	}); err != nil {
		t.Fatalf("import: %v %s", err, out)
	}
	doc := load(t, home)
	return doc.Keys[len(doc.Keys)-1]
}

func TestReadOnlyAndRetiredOperationsNeverInitializeStorage(t *testing.T) {
	for _, operation := range []string{"list", "add", "guide", "pin-host", "register", "delete"} {
		home := tempHome(t)
		if _, err := run(t, home, app.Request{Operation: operation}); !errors.Is(err, app.ErrInvalidInput) {
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
	if _, err := app.Snapshot(app.Options{Home: home}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh")); !os.IsNotExist(err) {
		t.Fatal("snapshot initialized storage")
	}
}

func TestImportPairValidationAndConfirmationPreserveSources(t *testing.T) {
	for _, mode := range []string{"pair", "public-only", "mismatch", "declined", "missing-confirmation", "source-changed", "private-changed"} {
		t.Run(mode, func(t *testing.T) {
			home := tempHome(t)
			pub := filepath.Join(home, "source.pub")
			priv := filepath.Join(home, "source")
			write(t, pub, publicKey)
			write(t, priv, "private fixture")
			request := app.Request{Operation: "import", Name: "source", PublicPath: pub, PrivatePath: priv}
			if mode == "public-only" {
				request.PrivatePath = ""
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
			if mode == "public-only" && !errors.Is(err, app.ErrInvalidInput) {
				t.Fatalf("public-only import not rejected as input: %v", err)
			}
			if (mode == "pair") != (err == nil) {
				t.Fatalf("result: %v", err)
			}
			doc := load(t, home)
			if mode == "pair" {
				if len(doc.Keys) != 1 || doc.Keys[0].Origin != "imported" || doc.Keys[0].PrivatePath != priv {
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
			if _, err := run(t, home, app.Request{Operation: "create", Name: "second"}); err == nil || !strings.Contains(err.Error(), "unfinished creation") {
				t.Fatal("second creation allocated while one is unfinished", err)
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
			if len(doc.Keys) != 1 || doc.Keys[0].PrivatePath != original || doc.Keys[0].ID != creation.KeyID || len(doc.Creations) != 0 {
				t.Fatal("completed creation not replaced by its key", doc)
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
			text := o.Output.(*bytes.Buffer).String()
			for _, line := range strings.Split(text, "\n") {
				if strings.HasPrefix(line, "Public: ") {
					write(t, strings.TrimPrefix(line, "Public: "), "occupied")
				}
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

func TestRemoveKeyKeepsFilesAndOnlyDropsMetadata(t *testing.T) {
	home := tempHome(t)
	first := imported(t, home, "first", publicKey)
	second := imported(t, home, "second", otherPublicKey)
	if _, err := run(t, home, app.Request{Operation: "remove", Key: first.ID}, func(o *app.Options) {
		o.Confirm = func(string) (bool, error) { return false, nil }
	}); !errors.Is(err, app.ErrCancelled) || len(load(t, home).Keys) != 2 {
		t.Fatal("declined removal changed state", err)
	}
	if _, err := run(t, home, app.Request{Operation: "remove", Key: "first"}); err != nil {
		t.Fatal(err)
	}
	doc := load(t, home)
	if len(doc.Keys) != 1 || doc.Keys[0].ID != second.ID {
		t.Fatal("wrong key removed", doc)
	}
	if string(read(t, first.PublicPath)) != publicKey || string(read(t, first.PrivatePath)) != "private fixture first" {
		t.Fatal("removal changed key files")
	}
	// Removing metadata only: a removed pair can be adopted again with a new ID.
	again := imported(t, home, "first", publicKey)
	if again.ID == first.ID {
		t.Fatal("removed ID reused")
	}
}

func TestLegacyStateIsArchivedAndReducedOnFirstChange(t *testing.T) {
	home := tempHome(t)
	keep := filepath.Join(home, "keep")
	write(t, keep, "private keep")
	write(t, keep+".pub", publicKey)
	write(t, filepath.Join(home, "agent.pub"), otherPublicKey)
	privateDigest := sha256.Sum256([]byte("private keep"))
	keepKey := map[string]any{
		"id": "key-keep", "name": "keep", "fingerprint": "SHA256:" + strings.Repeat("A", 43), "public_path": keep + ".pub",
		"private_path": keep, "private_sha256": hex.EncodeToString(privateDigest[:]), "connection_check": "unverified",
		"origin": "imported", "imported_at": timestamp,
	}
	legacy := map[string]any{
		"version": 2,
		"keys": []any{keepKey, map[string]any{
			"id": "key-agent", "name": "agent", "fingerprint": "SHA256:" + strings.Repeat("B", 43), "public_path": filepath.Join(home, "agent.pub"),
			"connection_check": "unverified", "origin": "imported", "imported_at": timestamp,
		}},
		"hosts":         []any{map[string]any{"id": "host-1", "alias": "server", "hostname": "server.example", "user": "alice", "port": 22, "os": "linux"}},
		"registrations": []any{},
		"associations":  []any{map[string]any{"host_id": "host-1", "key_id": "key-keep"}},
		"creations": []any{map[string]any{
			"id": "creation-done", "key_id": "key-keep", "name": "keep", "private_path": keep, "public_path": keep + ".pub",
			"created_at": timestamp, "stage": "complete",
		}},
	}
	data, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(state.Directory(home), "state.json")
	write(t, path, string(data))

	view, err := app.Snapshot(app.Options{Home: home})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"schema v2", "public-only key \"agent\"", "1 destination"} {
		if !strings.Contains(view.Review, want) {
			t.Fatalf("migration notice lacks %q: %s", want, view.Review)
		}
	}
	if len(view.State.Keys) != 1 || view.State.Keys[0].ID != "key-keep" || len(view.State.Creations) != 0 {
		t.Fatal("legacy conversion kept wrong records", view.State)
	}
	if !bytes.Equal(data, read(t, path)) {
		t.Fatal("reading migrated the file")
	}
	if matches, _ := filepath.Glob(filepath.Join(state.Directory(home), state.ArchivePattern)); len(matches) != 0 {
		t.Fatal("reading archived the file")
	}

	// The fingerprint in the fixture is synthetic, so remove (no file checks) is the first change.
	output, err := run(t, home, app.Request{Operation: "remove", Key: "key-keep"})
	if err != nil {
		t.Fatal(err)
	}
	notice, preview := strings.Index(output, "1 destination"), strings.Index(output, "Remove managed key")
	if notice < 0 || preview < 0 || notice > preview {
		t.Fatalf("migration notice must precede the confirmed preview: %s", output)
	}
	matches, err := filepath.Glob(filepath.Join(state.Directory(home), state.ArchivePattern))
	if err != nil || len(matches) != 1 || !strings.Contains(filepath.Base(matches[0]), "state.v2-") {
		t.Fatal("legacy archive missing", matches, err)
	}
	if !bytes.Equal(data, read(t, matches[0])) || !bytes.Equal(data, read(t, path+".bak")) {
		t.Fatal("archive or backup is not the exact legacy file")
	}
	after := load(t, home)
	if after.Version != state.Version || after.Migration != nil || len(after.Keys) != 0 {
		t.Fatal("state not saved as current schema", after)
	}
	if strings.Contains(string(read(t, path)), "hosts") {
		t.Fatal("legacy fields written to the current schema")
	}
	// A second change must not archive again.
	if _, err := run(t, home, app.Request{Operation: "create", Name: "next"}); err != nil {
		t.Fatal(err)
	}
	if again, _ := filepath.Glob(filepath.Join(state.Directory(home), state.ArchivePattern)); len(again) != 1 {
		t.Fatal("current schema archived again", again)
	}
}

func TestLocksAndBackupFailurePreserveState(t *testing.T) {
	for _, obstacle := range []string{"lock", "backup"} {
		t.Run(obstacle, func(t *testing.T) {
			home := tempHome(t)
			key := imported(t, home, "key", publicKey)
			path := filepath.Join(state.Directory(home), "state.json")
			before := read(t, path)
			if obstacle == "lock" {
				write(t, filepath.Join(state.Directory(home), "state.lock"), "unknown lock")
			} else if err := os.Mkdir(path+".bak", 0700); err != nil {
				t.Fatal(err)
			}
			out, err := run(t, home, app.Request{Operation: "remove", Key: key.ID})
			if err == nil || strings.Contains(out, "removed") || !bytes.Equal(before, read(t, path)) {
				t.Fatal(err, out)
			}
		})
	}
}

func TestInvalidStateIsRejectedWithoutOverwrite(t *testing.T) {
	for _, invalid := range []string{
		`{`, `null`, `{}`, `{"version":99,"keys":[]}`,
		`{"version":3,"version":3,"keys":[]}`,
		`{"version":3,"Version":3,"keys":[]}`,
		`{"version":3,"keys":[],"secret":"no"}`,
		`{"version":3,"keys":[],"hosts":[]}`,
		`{"version":3}`,
		`{"version":3,"keys":[]} {}`,
		`{"version":2,"keys":[],"registrations":[]}`,
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
	key := imported(t, home, "key", publicKey)
	write(t, key.PrivatePath, "changed private")
	if _, err := app.PublicKey(home, key.ID); err == nil {
		t.Fatal("private replacement ignored")
	}
	path := filepath.Join(home, "second.pub")
	write(t, path, otherPublicKey)
	write(t, filepath.Join(home, "second"), "private second")
	if _, err := run(t, home, app.Request{Operation: "import", Name: "second", PublicPath: path, PrivatePath: filepath.Join(home, "second")}); err == nil {
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

func TestAmbiguousKeyNameIsRejected(t *testing.T) {
	home := tempHome(t)
	first := imported(t, home, "first", publicKey)
	imported(t, home, "second", otherPublicKey)
	doc := load(t, home)
	doc.Keys[0].Name, doc.Keys[1].Name = "shared", "shared"
	saveFixture(t, home, doc)
	if _, err := app.PublicKey(home, "shared"); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatal("ambiguous name accepted", err)
	}
	if _, err := app.PublicKey(home, first.ID); err != nil {
		t.Fatal(err)
	}
}

func TestStateValidationRejectsCorruptMetadataWithoutOverwrite(t *testing.T) {
	creation := func(d *state.State) state.Creation {
		dir := filepath.Dir(d.Keys[0].PublicPath)
		return state.Creation{ID: "creation", KeyID: "key-new", Name: "new", PrivatePath: filepath.Join(dir, "new"), PublicPath: filepath.Join(dir, "new.pub"), CreatedAt: timestamp, Stage: "planned"}
	}
	for name, corrupt := range map[string]func(*state.State){
		"nonhex-private-digest": func(d *state.State) { d.Keys[0].PrivateSHA256 = strings.Repeat("z", 64) },
		"missing-private":       func(d *state.State) { d.Keys[0].PrivatePath, d.Keys[0].PrivateSHA256 = "", "" },
		"invalid-origin":        func(d *state.State) { d.Keys[0].Origin = "guessed" },
		"invalid-timestamp":     func(d *state.State) { d.Keys[0].ImportedAt = "yesterday" },
		"mixed-provenance":      func(d *state.State) { d.Keys[0].CreatedAt = timestamp },
		"same-normalized-path": func(d *state.State) {
			d.Keys[0].PrivatePath = filepath.Dir(d.Keys[0].PublicPath) + string(filepath.Separator) + "." + string(filepath.Separator) + filepath.Base(d.Keys[0].PublicPath)
		},
		"duplicate-key": func(d *state.State) { d.Keys = append(d.Keys, d.Keys[0]) },
		"creation-reuses-key-id": func(d *state.State) {
			c := creation(d)
			c.KeyID = d.Keys[0].ID
			d.Creations = []state.Creation{c}
		},
		"completed-creation-retained": func(d *state.State) {
			c := creation(d)
			c.Stage = "complete"
			d.Creations = []state.Creation{c}
		},
		"creation-path-conflict": func(d *state.State) {
			c := creation(d)
			c.PrivatePath, c.PublicPath = d.Keys[0].PrivatePath, d.Keys[0].PrivatePath+".pub"
			d.Creations = []state.Creation{c}
		},
	} {
		t.Run(name, func(t *testing.T) {
			home := tempHome(t)
			imported(t, home, "key", publicKey)
			document := load(t, home)
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
