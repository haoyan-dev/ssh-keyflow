package app_test

import (
	"bytes"
	"context"
	"crypto/sha256"
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

func TestReportsManualRegisterPreservesPriorEvidence(t *testing.T) {
	for _, prior := range []string{"unrecorded", "verified"} {
		t.Run(prior, func(t *testing.T) {
			home, external := prepareRegistration(t, true)
			options := app.Options{Home: home, Input: strings.NewReader(""), Output: io.Discard, Runner: external, Confirm: func(string) (bool, error) { return true, nil }}
			if prior == "verified" {
				options.Request = &app.Request{Operation: "register", Host: "server", Key: "fixture", Bootstrap: "password"}
				if err := app.Run(context.Background(), options); err != nil {
					t.Fatal(err)
				}
			} else if err := state.Update(home, func(document *state.State) error {
				document.Registrations, document.Onboardings = []state.Registration{}, nil
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			before, err := app.Snapshot(options)
			if err != nil {
				t.Fatal(err)
			}
			var reports []app.Report
			options.Report = func(report app.Report) { reports = append(reports, report) }
			options.Request = &app.Request{Operation: "register", Host: "server", Key: "fixture", Bootstrap: "manual"}
			if err := app.Run(context.Background(), options); err != nil {
				t.Fatal(err)
			}
			after, err := app.Snapshot(app.Options{Home: home})
			if err != nil || !reflect.DeepEqual(before.State, after.State) {
				t.Fatalf("manual guidance changed prior evidence: %v", err)
			}
			manual := requireReport(t, reports, app.ReportManual, app.OutcomeRequired)
			if manual.HostID != before.State.Hosts[0].ID || manual.KeyID != before.State.Keys[0].ID || manual.Saved {
				t.Fatalf("manual result invents saved work: %+v", manual)
			}
			if prior == "unrecorded" && manual.RegistrationID != "" {
				t.Fatalf("invented recovery task: %+v", manual)
			}
			if prior == "verified" && manual.RegistrationID != before.State.Registrations[0].ID {
				t.Fatalf("lost existing pair: %+v", manual)
			}
			requireReport(t, reports, app.ReportVerification, app.OutcomeNotChecked)
			rejectReport(t, reports, app.ReportVerification, app.OutcomeVerified)
		})
	}
}

func requireReport(t *testing.T, reports []app.Report, kind app.ReportKind, outcome app.ReportOutcome) app.Report {
	t.Helper()
	for _, report := range reports {
		if report.Kind == kind && report.Outcome == outcome {
			return report
		}
	}
	t.Fatalf("missing %s/%s in %+v", kind, outcome, reports)
	return app.Report{}
}

func rejectReport(t *testing.T, reports []app.Report, kind app.ReportKind, outcome app.ReportOutcome) {
	t.Helper()
	for _, report := range reports {
		if report.Kind == kind && report.Outcome == outcome {
			t.Fatalf("unexpected %s/%s: %+v", kind, outcome, report)
		}
	}
}

func TestReportsRegistrationAndDestinationVerification(t *testing.T) {
	for _, scenario := range []string{"verified", "failed", "public-only"} {
		t.Run(scenario, func(t *testing.T) {
			home, external := prepareRegistration(t, scenario != "public-only")
			if scenario == "failed" {
				external.ssh = func(command app.Command) error {
					payload := command.Args[len(command.Args)-1]
					switch {
					case strings.Contains(payload, "KEYFLOW_PROBE"):
						fmt.Fprintln(command.Stdout, "KEYFLOW_TARGET:/home/me/.ssh/authorized_keys")
					case strings.Contains(payload, "KEYFLOW_REGISTER"):
						fmt.Fprintln(command.Stdout, "KEYFLOW_REGISTERED")
					case strings.Contains(payload, "KEYFLOW_VERIFIED"):
						return errors.New("fixture verification denied")
					default:
						t.Fatalf("unexpected command: %s", payload)
					}
					return nil
				}
			}
			var reports []app.Report
			err := app.Run(context.Background(), app.Options{Home: home, Input: strings.NewReader(""), Output: io.Discard, Runner: external,
				Request: &app.Request{Operation: "register", Host: "server", Key: "fixture", Bootstrap: "password", Configure: scenario == "failed"},
				Confirm: func(string) (bool, error) { return true, nil }, Report: func(report app.Report) { reports = append(reports, report) },
			})
			view, loadErr := app.Snapshot(app.Options{Home: home})
			if loadErr != nil || (err != nil) != (scenario == "failed") {
				t.Fatalf("run=%v snapshot=%v", err, loadErr)
			}
			registration := requireReport(t, reports, app.ReportRegistration, app.OutcomeApplied)
			saved := view.State.Registrations[0]
			if !registration.Saved || registration.RegistrationID != saved.ID || registration.Status != "registered" || saved.Status != "registered" {
				t.Fatalf("lost registration checkpoint: %+v %+v", registration, saved)
			}
			outcome := app.OutcomeVerified
			if scenario == "failed" {
				outcome = app.OutcomeFailed
			}
			if scenario == "public-only" {
				outcome = app.OutcomeUnavailable
			}
			verification := requireReport(t, reports, app.ReportVerification, outcome)
			if verification.HostID != saved.HostID || verification.KeyID != saved.KeyID || verification.RegistrationID != saved.ID {
				t.Fatalf("verification not bound to pair: %+v", verification)
			}
			if scenario == "failed" {
				requireReport(t, reports, app.ReportAlias, app.OutcomeSkipped)
				requireReport(t, reports, app.ReportInclude, app.OutcomeSkipped)
				if len(view.State.Configs) != 0 || saved.ConnectionCheck != "failed" {
					t.Fatalf("failed verification continued config: %+v", view.State)
				}
			}
			if scenario == "public-only" && saved.ConnectionCheck != "unverified" {
				t.Fatalf("invented private-key verification: %+v", saved)
			}
		})
	}
}

func TestReportsIndependentConfigurationApprovals(t *testing.T) {
	for _, scenario := range []string{"alias-declined", "include-declined", "include-failed", "complete"} {
		t.Run(scenario, func(t *testing.T) {
			home, external := prepareRegistration(t, true)
			original := "Host *\r\n  ServerAliveInterval 30\r\n"
			mainPath := writeFixture(t, home, ".ssh/config", original)
			var reports []app.Report
			err := app.Run(context.Background(), app.Options{Home: home, Input: strings.NewReader(""), Output: io.Discard, Runner: external,
				Request: &app.Request{Operation: "register", Host: "server", Key: "fixture", Bootstrap: "password", Configure: true},
				Confirm: func(label string) (bool, error) {
					if strings.Contains(label, "Generate SSH alias") && scenario == "alias-declined" {
						return false, nil
					}
					if strings.Contains(label, "exact backup") {
						if scenario == "include-declined" {
							return false, nil
						}
						if scenario == "include-failed" {
							writeFixture(t, home, ".ssh/config", original+"# external edit\r\n")
						}
					}
					return true, nil
				}, Report: func(report app.Report) { reports = append(reports, report) },
			})
			view, loadErr := app.Snapshot(app.Options{Home: home})
			if loadErr != nil || view.State.Registrations[0].Status != "registered" || view.State.Registrations[0].ConnectionCheck != "verified" {
				t.Fatalf("lost prior registration: %v %+v", loadErr, view.State)
			}
			requireReport(t, reports, app.ReportRegistration, app.OutcomeApplied)
			if scenario == "alias-declined" {
				if !errors.Is(err, app.ErrCancelled) || len(view.State.Configs) != 0 || view.State.Hosts[0].ConfigKeyID != "" {
					t.Fatalf("declined alias saved config: %v %+v", err, view.State)
				}
				requireReport(t, reports, app.ReportAlias, app.OutcomeDeclined)
				if requireReport(t, reports, app.ReportConfirmation, app.OutcomeDeclined).NoChanges {
					t.Fatal("partial cancellation claimed no changes")
				}
				requireReport(t, reports, app.ReportInclude, app.OutcomeSkipped)
				rejectReport(t, reports, app.ReportAlias, app.OutcomeApplied)
			} else {
				alias := requireReport(t, reports, app.ReportAlias, app.OutcomeApplied)
				if !alias.Saved || alias.Path == "" || len(view.State.Configs) != 1 || view.State.Configs[0].PendingSHA256 != "" || view.State.Hosts[0].ConfigKeyID != view.State.Keys[0].ID {
					t.Fatalf("alias checkpoint missing: %+v %+v", alias, view.State)
				}
				outcome := app.OutcomeApplied
				switch scenario {
				case "include-declined":
					outcome = app.OutcomeDeclined
					if !errors.Is(err, app.ErrCancelled) {
						t.Fatalf("decline error changed: %v", err)
					}
				case "include-failed":
					outcome = app.OutcomeRequiresReview
					if err == nil {
						t.Fatal("external edit was accepted")
					}
				case "complete":
					if err != nil || len(view.State.ConfigEdits) != 1 || !view.State.ConfigEdits[0].Complete {
						t.Fatalf("incomplete Include: %v %+v", err, view.State)
					}
				}
				include := requireReport(t, reports, app.ReportInclude, outcome)
				if include.Path != mainPath || include.Saved != (scenario == "complete") {
					t.Fatalf("wrong Include evidence: %+v", include)
				}
				if scenario != "complete" && len(view.State.ConfigEdits) != 0 {
					t.Fatalf("invented durable Include task: %+v", view.State.ConfigEdits)
				}
				if scenario != "complete" {
					rejectReport(t, reports, app.ReportInclude, app.OutcomeApplied)
				}
			}
			current, readErr := os.ReadFile(mainPath)
			if readErr != nil {
				t.Fatal(readErr)
			}
			expected := original
			if scenario == "complete" {
				expected = "Include \"ssh-keyflow/config\"\r\n" + original
			}
			if scenario == "include-failed" {
				expected += "# external edit\r\n"
			}
			if string(current) != expected {
				t.Fatalf("main config changed unexpectedly: %s", current)
			}
		})
	}
}

func TestReportsUnknownRemoteAndFailedRegistrationSave(t *testing.T) {
	for _, scenario := range []string{"remote-error", "missing-marker", "cancelled", "save-failed"} {
		t.Run(scenario, func(t *testing.T) {
			home, external := prepareRegistration(t, true)
			external.ssh = func(command app.Command) error {
				payload := command.Args[len(command.Args)-1]
				if strings.Contains(payload, "KEYFLOW_PROBE") {
					fmt.Fprintln(command.Stdout, "KEYFLOW_TARGET:/home/me/.ssh/authorized_keys")
					return nil
				}
				if !strings.Contains(payload, "KEYFLOW_REGISTER") {
					t.Fatalf("continued after unknown remote: %s", payload)
				}
				switch scenario {
				case "save-failed":
					blockStateSave(t, home)
					fmt.Fprintln(command.Stdout, "KEYFLOW_REGISTERED")
					return nil
				case "missing-marker":
					return nil
				case "cancelled":
					return context.Canceled
				default:
					return errors.New("fixture transport lost after write")
				}
			}
			var reports []app.Report
			err := app.Run(context.Background(), app.Options{Home: home, Input: strings.NewReader(""), Output: io.Discard, Runner: external,
				Request: &app.Request{Operation: "register", Host: "server", Key: "fixture", Bootstrap: "password", Configure: true}, Confirm: func(string) (bool, error) { return true, nil },
				Report: func(report app.Report) { reports = append(reports, report) },
			})
			if err == nil {
				t.Fatal("uncertain result returned success")
			}
			view, loadErr := app.Snapshot(app.Options{Home: home})
			if loadErr != nil || view.State.Registrations[0].Status != "unknown" {
				t.Fatalf("uncertainty not preserved: %v %+v", loadErr, view.State)
			}
			unknown := requireReport(t, reports, app.ReportRegistration, app.OutcomeUnknown)
			if unknown.Saved != (scenario != "save-failed") {
				t.Fatalf("failed save reported committed: %+v", unknown)
			}
			rejectReport(t, reports, app.ReportRegistration, app.OutcomeApplied)
			rejectReport(t, reports, app.ReportVerification, app.OutcomeVerified)
			requireReport(t, reports, app.ReportAlias, app.OutcomeSkipped)
			requireReport(t, reports, app.ReportInclude, app.OutcomeSkipped)
		})
	}
}

func blockStateSave(t *testing.T, home string) {
	t.Helper()
	path := filepath.Join(state.Directory(home), "state.json.bak")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
}

func TestReportsConfirmationDenialBeforeEffects(t *testing.T) {
	for _, scenario := range []string{"declined", "interrupted"} {
		t.Run(scenario, func(t *testing.T) {
			home := fixtureTempDir(t)
			public := writeFixture(t, home, "source.pub", publicKey)
			var reports []app.Report
			err := app.Run(context.Background(), app.Options{Home: home, Output: io.Discard, Runner: runner{},
				Request: &app.Request{Operation: "import", Name: "declined", PublicPath: public},
				Confirm: func(string) (bool, error) {
					if scenario == "interrupted" {
						return false, io.EOF
					}
					return false, nil
				},
				Report: func(report app.Report) { reports = append(reports, report) },
			})
			if err == nil {
				t.Fatal("confirmation failure returned success")
			}
			outcome := app.OutcomeDeclined
			if scenario == "interrupted" {
				outcome = app.OutcomeUnknown
			}
			confirmation := requireReport(t, reports, app.ReportConfirmation, outcome)
			if scenario == "declined" && !confirmation.NoChanges {
				t.Fatal("authoritative pre-effect cancellation not identified")
			}
			requireReport(t, reports, app.ReportExecution, app.OutcomeStopped)
			for _, report := range reports {
				if report.Saved {
					t.Fatalf("denial before effects reported saved: %+v", report)
				}
			}
			view, loadErr := app.Snapshot(app.Options{Home: home})
			if loadErr != nil || len(view.State.Keys) != 0 {
				t.Fatalf("denial changed state: %v %+v", loadErr, view.State)
			}
		})
	}
}

func TestReportsImportOnlyAfterPersistence(t *testing.T) {
	for _, scenario := range []string{"saved", "save-failed"} {
		t.Run(scenario, func(t *testing.T) {
			home := fixtureTempDir(t)
			public := writeFixture(t, home, "source.pub", publicKey)
			if scenario == "save-failed" {
				// Keep a valid existing state so the backup obstruction exercises Save.
				if err := state.CheckWritable(home); err != nil {
					t.Fatal(err)
				}
				if err := state.Update(home, func(*state.State) error { return nil }); err != nil {
					t.Fatal(err)
				}
				blockStateSave(t, home)
			}
			var reports []app.Report
			err := app.Run(context.Background(), app.Options{Home: home, Output: io.Discard, Runner: runner{},
				Request: &app.Request{Operation: "import", Name: "imported", PublicPath: public}, Confirm: func(string) (bool, error) { return true, nil },
				Report: func(report app.Report) { reports = append(reports, report) },
			})
			view, loadErr := app.Snapshot(app.Options{Home: home})
			if loadErr != nil {
				t.Fatal(loadErr)
			}
			if scenario == "saved" {
				report := requireReport(t, reports, app.ReportImport, app.OutcomeApplied)
				if err != nil || !report.Saved || len(view.State.Keys) != 1 || report.KeyID != view.State.Keys[0].ID {
					t.Fatalf("import report disagrees with saved state: %v %+v %+v", err, report, view.State)
				}
			} else {
				unknown := requireReport(t, reports, app.ReportImport, app.OutcomeUnknown)
				if err == nil || unknown.Saved || len(view.State.Keys) != 0 {
					t.Fatalf("failed import reported saved: %v %+v %+v", err, unknown, view.State)
				}
				rejectReport(t, reports, app.ReportImport, app.OutcomeApplied)
			}
		})
	}
}

func TestReportsCreationInterruptionAndResume(t *testing.T) {
	home := fixtureTempDir(t)
	external := onboardingRunner{testingContext: t, home: home, generate: func(command app.Command, path string) error {
		if err := os.WriteFile(path, []byte("private fixture"), 0600); err != nil {
			t.Fatal(err)
		}
		return context.Canceled
	}}
	var reports []app.Report
	options := app.Options{Home: home, Output: io.Discard, Runner: external, Request: &app.Request{Operation: "create", Name: "interrupted"}, Confirm: func(string) (bool, error) { return true, nil }, Report: func(report app.Report) { reports = append(reports, report) }}
	err := app.Run(context.Background(), options)
	view, loadErr := app.Snapshot(options)
	if !errors.Is(err, context.Canceled) || loadErr != nil || len(view.State.Keys) != 0 || len(view.State.Creations) != 1 || view.State.Creations[0].PrivateSHA256 == "" {
		t.Fatalf("partial creation evidence lost: %v %v %+v", err, loadErr, view.State)
	}
	pending := requireReport(t, reports, app.ReportCreation, app.OutcomePending)
	unknown := requireReport(t, reports, app.ReportCreation, app.OutcomeUnknown)
	creation := view.State.Creations[0]
	if !pending.Saved || pending.CreationID != creation.ID || unknown.Saved || unknown.KeyID != creation.KeyID {
		t.Fatalf("wrong interrupted identity: %+v %+v", pending, unknown)
	}
	rejectReport(t, reports, app.ReportCreation, app.OutcomeApplied)
	reports = nil
	options.Runner = onboardingRunner{testingContext: t, home: home, generate: func(app.Command, string) error { t.Fatal("resume generated over existing private file"); return nil }}
	options.Request = &app.Request{Operation: "resume", Resume: creation.ID}
	if err := app.Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	view, loadErr = app.Snapshot(options)
	complete := requireReport(t, reports, app.ReportCreation, app.OutcomeApplied)
	if loadErr != nil || !complete.Saved || complete.CreationID != creation.ID || len(view.State.Keys) != 1 || view.State.Keys[0].ID != creation.KeyID {
		t.Fatalf("resume report lost identity: %v %+v %+v", loadErr, complete, view.State)
	}
}

func TestReportsOnboardingSavedBeforeCreationCancellation(t *testing.T) {
	home := fixtureTempDir(t)
	var reports []app.Report
	options := app.Options{Home: home, Output: io.Discard, Runner: onboardingRunner{testingContext: t, home: home, generate: func(app.Command, string) error { return context.Canceled }},
		Request: &app.Request{Operation: "add", Name: "new", Destination: app.Destination{Alias: "new", Hostname: "new.example", User: "me", Port: 22, OS: "linux"}},
		Confirm: func(string) (bool, error) { return true, nil }, Report: func(report app.Report) { reports = append(reports, report) },
	}
	if err := app.Run(context.Background(), options); !errors.Is(err, context.Canceled) {
		t.Fatalf("unexpected error: %v", err)
	}
	view, err := app.Snapshot(options)
	if err != nil || len(view.State.Onboardings) != 1 || len(view.State.Hosts) != 1 {
		t.Fatalf("lost onboarding: %v %+v", err, view.State)
	}
	saved := requireReport(t, reports, app.ReportOnboarding, app.OutcomePending)
	if !saved.Saved || saved.HostID != view.State.Hosts[0].ID || saved.OnboardingID != view.State.Onboardings[0].ID || saved.CreationID != view.State.Creations[0].ID {
		t.Fatalf("unlinked checkpoint: %+v", saved)
	}
}

func TestReportsUnregisterManualRemovalAndUnknown(t *testing.T) {
	for _, scenario := range []string{"manual", "removed", "already-absent", "unknown", "final-declined"} {
		t.Run(scenario, func(t *testing.T) {
			home, external := prepareRegistration(t, true)
			before, _ := app.Snapshot(app.Options{Home: home})
			writes := 0
			external.ssh = func(command app.Command) error {
				payload := command.Args[len(command.Args)-1]
				switch {
				case strings.Contains(payload, "KEYFLOW_ALTERNATE"):
					fmt.Fprintln(command.Stdout, "KEYFLOW_TARGET:/home/me/.ssh/authorized_keys")
				case strings.Contains(payload, "KEYFLOW_REVOCATION_READ"):
					presence := "KEYFLOW_PRESENT"
					if scenario == "already-absent" {
						presence = "KEYFLOW_ABSENT"
					}
					fmt.Fprintln(command.Stdout, presence)
				case strings.Contains(payload, "KEYFLOW_UNREGISTER"):
					writes++
					if scenario == "unknown" {
						return context.Canceled
					}
					fmt.Fprintln(command.Stdout, "KEYFLOW_REMOVED")
				default:
					t.Fatalf("unexpected SSH payload: %s", payload)
				}
				return nil
			}
			mode := "password"
			if scenario == "manual" {
				mode = "manual"
			}
			var reports []app.Report
			err := app.Run(context.Background(), app.Options{Home: home, Input: strings.NewReader(""), Output: io.Discard, Runner: external,
				Request: &app.Request{Operation: "unregister", Registration: before.State.Registrations[0].ID, Bootstrap: mode},
				Confirm: func(label string) (bool, error) {
					return !(scenario == "final-declined" && strings.Contains(label, "Final confirmation")), nil
				},
				Report: func(report app.Report) { reports = append(reports, report) },
			})
			view, loadErr := app.Snapshot(app.Options{Home: home})
			if loadErr != nil {
				t.Fatal(loadErr)
			}
			if scenario == "final-declined" {
				if !errors.Is(err, app.ErrCancelled) || writes != 0 {
					t.Fatalf("unapproved write: %v writes=%d", err, writes)
				}
				requireReport(t, reports, app.ReportConfirmation, app.OutcomeDeclined)
				requireReport(t, reports, app.ReportRegistration, app.OutcomePending)
				rejectReport(t, reports, app.ReportRegistration, app.OutcomeApplied)
			} else if scenario == "manual" || scenario == "unknown" {
				if (err != nil) != (scenario == "unknown") {
					t.Fatalf("error compatibility changed: %v", err)
				}
				manual := requireReport(t, reports, app.ReportManual, app.OutcomeRequired)
				if manual.Operation != "unregister" || manual.RegistrationID != view.State.Registrations[0].ID || manual.HostID != view.State.Hosts[0].ID {
					t.Fatalf("manual target missing: %+v", manual)
				}
				requireReport(t, reports, app.ReportRegistration, app.OutcomeUnknown)
				if view.State.Registrations[0].Operation != "unregister" || view.State.Registrations[0].Stage != "unknown" {
					t.Fatalf("manual removal falsely complete: %+v", view.State)
				}
				if scenario == "manual" && writes != 0 {
					t.Fatal("manual contacted SSH")
				}
			} else {
				complete := requireReport(t, reports, app.ReportRegistration, app.OutcomeApplied)
				if err != nil || !complete.Saved || complete.Status != "removed" || complete.Operation != "unregister" || view.State.Registrations[0].Status != "removed" {
					t.Fatalf("removal report disagrees: %v %+v %+v", err, complete, view.State)
				}
				if scenario == "already-absent" && writes != 0 {
					t.Fatal("absent key retried mutation")
				}
			}
			for _, path := range []string{before.State.Keys[0].PublicPath, before.State.Keys[0].PrivatePath} {
				if _, err := os.Stat(path); err != nil {
					t.Fatalf("unregister removed local file: %v", err)
				}
			}
		})
	}
}

func TestReportsDeletionCheckpointsAndPartialFailure(t *testing.T) {
	for _, scenario := range []string{"complete", "save-after-removal-failed"} {
		t.Run(scenario, func(t *testing.T) {
			home := fixtureTempDir(t)
			options := app.Options{Home: home, Output: io.Discard, Runner: onboardingRunner{testingContext: t, home: home}, Confirm: func(string) (bool, error) { return true, nil }, Request: &app.Request{Operation: "create", Name: "local"}}
			if err := app.Run(context.Background(), options); err != nil {
				t.Fatal(err)
			}
			before, _ := app.Snapshot(options)
			key := before.State.Keys[0]
			if scenario != "complete" {
				if err := state.Update(home, func(document *state.State) error {
					document.Deletions = []state.LocalDeletion{{ID: "deletion-fixture", KeyID: key.ID, Stage: "pending", CreatedAt: "2026-11-07T00:00:00Z", PublicSHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(publicKey))), PrivateSHA256: key.PrivateSHA256}}
					return nil
				}); err != nil {
					t.Fatal(err)
				}
				options.Confirm = func(string) (bool, error) { blockStateSave(t, home); return true, nil }
			}
			var reports []app.Report
			options.Report = func(report app.Report) { reports = append(reports, report) }
			options.Request = &app.Request{Operation: "delete", Key: key.ID}
			err := app.Run(context.Background(), options)
			view, loadErr := app.Snapshot(options)
			if loadErr != nil {
				t.Fatal(loadErr)
			}
			if scenario == "complete" {
				complete := requireReport(t, reports, app.ReportDeletion, app.OutcomeApplied)
				pending := requireReport(t, reports, app.ReportDeletion, app.OutcomePending)
				if err != nil || !complete.Saved || !pending.Saved || complete.KeyID != key.ID || complete.DeletionID != view.State.Deletions[0].ID || len(view.State.Keys) != 0 || view.State.DeletedKeys[0].Key.ID != key.ID {
					t.Fatalf("deletion facts lost tombstone: %v %+v %+v", err, complete, view.State)
				}
			} else {
				unknown := requireReport(t, reports, app.ReportDeletion, app.OutcomeUnknown)
				if err == nil || unknown.Saved || unknown.DeletionID != "deletion-fixture" || len(view.State.Keys) != 1 {
					t.Fatalf("partial deletion falsely complete: %v %+v %+v", err, unknown, view.State)
				}
				if _, err := os.Stat(key.PublicPath); !os.IsNotExist(err) {
					t.Fatalf("fixture did not exercise file removal before failed save: %v", err)
				}
				if _, err := os.Stat(key.PrivatePath); err != nil {
					t.Fatalf("continued removal after failed checkpoint: %v", err)
				}
				rejectReport(t, reports, app.ReportDeletion, app.OutcomeApplied)
			}
		})
	}
}

func TestReportsOnboardingManualIntentAndReloadFailure(t *testing.T) {
	home := fixtureTempDir(t)
	public := writeFixture(t, home, "key.pub", publicKey)
	options := app.Options{Home: home, Output: io.Discard, Runner: runner{}, Request: &app.Request{Operation: "import", Name: "reuse", PublicPath: public}, Confirm: func(string) (bool, error) { return true, nil }}
	if err := app.Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	var reports []app.Report
	options.Report = func(report app.Report) { reports = append(reports, report) }
	options.Request = &app.Request{Operation: "add", Key: "reuse", Destination: app.Destination{Alias: "new", Hostname: "new.example", User: "me", Port: 22, OS: "linux"}, Bootstrap: "manual"}
	if err := app.Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	view, err := app.Snapshot(options)
	if err != nil {
		t.Fatal(err)
	}
	saved := requireReport(t, reports, app.ReportOnboarding, app.OutcomePending)
	manual := requireReport(t, reports, app.ReportManual, app.OutcomeRequired)
	if !saved.Saved || saved.HostID != view.State.Hosts[0].ID || manual.RegistrationID != view.State.Registrations[0].ID || manual.OnboardingID != view.State.Onboardings[0].ID {
		t.Fatalf("lost saved/manual distinction: %+v %+v %+v", saved, manual, view.State)
	}
	snapshotReports := append([]app.Report(nil), reports...)
	writeFixture(t, home, ".ssh/ssh-keyflow/state.json", "{")
	if _, err := app.Snapshot(options); err == nil {
		t.Fatal("reload failure was hidden")
	}
	if !reflect.DeepEqual(reports, snapshotReports) {
		t.Fatal("Snapshot replayed or changed operation reports")
	}
}

func TestReportsConfigurationRemovalAfterUnregister(t *testing.T) {
	home, external := prepareRegistration(t, true)
	options := app.Options{Home: home, Input: strings.NewReader(""), Output: io.Discard, Runner: external, Confirm: func(string) (bool, error) { return true, nil }, Request: &app.Request{Operation: "register", Host: "server", Key: "fixture", Bootstrap: "password", Configure: true}}
	if err := app.Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	before, _ := app.Snapshot(options)
	mainPath := filepath.Join(home, ".ssh", "config")
	original, _ := os.ReadFile(mainPath)
	external.ssh = func(command app.Command) error {
		payload := command.Args[len(command.Args)-1]
		switch {
		case strings.Contains(payload, "KEYFLOW_ALTERNATE"):
			fmt.Fprintln(command.Stdout, "KEYFLOW_TARGET:/home/me/.ssh/authorized_keys")
		case strings.Contains(payload, "KEYFLOW_REVOCATION_READ"):
			fmt.Fprintln(command.Stdout, "KEYFLOW_PRESENT")
		case strings.Contains(payload, "KEYFLOW_UNREGISTER"):
			fmt.Fprintln(command.Stdout, "KEYFLOW_REMOVED")
		default:
			t.Fatalf("unexpected SSH: %s", payload)
		}
		return nil
	}
	options.Runner = external
	var reports []app.Report
	options.Report = func(report app.Report) { reports = append(reports, report) }
	options.Request = &app.Request{Operation: "unregister", Registration: before.State.Registrations[0].ID, Bootstrap: "password"}
	if err := app.Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	alias := requireReport(t, reports, app.ReportAlias, app.OutcomeApplied)
	if !alias.Saved || alias.Operation != "remove" || alias.KeyID != before.State.Keys[0].ID {
		t.Fatalf("configuration removal not reported: %+v", alias)
	}
	after, err := app.Snapshot(options)
	actual, _ := os.ReadFile(mainPath)
	if err != nil || after.State.Hosts[0].ConfigKeyID != "" || string(actual) != string(original) || !reflect.DeepEqual(before.State.ConfigEdits, after.State.ConfigEdits) {
		t.Fatalf("removal damaged Include: %v %+v", err, after.State)
	}
}

func TestReportsRegistrationDenialKeepsResolvedTarget(t *testing.T) {
	home, external := prepareRegistration(t, true)
	before, err := app.Snapshot(app.Options{Home: home})
	if err != nil {
		t.Fatal(err)
	}
	var reports []app.Report
	external.ssh = func(app.Command) error { t.Fatal("denial contacted SSH"); return nil }
	err = app.Run(context.Background(), app.Options{Home: home, Output: io.Discard, Runner: external,
		Request: &app.Request{Operation: "register", Host: "server", Key: "fixture", Bootstrap: "password"},
		Confirm: func(string) (bool, error) { return false, nil }, Report: func(report app.Report) { reports = append(reports, report) },
	})
	denial := requireReport(t, reports, app.ReportConfirmation, app.OutcomeDeclined)
	stopped := requireReport(t, reports, app.ReportExecution, app.OutcomeStopped)
	if !errors.Is(err, app.ErrCancelled) || !denial.NoChanges || denial.HostID != before.State.Hosts[0].ID || denial.KeyID != before.State.Keys[0].ID || stopped.ReportTarget != denial.ReportTarget {
		t.Fatalf("denial lost target or claimed effects: %v %+v %+v", err, denial, stopped)
	}
	after, err := app.Snapshot(app.Options{Home: home})
	if err != nil || !reflect.DeepEqual(before.State, after.State) {
		t.Fatalf("denial changed evidence: %v", err)
	}
}

func TestReportsOnboardingResumePreservesCompletedRegistrationEvidence(t *testing.T) {
	home, external := prepareRegistration(t, true)
	options := app.Options{Home: home, Input: strings.NewReader(""), Output: io.Discard, Runner: external, Confirm: func(string) (bool, error) { return true, nil }, Request: &app.Request{Operation: "register", Host: "server", Key: "fixture", Bootstrap: "password"}}
	if err := app.Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	before, err := app.Snapshot(options)
	if err != nil {
		t.Fatal(err)
	}
	var reports []app.Report
	options.Report = func(report app.Report) { reports = append(reports, report) }
	options.Request = &app.Request{Operation: "resume", Resume: before.State.Onboardings[0].ID, Bootstrap: "manual"}
	if err := app.Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	after, err := app.Snapshot(options)
	if err != nil || !reflect.DeepEqual(before.State.Registrations, after.State.Registrations) {
		t.Fatalf("resume changed saved registration: %v", err)
	}
	requireReport(t, reports, app.ReportManual, app.OutcomeRequired)
	requireReport(t, reports, app.ReportVerification, app.OutcomeNotChecked)
	rejectReport(t, reports, app.ReportRegistration, app.OutcomePending)
	rejectReport(t, reports, app.ReportRegistration, app.OutcomeApplied)
	rejectReport(t, reports, app.ReportVerification, app.OutcomeVerified)
}

func TestReportsResumeDenialKeepsSavedIdentity(t *testing.T) {
	for _, kind := range []string{"creation", "onboarding"} {
		t.Run(kind, func(t *testing.T) {
			home := fixtureTempDir(t)
			var external app.Runner = onboardingRunner{testingContext: t, home: home, generate: func(app.Command, string) error { return context.Canceled }}
			options := app.Options{Home: home, Input: strings.NewReader(""), Output: io.Discard, Runner: external, Confirm: func(string) (bool, error) { return true, nil }, Request: &app.Request{Operation: "create", Name: "pending"}}
			if kind == "onboarding" {
				home, external = prepareRegistration(t, true)
				options.Home, options.Runner = home, external
			} else if err := app.Run(context.Background(), options); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			before, err := app.Snapshot(options)
			if err != nil {
				t.Fatal(err)
			}
			id, keyID, hostID := "", "", ""
			if kind == "creation" {
				id, keyID = before.State.Creations[0].ID, before.State.Creations[0].KeyID
			} else {
				id, keyID, hostID = before.State.Onboardings[0].ID, before.State.Onboardings[0].KeyID, before.State.Onboardings[0].HostID
			}
			var reports []app.Report
			options.Request = &app.Request{Operation: "resume", Resume: id}
			options.Confirm = func(string) (bool, error) { return false, nil }
			options.Report = func(report app.Report) { reports = append(reports, report) }
			err = app.Run(context.Background(), options)
			denial := requireReport(t, reports, app.ReportConfirmation, app.OutcomeDeclined)
			if !errors.Is(err, app.ErrCancelled) || !denial.NoChanges || denial.KeyID != keyID || denial.HostID != hostID || (kind == "creation" && denial.CreationID != id) || (kind == "onboarding" && denial.OnboardingID != id) {
				t.Fatalf("lost saved resume identity: %v %+v", err, denial)
			}
			after, err := app.Snapshot(options)
			if err != nil || !reflect.DeepEqual(before.State, after.State) {
				t.Fatalf("denial changed evidence: %v", err)
			}
		})
	}
}

func TestReportsIncludeRecoveryAndLateFailure(t *testing.T) {
	for _, scenario := range []string{"existing-top-level", "scoped", "after-backup", "after-config", "late-failure"} {
		t.Run(scenario, func(t *testing.T) {
			home, external := prepareRegistration(t, true)
			original := "Host old\n HostName old.example\n"
			path := writeFixture(t, home, ".ssh/config", original)
			options := app.Options{Home: home, Input: strings.NewReader(""), Output: io.Discard, Runner: external, Confirm: func(string) (bool, error) { return true, nil }, Request: &app.Request{Operation: "register", Host: "server", Key: "fixture", Bootstrap: "password", Configure: true}}
			if err := app.Run(context.Background(), options); err != nil {
				t.Fatal(err)
			}
			before, err := app.Snapshot(options)
			if err != nil {
				t.Fatal(err)
			}
			edit := before.State.ConfigEdits[0]
			if scenario == "scoped" {
				writeFixture(t, home, ".ssh/config", "Host other\n Include ssh-keyflow/config\n")
			} else if scenario == "after-backup" || scenario == "late-failure" {
				writeFixture(t, home, ".ssh/config", original)
			}
			if strings.HasPrefix(scenario, "after-") || scenario == "late-failure" {
				if err := state.Update(home, func(document *state.State) error { document.ConfigEdits[0].Complete = false; return nil }); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "late-failure" {
				if err := os.Remove(edit.BackupPath); err != nil {
					t.Fatal(err)
				}
				options.Confirm = func(label string) (bool, error) {
					if strings.Contains(label, "exact backup") {
						if err := os.Mkdir(edit.BackupPath, 0700); err != nil {
							t.Fatal(err)
						}
					}
					return true, nil
				}
			}
			var reports []app.Report
			options.Report = func(report app.Report) { reports = append(reports, report) }
			err = app.Run(context.Background(), options)
			requireReport(t, reports, app.ReportRegistration, app.OutcomeApplied)
			requireReport(t, reports, app.ReportVerification, app.OutcomeVerified)
			requireReport(t, reports, app.ReportAlias, app.OutcomeApplied)
			expected, saved := app.OutcomeReconciled, false
			switch scenario {
			case "scoped":
				expected = app.OutcomeRequiresReview
			case "after-backup":
				expected, saved = app.OutcomeApplied, true
			case "after-config":
				saved = true
			case "late-failure":
				expected = app.OutcomeUnknown
			}
			report := requireReport(t, reports, app.ReportInclude, expected)
			if report.Path != path || report.Saved != saved || (err != nil) != (scenario == "late-failure") {
				t.Fatalf("wrong recovery fact: %v %+v", err, report)
			}
			after, loadErr := app.Snapshot(options)
			if loadErr != nil {
				t.Fatal(loadErr)
			}
			actual, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			want := "Include \"ssh-keyflow/config\"\n" + original
			if scenario == "scoped" {
				want = "Host other\n Include ssh-keyflow/config\n"
			}
			if scenario == "late-failure" {
				want = original
				pending := requireReport(t, reports, app.ReportInclude, app.OutcomePending)
				if !pending.Saved || after.State.ConfigEdits[0].Complete {
					t.Fatalf("lost pending Include journal: %+v %+v", pending, after.State.ConfigEdits)
				}
			} else if !after.State.ConfigEdits[0].Complete {
				t.Fatal("successful recovery left journal pending")
			}
			if string(actual) != want {
				t.Fatalf("unexpected Include bytes: %q", actual)
			}
			if expected != app.OutcomeApplied {
				rejectReport(t, reports, app.ReportInclude, app.OutcomeApplied)
			}
		})
	}
}

func TestReportsAliasFailedSavePreservesRegistration(t *testing.T) {
	home, external := prepareRegistration(t, true)
	var reports []app.Report
	err := app.Run(context.Background(), app.Options{Home: home, Input: strings.NewReader(""), Output: io.Discard, Runner: external, Request: &app.Request{Operation: "register", Host: "server", Key: "fixture", Bootstrap: "password", Configure: true},
		Confirm: func(label string) (bool, error) {
			if strings.Contains(label, "Generate SSH alias") {
				blockStateSave(t, home)
			}
			return true, nil
		}, Report: func(report app.Report) { reports = append(reports, report) },
	})
	if err == nil {
		t.Fatal("alias save unexpectedly succeeded")
	}
	requireReport(t, reports, app.ReportRegistration, app.OutcomeApplied)
	requireReport(t, reports, app.ReportVerification, app.OutcomeVerified)
	unknown := requireReport(t, reports, app.ReportAlias, app.OutcomeUnknown)
	if unknown.Saved {
		t.Fatalf("failed save claimed committed alias: %+v", unknown)
	}
	rejectReport(t, reports, app.ReportAlias, app.OutcomeApplied)
	requireReport(t, reports, app.ReportInclude, app.OutcomeSkipped)
	view, loadErr := app.Snapshot(app.Options{Home: home})
	if loadErr != nil || view.State.Registrations[0].ConnectionCheck != "verified" || view.State.Hosts[0].ConfigKeyID != "" || len(view.State.Configs) != 0 {
		t.Fatalf("lost saved registration or fabricated config: %v %+v", loadErr, view.State)
	}
}

func TestReportsDoNotChangeOutputApprovalsOrCommandIntent(t *testing.T) {
	for _, scenario := range []string{"registered", "manual", "denied", "alias-denied"} {
		t.Run(scenario, func(t *testing.T) {
			home, external := prepareRegistration(t, true)
			filename := filepath.Join(state.Directory(home), "state.json")
			initial, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			var outputs, results []string
			var approvals, commands [][]string
			var registrations [][]state.Registration
			for _, enabled := range []bool{false, true} {
				if err := os.WriteFile(filename, initial, 0600); err != nil {
					t.Fatal(err)
				}
				var output bytes.Buffer
				var labels, intents []string
				recording := external
				recording.ssh = func(command app.Command) error {
					intents = append(intents, strings.Join(command.Args, "\n"))
					return external.Run(context.Background(), command)
				}
				mode := "password"
				if scenario == "manual" {
					mode = "manual"
				}
				options := app.Options{Home: home, Input: strings.NewReader(""), Output: &output, Runner: recording, Request: &app.Request{Operation: "register", Host: "server", Key: "fixture", Bootstrap: mode, Configure: scenario == "alias-denied"}, Confirm: func(label string) (bool, error) {
					labels = append(labels, label)
					return scenario != "denied" && !(scenario == "alias-denied" && strings.Contains(label, "Generate SSH alias")), nil
				}}
				if enabled {
					options.Report = func(app.Report) {}
				}
				err := app.Run(context.Background(), options)
				result := ""
				if err != nil {
					result = err.Error()
				}
				view, loadErr := app.Snapshot(options)
				if loadErr != nil {
					t.Fatal(loadErr)
				}
				outputs = append(outputs, output.String())
				results = append(results, result)
				approvals = append(approvals, labels)
				commands = append(commands, intents)
				registrations = append(registrations, view.State.Registrations)
			}
			if outputs[0] != outputs[1] || results[0] != results[1] || !reflect.DeepEqual(approvals[0], approvals[1]) || !reflect.DeepEqual(commands[0], commands[1]) || !reflect.DeepEqual(registrations[0], registrations[1]) {
				t.Fatalf("sink altered operation behavior: results=%v approvals=%v commands=%v", results, approvals, commands)
			}
		})
	}
}

func TestReportsFailedNewRegistrationSaveRetainsAllocatedTarget(t *testing.T) {
	home, external := prepareRegistration(t, true)
	if err := state.Update(home, func(document *state.State) error {
		document.Registrations = []state.Registration{}
		document.Onboardings = nil
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	external.ssh = func(command app.Command) error {
		payload := command.Args[len(command.Args)-1]
		if strings.Contains(payload, "KEYFLOW_PROBE") {
			fmt.Fprintln(command.Stdout, "KEYFLOW_TARGET:/home/me/.ssh/authorized_keys")
			return nil
		}
		if !strings.Contains(payload, "KEYFLOW_REGISTER") {
			t.Fatalf("unexpected command: %s", payload)
		}
		blockStateSave(t, home)
		fmt.Fprintln(command.Stdout, "KEYFLOW_REGISTERED")
		return nil
	}
	var reports []app.Report
	err := app.Run(context.Background(), app.Options{Home: home, Input: strings.NewReader(""), Output: io.Discard, Runner: external, Request: &app.Request{Operation: "register", Host: "server", Key: "fixture", Bootstrap: "password"}, Confirm: func(string) (bool, error) { return true, nil }, Report: func(report app.Report) { reports = append(reports, report) }})
	unknown := requireReport(t, reports, app.ReportRegistration, app.OutcomeUnknown)
	stopped := requireReport(t, reports, app.ReportExecution, app.OutcomeStopped)
	if err == nil || unknown.Saved || unknown.RegistrationID == "" || stopped.ReportTarget != unknown.ReportTarget {
		t.Fatalf("lost allocated uncertain target: %v %+v %+v", err, unknown, stopped)
	}
}
