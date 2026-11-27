package app_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/state"
)

type failingOutput struct {
	marker string
	err    error
	failed bool
}

func (output *failingOutput) Write(data []byte) (int, error) {
	if !output.failed && strings.HasPrefix(string(data), output.marker) {
		output.failed = true
		return 0, output.err
	}
	return len(data), nil
}

func TestMutationOutputFailureStopsBeforeConfirmationAndSave(t *testing.T) {
	for _, operation := range []string{"import", "create", "remove", "resume"} {
		for _, phase := range []string{"preview", "confirmation"} {
			t.Run(operation+"/"+phase, func(t *testing.T) {
				home := tempHome(t)
				key := imported(t, home, "existing", publicKey)
				request := app.Request{Operation: operation, Key: key.ID}
				preview := ""
				switch operation {
				case "import":
					request.Name = "incoming"
					request.PublicPath = filepath.Join(home, "incoming.pub")
					request.PrivatePath = filepath.Join(home, "incoming")
					write(t, request.PublicPath, otherPublicKey)
					write(t, request.PrivatePath, "private incoming")
					preview = "Import "
				case "create":
					request.Name = "new"
					preview = "Create Ed25519 "
				case "remove":
					preview = "Remove managed key "
				case "resume":
					document := load(t, home)
					privatePath := filepath.Join(state.Directory(home), "keys", "id_ed25519_output")
					creation := state.Creation{
						ID: "creation-output", KeyID: "key-output", Name: "resumed",
						PrivatePath: privatePath, PublicPath: privatePath + ".pub",
						CreatedAt: timestamp, Stage: "planned",
					}
					document.Creations = append(document.Creations, creation)
					saveFixture(t, home, document)
					request.Resume = creation.ID
					preview = "Resume "
				}

				before := outputTestFiles(t, home)
				writeErr := errors.New("preview output unavailable")
				output := &failingOutput{marker: preview, err: writeErr}
				if phase == "confirmation" {
					output.marker = "Type yes "
				}
				confirmations, nativeCalls := 0, 0
				_, err := run(t, home, request, func(options *app.Options) {
					options.Output = output
					options.Confirm = func(string) (bool, error) {
						confirmations++
						return true, nil
					}
					options.Runner = localRunner{
						t: t, home: home, input: options.Input, derived: otherPublicKey,
						before: func(app.Command) { nativeCalls++ },
					}
				})
				if !output.failed {
					t.Fatal("requested output failure was never exercised")
				}
				if !errors.Is(err, writeErr) {
					t.Errorf("expected writer error, got %v", err)
				}
				if confirmations != 0 {
					t.Errorf("confirmation called %d times after output failure", confirmations)
				}
				if operation != "import" && nativeCalls != 0 {
					t.Errorf("native process called %d times after output failure", nativeCalls)
				}
				if after := outputTestFiles(t, home); !reflect.DeepEqual(before, after) {
					t.Error("state, backup, source or managed files changed after output failure")
				}
			})
		}
	}
}

func outputTestFiles(t *testing.T, home string) map[string]string {
	t.Helper()
	files := make(map[string]string)
	err := filepath.WalkDir(home, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[path] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}
