package tui_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/state"
)

func pendingOnboardingHome(t *testing.T) (string, state.State) {
	t.Helper()
	home := t.TempDir()
	path := filepath.Join(state.Directory(home), "keys", "reserved")
	doc := state.State{Version: 1, Keys: []state.Key{}, Registrations: []state.Registration{},
		Hosts:       []state.Host{{ID: "host-saved", Alias: "saved-build", Hostname: "saved.example", User: "alice", Port: 2222, OS: "linux"}},
		Creations:   []state.Creation{{ID: "creation-saved", KeyID: "key-reserved", Name: "Saved new key", PrivatePath: path, PublicPath: path + ".pub", CreatedAt: "2026-11-07T00:00:00Z", Stage: "pending"}},
		Onboardings: []state.Onboarding{{ID: "onboarding-saved", HostID: "host-saved", KeyID: "key-reserved", CreationID: "creation-saved", Stage: "key_pending", CreatedAt: "2026-11-07T00:00:00Z"}}}
	saveNavigationState(t, home, doc)
	return home, doc
}

func TestReviewFixPendingOnboardingReachesReviewAndHandsOffSavedIntent(t *testing.T) {
	home, doc := pendingOnboardingHome(t)
	path := filepath.Join(state.Directory(home), "state.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var probes atomic.Int32
	native := &handoffIO{answer: strings.NewReader("no\n")}
	session := startHandoff(t, app.Options{Home: home, Input: native, Output: native, Runner: navigationRunner{&probes}}, nil)
	session.send(press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyEnter), press('s'), reviewKey())
	screen := session.screen(t)
	if !strings.Contains(screen, "Review resume") || !strings.Contains(screen, "Saved operation: onboarding-saved") {
		t.Fatalf("valid reserved identity cannot reach Review:\n%s", screen)
	}
	session.send(press(tea.KeyEnter))
	session.returned(t)
	native.Lock()
	output := native.output.String()
	native.Unlock()
	if !strings.Contains(output, "Saved new key") || !strings.Contains(output, doc.Creations[0].PrivatePath) {
		t.Fatalf("handoff lost saved creation intent: %s", output)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if probes.Load() != 0 || !bytes.Equal(before, after) {
		t.Fatal("denied saved intent executed or rewrote state")
	}
	if _, err := os.Stat(doc.Creations[0].PrivatePath); !os.IsNotExist(err) {
		t.Fatal("preparation or denial created key")
	}
}

func TestReviewFixPasteEditsAtCaretAndRetainsDirtyDraftThroughHelp(t *testing.T) {
	home := t.TempDir()
	journey := []tea.Msg{press('c'), text("ac"), press(tea.KeyLeft), tea.PasteMsg{Content: "q?rd"}, press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyEnter), tea.PasteMsg{Content: "ignored help paste"}, press(tea.KeyEscape), press(tea.KeyTab), press(tea.KeyEnter)}
	screen := runScreen(t, home, journey...)
	if !strings.Contains(screen, "> Keep editing") {
		t.Fatalf("paste did not mark draft dirty: %s", screen)
	}
	journey = append(journey, press(tea.KeyEnter), reviewKey())
	screen = runScreen(t, home, journey...)
	if !strings.Contains(screen, "Management name: aq?rdc") || strings.Contains(screen, "ignored help paste") {
		t.Fatalf("paste/caret/help retention failed:\n%s", screen)
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh")); !os.IsNotExist(err) {
		t.Fatal("paste executed an operation")
	}
}

func TestReviewFixPasteFiltersListAndPickerAtCaret(t *testing.T) {
	for _, picker := range []bool{false, true} {
		t.Run(map[bool]string{false: "list", true: "picker"}[picker], func(t *testing.T) {
			home, _ := navigationHome(t)
			journey := []tea.Msg{}
			if picker {
				journey = append(journey, press('r'))
			}
			prefix, want := "host-", "Destination ID: host-7"
			if picker {
				prefix, want = "key-", "Laptop [key-7]"
			}
			journey = append(journey, press('/'), text(prefix+"x"), press(tea.KeyLeft), tea.PasteMsg{Content: "7"}, press(tea.KeyDelete), press(tea.KeyEnter), press(tea.KeyEnter))
			if picker {
				journey = append(journey, reviewKey())
			} else {
				journey = append(journey, press('v'))
			}
			screen := runScreen(t, home, journey...)
			if !strings.Contains(screen, want) {
				t.Fatalf("paste filter failed:\n%s", screen)
			}
		})
	}
	home, _ := navigationHome(t)
	screen := runScreen(t, home, press('/'), tea.PasteMsg{Content: "q?rd"}, press(tea.KeyEscape), press('?'), tea.PasteMsg{Content: "ignored"}, press(tea.KeyEscape))
	if !strings.Contains(screen, "q?rd") || !strings.Contains(screen, "No matching") || strings.Contains(screen, "ignored") {
		t.Fatalf("literal paste/search help retention failed:\n%s", screen)
	}
}

func TestReviewFixPasteClearsHiddenChoicesAndCannotEditFixedControls(t *testing.T) {
	home, _ := navigationHome(t)
	journey := []tea.Msg{press('r'), press(tea.KeyEnter), tea.PasteMsg{Content: "other-host"}, press(tea.KeyTab), tea.PasteMsg{Content: "other-key"}, press(tea.KeyTab), clearKey(), tea.PasteMsg{Content: "key"}, press(tea.KeyTab), tea.PasteMsg{Content: "C:/old-identity"}, tea.KeyPressMsg(tea.Key{Code: tea.KeyTab, Mod: tea.ModShift}), clearKey(), tea.PasteMsg{Content: "manual"}, reviewKey(), tea.PasteMsg{Content: "q?rc\n\rctrl+s"}}
	screen := runScreen(t, home, journey...)
	for _, want := range []string{"Review register", "host-1", "key-1", "Auth: manual"} {
		if !strings.Contains(screen, want) {
			t.Fatalf("missing %q:\n%s", want, screen)
		}
	}
	for _, unwanted := range []string{"other-host", "other-key", "C:/old-identity"} {
		if strings.Contains(screen, unwanted) {
			t.Fatalf("fixed/hidden value leaked %q:\n%s", unwanted, screen)
		}
	}
	journey = append(journey, press(tea.KeyEscape), press(tea.KeyRight), press(tea.KeyRight), reviewKey())
	screen = runScreen(t, home, journey...)
	if strings.Contains(screen, "Review register") || !strings.Contains(screen, "enter a private key path") {
		t.Fatalf("hidden identity was not cleared:\n%s", screen)
	}
	// Paste updates controlling OS immediately, clearing the now-inapplicable account.
	journey = destinationDraft("windows")
	// destinationDraft requires nonempty text, so replace the final message.
	journey[len(journey)-1] = tea.PasteMsg{Content: "windows"}
	journey = append(journey, press(tea.KeyTab), tea.PasteMsg{Content: "administrator"}, tea.KeyPressMsg(tea.Key{Code: tea.KeyTab, Mod: tea.ModShift}), clearKey(), tea.PasteMsg{Content: "linux"}, reviewKey(), press(tea.KeyEnter), press(tea.KeyEnter), reviewKey())
	screen = runScreen(t, home, journey...)
	if !strings.Contains(screen, "Review add") || strings.Contains(screen, "Windows account:") {
		t.Fatalf("pasted OS failed conditional clearing:\n%s", screen)
	}
}

func TestReviewFixPasteBelowMinimumRetainsFormAndSearch(t *testing.T) {
	for _, search := range []bool{false, true} {
		t.Run(map[bool]string{false: "form", true: "search"}[search], func(t *testing.T) {
			home, _ := navigationHome(t)
			journey := []tea.Msg{press('c'), tea.PasteMsg{Content: "saved"}}
			if search {
				journey = []tea.Msg{press('/'), tea.PasteMsg{Content: "host-7"}}
			}
			journey = append(journey, tea.WindowSizeMsg{Width: 31, Height: 11}, tea.PasteMsg{Content: "ignored"}, reviewKey(), press(tea.KeyEnter), tea.WindowSizeMsg{Width: 80, Height: 24})
			if search {
				journey = append(journey, press(tea.KeyEnter), press(tea.KeyEnter), press('v'))
			} else {
				journey = append(journey, reviewKey())
			}
			screen := runScreen(t, home, journey...)
			want := "Management name: saved"
			if search {
				want = "Destination ID: host-7"
			}
			if !strings.Contains(screen, want) || strings.Contains(screen, "ignored") {
				t.Fatalf("undersize paste changed input:\n%s", screen)
			}
		})
	}
}

func TestReviewFixResumeReviewShowsEverySavedIdentity(t *testing.T) {
	for _, kind := range []string{"creation", "deletion", "onboarding-new", "onboarding-existing"} {
		t.Run(kind, func(t *testing.T) {
			home, doc := pendingOnboardingHome(t)
			want := []string{"Review resume"}
			switch kind {
			case "creation":
				doc.Onboardings = nil
				want = append(want, "New Ed25519 key: Saved new key", "key-reserved", doc.Creations[0].PrivatePath, doc.Creations[0].PublicPath)
			case "onboarding-new":
				want = append(want, "saved-build", "alice@saved.example:2222", "New Ed25519 key: Saved new key", "creation-saved", doc.Creations[0].PrivatePath)
			default:
				home, doc = navigationHome(t)
				doc.Keys, doc.Hosts, doc.Registrations = doc.Keys[:1], doc.Hosts[:1], nil
				doc.Registrations = []state.Registration{}
				want = append(want, "Laptop [key-1]", doc.Keys[0].Fingerprint)
				if kind == "deletion" {
					doc.Deletions = []state.LocalDeletion{{ID: "deletion-saved", KeyID: "key-1", CreatedAt: "2026-11-07T00:00:00Z", Stage: "pending", PublicSHA256: strings.Repeat("a", 64), PrivateRemoved: true}}
				} else {
					doc.Onboardings = []state.Onboarding{{ID: "onboarding-existing", HostID: "host-1", KeyID: "key-1", Stage: "planned", CreatedAt: "2026-11-07T00:00:00Z"}}
					want = append(want, "build-1", "alice@server-1.example:22")
				}
			}
			saveNavigationState(t, home, doc)
			id := "onboarding-saved"
			if kind == "creation" {
				id = "creation-saved"
			}
			if kind == "deletion" {
				id = "Interrupted local deletion"
			}
			if kind == "onboarding-existing" {
				id = "onboarding-existing"
			}
			screen := runSizedScreen(t, home, 300, 60, press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab), press('/'), text(id), press(tea.KeyEnter), press(tea.KeyEnter), press('s'), reviewKey())
			for _, v := range want {
				if !strings.Contains(screen, v) {
					t.Fatalf("%s Review lacks %q:\n%s", kind, v, screen)
				}
			}
			if (kind == "creation" || kind == "deletion") && (strings.Contains(screen, "Auth:") || strings.Contains(screen, "SSH configuration:")) {
				t.Fatalf("local resume has remote options: %s", screen)
			}
		})
	}
}

func TestReviewFixRefreshKeepsExplicitResultDetailsAndOrigin(t *testing.T) {
	for _, link := range []rune{'v', 't'} {
		t.Run(string(link), func(t *testing.T) {
			home, view := handoffHome(t)
			type corruptState struct{}
			var probes atomic.Int32
			native := &handoffIO{answer: strings.NewReader("no\n")}
			session := startHandoff(t, app.Options{Home: home, Input: native, Output: native, Runner: navigationRunner{&probes}}, func(_ tea.Model, msg tea.Msg) tea.Msg {
				switch msg := msg.(type) {
				case corruptState:
					if err := os.WriteFile(filepath.Join(state.Directory(home), "state.json"), []byte("invalid fixture"), 0600); err != nil {
						t.Fatal(err)
					}
					return press(tea.KeyF5)
				case replaceSavedState:
					if err := msg.document.Validate(); err != nil {
						t.Errorf("invalid replacement fixture: %v", err)
						return nil
					}
					saveNavigationState(t, home, msg.document)
					return press(tea.KeyF5)
				}
				return msg
			})
			session.send(press('r'), press(tea.KeyEnter), reviewKey(), press(tea.KeyEnter))
			session.returned(t)
			session.send(tea.WindowSizeMsg{Width: 80, Height: 18}, press(tea.KeyDown), press(tea.KeyDown), press(tea.KeyDown))
			origin := strings.Split(session.screen(t), "\n")[2]
			session.send(press(link), press(tea.KeyDown), press(tea.KeyDown), press(tea.KeyDown))
			detail := strings.Split(session.screen(t), "\n")[2]
			session.send(press(tea.KeyF5))
			screen := session.screen(t)
			if !strings.Contains(screen, "esc back") || strings.Split(screen, "\n")[2] != detail || strings.Contains(screen, "Managed destinations") {
				t.Fatalf("refresh lost explicit details/scroll:\n%s", screen)
			}
			session.send(corruptState{}, press(tea.KeyF5))
			screen = session.screen(t)
			if !strings.Contains(screen, "stale") || !strings.Contains(screen, "Refresh failed") {
				t.Fatalf("failed reload hid stale truth:\n%s", screen)
			}
			session.send(replaceSavedState{view.State})
			screen = session.screen(t)
			if strings.Contains(screen, "stale") || strings.Split(screen, "\n")[2] != detail {
				t.Fatalf("recovery lost details/scroll:\n%s", screen)
			}
			session.send(press(tea.KeyEscape))
			screen = session.screen(t)
			if strings.Split(screen, "\n")[2] != origin || !strings.Contains(screen, "Tab 1/") {
				t.Fatalf("refresh lost result origin/scroll:\n%s", screen)
			}
			session.send(press(link))
			removed := view.State
			removed.Registrations = []state.Registration{}
			removed.Onboardings = nil
			removed.History = nil
			session.send(replaceSavedState{removed}, press('r'), press('s'), press('d'))
			screen = session.screen(t)
			if !strings.Contains(screen, "target no longer available") || strings.Contains(screen, "Review ") || strings.Contains(screen, "Register public key") {
				t.Fatalf("removed target not truthful/read-only:\n%s", screen)
			}
			session.send(press(tea.KeyEscape))
			screen = session.screen(t)
			if strings.Split(screen, "\n")[2] != origin || !strings.Contains(screen, "Tab 1/") {
				t.Fatalf("removed target lost result origin:\n%s", screen)
			}
			if probes.Load() != 0 {
				t.Fatal("refresh/navigation executed command")
			}
		})
	}
}

func TestReviewFixPendingOnboardingRejectsLinkedCreationDrift(t *testing.T) {
	for _, phase := range []string{"review", "handoff"} {
		t.Run(phase, func(t *testing.T) {
			home, doc := pendingOnboardingHome(t)
			type changeCreation struct{}
			var probes atomic.Int32
			native := &handoffIO{answer: strings.NewReader("no\n")}
			session := startHandoff(t, app.Options{Home: home, Input: native, Output: native, Runner: navigationRunner{&probes}}, func(_ tea.Model, msg tea.Msg) tea.Msg {
				if _, ok := msg.(changeCreation); ok {
					doc.Creations[0].Name = "Externally changed intent"
					saveNavigationState(t, home, doc)
					return nil
				}
				return msg
			})
			session.send(press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyEnter), press('s'))
			if phase == "handoff" {
				session.send(reviewKey())
				if !strings.Contains(session.screen(t), "Review resume") {
					t.Fatal("initial valid intent did not reach Review")
				}
			}
			session.send(changeCreation{})
			if phase == "handoff" {
				session.send(press(tea.KeyEnter))
			} else {
				session.send(reviewKey())
			}
			screen := session.screen(t)
			if !strings.Contains(screen, "changed or is missing") || strings.Contains(screen, "Review resume") {
				t.Fatalf("changed linked creation was accepted:\n%s", screen)
			}
			native.Lock()
			output := native.output.String()
			native.Unlock()
			if probes.Load() != 0 || strings.Contains(output, "Type yes") {
				t.Fatal("drift reached native execution")
			}
		})
	}
}

func TestReviewFixPasteDoesNotActivateFormControls(t *testing.T) {
	for _, tabs := range []int{1, 2, 3} {
		t.Run(string(rune('0'+tabs)), func(t *testing.T) {
			home := t.TempDir()
			journey := []tea.Msg{press('c'), tea.PasteMsg{Content: "saved"}}
			for i := 0; i < tabs; i++ {
				journey = append(journey, press(tea.KeyTab))
			}
			before := runScreen(t, home, journey...)
			journey = append(journey, tea.PasteMsg{Content: "q?rc\n\rctrl+s"})
			screen := runScreen(t, home, journey...)
			if screen != before {
				t.Fatalf("paste activated fixed control:\n%s", screen)
			}
			if _, err := os.Stat(filepath.Join(home, ".ssh")); !os.IsNotExist(err) {
				t.Fatal("paste executed a control")
			}
		})
	}
}

func TestReviewFixScrolledFormSurvivesCursorReplies(t *testing.T) {
	type captureReplies bool
	type deliverReply struct{ message tea.Msg }
	type unrelatedReply struct{}
	replies := make(chan tea.Msg, 32)
	holding := false
	var probes atomic.Int32
	session := startHandoff(t, app.Options{Home: t.TempDir(), Input: strings.NewReader(""), Output: io.Discard, Runner: navigationRunner{&probes}}, func(_ tea.Model, message tea.Msg) tea.Msg {
		switch message := message.(type) {
		case captureReplies:
			holding = bool(message)
			return nil
		case deliverReply:
			return message.message
		case tea.WindowSizeMsg:
			return tea.WindowSizeMsg{Width: 32, Height: 12}
		case tea.KeyPressMsg, tea.PasteMsg, tea.BatchMsg, tea.QuitMsg:
			return message
		}
		if holding {
			replies <- message
			return nil
		}
		return message
	})
	path := "C:/begin-part/" + strings.Repeat("directory/", 25) + "final-public-key.pub"
	session.send(tea.WindowSizeMsg{Width: 32, Height: 12}, press('i'), tea.PasteMsg{Content: "saved import"}, press(tea.KeyTab), tea.PasteMsg{Content: path})
	session.screen(t)
	// Hold natural textinput command replies as opaque public-program messages.
	scrollToTail := func(want string) string {
		for page := 0; page < 12; page++ {
			screen := session.screen(t)
			if strings.Contains(compactPanelText(screen), compactPanelText(want)) && !strings.Contains(ansi.Strip(screen), "┌ Import existing key") {
				return screen
			}
			session.send(press(tea.KeyPgDown))
		}
		t.Fatalf("path tail %q inaccessible", want)
		return ""
	}
	session.send(captureReplies(true), press(tea.KeyLeft))
	scrollToTail("final-public-key.pub")
	scrolled := session.screen(t)
	if !strings.Contains(compactPanelText(scrolled), "final-public-key.pub") || strings.Contains(strings.Join(strings.Split(scrolled, "\n")[2:], "\n"), "Import existing key") {
		t.Fatalf("fixture did not scroll to full path tail:\n%s", scrolled)
	}
	var delayed []tea.Msg
	deadline := time.NewTimer(1200 * time.Millisecond)
	defer deadline.Stop()
collect:
	for {
		select {
		case reply := <-replies:
			delayed = append(delayed, reply)
		case <-deadline.C:
			break collect
		case <-session.ctx.Done():
			t.Fatal("cursor command replies timed out")
		}
	}
	if len(delayed) == 0 {
		t.Fatal("no natural cursor command reply was observed")
	}
	for _, reply := range delayed {
		session.send(deliverReply{reply})
	}
	if screen := session.screen(t); screen != scrolled {
		t.Fatalf("cursor reply reset scrolled editable path:\nbefore=%s\nafter=%s", scrolled, screen)
	}
	session.send(deliverReply{unrelatedReply{}})
	if screen := session.screen(t); screen != scrolled {
		t.Fatalf("unrelated response reset viewport:\n%s", screen)
	}
	// A reply issued by the public-path input must not disturb the next input.
	session.send(press(tea.KeyTab), tea.PasteMsg{Content: path + ".private"})
	scrollToTail("final-public-key.pub.private")
	nextField := session.screen(t)
	if !strings.Contains(compactPanelText(nextField), "final-public-key.pub.private") {
		t.Fatalf("next editable field was not scrolled:\n%s", nextField)
	}
	for _, reply := range delayed {
		session.send(deliverReply{reply})
	}
	if screen := session.screen(t); screen != nextField {
		t.Fatalf("stale input reply changed next field viewport:\n%s", screen)
	}
	session.send(press(tea.KeyLeft))
	if screen := session.screen(t); !strings.Contains(screen, "> Private key path") {
		t.Fatalf("actual caret navigation did not reveal focused field:\n%s", screen)
	}
	session.send(press(tea.KeyPgDown), press(tea.KeyPgDown), press(tea.KeyPgDown), tea.PasteMsg{Content: "q?"})
	if screen := session.screen(t); !strings.Contains(screen, "> Private key path") {
		t.Fatalf("actual edit did not reveal focused field:\n%s", screen)
	}
	if probes.Load() != 0 {
		t.Fatal("cursor/edit journey executed an operation")
	}
}
