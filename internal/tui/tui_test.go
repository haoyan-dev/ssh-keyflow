package tui_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/tui"
)

func fixtureTempDir(test *testing.T) string {
	test.Helper()
	// macOS temporary directories can have a symlinked /var ancestor.
	directory, err := filepath.EvalSymlinks(test.TempDir())
	if err != nil {
		test.Fatal(err)
	}
	return directory
}

func runScreen(test *testing.T, home string, messages ...tea.Msg) string {
	return runSizedScreen(test, home, 80, 24, messages...)
}

func runSizedScreen(test *testing.T, home string, width, height int, messages ...tea.Msg) string {
	return runSizedScreenOptions(test, home, width, height, nil, messages...)
}

func runSizedScreenOptions(test *testing.T, home string, width, height int, options []tea.ProgramOption, messages ...tea.Msg) string {
	return runSizedModelOptions(test, home, width, height, options, messages...).View().Content
}

func runSizedModelOptions(test *testing.T, home string, width, height int, options []tea.ProgramOption, messages ...tea.Msg) tea.Model {
	test.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var probes atomic.Int32
	program, err := tui.NewProgram(ctx, app.Options{Home: home, Input: strings.NewReader(""), Output: io.Discard, Runner: navigationRunner{&probes}}, append([]tea.ProgramOption{tea.WithWindowSize(width, height), tea.WithoutRenderer()}, options...)...)
	if err != nil {
		test.Fatal(err)
	}
	type result struct {
		model tea.Model
		err   error
	}
	done := make(chan result, 1)
	go func() { model, err := program.Run(); done <- result{model, err} }()
	program.Send(tea.WindowSizeMsg{Width: width, Height: height})
	for _, message := range messages {
		program.Send(message)
	}
	program.Send(tea.Quit())
	final := <-done
	if final.err != nil {
		test.Fatal(final.err)
	}
	if probes.Load() != 0 {
		test.Fatal("navigation probed dependencies or executed a command")
	}
	return final.model
}

func press(code rune) tea.Msg { return tea.KeyPressMsg(tea.Key{Code: code}) }

func TestTUIStartsAtReadOnlyListsAndQuits(test *testing.T) {
	home := test.TempDir()
	text := runScreen(test, home)
	for _, expected := range []string{"SSH KeyFlow", "Destinations", "Keys", "Registrations", "Tasks"} {
		if !strings.Contains(text, expected) {
			test.Fatalf("missing %q in %s", expected, text)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh")); !os.IsNotExist(err) {
		test.Fatal("TUI listing initialized storage")
	}
}

func TestTUIFormCanBeCancelledBeforeMutation(test *testing.T) {
	home := test.TempDir()
	opened := runScreen(test, home, press(tea.KeyTab), press('c'))
	if !strings.Contains(opened, "Create Ed25519 key") || !strings.Contains(opened, "Management name") {
		test.Fatalf("form not opened: %s", opened)
	}
	cancelled := runScreen(test, home, press(tea.KeyTab), press('c'), tea.KeyPressMsg(tea.Key{Text: "draft", Code: 'd'}), press(tea.KeyEscape), press(tea.KeyTab), press(tea.KeyEnter))
	if !strings.Contains(cancelled, "Changes cancelled") || !strings.Contains(cancelled, "Managed keys") {
		test.Fatalf("cancel failed: %s", cancelled)
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh")); !os.IsNotExist(err) {
		test.Fatal("cancelled form initialized storage")
	}
}

func TestTUIReadOnlyBackDoesNotReportAChangeWasCancelled(test *testing.T) {
	text := runScreen(test, test.TempDir(), press('g'), press(tea.KeyEscape))
	if strings.Contains(text, "Changes cancelled") {
		test.Fatal("returning from read-only discovery reported a cancelled mutation")
	}
}

func TestTUINarrowFormKeepsInputAndControlsVisible(test *testing.T) {
	text := runSizedScreen(test, test.TempDir(), 32, 12, press('c'))
	if lipgloss.Height(text) > 12 || lipgloss.Width(text) > 32 {
		test.Fatalf("narrow screen overflows: %dx%d\n%s", lipgloss.Width(text), lipgloss.Height(text), text)
	}
	for _, expected := range []string{"Management name", "ctrl+s", "esc"} {
		if !strings.Contains(text, expected) {
			test.Fatalf("missing %q: %s", expected, text)
		}
	}
}

func TestTUIAddDestinationRequiresExplicitOSBeforeReview(test *testing.T) {
	home := test.TempDir()
	form := destinationDraft("macos")
	// Leave OS untouched until the user explicitly selects it.
	form = form[:len(form)-1]
	unconfirmed := runScreen(test, home, append(form, reviewKey())...)
	if strings.Contains(unconfirmed, "Review add") || !strings.Contains(unconfirmed, "OS: select linux, macos or windows") {
		test.Fatalf("missing OS was accepted as confirmed:\n%s", unconfirmed)
	}
	form = append(form, text("macos"), reviewKey(), text("new key"), reviewKey())
	confirmed := runScreen(test, home, form...)
	if !strings.Contains(confirmed, "Review add") || !strings.Contains(confirmed, "OS: linux/macos/windows: macos") {
		test.Fatalf("explicit OS could not reach review:\n%s", confirmed)
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh")); !os.IsNotExist(err) {
		test.Fatal("form review initialized storage")
	}
}

type publicRunner struct{}

func (publicRunner) LookPath(name string) (string, error)   { return name, nil }
func (publicRunner) Run(context.Context, app.Command) error { return nil }

func TestTUISelectedDetailsAndDiscoveryUseCurrentOperations(test *testing.T) {
	home := test.TempDir()
	source := filepath.Join(home, ".ssh", "source.pub")
	if err := os.MkdirAll(filepath.Dir(source), 0700); err != nil {
		test.Fatal(err)
	}
	public := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB fixture"
	if err := os.WriteFile(source, []byte(public), 0600); err != nil {
		test.Fatal(err)
	}
	options := app.Options{Home: home, Output: io.Discard, Runner: publicRunner{}, Confirm: func(string) (bool, error) { return true, nil }}
	options.Request = &app.Request{Operation: "import", Name: "Managed example", PublicPath: source}
	if err := app.Run(context.Background(), options); err != nil {
		test.Fatal(err)
	}
	options.Request = &app.Request{Operation: "add", Key: "Managed example", Bootstrap: "manual",
		Destination: app.Destination{Alias: "example", Hostname: "example.test", User: "me", Port: 22, OS: "linux"}}
	if err := app.Run(context.Background(), options); err != nil {
		test.Fatal(err)
	}
	for _, messages := range [][]tea.Msg{nil, {press('?')}, {press(tea.KeyTab)}, {press(tea.KeyTab), press('?')}} {
		screen := runSizedScreen(test, home, 32, 12, messages...)
		if lipgloss.Height(screen) > 12 || lipgloss.Width(screen) > 32 {
			test.Fatalf("populated narrow overflow: %s", screen)
		}
	}
	details := runScreen(test, home, press(tea.KeyTab), press(tea.KeyEnter))
	if !strings.Contains(details, "Managed example") || !strings.Contains(details, "fingerprint") || !strings.Contains(details, "imported") {
		test.Fatalf("selected key details unavailable: %s", details)
	}
	discovery := runScreen(test, home, press('g'))
	if !strings.Contains(discovery, "Discovery (read-only") || strings.Contains(discovery, "use 1/r") {
		test.Fatalf("discovery refers to removed menu: %s", discovery)
	}
	review := runScreen(test, home, press(tea.KeyTab), press(tea.KeyTab), press('r'), tea.KeyPressMsg(tea.Key{Code: 's', Mod: tea.ModCtrl}))
	if !strings.Contains(review, "Review register") {
		test.Fatalf("selected registration did not supply its destination and key: %s", review)
	}
}

func TestTUIListAndScrollableHelpFitResizedTerminal(test *testing.T) {
	home := test.TempDir()
	for _, messages := range [][]tea.Msg{
		nil, {press('?')}, {press('?'), press(tea.KeyPgDown), press(tea.KeyPgDown), press(tea.KeyPgDown)},
		{tea.WindowSizeMsg{Width: 80, Height: 24}, press('?'), tea.WindowSizeMsg{Width: 32, Height: 12}},
		{tea.WindowSizeMsg{Width: 32, Height: 12}, tea.WindowSizeMsg{Width: 80, Height: 24}, tea.WindowSizeMsg{Width: 32, Height: 12}},
	} {
		text := runSizedScreen(test, home, 32, 12, messages...)
		if lipgloss.Height(text) > 12 || lipgloss.Width(text) > 32 {
			test.Fatalf("overflow %dx%d: %s", lipgloss.Width(text), lipgloss.Height(text), text)
		}
		if !strings.Contains(text, "q quit") && !strings.Contains(text, "esc/? back") {
			test.Fatalf("missing controls: %s", text)
		}
	}
	top := runSizedScreen(test, home, 32, 12, press('?'))
	bottom := runSizedScreen(test, home, 32, 12, press('?'), press(tea.KeyEnd))
	if !strings.Contains(top, "Keyboard help") || !strings.Contains(bottom, "Discovery") || top == bottom {
		test.Fatalf("help not scrollable: top=%s bottom=%s", top, bottom)
	}
}

// Runner is an external process boundary; navigation must never reach it.
type navigationRunner struct{ calls *atomic.Int32 }

func (r navigationRunner) LookPath(name string) (string, error) {
	r.calls.Add(1)
	return "", fmt.Errorf("unexpected dependency probe: %s", name)
}
func (r navigationRunner) Run(context.Context, app.Command) error {
	r.calls.Add(1)
	return fmt.Errorf("unexpected process execution")
}
