package tui_test

import (
	"bytes"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/tui"
	"sync"
	"sync/atomic"
	"time"

	"ssh-keyflow/internal/state"
	"strings"
	"testing"
)

func reviewKey() tea.Msg { return tea.KeyPressMsg(tea.Key{Code: 's', Mod: tea.ModCtrl}) }
func clearKey() tea.Msg  { return tea.KeyPressMsg(tea.Key{Code: 'u', Mod: tea.ModCtrl}) }
func destinationDraft(osName string) []tea.Msg {
	return []tea.Msg{press('a'), text("作業-box"), press(tea.KeyEnter), text("server.example"), press(tea.KeyEnter), text("alice"), press(tea.KeyEnter), press(tea.KeyEnter), text(osName)}
}

func TestFormsDestinationStepValidatesAndKeepsEarlierInputs(t *testing.T) {
	home := t.TempDir()
	journey := destinationDraft("linux")
	screen := runScreen(t, home, journey...)
	if !strings.Contains(screen, "Destination · step 1/3") || strings.Contains(screen, "Windows account") || strings.Contains(screen, "Authentication key path") {
		t.Fatalf("not a focused destination step:\n%s", screen)
	}
	journey = append(journey, reviewKey())
	screen = runScreen(t, home, journey...)
	if !strings.Contains(screen, "Key · step 2/3") || !strings.Contains(screen, "Management name") {
		t.Fatalf("review did not focus first invalid field across steps:\n%s", screen)
	}
	journey = append(journey, text("日本語 key"), reviewKey())
	screen = runScreen(t, home, journey...)
	for _, want := range []string{"Review add", "作業-box", "server.example", "日本語 key", "manual", "no"} {
		if !strings.Contains(screen, want) {
			t.Fatalf("missing %q:\n%s", want, screen)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh")); !os.IsNotExist(err) {
		t.Fatal("preparation wrote storage")
	}
}

func TestFormsConditionalChoicesEmitOnlyApplicableRequestValues(t *testing.T) {
	home, _ := navigationHome(t)
	journey := []tea.Msg{press('r'), press(tea.KeyEnter), press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyRight), press(tea.KeyRight), press(tea.KeyTab), text("C:/bootstrap/id"), press(tea.KeyTab), press(tea.KeyRight), reviewKey()}
	first := runScreen(t, home, journey...)
	for _, want := range []string{"Review register", "host-1", "key-1", "Authentication key path: C:/bootstrap/id", "Auth: key", "SSH configuration: yes"} {
		if !strings.Contains(first, want) {
			t.Fatalf("missing %q:\n%s", want, first)
		}
	}
	journey = append(journey, press(tea.KeyEscape), tea.KeyPressMsg(tea.Key{Code: tea.KeyTab, Mod: tea.ModShift}), tea.KeyPressMsg(tea.Key{Code: tea.KeyTab, Mod: tea.ModShift}), press(tea.KeyLeft), press(tea.KeyLeft), reviewKey())
	manual := runScreen(t, home, journey...)
	if strings.Contains(manual, "C:/bootstrap/id") || !strings.Contains(manual, "Auth: manual") {
		t.Fatalf("hidden identity survived: %+v", manual)
	}
}

func TestFormsReresolveFixedIDsAndRetainDraftOnChangedTargets(t *testing.T) {
	for _, change := range []string{"display", "path", "missing"} {
		t.Run(change, func(t *testing.T) {
			home, document := navigationHome(t)
			updated := false
			options := []tea.ProgramOption{tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
				if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "ctrl+s" && !updated {
					updated = true
					switch change {
					case "display":
						document.Keys[6].Name = "新しい表示"
						document.Keys[0], document.Keys[6] = document.Keys[6], document.Keys[0]
					case "path":
						document.Keys[6].PublicPath = filepath.Join(home, "changed.pub")
					case "missing":
						document.Keys = document.Keys[:6]
						document.Registrations = document.Registrations[:6]
					}
					saveNavigationState(t, home, document)
				}
				return msg
			})}
			journey := []tea.Msg{press('r'), press('/'), text("key-7"), press(tea.KeyEnter), press(tea.KeyEnter), reviewKey()}
			final := runSizedModelOptions(t, home, 80, 24, options, journey...)
			ready := strings.Contains(final.View().Content, "Review register")
			screen := final.View().Content
			if change == "display" {
				if !ready || !strings.Contains(screen, "key-7") || !strings.Contains(screen, "host-1") || !strings.Contains(screen, "新しい表示") {
					t.Fatalf("fixed identity drifted after rename/reorder: ready=%v\n%s", ready, screen)
				}
			} else {
				if ready || !strings.Contains(screen, "key-7") || !strings.Contains(screen, "target") {
					t.Fatalf("changed target accepted or draft lost:\n%s", screen)
				}
			}
		})
	}
}

func TestFormsKeyPickerCancelRetainsDraftAndExclusiveKeyChoice(t *testing.T) {
	home, _ := navigationHome(t)
	journey := append(destinationDraft("windows"), press(tea.KeyEnter), text("standard"), reviewKey())
	before := runScreen(t, home, journey...)
	after := runScreen(t, home, append(journey, press(tea.KeyEnter), press('/'), text("not-present?q"), press(tea.KeyEscape), press(tea.KeyEscape))...)
	if before != after {
		t.Fatalf("picker cancel lost draft/focus:\nbefore=%s\nafter=%s", before, after)
	}
	// Choose Create, then switch back to Existing; no hidden new name may survive.
	journey = append(journey, tea.KeyPressMsg(tea.Key{Code: tea.KeyTab, Mod: tea.ModShift}), press(tea.KeyRight), press(tea.KeyTab), text("discarded new name"), reviewKey(), press(tea.KeyEscape), tea.KeyPressMsg(tea.Key{Code: tea.KeyTab, Mod: tea.ModShift}), press(tea.KeyRight), press(tea.KeyTab), press(tea.KeyEnter), press('/'), text("key-8"), press(tea.KeyEnter), press(tea.KeyEnter), reviewKey())
	screen := runScreen(t, home, journey...)
	for _, want := range []string{"Review add", "key-8", "Windows account: standard", "verification unavailable"} {
		if !strings.Contains(screen, want) {
			t.Fatalf("missing %q:\n%s", want, screen)
		}
	}
	if strings.Contains(screen, "discarded new name") {
		t.Fatalf("exclusive choice retained stale name:\n%s", screen)
	}
	// Back through Key to Destination; changing Windows to Linux clears account.
	journey = append(journey, press(tea.KeyEscape), press(tea.KeyEscape), press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyRight), reviewKey())
	screen = runScreen(t, home, journey...)
	if !strings.Contains(screen, "Review add") || strings.Contains(screen, "Windows account:") || !strings.Contains(screen, "OS: linux/macos/windows: linux") {
		t.Fatalf("hidden account survived OS switch:\n%s", screen)
	}
}

func TestFormsLocalResumeSelectorsOmitRemoteOptions(t *testing.T) {
	for _, kind := range []string{"creation", "deletion"} {
		t.Run(kind, func(t *testing.T) {
			home, doc := navigationHome(t)
			doc.Keys, doc.Hosts, doc.Registrations = doc.Keys[:1], doc.Hosts[:1], []state.Registration{}
			if kind == "creation" {
				path := filepath.Join(home, "new-key")
				doc.Creations = []state.Creation{{ID: "creation-local", KeyID: "future-key", Name: "新しい鍵", PrivatePath: path, PublicPath: path + ".pub", CreatedAt: "2026-11-07T00:00:00Z", Stage: "planned"}}
			} else {
				doc.Deletions = []state.LocalDeletion{{ID: "deletion-local", KeyID: "key-1", CreatedAt: "2026-11-07T00:00:00Z", Stage: "pending", PublicSHA256: strings.Repeat("a", 64), PrivateRemoved: true}}
			}
			saveNavigationState(t, home, doc)
			screen := runScreen(t, home, press('s'), press(tea.KeyEnter), reviewKey())
			if !strings.Contains(screen, "Review resume") || !strings.Contains(screen, kind+"-local") || strings.Contains(screen, "Auth:") || strings.Contains(screen, "SSH configuration:") {
				t.Fatalf("local resume wrong fields:\n%s", screen)
			}
		})
	}
}

func TestFormsValidationNamesFirstInvalidInput(t *testing.T) {
	for _, tc := range []struct {
		name, value, want string
		index             int
	}{
		{"alias", "", "Destination alias:", 0}, {"host", "-unsafe", "Host / IP:", 1}, {"user", "", "Login user:", 2},
		{"port", "0", "1..65535", 3}, {"port-high", "65536", "1..65535", 3}, {"port-text", "abc", "1..65535", 3},
		{"os", "unknown", "OS: select linux, macos or windows", 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			journey := destinationDraft("macos")
			// Move back from OS to the invalid field without changing valid entries.
			for i := 4; i > tc.index; i-- {
				journey = append(journey, tea.KeyPressMsg(tea.Key{Code: tea.KeyTab, Mod: tea.ModShift}))
			}
			journey = append(journey, clearKey())
			if tc.value != "" {
				journey = append(journey, text(tc.value))
			}
			journey = append(journey, reviewKey())
			screen := runScreen(t, t.TempDir(), journey...)
			if strings.Contains(screen, "Review add") || !strings.Contains(screen, tc.want) {
				t.Fatalf("wrong validation/focus:\n%s", screen)
			}
		})
	}
	screen := runScreen(t, t.TempDir(), press('i'), text("valid name"), reviewKey())
	if !strings.Contains(screen, "Existing public key path:") {
		t.Fatalf("path guidance missing:\n%s", screen)
	}
}

func TestFormsEditingAndHelpRetainCursorAndDraft(t *testing.T) {
	home := t.TempDir()
	journey := []tea.Msg{press('c'), text("ac"), press(tea.KeyLeft), text("b")}
	screen := runScreen(t, home, append(journey, reviewKey())...)
	if !strings.Contains(screen, "Management name: abc") {
		t.Fatalf("editing moved the cursor: %s", screen)
	}
	journey = append(journey, press(tea.KeyTab), press(tea.KeyTab))
	before := runScreen(t, home, journey...)
	after := runScreen(t, home, append(journey, press(tea.KeyEnter), press(tea.KeyEscape))...)
	if before != after {
		t.Fatalf("Help lost form position/draft:\nbefore=%s\nafter=%s", before, after)
	}
}

func TestFormsCompactValidationAndUnregisterGuidance(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {60, 18}, {32, 12}} {
		journey := append(destinationDraft("linux"), tea.KeyPressMsg(tea.Key{Code: tea.KeyTab, Mod: tea.ModShift}), clearKey(), text("0"), reviewKey())
		screen := runSizedScreen(t, t.TempDir(), size[0], size[1], journey...)
		if lipgloss.Width(screen) > size[0] || lipgloss.Height(screen) > size[1] || !strings.Contains(screen, "1..65535") || !strings.Contains(screen, "ctrl+s") {
			t.Fatalf("focused error/controls inaccessible at %v:\n%s", size, screen)
		}
	}
	home, _ := navigationHome(t)
	screen := runScreen(t, home, press('u'))
	for _, want := range []string{"build-1", "Laptop", "alternate authentication", "local key files"} {
		if !strings.Contains(screen, want) {
			t.Fatalf("unregister missing %q:\n%s", want, screen)
		}
	}
	review := runScreen(t, home, press('u'), reviewKey())
	if strings.Contains(review, "SSH configuration:") {
		t.Fatalf("unregister offered configuration:\n%s", review)
	}
}

type formNativeIO struct {
	sync.Mutex
	approved bool
	answer   *strings.Reader
}

func (n *formNativeIO) Read(p []byte) (int, error) {
	n.Lock()
	defer n.Unlock()
	if !n.approved {
		return 0, io.EOF
	}
	return n.answer.Read(p)
}
func (n *formNativeIO) Write(p []byte) (int, error) {
	n.Lock()
	defer n.Unlock()
	if strings.Contains(string(p), "Type yes to register this managed public key:") {
		n.approved = true
	}
	return len(p), nil
}

type formBarrier struct{ view chan string }
type intentRunner struct {
	commands chan app.Command
	calls    atomic.Int32
}

func (r *intentRunner) LookPath(name string) (string, error) { r.calls.Add(1); return name, nil }
func (r *intentRunner) Run(_ context.Context, cmd app.Command) error {
	r.calls.Add(1)
	r.commands <- cmd
	return fmt.Errorf("fixture stops at first probe")
}

func TestFormsHandoffEmitsChosenAuthenticationIntent(t *testing.T) {
	for _, mode := range []string{"key", "password"} {
		t.Run(mode, func(t *testing.T) {
			home := t.TempDir()
			source := filepath.Join(home, "source.pub")
			public := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB fixture"
			if err := os.WriteFile(source, []byte(public), 0600); err != nil {
				t.Fatal(err)
			}
			opts := app.Options{Home: home, Output: io.Discard, Runner: publicRunner{}, Confirm: func(string) (bool, error) { return true, nil }}
			opts.Request = &app.Request{Operation: "import", Name: "Intent fixture", PublicPath: source}
			if err := app.Run(context.Background(), opts); err != nil {
				t.Fatal(err)
			}
			opts.Request = &app.Request{Operation: "add", Key: "Intent fixture", Bootstrap: "manual", Destination: app.Destination{Alias: "fixture", Hostname: "example.test", User: "me", Port: 2222, OS: "linux"}}
			if err := app.Run(context.Background(), opts); err != nil {
				t.Fatal(err)
			}
			identity := filepath.Join(home, "disposable-identity")
			if err := os.WriteFile(identity, []byte("fixture only; never a credential"), 0600); err != nil {
				t.Fatal(err)
			}
			statePath := filepath.Join(state.Directory(home), "state.json")
			before, err := os.ReadFile(statePath)
			if err != nil {
				t.Fatal(err)
			}
			runner := &intentRunner{commands: make(chan app.Command, 1)}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			native := &formNativeIO{answer: strings.NewReader("yes\n")}
			program, err := tui.NewProgram(ctx, app.Options{Home: home, Input: native, Output: native, Runner: runner}, tea.WithoutRenderer(), tea.WithFilter(func(m tea.Model, msg tea.Msg) tea.Msg {
				if barrier, ok := msg.(formBarrier); ok {
					barrier.view <- m.View().Content
					return nil
				}
				return msg
			}))
			if err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { _, err := program.Run(); done <- err }()
			defer program.Quit()
			journey := []tea.Msg{press('r'), press(tea.KeyEnter), press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyRight), press(tea.KeyRight), press(tea.KeyTab), text(identity)}
			if mode == "password" {
				journey = append(journey, tea.KeyPressMsg(tea.Key{Code: tea.KeyTab, Mod: tea.ModShift}), press(tea.KeyLeft))
			}
			journey = append(journey, reviewKey())
			for _, msg := range journey {
				program.Send(msg)
			}
			views := make(chan string, 1)
			program.Send(formBarrier{views})
			var view string
			select {
			case view = <-views:
			case <-ctx.Done():
				t.Fatal("review timed out")
			}
			after, err := os.ReadFile(statePath)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(view, "Review register") || runner.calls.Load() != 0 || !bytes.Equal(before, after) {
				t.Fatalf("preparation changed state or executed:\n%s", view)
			}
			program.Send(press(tea.KeyEnter))
			var command app.Command
			select {
			case command = <-runner.commands:
			case <-ctx.Done():
				t.Fatal("handoff never emitted a probe")
			}
			args := strings.Join(command.Args, "\n")
			for _, want := range []string{"-l\nme", "--\nexample.test", "-p\n2222", "KEYFLOW_PROBE"} {
				if !strings.Contains(args, want) {
					t.Fatalf("wrong target/probe: %s", args)
				}
			}
			if command.Name != "ssh" || !command.Interactive {
				t.Fatalf("wrong command: %+v", command)
			}
			if mode == "key" {
				if !strings.Contains(args, "-i\n"+identity) || !strings.Contains(args, "PreferredAuthentications=publickey") {
					t.Fatalf("key intent lost: %s", args)
				}
			} else if strings.Contains(args, identity) || strings.Contains(args, "-i\n") || !strings.Contains(args, "PreferredAuthentications=keyboard-interactive,password") {
				t.Fatalf("stale identity/password intent: %s", args)
			}
			program.Send(tea.Quit())
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestFormsLongValuesAndErrorsAreVerticallyAccessible(t *testing.T) {
	home := t.TempDir()
	path := "C:/begin-part/" + strings.Repeat("directory/", 25) + "final-public-key.pub"
	journey := []tea.Msg{press('i'), text("import label"), press(tea.KeyTab), text(path)}
	for _, size := range [][2]int{{80, 24}, {60, 18}, {32, 12}} {
		top := runSizedScreen(t, home, size[0], size[1], append(journey, press(tea.KeyPgUp))...)
		if !strings.Contains(top, "C:/begin-part/") {
			t.Fatalf("path beginning requires horizontal scrolling at %v:\n%s", size, top)
		}
		screen := runSizedScreen(t, home, size[0], size[1], append(journey, press(tea.KeyPgDown), press(tea.KeyPgDown), press(tea.KeyPgDown))...)
		if !strings.Contains(strings.ReplaceAll(screen, "\n", ""), "final-public-key.pub") || lipgloss.Width(screen) > size[0] || lipgloss.Height(screen) > size[1] {
			t.Fatalf("long path inaccessible at %v:\n%s", size, screen)
		}
	}
}

func TestFormsReresolveAgainBeforeHandoff(t *testing.T) {
	home, doc := navigationHome(t)
	options := []tea.ProgramOption{tea.WithFilter(func(m tea.Model, msg tea.Msg) tea.Msg {
		if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "enter" && strings.Contains(m.View().Content, "Review register") {
			doc.Hosts[0].Hostname = "changed.example"
			saveNavigationState(t, home, doc)
		}
		return msg
	})}
	screen := runSizedScreenOptions(t, home, 80, 24, options, press('r'), press(tea.KeyEnter), reviewKey(), press(tea.KeyEnter))
	if strings.Contains(screen, "Review register") || !strings.Contains(screen, "changed or is missing") || !strings.Contains(screen, "draft retained") {
		t.Fatalf("handoff accepted changed target:\n%s", screen)
	}
}
