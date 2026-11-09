package app_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/state"
)

func removalFixture(t *testing.T) (string, state.Key, state.Key) {
	t.Helper()
	home := tempHome(t)
	key := imported(t, home, "first", publicKey, true)
	other := imported(t, home, "second", otherPublicKey, false)
	for _, alias := range []string{"one", "two"} {
		if _, err := run(t, home, app.Request{Operation: "add", Key: key.ID, Destination: destination(alias, "linux", "")}); err != nil {
			t.Fatal(err)
		}
		if _, err := run(t, home, app.Request{Operation: "associate", Host: alias, Key: other.ID}); err != nil {
			t.Fatal(err)
		}
	}
	return home, key, other
}

func TestRemovalClearsOnlySelectedMetadataAndPreservesFiles(t *testing.T) {
	for _, operation := range []string{"remove-key", "remove-host", "dissociate"} {
		t.Run(operation, func(t *testing.T) {
			home, key, other := removalFixture(t)
			before := load(t, home)
			statePath := filepath.Join(state.Directory(home), "state.json")
			previous := read(t, statePath)
			config := filepath.Join(home, ".ssh", "config")
			write(t, config, "Host one\n  HostName example.test\n")
			var reports []app.Report
			out, err := run(t, home, app.Request{Operation: operation, Key: key.ID, Host: "ONE"}, func(o *app.Options) {
				o.Runner = forbiddenRunner{t}
				o.Report = func(r app.Report) { reports = append(reports, r) }
			})
			if err != nil {
				t.Fatalf("%v: %s", err, out)
			}
			doc := load(t, home)
			wantKeys, wantHosts, wantLinks := 2, 2, 3
			if operation == "remove-key" {
				wantKeys, wantLinks = 1, 2
				if !reflect.DeepEqual(doc.RemovedKeys, []state.Key{key}) {
					t.Fatal("removed key identity was not preserved")
				}
			}
			if operation == "remove-host" {
				wantHosts, wantLinks = 1, 2
				if !reflect.DeepEqual(doc.RemovedHosts, before.Hosts[:1]) {
					t.Fatal("removed destination identity was not preserved")
				}
			}
			if len(doc.Keys) != wantKeys || len(doc.Hosts) != wantHosts || len(doc.Associations) != wantLinks {
				t.Fatalf("wrong remaining records: %+v", doc)
			}
			for _, link := range doc.Associations {
				if link.KeyID == key.ID && link.HostID == before.Hosts[0].ID {
					t.Fatal("selected association remains")
				}
			}
			if string(read(t, statePath+".bak")) != string(previous) || string(read(t, key.PublicPath)) != publicKey ||
				string(read(t, key.PrivatePath)) != "private fixture" || string(read(t, other.PublicPath)) != otherPublicKey ||
				string(read(t, config)) != "Host one\n  HostName example.test\n" {
				t.Fatal("files or state backup changed unexpectedly")
			}
			if len(reports) != 1 || reports[0].Kind != app.ReportRemoval || !reports[0].Saved {
				t.Fatalf("missing saved report: %+v", reports)
			}
			if !strings.Contains(out, "Key files, SSH configuration, remote authorized keys and legacy history are preserved") {
				t.Fatal("confirmation did not explain effects")
			}
		})
	}
}

type forbiddenRunner struct{ t *testing.T }

func (r forbiddenRunner) LookPath(string) (string, error) {
	r.t.Fatal("metadata removal probed an external command")
	return "", nil
}
func (r forbiddenRunner) Run(_ context.Context, _ app.Command) error {
	r.t.Fatal("metadata removal ran an external command")
	return nil
}

func TestRemovalCancellationAndFailuresDoNotSave(t *testing.T) {
	for _, operation := range []string{"remove-key", "remove-host", "dissociate"} {
		for _, failure := range []string{"decline", "missing-confirmation", "confirmation-error", "output", "lock", "backup"} {
			t.Run(operation+"/"+failure, func(t *testing.T) {
				home, key, _ := removalFixture(t)
				path := filepath.Join(state.Directory(home), "state.json")
				before := read(t, path)
				if failure == "lock" {
					write(t, filepath.Join(state.Directory(home), "state.lock"), "held")
				}
				if failure == "backup" {
					if err := os.Remove(path + ".bak"); err != nil {
						t.Fatal(err)
					}
					if err := os.Mkdir(path+".bak", 0700); err != nil {
						t.Fatal(err)
					}
				}
				var saved bool
				_, err := run(t, home, app.Request{Operation: operation, Key: key.ID, Host: "one"}, func(o *app.Options) {
					o.Report = func(r app.Report) { saved = saved || r.Saved }
					switch failure {
					case "decline":
						o.Confirm = func(string) (bool, error) { return false, nil }
					case "missing-confirmation":
						o.Confirm = nil
					case "confirmation-error":
						o.Confirm = func(string) (bool, error) { return false, io.ErrUnexpectedEOF }
					case "output":
						o.Output = &failingOutput{err: io.ErrClosedPipe}
					}
				})
				if err == nil || saved || string(read(t, path)) != string(before) {
					t.Fatalf("failure changed state: %v saved=%v", err, saved)
				}
			})
		}
	}
}

func TestRemovalRetainsLegacyHistoryAndAllowsIdentityReuse(t *testing.T) {
	home, key, other := removalFixture(t)
	doc := load(t, home)
	host := doc.Hosts[0]
	doc.Registrations = []state.Registration{{ID: "legacy", KeyID: key.ID, HostID: host.ID, Status: "unknown", ConnectionCheck: "unknown", Operation: "register", Stage: "unknown"}}
	doc.History = []state.RegistrationEvent{{RegistrationID: "legacy", At: timestamp, Status: "unknown", ConnectionCheck: "unknown", Operation: "register", Stage: "unknown"}}
	doc.Onboardings = []state.Onboarding{{ID: "onboarding", HostID: host.ID, KeyID: key.ID, Stage: "manual", CreatedAt: timestamp}}
	doc.Hosts[0].ConfigKeyID = key.ID
	saveFixture(t, home, doc)
	for _, request := range []app.Request{{Operation: "remove-key", Key: "first"}, {Operation: "remove-host", Host: host.ID}} {
		if _, err := run(t, home, request); err != nil {
			t.Fatal(err)
		}
	}
	after := load(t, home)
	if !reflect.DeepEqual(doc.Registrations, after.Registrations) || !reflect.DeepEqual(doc.History, after.History) || !reflect.DeepEqual(doc.Onboardings, after.Onboardings) {
		t.Fatal("legacy records changed")
	}
	reimported := imported(t, home, "first", publicKey, true)
	if reimported.ID == key.ID {
		t.Fatal("reimport reused historical ID")
	}
	if _, err := run(t, home, app.Request{Operation: "add", Key: other.ID, Destination: destination("one", "linux", "")}); err != nil {
		t.Fatal("removed alias could not be reused", err)
	}
	for _, request := range []app.Request{{Operation: "remove-key", Key: key.ID}, {Operation: "remove-host", Host: host.ID}, {Operation: "dissociate", Key: other.ID, Host: "one-missing"}} {
		if _, err := run(t, home, request); !errors.Is(err, app.ErrInvalidInput) {
			t.Fatal("removed/missing target accepted", err)
		}
	}
}

func TestRemovedCreatedKeyRetainsCreationAndDoesNotRequireFiles(t *testing.T) {
	home := tempHome(t)
	if _, err := run(t, home, app.Request{Operation: "create", Name: "created"}); err != nil {
		t.Fatal(err)
	}
	doc := load(t, home)
	key := doc.Keys[0]
	if err := os.Remove(key.PublicPath); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, home, app.Request{Operation: "remove-key", Key: key.ID}); err != nil {
		t.Fatal(err)
	}
	after := load(t, home)
	if len(after.Keys) != 0 || !reflect.DeepEqual(doc.Creations, after.Creations) || len(after.RemovedKeys) != 1 {
		t.Fatal("created-key removal lost history")
	}
	view, err := app.Snapshot(app.Options{Home: home})
	if err != nil || view.Review != "" {
		t.Fatalf("removed key still requires files: %v %s", err, view.Review)
	}
}

func TestRemovedRecordsCannotRemainAssociatedOrCorruptHistory(t *testing.T) {
	for _, corruption := range []string{"archived-key-link", "archived-host-link", "duplicate-key-id", "duplicate-host-id", "bad-archive-key", "bad-archive-host"} {
		t.Run(corruption, func(t *testing.T) {
			home, key, _ := removalFixture(t)
			before := load(t, home)
			host := before.Hosts[0]
			for _, request := range []app.Request{{Operation: "remove-key", Key: key.ID}, {Operation: "remove-host", Host: host.ID}} {
				if _, err := run(t, home, request); err != nil {
					t.Fatal(err)
				}
			}
			doc := load(t, home)
			switch corruption {
			case "archived-key-link":
				doc.Associations = append(doc.Associations, state.Association{HostID: doc.Hosts[0].ID, KeyID: key.ID})
			case "archived-host-link":
				doc.Associations = append(doc.Associations, state.Association{HostID: host.ID, KeyID: doc.Keys[0].ID})
			case "duplicate-key-id":
				doc.RemovedKeys = append(doc.RemovedKeys, doc.Keys[0])
			case "duplicate-host-id":
				doc.RemovedHosts = append(doc.RemovedHosts, doc.Hosts[0])
			case "bad-archive-key":
				doc.RemovedKeys[0].Fingerprint = "invalid"
			case "bad-archive-host":
				doc.RemovedHosts[0].Port = 0
			}
			if err := doc.Validate(); err == nil {
				t.Fatal("invalid archived state accepted")
			}
		})
	}
}

func TestRemovalRejectsAmbiguousNamesAndMissingAssociationsWithoutChanges(t *testing.T) {
	home, first, _ := removalFixture(t)
	doc := load(t, home)
	doc.Keys[1].Name = first.Name
	before := saveFixture(t, home, doc)
	for _, request := range []app.Request{
		{Operation: "remove-key", Key: first.Name},
		{Operation: "dissociate", Host: "one", Key: first.Name},
		{Operation: "remove-key", Key: "missing"},
		{Operation: "remove-host", Host: "missing"},
	} {
		if _, err := run(t, home, request); !errors.Is(err, app.ErrInvalidInput) {
			t.Fatal("ambiguous/missing target accepted", err)
		}
		if string(read(t, filepath.Join(state.Directory(home), "state.json"))) != string(before) {
			t.Fatal("invalid target changed state")
		}
	}
	request := app.Request{Operation: "dissociate", Host: "one", Key: first.ID}
	if _, err := run(t, home, request); err != nil {
		t.Fatal(err)
	}
	before = read(t, filepath.Join(state.Directory(home), "state.json"))
	if _, err := run(t, home, request); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatal("nonexistent association accepted", err)
	}
	if string(read(t, filepath.Join(state.Directory(home), "state.json"))) != string(before) {
		t.Fatal("nonexistent association changed state")
	}
}
