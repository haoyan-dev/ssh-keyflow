package tui_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/state"
)

// Optional plain-text evidence from the same public-event journeys used below.
// KEYFLOW_CAPTURE is a test-only destination, never a product setting.
func TestCaptureReadability(t *testing.T) {
	root := os.Getenv("KEYFLOW_CAPTURE")
	if root == "" {
		t.Skip("capture directory not requested")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	home, _ := navigationHome(t)
	pages := map[string][]tea.Msg{
		"destinations":  nil,
		"keys":          {press(tea.KeyTab)},
		"registrations": {press(tea.KeyTab), press(tea.KeyTab)},
		"tasks":         {press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab)},
		"destination":   {press(tea.KeyEnter)},
		"task":          {press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyEnter)},
		"picker":        {press('r')},
		"add-start":     {press('a')},
		"add-os":        destinationDraft("linux"),
		"register":      {press('r'), press(tea.KeyEnter)},
		"review":        {press('r'), press(tea.KeyEnter), reviewKey()},
		"create-review": {press('c'), text("Work laptop"), reviewKey()},
		"discard":       {press('c'), text("Work laptop"), interruptKey()},
		"help":          {press('?')},
		"history":       {press('h')},
		"discovery":     {press('g')},
	}
	for _, size := range [][2]int{{80, 24}, {60, 18}, {32, 12}} {
		for name, events := range pages {
			screen := runSizedScreen(t, home, size[0], size[1], events...)
			path := filepath.Join(root, name+"-"+sizeName(size)+".txt")
			if err := os.WriteFile(path, []byte(ansi.Strip(screen)+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestReadableFirstViewports(t *testing.T) {
	home, _ := navigationHome(t)
	compact := compactPanelText
	for _, size := range [][2]int{{80, 24}, {60, 18}, {32, 12}} {
		t.Run(sizeName(size), func(t *testing.T) {
			for _, tc := range []struct {
				journey []tea.Msg
				wants   []string
			}{
				{[]tea.Msg{press('r')}, []string{"Register > Choose key", "Destination: build-1", "1/9", "> Laptop [key-1]", "public only"}},
				{[]tea.Msg{press(tea.KeyEnter)}, []string{"Destination: build-1", "Saved status", "Key: Laptop", "Registration: registered", "Connection: failed", "> r Register"}},
				{[]tea.Msg{press(tea.KeyTab), press(tea.KeyTab)}, []string{"> build-1 / Laptop", "registered", "Connection: failed"}},
				{[]tea.Msg{press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab)}, []string{"key missing/unreadable", "Enter details"}},
				{[]tea.Msg{press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyEnter)}, []string{"Task", "public key missing/unreadable", "v technical", "> v Read full reason"}},
				{[]tea.Msg{press('h')}, []string{"Scope: All recorded", "No recorded events in this view."}},
				{[]tea.Msg{press('?')}, []string{"Help: Destinations", "Actions here", "r Register"}},
			} {
				screen := runSizedScreen(t, home, size[0], size[1], tc.journey...)
				if lipgloss.Width(screen) > size[0] || lipgloss.Height(screen) > size[1] {
					t.Fatalf("viewport overflow:\n%s", screen)
				}
				for _, want := range tc.wants {
					if !strings.Contains(compact(screen), compact(want)) {
						t.Fatalf("missing first-viewport information %q:\n%s", want, screen)
					}
				}
			}
		})
	}
}

func TestLongDuplicateKeyNamesRetainDiscriminators(t *testing.T) {
	home, document := navigationHome(t)
	document.Keys[0].Name = strings.Repeat("公開鍵", 20)
	document.Keys[1].Name = document.Keys[0].Name
	saveNavigationState(t, home, document)
	screen := runSizedScreen(t, home, 32, 12, press('r'))
	for _, want := range []string{"[key-1]", "[key-2]", "public only", "Destination: build-1"} {
		if !strings.Contains(screen, want) {
			t.Fatalf("narrow picker lost disambiguation %q:\n%s", want, screen)
		}
	}
}

func TestVisiblePagesMatchKeyboardPaging(t *testing.T) {
	home, _ := navigationHome(t)
	for _, size := range [][2]int{{80, 24}, {60, 18}, {32, 12}} {
		for _, picker := range []bool{false, true} {
			journey := []tea.Msg{}
			if picker {
				journey = append(journey, press('r'))
			}
			first := runSizedScreen(t, home, size[0], size[1], journey...)
			row := "alice@server-"
			if picker {
				row = "public only"
			}
			capacity := strings.Count(first, row)
			if capacity < 1 || capacity >= 9 {
				t.Fatalf("fixture does not span pages:\n%s", first)
			}
			next := append(append([]tea.Msg{}, journey...), press(tea.KeyPgDown))
			screen := runSizedScreen(t, home, size[0], size[1], next...)
			if !strings.Contains(screen, fmt.Sprintf("%d/9", capacity+1)) {
				t.Fatalf("PageDown skipped visible rows at %v (picker %v):\n%s", size, picker, screen)
			}
			back := runSizedScreen(t, home, size[0], size[1], append(next, press(tea.KeyPgUp))...)
			if back != first {
				t.Fatalf("PageUp lost page/selection at %v (picker %v)", size, picker)
			}
			next = append(next, press(tea.KeyEnter))
			if picker {
				next = append(next, reviewKey())
			} else {
				next = append(next, press('v'))
			}
			target := fmt.Sprintf("host-%d", capacity+1)
			if picker {
				target = fmt.Sprintf("key-%d", capacity+1)
			}
			// Start at the tested size, then widen after choosing for references.
			screen = runSizedScreen(t, home, size[0], size[1], append(next, tea.WindowSizeMsg{Width: 80, Height: 24})...)
			if !strings.Contains(screen, target) {
				t.Fatalf("paging changed fixed target %s:\n%s", target, screen)
			}
		}
	}
}

func TestLongPickerTitleRetainsPositionWhenPaging(t *testing.T) {
	home, document := navigationHome(t)
	for i := range document.Registrations {
		document.Registrations[i].HostID = document.Hosts[0].ID
	}
	saveNavigationState(t, home, document)
	first := runSizedScreen(t, home, 32, 12, press('u'))
	if !strings.Contains(first, "1/9") {
		t.Fatalf("long picker title hid position:\n%s", first)
	}
	capacity := strings.Count(first, "registered")
	if capacity < 1 || capacity >= 9 {
		t.Fatalf("long picker fixture did not span pages:\n%s", first)
	}
	next := runSizedScreen(t, home, 32, 12, press('u'), press(tea.KeyPgDown))
	if !strings.Contains(next, fmt.Sprintf("%d/9", capacity+1)) {
		t.Fatalf("long picker paging lost position:\n%s", next)
	}
}

func TestSavedWorkAndUnregisterPickersKeepDistinctReferences(t *testing.T) {
	home, document := navigationHome(t)
	document.Registrations = document.Registrations[:2]
	for i := range document.Registrations {
		document.Registrations[i].HostID = document.Hosts[0].ID
		document.Registrations[i].Stage = "pending"
	}
	saveNavigationState(t, home, document)
	for _, size := range [][2]int{{80, 24}, {60, 18}, {32, 12}} {
		for _, action := range []rune{'s', 'u'} {
			screen := runSizedScreen(t, home, size[0], size[1], press(action))
			for _, reference := range []string{"[registration-1]", "[registration-2]"} {
				if !strings.Contains(ansi.Strip(screen), reference) {
					t.Fatalf("%c picker lost distinguishing %s at %v:\n%s", action, reference, size, screen)
				}
			}
			// The second visible reference must still select its exact saved ID.
			review := runSizedScreen(t, home, size[0], size[1], press(action), press(tea.KeyDown), press(tea.KeyEnter), reviewKey(), tea.WindowSizeMsg{Width: 80, Height: 24}, press(tea.KeyHome))
			if !strings.Contains(compactPanelText(review), "registration-2") {
				t.Fatalf("%c picker changed the chosen target at %v:\n%s", action, size, review)
			}
		}
	}
}

func TestResizedDraftOverlaysRevealOriginalFocus(t *testing.T) {
	home, _ := navigationHome(t)
	start := []tea.Msg{press('r'), press(tea.KeyEnter), press(tea.KeyTab), press(tea.KeyTab)}
	for _, tc := range []struct {
		name, want string
		overlay    []tea.Msg
		back       tea.Msg
	}{
		{"review", "> Auth:", []tea.Msg{reviewKey()}, press('e')},
		{"discard", "> Auth:", []tea.Msg{press(tea.KeyRight), interruptKey()}, press(tea.KeyEscape)},
		{"help", "> [Help]", []tea.Msg{press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyEnter)}, press(tea.KeyEscape)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			journey := append(append([]tea.Msg{}, start...), tc.overlay...)
			journey = append(journey, tea.WindowSizeMsg{Width: 32, Height: 12}, tc.back)
			screen := runSizedScreen(t, home, 80, 24, journey...)
			if !strings.Contains(screen, tc.want) || lipgloss.Height(screen) > 12 || lipgloss.Width(screen) > 32 {
				t.Fatalf("resized return hid focus:\n%s", screen)
			}
		})
	}
}

func TestLongAuthenticationValueKeepsFormContextAndGuidance(t *testing.T) {
	home, _ := navigationHome(t)
	path := "C:/alternate/" + strings.Repeat("directory/", 25) + "identity-tail"
	journey := []tea.Msg{press('u'), press(tea.KeyTab), press(tea.KeyRight), press(tea.KeyRight), press(tea.KeyTab), tea.PasteMsg{Content: path}}
	journey = append(journey, press(tea.KeyPgUp), press(tea.KeyPgUp), press(tea.KeyPgUp))
	var seen strings.Builder
	for page := 0; page < 14; page++ {
		screen := runSizedScreen(t, home, 32, 12, journey...)
		if lipgloss.Height(screen) > 12 || lipgloss.Width(screen) > 32 {
			t.Fatalf("long authentication form overflow:\n%s", screen)
		}
		seen.WriteString(compactPanelText(screen))
		journey = append(journey, press(tea.KeyPgDown))
	}
	for _, want := range []string{"Registration to remove", "Alternate private key path", "excluding the target fingerprint", "identity-tail", "[Continue]", "[Help]", "[Cancel]"} {
		if !strings.Contains(seen.String(), strings.Join(strings.Fields(want), "")) {
			t.Fatalf("long field hid %q", want)
		}
	}
}

func TestUnregisterPickerNamesFixedOrigin(t *testing.T) {
	home, document := navigationHome(t)
	document.Registrations[1].HostID = document.Hosts[0].ID
	saveNavigationState(t, home, document)
	screen := runSizedScreen(t, home, 32, 12, press('u'))
	if !strings.Contains(screen, "Destination: build-1") || !strings.Contains(screen, "Unregister > Choose") {
		t.Fatalf("unregister picker lost origin:\n%s", screen)
	}
}

func TestAddKeyPickerReopenRetainsDestinationContext(t *testing.T) {
	home, _ := navigationHome(t)
	journey := append(destinationDraft("linux"), press(tea.KeyTab), press(tea.KeyEnter), press(tea.KeyTab), press(tea.KeyEnter), press(tea.KeyEnter))
	// Returning from the picker focuses Continue; step back to Managed key.
	journey = append(journey, tea.KeyPressMsg(tea.Key{Code: tea.KeyTab, Mod: tea.ModShift}), press(tea.KeyEnter))
	for _, size := range [][2]int{{80, 24}, {60, 18}, {32, 12}} {
		screen := runSizedScreen(t, home, size[0], size[1], journey...)
		if !strings.Contains(screen, "Destination: 作業-box") || strings.Contains(screen, "Key: Laptop") {
			t.Fatalf("reopened Add picker lost draft destination:\n%s", screen)
		}
	}
}

func TestSameNameRegistrationsRemainVisuallyDistinctAndTargetBound(t *testing.T) {
	home, document := navigationHome(t)
	document.Registrations[1].HostID = document.Hosts[0].ID
	document.Keys[0].Name = strings.Repeat("公開鍵", 20)
	document.Keys[1].Name = document.Keys[0].Name
	saveNavigationState(t, home, document)
	for _, size := range [][2]int{{80, 24}, {60, 18}, {32, 12}} {
		journey := []tea.Msg{press(tea.KeyTab), press(tea.KeyTab)}
		screen := runSizedScreen(t, home, size[0], size[1], journey...)
		for _, want := range []string{"[registration-1]", "Connection: failed"} {
			if !strings.Contains(screen, want) {
				t.Fatalf("same-name registration lost %q at %v:\n%s", want, size, screen)
			}
		}
		second := runSizedScreen(t, home, size[0], size[1], append(journey, press(tea.KeyDown))...)
		if !strings.Contains(second, "[registration-2]") {
			t.Fatalf("second same-name registration lost discriminator:\n%s", second)
		}
		journey = append(journey, press(tea.KeyDown), press(tea.KeyEnter), press('v'), tea.WindowSizeMsg{Width: 100, Height: 80})
		screen = runSizedScreen(t, home, size[0], size[1], journey...)
		for _, want := range []string{"Registration ID: registration-2", "Key ID: key-2", "Destination ID: host-1"} {
			if !strings.Contains(screen, want) {
				t.Fatalf("same-name registration changed target %q:\n%s", want, screen)
			}
		}
	}
}

func TestReadOnlyResultTasksMatchNextStepsAndHistoryScope(t *testing.T) {
	home, runner := resultHome(t, true)
	options := app.Options{Home: home, Runner: runner, Output: io.Discard, Confirm: func(string) (bool, error) { return true, nil },
		Request: &app.Request{Operation: "register", Host: "result", Key: "Result fixture", Bootstrap: "password"}}
	if err := app.Run(t.Context(), options); err != nil {
		t.Fatal(err)
	}
	view, err := app.Snapshot(options)
	if err != nil {
		t.Fatal(err)
	}
	document := view.State
	document.Hosts[0].ConfigKeyID = document.Keys[0].ID
	document.Configs = []state.Config{{Path: filepath.Join(state.Directory(home), "config"), SHA256: strings.Repeat("a", 64), PendingSHA256: strings.Repeat("b", 64)}}
	type pendingConfiguration struct{}
	native := &handoffIO{answer: strings.NewReader("yes\n")}
	session := startHandoff(t, app.Options{Home: home, Runner: runner, Input: native, Output: native}, func(_ tea.Model, msg tea.Msg) tea.Msg {
		if _, ok := msg.(pendingConfiguration); ok {
			saveNavigationState(t, home, document)
			return press(tea.KeyF5)
		}
		return msg
	})
	session.send(press('r'), press(tea.KeyEnter), reviewKey(), press(tea.KeyEnter))
	session.returned(t)
	session.send(pendingConfiguration{})
	screen := session.screen(t)
	if !strings.Contains(screen, "no supported continuation") || strings.Contains(screen, "Continue saved work") {
		t.Fatalf("read-only task offered continuation:\n%s", screen)
	}
	session.send(press('v'), press('h'))
	screen = session.screen(t)
	if !strings.Contains(screen, "Scope: registration: result / Result fixture") {
		t.Fatalf("result entity history scope missing:\n%s", screen)
	}
}

func TestResultInstructionsAreDirectReadOnlyAndRestoreOrigin(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {60, 18}, {32, 12}} {
		t.Run(sizeName(size), func(t *testing.T) {
			home, _ := handoffHome(t)
			var probes atomic.Int32
			native := &handoffIO{answer: strings.NewReader("yes\n")}
			session := startHandoff(t, app.Options{Home: home, Runner: navigationRunner{&probes}, Input: native, Output: native}, nil)
			session.send(press('r'), press(tea.KeyEnter), reviewKey(), press(tea.KeyEnter))
			session.returned(t)
			session.send(tea.WindowSizeMsg{Width: size[0], Height: size[1]}, press(tea.KeyDown))
			origin := session.screen(t)
			if !strings.Contains(origin, "o instructions") || !strings.Contains(origin, "Tab 1/") {
				t.Fatalf("instructions or action position hidden:\n%s", origin)
			}
			session.send(press('o'))
			instructions := session.screen(t)
			if !strings.Contains(instructions, "Session output") || !strings.Contains(compactPanelText(instructions), "notsavedafterexit") {
				t.Fatalf("instructions boundary missing:\n%s", instructions)
			}
			captureGroupedScreen(t, "instructions", size, false, instructions)
			if root := os.Getenv("KEYFLOW_CAPTURE"); root != "" {
				if err := os.WriteFile(filepath.Join(root, "instructions-"+sizeName(size)+".txt"), []byte(ansi.Strip(instructions)), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, "result-"+sizeName(size)+".txt"), []byte(ansi.Strip(origin)), 0600); err != nil {
					t.Fatal(err)
				}
			}
			session.send(press(tea.KeyEnd))
			captureGroupedScreen(t, "instructions", size, true, session.screen(t))
			session.send(press('?'), press(tea.KeyEscape), press('r'), press('s'))
			if !strings.Contains(session.screen(t), "read-only") {
				t.Fatal("instruction document accepted operation keys")
			}
			session.send(press(tea.KeyEscape))
			// The blocked action notice is separate from the restored viewport.
			session.send(press('o'), press(tea.KeyEscape))
			restored := session.screen(t)
			if strings.Split(restored, "\n")[2] != strings.Split(origin, "\n")[2] {
				t.Fatalf("instructions lost result scroll origin:\n%s", restored)
			}
			if probes.Load() != 0 {
				t.Fatal("manual result or document navigation executed a command")
			}
		})
	}
}

func sizeName(size [2]int) string {
	switch size[0] {
	case 80:
		return "80x24"
	case 60:
		return "60x18"
	default:
		return "32x12"
	}
}
