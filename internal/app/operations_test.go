package app_test

import (
	"bytes"
	"context"
	"errors"
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
