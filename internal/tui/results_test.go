package tui_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/state"
	"ssh-keyflow/internal/tui"
)

const resultPublic = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB fixture"

// Only the process boundary is fake. Persistence, reports, confirmations and
// result navigation use the production app.Run and NewProgram boundaries.
type resultRunner struct {
	calls   atomic.Int32
	failure string
}

func (r *resultRunner) LookPath(name string) (string, error) { return name, nil }
func (r *resultRunner) Run(_ context.Context, c app.Command) error {
	r.calls.Add(1)
	if c.Name == "ssh-keygen" {
		fmt.Fprintln(c.Stdout, resultPublic)
		return nil
	}
	if c.Name != "ssh" || !c.Interactive || c.Stdin == nil {
		return errors.New("unexpected external command")
	}
	payload := c.Args[len(c.Args)-1]
	switch {
	case strings.Contains(payload, "KEYFLOW_PROBE"):
		fmt.Fprintln(c.Stdout, "KEYFLOW_TARGET:/home/me/.ssh/authorized_keys")
	case strings.Contains(payload, "KEYFLOW_RECONCILE"):
		fmt.Fprintln(c.Stdout, "KEYFLOW_PRESENT")
	case strings.Contains(payload, "KEYFLOW_VERIFIED"):
		if r.failure == "verify" {
			return errors.New("fixture verification denied")
		}
		fmt.Fprintln(c.Stdout, "KEYFLOW_VERIFIED")
	case strings.Contains(payload, "KEYFLOW_REGISTER"):
		if r.failure == "remote" {
			return errors.New("fixture transport lost after write")
		}
		fmt.Fprintln(c.Stdout, "KEYFLOW_REGISTERED")
	default:
		return errors.New("unexpected fixture SSH protocol")
	}
	return nil
}
func resultHome(t *testing.T, private bool) (string, *resultRunner) {
	t.Helper()
	home := fixtureTempDir(t)
	public := filepath.Join(home, "fixture.pub")
	companion := ""
	if err := os.WriteFile(public, []byte(resultPublic), 0600); err != nil {
		t.Fatal(err)
	}
	if private {
		companion = filepath.Join(home, "fixture")
		if err := os.WriteFile(companion, []byte("private fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	runner := &resultRunner{}
	options := app.Options{Home: home, Output: io.Discard, Runner: runner, Confirm: func(string) (bool, error) { return true, nil }}
	options.Request = &app.Request{Operation: "import", Name: "Result fixture", PublicPath: public, PrivatePath: companion}
	if err := app.Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	options.Request = &app.Request{Operation: "add", Key: "Result fixture", Bootstrap: "manual", Destination: app.Destination{Alias: "result", Hostname: "example.test", User: "me", Port: 2222, OS: "linux"}}
	if err := app.Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	return home, runner
}
func automaticRegister(s handoffSession, configure bool) {
	s.send(press('r'), press(tea.KeyEnter), press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyRight))
	if configure {
		s.send(press(tea.KeyTab), press(tea.KeyRight))
	}
	s.send(reviewKey(), press(tea.KeyEnter))
}
func allResultText(t *testing.T, s handoffSession) string {
	t.Helper()
	s.send(press(tea.KeyHome))
	var text strings.Builder
	for i := 0; i < 32; i++ {
		text.WriteString(s.screen(t))
		s.send(press(tea.KeyDown), press(tea.KeyDown), press(tea.KeyDown))
	}
	return strings.Join(strings.Fields(text.String()), " ")
}
func TestResultRemoteUncertaintyKeepsStoppedSummaryAndExactContinuation(t *testing.T) {
	home, runner := resultHome(t, true)
	runner.failure = "remote"
	native := &handoffIO{answer: strings.NewReader("yes\nyes\n")}
	session := startHandoff(t, app.Options{Home: home, Runner: runner, Input: native, Output: native}, nil)
	automaticRegister(session, false)
	screen := session.returned(t)
	for _, want := range []string{"Operation stopped", "outcome unknown", "manual action required"} {
		if !strings.Contains(screen, want) {
			t.Fatalf("uncertain remote result hid %q:\n%s", want, screen)
		}
	}
	view, err := app.Snapshot(app.Options{Home: home})
	if err != nil {
		t.Fatal(err)
	}
	if view.State.Registrations[0].Status != "unknown" {
		t.Fatal("lost uncertainty")
	}
	calls := runner.calls.Load()
	session.send(press('t'))
	if !strings.Contains(session.screen(t), "reconcile remote state before retry") {
		t.Fatal("task lacks reconciliation guidance")
	}
	session.send(press('s'), reviewKey())
	screen = session.screen(t)
	if !strings.Contains(screen, view.State.Registrations[0].ID) || !strings.Contains(screen, "Review resume") {
		t.Fatalf("wrong saved continuation:\n%s", screen)
	}
	if runner.calls.Load() != calls {
		t.Fatal("navigation blindly retried SSH")
	}
}

func TestResultManualVerifiedPairPreservesPriorEvidenceAfterRestart(t *testing.T) {
	home, runner := resultHome(t, true)
	options := app.Options{Home: home, Runner: runner, Output: io.Discard, Input: strings.NewReader(""), Confirm: func(string) (bool, error) { return true, nil }, Request: &app.Request{Operation: "register", Host: "result", Key: "Result fixture", Bootstrap: "password"}}
	if err := app.Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	before, err := app.Snapshot(options)
	if err != nil {
		t.Fatal(err)
	}
	native := &handoffIO{answer: strings.NewReader("yes\n")}
	session := startHandoff(t, app.Options{Home: home, Runner: runner, Input: native, Output: native}, nil)
	session.send(press('r'), press(tea.KeyEnter), reviewKey(), press(tea.KeyEnter))
	screen := session.returned(t)
	if !strings.Contains(screen, "Manual action required") || !strings.Contains(screen, "No saved recovery task") {
		t.Fatalf("manual existing-pair result:\n%s", screen)
	}
	full := allResultText(t, session)
	for _, want := range []string{"verification: not-checked", "Connection: verified", "Register again after exit/restart", "ssh-ed25519"} {
		if !strings.Contains(full, want) {
			t.Fatalf("missing separate session/saved evidence %q", want)
		}
	}
	after, err := app.Snapshot(options)
	if err != nil || !reflect.DeepEqual(before.State, after.State) {
		t.Fatalf("manual changed prior verified state: %v", err)
	}
	restart := runScreen(t, home, press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab))
	if !strings.Contains(restart, "No recorded follow-up") {
		t.Fatalf("stale onboarding became task:\n%s", restart)
	}
	session.send(press(tea.KeyHome), press('v'), press('r'), reviewKey())
	review := session.screen(t)
	for _, id := range []string{before.State.Hosts[0].ID, before.State.Keys[0].ID} {
		if !strings.Contains(review, id) {
			t.Fatalf("result link lost exact target %s:\n%s", id, review)
		}
	}
}

func TestResultAutomaticRegistrationOutcomeMatrix(t *testing.T) {
	for _, scenario := range []string{"verified", "public-only", "verify-failed", "alias-declined", "include-declined", "include-failed", "configured"} {
		t.Run(scenario, func(t *testing.T) {
			home, runner := resultHome(t, scenario != "public-only")
			if scenario == "verify-failed" {
				runner.failure = "verify"
			}
			config := scenario != "verified"
			answers := "yes\nyes\nyes\nyes\n"
			if scenario == "verified" || scenario == "verify-failed" {
				answers = "yes\nyes\n"
			}
			if scenario == "public-only" {
				answers = "yes\nyes\nyes\n"
			}
			if scenario == "alias-declined" {
				answers = "yes\nyes\nno\n"
			}
			if scenario == "include-declined" {
				answers = "yes\nyes\nyes\nno\n"
			}
			mainPath := filepath.Join(home, ".ssh", "config")
			original := "Host *\r\n  ServerAliveInterval 30\r\n"
			if err := os.WriteFile(mainPath, []byte(original), 0600); err != nil {
				t.Fatal(err)
			}
			native := &handoffIO{answer: strings.NewReader(answers)}
			if scenario == "include-failed" {
				native.onPrompt = func(output string) {
					if strings.Contains(output, "exact backup") {
						if err := os.WriteFile(mainPath, []byte(original+"# external edit\r\n"), 0600); err != nil {
							t.Error(err)
						}
					}
				}
			}
			session := startHandoff(t, app.Options{Home: home, Runner: runner, Input: native, Output: native}, nil)
			automaticRegister(session, config)
			screen := session.returned(t)
			summary := "Completed; confirmed applicable work recorded"
			switch scenario {
			case "verify-failed":
				summary = "Registered; connection check failed"
			case "alias-declined":
				summary = "Stopped after registration; managed alias not generated this time"
			case "include-declined", "include-failed":
				summary = "Registration and managed alias saved; Include setup remains"
			}
			if !strings.Contains(screen, summary) {
				t.Fatalf("wrong %s summary:\n%s", scenario, screen)
			}
			full := allResultText(t, session)
			wants := []string{"registration: applied (saved checkpoint)"}
			switch scenario {
			case "public-only":
				wants = append(wants, "verification: unavailable", "alias: skipped", "include: skipped", "Private-key verification unavailable")
			case "verify-failed":
				wants = append(wants, "verification: failed", "alias: skipped", "include: skipped")
			case "alias-declined":
				wants = append(wants, "alias: declined", "prior bindings/files", "include: skipped")
			case "include-declined":
				wants = append(wants, "alias: applied (saved checkpoint)", "include: declined", "Main config unchanged by this request", "No saved recovery task", "configuration requested")
			case "include-failed":
				wants = append(wants, "alias: applied (saved checkpoint)", "include: requires-review", "Error:")
			case "configured":
				wants = append(wants, "alias: applied (saved checkpoint)", "include: applied (saved checkpoint)")
			}
			for _, want := range wants {
				if !strings.Contains(full, want) {
					t.Fatalf("%s omitted %q", scenario, want)
				}
			}
			view, err := app.Snapshot(app.Options{Home: home})
			if err != nil {
				t.Fatal(err)
			}
			if view.State.Registrations[0].Status != "registered" {
				t.Fatal("lost saved registration")
			}
			if scenario == "include-declined" || scenario == "include-failed" {
				if len(view.State.Configs) != 1 || len(view.State.ConfigEdits) != 0 || view.State.Hosts[0].ConfigKeyID != view.State.Keys[0].ID {
					t.Fatal("missing alias or fabricated Include intent")
				}
			}
			expected := original
			if scenario == "include-failed" {
				expected += "# external edit\r\n"
			}
			if scenario == "configured" {
				expected = "Include \"ssh-keyflow/config\"\r\n" + original
			}
			current, err := os.ReadFile(mainPath)
			if err != nil || string(current) != expected {
				t.Fatal("wrong main config effects", err, string(current))
			}
			calls := runner.calls.Load()
			session.send(press(tea.KeyHome), press('v'), press('v'))
			if !strings.Contains(session.screen(t), view.State.Registrations[0].ID) {
				t.Fatal("result entity link has wrong target")
			}
			if scenario == "verify-failed" {
				session.send(press(tea.KeyEscape), press(tea.KeyEscape), press('s'), reviewKey())
				review := session.screen(t)
				if !strings.Contains(review, "Review resume") || !strings.Contains(review, view.State.Registrations[0].ID) {
					t.Fatalf("wrong verification follow-up:\n%s", review)
				}
			} else {
				restart := runScreen(t, home, press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab))
				if !strings.Contains(restart, "No recorded follow-up") {
					t.Fatalf("fabricated persisted follow-up:\n%s", restart)
				}
			}
			if runner.calls.Load() != calls {
				t.Fatal("read-only result navigation retried execution")
			}
		})
	}
}

func TestResultCompactSizesAndUndersizeGuard(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {60, 18}, {32, 12}} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			home, view := handoffHome(t)
			native := &handoffIO{answer: strings.NewReader("yes\n")}
			var probes atomic.Int32
			session := startHandoff(t, app.Options{Home: home, Runner: navigationRunner{&probes}, Input: native, Output: native}, nil)
			session.send(press('r'), press(tea.KeyEnter), reviewKey(), press(tea.KeyEnter))
			session.returned(t)
			session.send(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			screen := session.screen(t)
			for _, want := range []string{"enter:", "Tab 1/", "? help", "esc back"} {
				if !strings.Contains(screen, want) {
					t.Fatalf("missing result control %q at %v:\n%s", want, size, screen)
				}
			}
			for _, line := range strings.Split(screen, "\n") {
				if ansi.StringWidth(line) > size[0] {
					t.Fatalf("result width exceeds %v: %q", size, line)
				}
			}
			if lipgloss.Height(screen) > size[1] {
				t.Fatalf("result height exceeds %v:\n%s", size, screen)
			}
			captureGroupedScreen(t, "result", size, false, screen)
			session.send(press(tea.KeyEnd))
			captureGroupedScreen(t, "result", size, true, session.screen(t))
			session.send(press('?'), press(tea.KeyEscape))
			if !strings.Contains(session.screen(t), "ssh-ed25519") && size[1] >= 18 {
				t.Fatal("help lost diagnostic viewport")
			}
			session.send(tea.WindowSizeMsg{Width: 31, Height: 11}, press('s'), press(tea.KeyEnter))
			if !strings.Contains(session.screen(t), "minimum 32x12") {
				t.Fatal("missing size guard")
			}
			session.send(tea.WindowSizeMsg{Width: size[0], Height: size[1]}, tea.WindowSizeMsg{Width: 80, Height: 24}, press(tea.KeyHome), press('s'), reviewKey())
			review := allResultText(t, session)
			if !strings.Contains(review, view.State.Registrations[0].ID) || !strings.Contains(review, "Review resume") {
				t.Fatal("resize retargeted continuation")
			}
			if probes.Load() != 0 {
				t.Fatal("undersize or result navigation executed a command")
			}
		})
	}
}

func TestResultManualUnregisterContinuationNeverOffersRegisterAgain(t *testing.T) {
	home, view := handoffHome(t)
	native := &handoffIO{answer: strings.NewReader("yes\nyes\n")}
	session := startHandoff(t, app.Options{Home: home, Runner: publicRunner{}, Input: native, Output: native}, nil)
	session.send(press('u'), reviewKey(), press(tea.KeyEnter))
	screen := session.returned(t)
	if !strings.Contains(screen, "Manual action required") {
		t.Fatalf("lost manual unregister result:\n%s", screen)
	}
	after, err := app.Snapshot(app.Options{Home: home})
	if err != nil {
		t.Fatal(err)
	}
	if after.State.Registrations[0].Operation != "unregister" {
		t.Fatal("lost unregister intent")
	}
	session.send(press('s'), reviewKey(), press(tea.KeyEnter))
	session.returned(t)
	session.send(press('r'))
	screen = session.screen(t)
	if !strings.Contains(screen, "Result:") || strings.Contains(screen, "Register public key") || strings.Contains(screen, "Register again") {
		t.Fatalf("resume unregister offered re-registration:\n%s", screen)
	}
	tasks := tui.ProjectTasks(after)
	if len(tasks) != 1 || tasks[0].ResumeID != view.State.Registrations[0].ID || !strings.Contains(tasks[0].Title, "Continue unregister") {
		t.Fatalf("stale onboarding replaced unregister: %+v", tasks)
	}
}

func TestTasksPendingDeletionSurvivesIntegrityWarningAndUsesLocalGuards(t *testing.T) {
	home, runner := resultHome(t, false)
	view, err := app.Snapshot(app.Options{Home: home})
	if err != nil {
		t.Fatal(err)
	}
	document := view.State
	document.Registrations[0].Operation = "unregister"
	document.Registrations[0].Status = "removed"
	document.Registrations[0].Stage = "complete"
	public, err := os.ReadFile(document.Keys[0].PublicPath)
	if err != nil {
		t.Fatal(err)
	}
	document.Deletions = []state.LocalDeletion{{ID: "deletion-recovery", KeyID: document.Keys[0].ID, Stage: "pending", CreatedAt: "2026-11-07T00:00:00Z", PublicSHA256: fmt.Sprintf("%x", sha256.Sum256(public)), PrivateRemoved: true}}
	saveNavigationState(t, home, document)
	if err := os.Remove(document.Keys[0].PublicPath); err != nil {
		t.Fatal(err)
	}
	view, err = app.Snapshot(app.Options{Home: home})
	if err != nil || view.Review == "" {
		t.Fatal("fixture lacks real integrity warning", err)
	}
	native := &handoffIO{answer: strings.NewReader("yes\n")}
	calls := runner.calls.Load()
	session := startHandoff(t, app.Options{Home: home, Runner: runner, Input: native, Output: native}, nil)
	session.send(press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab), press('/'), text("deletion-recovery"), press(tea.KeyEnter), press(tea.KeyDown), press(tea.KeyEnter), press('s'), reviewKey())
	screen := session.screen(t)
	if !strings.Contains(screen, "Review resume") || strings.Contains(screen, "Auth:") || strings.Contains(screen, "SSH configuration:") {
		t.Fatalf("guarded local recovery unavailable:\n%s", screen)
	}
	session.send(press(tea.KeyEnter))
	screen = session.returned(t)
	if !strings.Contains(screen, "Completed;") {
		t.Fatalf("dedicated guarded deletion could not reconcile absent file:\n%s", screen)
	}
	after, err := app.Snapshot(app.Options{Home: home})
	if err != nil || len(after.State.Keys) != 0 || after.State.Deletions[0].Stage != "complete" || after.State.DeletedKeys[0].Key.ID != document.Keys[0].ID {
		t.Fatal("deletion did not preserve journal/tombstone", err)
	}
	if runner.calls.Load() != calls {
		t.Fatal("local recovery invoked authentication")
	}
}

func TestResultAliasDenialKeepsPriorBindingAndIncludeEvidence(t *testing.T) {
	home, runner := resultHome(t, true)
	options := app.Options{Home: home, Runner: runner, Output: io.Discard, Input: strings.NewReader(""), Confirm: func(string) (bool, error) { return true, nil }, Request: &app.Request{Operation: "register", Host: "result", Key: "Result fixture", Bootstrap: "password", Configure: true}}
	if err := app.Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	before, err := app.Snapshot(options)
	if err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(home, ".ssh", "config")
	managedPath := before.State.Configs[0].Path
	mainBefore, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatal(err)
	}
	aliasBefore, err := os.ReadFile(managedPath)
	if err != nil {
		t.Fatal(err)
	}
	native := &handoffIO{answer: strings.NewReader("yes\nyes\nno\n")}
	session := startHandoff(t, app.Options{Home: home, Runner: runner, Input: native, Output: native}, nil)
	automaticRegister(session, true)
	if screen := session.returned(t); !strings.Contains(screen, "managed alias not generated this time") {
		t.Fatalf("denial misclassified:\n%s", screen)
	}
	full := allResultText(t, session)
	for _, want := range []string{"prior bindings/files", "saved binding", "Connection: verified", "alias: declined"} {
		if !strings.Contains(full, want) {
			t.Fatalf("lost prior evidence %q", want)
		}
	}
	after, err := app.Snapshot(options)
	if err != nil {
		t.Fatal(err)
	}
	mainAfter, _ := os.ReadFile(mainPath)
	aliasAfter, _ := os.ReadFile(managedPath)
	if string(mainAfter) != string(mainBefore) || string(aliasAfter) != string(aliasBefore) || !reflect.DeepEqual(before.State.Configs, after.State.Configs) || !reflect.DeepEqual(before.State.ConfigEdits, after.State.ConfigEdits) || after.State.Hosts[0].ConfigKeyID != before.State.Hosts[0].ConfigKeyID {
		t.Fatal("alias denial changed prior files/binding/Include evidence")
	}
	if len(tui.ProjectTasks(after)) != 0 {
		t.Fatal("denial fabricated pending config intent")
	}
}
