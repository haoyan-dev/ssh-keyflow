package app_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ssh-keyflow/internal/app"
)

func TestStructuredListDoesNotInitializeStorageOrRequireSSH(testingContext *testing.T) {
	home := fixtureTempDir(testingContext)
	var output bytes.Buffer
	err := app.Run(context.Background(), app.Options{Home: home, Input: strings.NewReader(""), Output: &output, Runner: &missingRunner{}, Request: &app.Request{Operation: "list"}})
	if err != nil {
		testingContext.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh")); !os.IsNotExist(err) {
		testingContext.Fatal("read-only listing initialized SSH storage")
	}
}

func TestStructuredImportUsesFieldsAndExplicitConfirmation(testingContext *testing.T) {
	home := fixtureTempDir(testingContext)
	public := writeFixture(testingContext, home, ".ssh/source.pub", publicKey)
	var output bytes.Buffer
	err := app.Run(context.Background(), app.Options{
		Home: home, Output: &output, Runner: runner{},
		Request: &app.Request{Operation: "import", Name: "structured", PublicPath: public},
		Confirm: func(message string) (bool, error) { return true, nil },
	})
	if err != nil {
		testingContext.Fatalf("import: %v %s", err, output.String())
	}
	snapshot, err := app.Snapshot(app.Options{Home: home})
	if err != nil || len(snapshot.State.Keys) != 1 || snapshot.State.Keys[0].Name != "structured" {
		testingContext.Fatalf("import result: %v %#v", err, snapshot)
	}
	if strings.Contains(output.String(), "Existing public or private key path:") {
		testingContext.Fatal("structured operation requested a menu field")
	}
}

func TestStructuredCreateAndDeleteRetainIdentity(testingContext *testing.T) {
	home := fixtureTempDir(testingContext)
	var output bytes.Buffer
	options := app.Options{Home: home, Output: &output, Runner: onboardingRunner{testingContext: testingContext, home: home}, Confirm: func(string) (bool, error) { return true, nil }}
	options.Request = &app.Request{Operation: "create", Name: "direct"}
	if err := app.Run(context.Background(), options); err != nil {
		testingContext.Fatalf("create: %v %s", err, output.String())
	}
	view, err := app.Snapshot(options)
	if err != nil || len(view.State.Keys) != 1 {
		testingContext.Fatalf("created key unavailable: %v", err)
	}
	key := view.State.Keys[0]
	options.Request = &app.Request{Operation: "delete", Key: key.ID}
	if err := app.Run(context.Background(), options); err != nil {
		testingContext.Fatalf("delete: %v %s", err, output.String())
	}
	view, err = app.Snapshot(options)
	if err != nil || len(view.State.Keys) != 0 || len(view.State.DeletedKeys) != 1 || view.State.DeletedKeys[0].Key.ID != key.ID {
		testingContext.Fatalf("deletion lost identity: %v %#v", err, view.State)
	}
}

func TestStructuredDestinationReusesImportedKeyAndOffersManualGuidance(testingContext *testing.T) {
	home := fixtureTempDir(testingContext)
	public := writeFixture(testingContext, home, ".ssh/source.pub", publicKey)
	var output bytes.Buffer
	options := app.Options{Home: home, Output: &output, Runner: runner{}, Confirm: func(string) (bool, error) { return true, nil }}
	options.Request = &app.Request{Operation: "import", Name: "reuse", PublicPath: public}
	if err := app.Run(context.Background(), options); err != nil {
		testingContext.Fatal(err)
	}
	options.Request = &app.Request{Operation: "add", Key: "reuse", Destination: app.Destination{Alias: "server", Hostname: "server.example", User: "alice", OS: "linux", Port: 22}, Bootstrap: "manual"}
	if err := app.Run(context.Background(), options); err != nil {
		testingContext.Fatalf("add: %v %s", err, output.String())
	}
	view, err := app.Snapshot(options)
	if err != nil || len(view.State.Hosts) != 1 || len(view.State.Registrations) != 1 || view.State.Registrations[0].Status != "unknown" || !strings.Contains(output.String(), "BEGIN REGISTRATION SCRIPT") {
		testingContext.Fatalf("manual onboarding: %v %#v %s", err, view.State, output.String())
	}
	options.Request = &app.Request{Operation: "register", Host: "server", Key: "reuse", Bootstrap: "manual"}
	if err := app.Run(context.Background(), options); err != nil {
		testingContext.Fatalf("register: %v", err)
	}
	options.Request = &app.Request{Operation: "resume", Resume: view.State.Onboardings[0].ID, Bootstrap: "manual"}
	if err := app.Run(context.Background(), options); err != nil {
		testingContext.Fatalf("resume: %v", err)
	}
	options.Request = &app.Request{Operation: "unregister", Registration: view.State.Registrations[0].ID, Bootstrap: "manual"}
	if err := app.Run(context.Background(), options); err != nil {
		testingContext.Fatalf("unregister: %v", err)
	}
	view, err = app.Snapshot(options)
	if err != nil || view.State.Registrations[0].Operation != "unregister" || view.State.Registrations[0].Stage != "unknown" {
		testingContext.Fatalf("manual revocation must remain unknown: %v %#v", err, view.State)
	}
}

func TestStructuredCancellationPreservesSourceAndDoesNotSaveState(testingContext *testing.T) {
	home := fixtureTempDir(testingContext)
	public := writeFixture(testingContext, home, ".ssh/source.pub", publicKey)
	err := app.Run(context.Background(), app.Options{
		Home: home, Output: &bytes.Buffer{}, Runner: runner{},
		Request: &app.Request{Operation: "import", Name: "declined", PublicPath: public},
		Confirm: func(string) (bool, error) { return false, nil },
	})
	if !errors.Is(err, app.ErrCancelled) {
		testingContext.Fatalf("cancellation reported success: %v", err)
	}
	if data, err := os.ReadFile(public); err != nil || string(data) != publicKey {
		testingContext.Fatalf("source changed: %v %s", err, data)
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh/ssh-keyflow/state.json")); !os.IsNotExist(err) {
		testingContext.Fatal("cancelled import saved state")
	}
}

func TestStructuredInvalidAuthenticationDoesNotInitializeStorage(testingContext *testing.T) {
	home := fixtureTempDir(testingContext)
	err := app.Run(context.Background(), app.Options{
		Home: home, Output: &bytes.Buffer{}, Runner: runner{},
		Request: &app.Request{Operation: "add", Name: "new", Destination: app.Destination{Alias: "server", Hostname: "server.example", User: "alice", OS: "linux", Port: 22}, Bootstrap: "guess"},
		Confirm: func(string) (bool, error) { return true, nil },
	})
	if err == nil {
		testingContext.Fatal("invalid authentication accepted")
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh")); !os.IsNotExist(err) {
		testingContext.Fatal("invalid input initialized storage")
	}
}

func TestStructuredAmbiguousKeyReportsInvalidInputWithoutChangingRecords(testingContext *testing.T) {
	home := fixtureTempDir(testingContext)
	options := app.Options{Home: home, Output: &bytes.Buffer{}, Runner: runner{}, Confirm: func(string) (bool, error) { return true, nil }}
	for index, public := range []string{publicKey, otherPublicKey} {
		path := writeFixture(testingContext, home, fmt.Sprintf(".ssh/source%d.pub", index), public)
		options.Request = &app.Request{Operation: "import", Name: "same-name", PublicPath: path}
		if err := app.Run(context.Background(), options); err != nil {
			testingContext.Fatal(err)
		}
	}
	options.Request = &app.Request{Operation: "delete", Key: "same-name"}
	if err := app.Run(context.Background(), options); !errors.Is(err, app.ErrInvalidInput) {
		testingContext.Fatalf("ambiguous target was not a usage error: %v", err)
	}
	view, err := app.Snapshot(options)
	if err != nil || len(view.State.Keys) != 2 || len(view.State.Deletions) != 0 {
		testingContext.Fatalf("ambiguous target changed records: %v %#v", err, view.State)
	}
}

func TestStructuredMissingTargetsFailBeforeStorageInitialization(testingContext *testing.T) {
	requests := []app.Request{
		{Operation: "add", Key: "missing", Destination: app.Destination{Alias: "server", Hostname: "server.example", User: "alice", Port: 22, OS: "linux"}},
		{Operation: "delete", Key: "missing"},
		{Operation: "register", Host: "missing", Key: "missing"},
		{Operation: "unregister", Registration: "missing"},
		{Operation: "resume", Resume: "missing"},
	}
	for _, request := range requests {
		testingContext.Run(request.Operation, func(testingContext *testing.T) {
			home := fixtureTempDir(testingContext)
			err := app.Run(context.Background(), app.Options{
				Home: home, Output: &bytes.Buffer{}, Runner: runner{}, Request: &request,
				Confirm: func(string) (bool, error) { return true, nil },
			})
			if !errors.Is(err, app.ErrInvalidInput) {
				testingContext.Fatalf("missing target was not a usage error: %v", err)
			}
			if _, err := os.Stat(filepath.Join(home, ".ssh")); !os.IsNotExist(err) {
				testingContext.Fatal("invalid target initialized storage")
			}
		})
	}
}

func TestStructuredRegistrationUsesOriginalInputAndConfigConsent(testingContext *testing.T) {
	home, external := prepareRegistration(testingContext, true)
	terminal := strings.NewReader("")
	var output bytes.Buffer
	err := app.Run(context.Background(), app.Options{
		Home: home, Input: terminal, Output: &output, Runner: external,
		Request: &app.Request{Operation: "register", Host: "server", Key: "fixture", Bootstrap: "password", Configure: true},
		Confirm: func(string) (bool, error) { return true, nil },
	})
	view, readErr := app.Snapshot(app.Options{Home: home})
	if err != nil || readErr != nil || view.State.Registrations[0].ConnectionCheck != "verified" || len(view.State.Configs) != 1 || len(view.State.ConfigEdits) != 1 || !view.State.ConfigEdits[0].Complete {
		testingContext.Fatalf("direct registration/configuration failed: %v %v %#v %s", err, readErr, view.State, output.String())
	}
}

func TestStructuredRegistrationReportsSpecifiedKeyVerificationFailure(testingContext *testing.T) {
	for _, verification := range []string{"denied", "missing-marker"} {
		testingContext.Run(verification, func(testingContext *testing.T) {
			home, external := prepareRegistration(testingContext, true)
			external.ssh = func(command app.Command) error {
				payload := command.Args[len(command.Args)-1]
				switch {
				case strings.Contains(payload, "KEYFLOW_PROBE"):
					fmt.Fprintln(command.Stdout, "KEYFLOW_TARGET:/home/me/.ssh/authorized_keys")
				case strings.Contains(payload, "KEYFLOW_REGISTER"):
					fmt.Fprintln(command.Stdout, "KEYFLOW_REGISTERED")
				case strings.Contains(payload, "KEYFLOW_VERIFIED"):
					if verification == "denied" {
						return fmt.Errorf("specified key denied")
					}
				}
				return nil
			}
			var output bytes.Buffer
			err := app.Run(context.Background(), app.Options{
				Home: home, Input: strings.NewReader(""), Output: &output, Runner: external,
				Request: &app.Request{Operation: "register", Host: "server", Key: "fixture", Bootstrap: "password", Configure: true},
				Confirm: func(string) (bool, error) { return true, nil },
			})
			view, readErr := app.Snapshot(app.Options{Home: home})
			if err == nil || !strings.Contains(err.Error(), "specified-key connection verification failed") {
				testingContext.Fatalf("verification failure reported success: %v\n%s", err, output.String())
			}
			if readErr != nil || view.State.Registrations[0].Status != "registered" || view.State.Registrations[0].ConnectionCheck != "failed" ||
				len(view.State.History) == 0 || view.State.History[len(view.State.History)-1].ConnectionCheck != "failed" || len(view.State.Configs) != 0 {
				testingContext.Fatalf("verification failure lost completed registration or continued configuration: %v %#v", readErr, view.State)
			}
			if !strings.Contains(output.String(), "registered; connection=failed") {
				testingContext.Fatalf("completed registration and failed connection are not distinguished: %s", output.String())
			}
		})
	}
}

func TestStructuredResumeShowsSavedTargetsBeforeConfirmation(testingContext *testing.T) {
	for _, kind := range []string{"creation", "onboarding", "pending-key"} {
		testingContext.Run(kind, func(testingContext *testing.T) {
			home := fixtureTempDir(testingContext)
			var output bytes.Buffer
			options := app.Options{Home: home, Output: &output, Confirm: func(string) (bool, error) { return true, nil }}
			if kind == "onboarding" {
				var external registrationRunner
				home, external = prepareRegistration(testingContext, true)
				options.Home, options.Runner = home, external
			} else {
				options.Runner = onboardingRunner{testingContext: testingContext, home: home, generate: func(app.Command, string) error {
					return fmt.Errorf("simulated interruption")
				}}
				options.Request = &app.Request{Operation: "create", Name: "saved creation"}
				if kind == "pending-key" {
					options.Request = &app.Request{Operation: "add", Name: "saved onboarding key",
						Destination: app.Destination{Alias: "pending", Hostname: "pending.example", User: "alice", Port: 2222, OS: "macos"}}
				}
				if err := app.Run(context.Background(), options); err == nil {
					testingContext.Fatal("fixture creation was not interrupted")
				}
			}
			view, err := app.Snapshot(options)
			if err != nil {
				testingContext.Fatal(err)
			}
			var id string
			var expected []string
			if kind == "creation" {
				creation := view.State.Creations[0]
				id = creation.ID
				expected = []string{creation.ID, creation.Name, creation.PrivatePath, creation.PublicPath, "stage=" + creation.Stage, "reconcile"}
			} else {
				job, host := view.State.Onboardings[0], view.State.Hosts[0]
				id = job.ID
				expected = []string{job.ID, "stage=" + job.Stage, host.Alias, fmt.Sprintf("%s@%s:%d", host.User, host.Hostname, host.Port), "OS=" + host.OS, "register"}
				if kind == "pending-key" {
					creation := view.State.Creations[0]
					expected = append(expected, creation.Name, creation.PrivatePath, creation.PublicPath, "stage="+creation.Stage)
				} else {
					key := view.State.Keys[0]
					expected = append(expected, key.ID, key.Name, key.Fingerprint, key.PrivatePath, key.PublicPath)
				}
			}
			statePath := filepath.Join(home, ".ssh", "ssh-keyflow", "state.json")
			before, err := os.ReadFile(statePath)
			if err != nil {
				testingContext.Fatal(err)
			}
			output.Reset()
			preview := ""
			options.Request = &app.Request{Operation: "resume", Resume: id}
			options.Confirm = func(string) (bool, error) { preview = output.String(); return false, nil }
			err = app.Run(context.Background(), options)
			if !errors.Is(err, app.ErrCancelled) {
				testingContext.Fatalf("resume rejection was not cancellation: %v", err)
			}
			for _, text := range expected {
				if !strings.Contains(preview, text) {
					testingContext.Fatalf("missing saved target %q before confirmation:\n%s", text, preview)
				}
			}
			after, err := os.ReadFile(statePath)
			if err != nil || !bytes.Equal(before, after) {
				testingContext.Fatalf("rejected resume changed saved state: %v", err)
			}
			for _, creation := range view.State.Creations {
				if _, err := os.Stat(creation.PrivatePath); !os.IsNotExist(err) {
					testingContext.Fatal("rejected resume generated a key")
				}
			}
		})
	}
}
