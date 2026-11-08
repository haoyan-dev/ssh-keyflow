package tui

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"ssh-keyflow/internal/app"
)

type nativeIO struct {
	sync.Mutex
	answer    *strings.Reader
	active    bool
	output    strings.Builder
	onConfirm func()
}

func (native *nativeIO) Read(p []byte) (int, error) {
	native.Lock()
	defer native.Unlock()
	if !native.active {
		return 0, io.EOF
	}
	return native.answer.Read(p)
}
func (native *nativeIO) Write(p []byte) (int, error) {
	native.Lock()
	defer native.Unlock()
	native.output.Write(p)
	if strings.Contains(native.output.String(), "Type yes") && !native.active {
		native.active = true
		if native.onConfirm != nil {
			native.onConfirm()
		}
	}
	return len(p), nil
}

type nativeRunner struct {
	input    io.Reader
	calls    atomic.Int32
	original atomic.Bool
}

func (runner *nativeRunner) LookPath(name string) (string, error) { return name, nil }
func (runner *nativeRunner) Run(_ context.Context, command app.Command) error {
	runner.calls.Add(1)
	runner.original.Store(command.Stdin == runner.input && command.Interactive)
	io.WriteString(command.Stdout, "native-output\x1b[2J\n")
	return context.Canceled
}

type barrier struct{ response chan model }
type session struct {
	program *tea.Program
	done    chan error
}

func startSession(t *testing.T, options app.Options) session {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	program, err := NewProgram(ctx, options, tea.WithoutRenderer(), tea.WithFilter(func(current tea.Model, message tea.Msg) tea.Msg {
		if request, ok := message.(barrier); ok {
			request.response <- current.(model)
			return nil
		}
		return message
	}))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := program.Run(); done <- err }()
	t.Cleanup(func() {
		program.Quit()
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(2 * time.Second):
			t.Error("native handoff did not terminate")
		}
		cancel()
	})
	return session{program, done}
}
func (session session) send(messages ...tea.Msg) {
	for _, message := range messages {
		session.program.Send(message)
	}
}
func (session session) snapshot(t *testing.T) model {
	t.Helper()
	response := make(chan model, 1)
	session.program.Send(barrier{response})
	select {
	case m := <-response:
		return m
	case <-time.After(3 * time.Second):
		t.Fatal("TUI barrier timed out")
		return model{}
	}
}
func (session session) result(t *testing.T) model {
	t.Helper()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	for {
		m := session.snapshot(t)
		if m.page == "result" {
			return m
		}
		select {
		case <-deadline.C:
			t.Fatal("native terminal did not return result")
			return model{}
		default:
		}
	}
}
func ctrlS() tea.KeyPressMsg { return tea.KeyPressMsg(tea.Key{Code: 's', Mod: tea.ModCtrl}) }
func TestNativeHandoffPreservesOriginalStdinAndInterruptedCreation(t *testing.T) {
	options := app.Options{Home: t.TempDir()}
	native := &nativeIO{answer: strings.NewReader("yes\n")}
	runner := &nativeRunner{input: native}
	options.Input, options.Output, options.Runner = native, native, runner
	var reports atomic.Int32
	options.Report = func(app.Report) { reports.Add(1) }
	session := startSession(t, options)
	session.send(press('c'), text("interrupted fixture"), ctrlS())
	m := session.result(t)
	if runner.calls.Load() != 1 || !runner.original.Load() {
		t.Fatal("native ssh-keygen stdin was wrapped, replayed, or not interactive")
	}
	if reports.Load() == 0 {
		t.Fatal("caller report callback was lost")
	}
	for _, want := range []string{"Operation stopped", "creation: pending", "(saved)", "native-output"} {
		if !strings.Contains(m.document, want) {
			t.Fatalf("missing %s: %s", want, m.document)
		}
	}
	if strings.Contains(m.View().Content, "\x1b[2J") {
		t.Fatal("native terminal control entered result page")
	}
	view, err := app.Snapshot(options)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.State.Creations) != 1 || view.State.Creations[0].Stage != "pending" {
		t.Fatal("creation checkpoint was lost")
	}
	for _, id := range []string{view.State.Creations[0].ID, view.State.Creations[0].KeyID} {
		if !strings.Contains(m.document, id) {
			t.Fatalf("missing saved identity %s", id)
		}
	}
	session.send(press('?'), press(tea.KeyEscape), press(tea.KeyEscape))
	m = session.snapshot(t)
	if m.page != "list" || len(m.entries()) != 1 || m.entries()[0].kind != "creation" {
		t.Fatal("unfinished creation inaccessible from Keys")
	}
	// A fresh native approval is required for the saved creation, too.
	native.Lock()
	native.answer = strings.NewReader("no\n")
	native.Unlock()
	session.send(press('s'))
	m = session.result(t)
	if runner.calls.Load() != 1 || m.request.Operation != "resume" || m.request.Resume != view.State.Creations[0].ID {
		t.Fatal("local resume did not bind saved creation or bypassed approval")
	}
}
func TestNativeConfirmationDenialStopsBeforeKeygenAndReleasesTerminal(t *testing.T) {
	home := t.TempDir()
	native := &nativeIO{answer: strings.NewReader("no\n")}
	runner := &nativeRunner{input: native}
	session := startSession(t, app.Options{Home: home, Input: native, Output: native, Runner: runner})
	session.send(press('c'), text("denied fixture"), ctrlS())
	m := session.result(t)
	if runner.calls.Load() != 0 || !strings.Contains(m.document, "confirmation: declined") {
		t.Fatalf("denial executed keygen or lost evidence: %s", m.document)
	}
	view, err := app.Snapshot(app.Options{Home: home})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.State.Keys) != 0 || len(view.State.Creations) != 0 {
		t.Fatal("denied creation saved an identity")
	}
	session.send(press(tea.KeyEscape), press('i'))
	if session.snapshot(t).page != "form" {
		t.Fatal("TUI did not accept input after native return")
	}
}
func TestNativeHandoffReloadFailureRetainsFactsAndRefreshNeverReexecutes(t *testing.T) {
	m := fixture(t)
	statePath := filepath.Join(filepath.Dir(m.view.State.Keys[0].PublicPath), ".ssh", "ssh-keyflow", "state.json")
	before, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	native := &nativeIO{answer: strings.NewReader("no\n"), onConfirm: func() {
		if err := os.WriteFile(statePath, []byte("broken"), 0600); err != nil {
			panic(err)
		}
	}}
	runner := &nativeRunner{input: native}
	options := m.options
	options.Input, options.Output, options.Runner = native, native, runner
	session := startSession(t, options)
	session.send(press('c'), text("stale fixture"), ctrlS())
	m = session.result(t)
	if !m.stale || !strings.Contains(m.document, "confirmation: declined") || !strings.Contains(m.document, "Reload failed") {
		t.Fatal(m.document)
	}
	if err := os.WriteFile(statePath, before, 0600); err != nil {
		t.Fatal(err)
	}
	session.send(press(tea.KeyF5))
	m = session.snapshot(t)
	if m.stale || !strings.Contains(m.document, "confirmation: declined") || runner.calls.Load() != 0 {
		t.Fatal("refresh lost denial evidence or replayed operation")
	}
}
func TestNativeConfirmationCancellationAndExecutionAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := readNativeConfirmation(ctx, strings.NewReader("yes\n")); err != context.Canceled {
		t.Fatalf("cancelled approval: %v", err)
	}
	runner := &nativeRunner{}
	operation := execution{ctx: ctx, options: app.Options{Runner: runner}, request: app.Request{Operation: "create", Name: "cancelled"}}
	if err := operation.Run(); err != context.Canceled || runner.calls.Load() != 0 {
		t.Fatal("cancelled handoff invoked backend")
	}
}
