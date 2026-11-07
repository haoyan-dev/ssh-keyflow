package tui_test

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
	"ssh-keyflow/internal/state"
	"ssh-keyflow/internal/tui"
)

type handoffIO struct {
	sync.Mutex
	answer   *strings.Reader
	active   bool
	corrupt  string
	output   strings.Builder
	onPrompt func(string)
}

func (n *handoffIO) Read(p []byte) (int, error) {
	n.Lock()
	defer n.Unlock()
	if !n.active {
		return 0, io.EOF
	}
	count, err := n.answer.Read(p)
	if n.corrupt != "" && count > 0 && strings.Contains(string(p[:count]), "\n") {
		if writeErr := os.WriteFile(n.corrupt, []byte("broken fixture state"), 0600); writeErr != nil {
			return 0, writeErr
		}
		n.corrupt = ""
	}
	return count, err
}
func (n *handoffIO) Write(p []byte) (int, error) {
	n.Lock()
	defer n.Unlock()
	n.output.Write(p)
	if n.onPrompt != nil {
		n.onPrompt(n.output.String())
	}
	if strings.Contains(n.output.String(), "Type yes") {
		n.active = true
	}
	return len(p), nil
}

func handoffHome(t *testing.T) (string, app.View) {
	t.Helper()
	home := fixtureTempDir(t)
	source := filepath.Join(home, "source.pub")
	if err := os.WriteFile(source, []byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	options := app.Options{Home: home, Output: io.Discard, Runner: publicRunner{}, Confirm: func(string) (bool, error) { return true, nil }}
	options.Request = &app.Request{Operation: "import", Name: "Handoff fixture", PublicPath: source}
	if err := app.Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	options.Request = &app.Request{Operation: "add", Key: "Handoff fixture", Bootstrap: "manual", Destination: app.Destination{Alias: "fixture", Hostname: "example.test", User: "me", Port: 2222, OS: "linux"}}
	if err := app.Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	view, err := app.Snapshot(options)
	if err != nil {
		t.Fatal(err)
	}
	return home, view
}

type handoffSession struct {
	program *tea.Program
	ctx     context.Context
	done    chan error
}

func startHandoff(t *testing.T, options app.Options, filter func(tea.Model, tea.Msg) tea.Msg) handoffSession {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	program, err := tui.NewProgram(ctx, options, tea.WithoutRenderer(), tea.WithFilter(func(m tea.Model, msg tea.Msg) tea.Msg {
		if b, ok := msg.(formBarrier); ok {
			b.view <- m.View().Content
			return nil
		}
		if filter != nil {
			return filter(m, msg)
		}
		return msg
	}))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	session := handoffSession{program, ctx, make(chan error, 1)}
	go func() {
		var runErr error
		defer func() { session.done <- runErr }()
		_, runErr = program.Run()
	}()
	t.Cleanup(func() {
		program.Quit()
		select {
		case err := <-session.done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(2 * time.Second):
			t.Error("handoff program did not stop during cleanup")
		}
		cancel()
	})
	return session
}
func (s handoffSession) send(messages ...tea.Msg) {
	for _, msg := range messages {
		s.program.Send(msg)
	}
}
func (s handoffSession) screen(t *testing.T) string {
	t.Helper()
	views := make(chan string, 1)
	s.program.Send(formBarrier{views})
	select {
	case screen := <-views:
		return screen
	case <-s.ctx.Done():
		t.Fatal("handoff view timed out")
		return ""
	}
}
func (s handoffSession) returned(t *testing.T) string {
	t.Helper()
	for {
		screen := s.screen(t)
		if strings.Contains(screen, "Result: ") {
			return screen
		}
	}
}

func TestHandoffRetainsManualFactsAndExplicitTarget(t *testing.T) {
	home, view := handoffHome(t)
	var probes atomic.Int32
	native := &handoffIO{answer: strings.NewReader("yes\n")}
	session := startHandoff(t, app.Options{Home: home, Input: native, Output: native, Runner: navigationRunner{&probes}}, nil)
	session.send(press('r'), press(tea.KeyEnter), reviewKey(), press(tea.KeyEnter))
	screen := session.returned(t)
	for _, want := range []string{"manual: required", "verification: not-checked", view.State.Hosts[0].ID, view.State.Keys[0].ID} {
		if !strings.Contains(screen, want) {
			t.Fatalf("lost typed manual result/target %q:\n%s", want, screen)
		}
	}
	if probes.Load() != 0 {
		t.Fatal("manual guidance executed a command")
	}
}

func TestHandoffDenialRetainsFactsThroughStaleReloadAndRefresh(t *testing.T) {
	home, view := handoffHome(t)
	statePath := filepath.Join(state.Directory(home), "state.json")
	before, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	native := &handoffIO{answer: strings.NewReader("no\n"), corrupt: statePath}
	var probes atomic.Int32
	var reports atomic.Int32
	session := startHandoff(t, app.Options{Home: home, Input: native, Output: native, Runner: navigationRunner{&probes}, Report: func(app.Report) { reports.Add(1) }}, nil)
	session.send(press('r'), press(tea.KeyEnter), reviewKey(), press(tea.KeyEnter))
	screen := session.returned(t)
	for _, want := range []string{"Cancelled before changes", "Reload failed", "stale", "confirmation: declined", view.State.Hosts[0].ID} {
		if !strings.Contains(screen, want) {
			t.Fatalf("lost denial/stale evidence %q:\n%s", want, screen)
		}
	}
	count := reports.Load()
	session.send(press('s'), press('r'), press('t'), press('v'), press(tea.KeyEnter))
	if screen := session.screen(t); !strings.Contains(screen, "stale") || strings.Contains(screen, "Resume saved operation") || strings.Contains(screen, "Register public key") {
		t.Fatalf("stale result exposed mutation: %s", screen)
	}
	session.send(press(tea.KeyF5))
	screen = session.screen(t)
	if !strings.Contains(screen, "confirmation: declined") || !strings.Contains(screen, "Reload failed") {
		t.Fatalf("failed Refresh hid facts:\n%s", screen)
	}
	session.send(press(tea.KeyEnd))
	if !strings.Contains(session.screen(t), "Type yes to register") {
		t.Fatal("native denial diagnostics lost")
	}
	if err := os.WriteFile(statePath, before, 0600); err != nil {
		t.Fatal(err)
	}
	session.send(press(tea.KeyF5), press(tea.KeyHome))
	screen = session.screen(t)
	if strings.Contains(screen, "stale") || !strings.Contains(screen, "confirmation: declined") {
		t.Fatalf("refresh lost facts or kept stale guard:\n%s", screen)
	}
	if reports.Load() != count || probes.Load() != 0 {
		t.Fatal("Refresh replayed execution")
	}
	after, err := app.Snapshot(app.Options{Home: home})
	if err != nil {
		t.Fatal(err)
	}
	if len(after.State.Registrations) != len(view.State.Registrations) {
		t.Fatal("denial changed registration evidence")
	}
}

func TestHandoffAcceptsContinueOnlyOnceBeforeNativeReturn(t *testing.T) {
	home, _ := handoffHome(t)
	native := &handoffIO{answer: strings.NewReader("no\nno\n")}
	var denials atomic.Int32
	var probes atomic.Int32
	var delayed []tea.Msg
	enters := 0
	var program *tea.Program
	session := startHandoff(t, app.Options{Home: home, Input: native, Output: native, Runner: navigationRunner{&probes}, Report: func(report app.Report) {
		if report.Kind == app.ReportConfirmation {
			denials.Add(1)
		}
	}}, func(m tea.Model, msg tea.Msg) tea.Msg {
		// Deliver paste and two Continue events while native execution is pending.
		if key, ok := msg.(tea.KeyPressMsg); ok {
			if key.String() == "enter" && strings.Contains(m.View().Content, "Review register") {
				enters++
				if enters == 2 {
					queued := append([]tea.Msg(nil), delayed...)
					go func() {
						for _, message := range queued {
							program.Send(message)
						}
					}()
				}
			}
			return msg
		}
		if _, ok := msg.(tea.PasteMsg); ok {
			return msg
		}
		if enters == 1 {
			delayed = append(delayed, msg)
			return nil
		}
		return msg
	})
	program = session.program
	session.send(press('r'), press(tea.KeyEnter), reviewKey(), press(tea.KeyEnter), tea.PasteMsg{Content: "q?rc\n\rctrl+s"}, press(tea.KeyEnter))
	session.returned(t)
	// Drain preceding command/completion events before checking native approvals.
	session.screen(t)
	if denials.Load() != 1 {
		t.Fatalf("Continue launched %d native approvals", denials.Load())
	}
	if probes.Load() != 0 {
		t.Fatal("denied native execution ran a command")
	}
}

// This is a simulated child interruption, not a native OS signal acceptance test.
type interruptedCreation struct {
	input    io.Reader
	calls    atomic.Int32
	original atomic.Bool
}

func (r *interruptedCreation) LookPath(name string) (string, error) { return name, nil }
func (r *interruptedCreation) Run(_ context.Context, command app.Command) error {
	r.calls.Add(1)
	r.original.Store(command.Stdin == r.input && command.Interactive)
	io.WriteString(command.Stdout, "\x1b[2Jnative-output-q-c-r\x1b[0m\n")
	return context.Canceled
}
func TestHandoffInterruptedCreationRetainsSavedCheckpointAndOriginalInput(t *testing.T) {
	home := fixtureTempDir(t)
	native := &handoffIO{answer: strings.NewReader("yes\n")}
	runner := &interruptedCreation{input: native}
	var reports []app.Report
	session := startHandoff(t, app.Options{Home: home, Input: native, Output: native, Runner: runner, Report: func(report app.Report) { reports = append(reports, report) }}, nil)
	session.send(press('c'), text("interrupted fixture"), reviewKey(), press(tea.KeyEnter))
	screen := session.returned(t)
	for _, want := range []string{"Operation stopped", "creation: pending (saved checkpoint)", "creation: unknown", "creation interrupted"} {
		if !strings.Contains(screen, want) {
			t.Fatalf("lost checkpoint/error %q:\n%s", want, screen)
		}
	}
	if runner.calls.Load() != 1 || !runner.original.Load() {
		t.Fatal("native input was wrapped, replayed, or lost interactive intent")
	}
	after, err := app.Snapshot(app.Options{Home: home})
	if err != nil {
		t.Fatal(err)
	}
	if len(after.State.Creations) != 1 || after.State.Creations[0].Stage != "pending" {
		t.Fatalf("lost durable creation checkpoint: %+v", after.State.Creations)
	}
	for _, id := range []string{after.State.Creations[0].ID, after.State.Creations[0].KeyID} {
		if !strings.Contains(screen, id) {
			t.Fatalf("lost allocated checkpoint target %s:\n%s", id, screen)
		}
	}
	if len(reports) == 0 {
		t.Fatal("caller report sink was not retained")
	}
	session.send(press(tea.KeyEnd))
	screen = session.screen(t)
	if !strings.Contains(screen, "native-output-q-c-r") || strings.Contains(screen, "\x1b[2J") {
		t.Fatalf("diagnostic output was lost or not sanitized:\n%s", screen)
	}
	session.send(press('?'), press(tea.KeyEscape), press(tea.KeyHome))
	if !strings.Contains(session.screen(t), "Operation stopped") {
		t.Fatal("returned TUI did not retain result through Help")
	}
}
