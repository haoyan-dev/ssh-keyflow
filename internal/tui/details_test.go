package tui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"ssh-keyflow/internal/state"
)

func TestDetailPanelsAndSemanticColorsRetainTextLabels(t *testing.T) {
	home, document := navigationHome(t)
	document.Registrations[0].Status = "pending"
	document.Registrations[0].Stage = "pending"
	document.Registrations[0].ConnectionCheck = "unknown"
	document.Registrations = append(document.Registrations, state.Registration{
		ID: "registration-extra", HostID: "host-1", KeyID: "key-2",
		Status: "registered", Operation: "register", Stage: "complete", ConnectionCheck: "verified",
	})
	saveNavigationState(t, home, document)
	screen := runSizedScreen(t, home, 100, 100, press(tea.KeyEnter))
	for _, want := range []string{
		"\x1b[1;36mSaved status", "\x1b[1;36mNext actions",
		"\x1b[32m1 registered", "\x1b[32m1 verified previously",
		"\x1b[33m1 awaiting registration", "\x1b[33m1 outcome unknown",
		"┌", "┐", "└", "┘",
	} {
		if !strings.Contains(screen, want) {
			t.Fatalf("missing visual cue %q:\n%q", want, screen)
		}
	}
	plain := ansi.Strip(screen)
	for _, want := range []string{"Registration:", "Connection:", "Saved evidence only; not a live SSH check.", "Next actions", "r Register public key", "Saved work unfinished"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("monochrome lost meaning %q:\n%s", want, plain)
		}
	}
	key := runSizedScreen(t, home, 100, 100, press(tea.KeyTab), press(tea.KeyEnter))
	if !strings.Contains(key, "\x1b[31mLocal deletion blocked:") {
		t.Fatalf("blocked action lost red cue:\n%q", key)
	}
	document.Registrations[0].Status = "failed"
	document.Registrations[0].ConnectionCheck = "failed"
	saveNavigationState(t, home, document)
	failed := runSizedScreen(t, home, 100, 100, press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyEnter))
	for _, want := range []string{"\x1b[31mRegistration: attempt failed", "\x1b[31mConnection: failed at last check"} {
		if !strings.Contains(failed, want) {
			t.Fatalf("failed evidence lost cue %q:\n%q", want, failed)
		}
	}
	document.Registrations[0].Status = "removed"
	document.Registrations[0].Operation = "unregister"
	document.Registrations[0].Stage = "complete"
	saveNavigationState(t, home, document)
	removed := runSizedScreen(t, home, 100, 100, press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyEnter))
	if !strings.Contains(removed, "Registration: removed (recorded)") ||
		strings.Contains(removed, "\x1b[32mRegistration: removed") {
		t.Fatalf("removed registration gained success coloring:\n%q", removed)
	}
}

func TestDetailPanelsReflowAcrossResizedDocumentOrigins(t *testing.T) {
	home, document := navigationHome(t)
	document.Hosts[0].Alias = strings.Repeat("東京", 20)
	document.Keys[0].Name = strings.Repeat("公開鍵", 15)
	document.Hosts[0].ConfigKeyID = document.Keys[0].ID
	saveNavigationState(t, home, document)
	path := filepath.Join(state.Directory(home), "state.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for tabs := 0; tabs < 4; tabs++ {
		var open []tea.Msg
		for i := 0; i < tabs; i++ {
			open = append(open, press(tea.KeyTab))
		}
		open = append(open, press(tea.KeyEnter))
		for _, widths := range [][2]int{{100, 60}, {80, 59}, {80, 32}, {32, 80}} {
			t.Run(fmt.Sprintf("kind%d/%dto%d", tabs, widths[0], widths[1]), func(t *testing.T) {
				resize := tea.WindowSizeMsg{Width: widths[1], Height: 80}
				expected := runSizedScreen(t, home, widths[1], 80, open...)
				for _, overlay := range [][]tea.Msg{nil, {press('?')}, {press('v'), press('?')}} {
					journey := append([]tea.Msg{}, open...)
					journey = append(journey, overlay...)
					journey = append(journey, resize)
					for range overlay {
						journey = append(journey, press(tea.KeyEscape))
					}
					actual := runSizedScreen(t, home, widths[0], 80, journey...)
					if actual != expected {
						t.Fatalf("resized origin differs (overlay %v):\nexpected=%s\nactual=%s", overlay, expected, actual)
					}
					plain := ansi.Strip(actual)
					panelCount := 0
					if widths[1] >= 60 {
						panelCount = 2
					}
					if strings.Count(plain, "┌") != panelCount {
						t.Fatalf("panels did not collapse or spread to related rows:\n%s", plain)
					}
					if lipgloss.Width(actual) > widths[1] || lipgloss.Height(actual) > 80 {
						t.Fatalf("resized panel overflow:\n%s", actual)
					}
					compact := strings.Join(strings.Fields(plain), "")
					// Borders are layout, not characters in a wrapped saved value.
					compact = strings.ReplaceAll(compact, "│", "")
					if tabs < 3 && !strings.Contains(compact, document.Keys[0].Name) {
						t.Fatalf("wrapped Unicode key name lost:\n%s", plain)
					}
				}
			})
		}
	}
	// Returning from a resized overlay retains the same scroll offset as a
	// direct resize, rather than jumping back to the overview's identity.
	open := []tea.Msg{press(tea.KeyEnter), press(tea.KeyPgDown)}
	resize := tea.WindowSizeMsg{Width: 32, Height: 12}
	direct := runSizedScreen(t, home, 80, 24, append(append([]tea.Msg{}, open...), resize)...)
	back := runSizedScreen(t, home, 80, 24, append(open, press('v'), press('?'), resize, press(tea.KeyEscape), press(tea.KeyEscape))...)
	if back != direct {
		t.Fatalf("resized overlay lost scroll:\ndirect=%s\nback=%s", direct, back)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatal("panel/resize navigation rewrote saved state")
	}
}

func TestDetailOverviewsExplainIdentityEvidenceAndActions(t *testing.T) {
	home, document := navigationHome(t)
	document.Keys[1].Name = "Desktop"
	document.Registrations[0].Status = "unknown"
	document.Registrations[0].Stage = "pending"
	document.Registrations = append(document.Registrations, state.Registration{
		ID: "registration-extra", HostID: "host-1", KeyID: "key-2",
		Status: "registered", Operation: "register", Stage: "complete", ConnectionCheck: "verified",
	})
	saveNavigationState(t, home, document)
	for _, tc := range []struct {
		name    string
		journey []tea.Msg
		wants   []string
	}{
		{"destination", []tea.Msg{press(tea.KeyEnter)}, []string{"Destination: build-1", "alice@server-1.example:22", "Registration: 1 registered, 1 outcome unknown", "Connection: 1 verified previously, 1 failed at last check", "Key registrations", "Laptop", "Desktop"}},
		{"key", []tea.Msg{press(tea.KeyTab), press(tea.KeyEnter)}, []string{"Key: Laptop", "public only", "private-key verification unavailable", "Local deletion blocked", "Destinations"}},
		{"registration", []tea.Msg{press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyEnter)}, []string{"Registration: build-1 / Laptop", "Registration: outcome unknown", "Connection: failed at last check", "r Register this destination/key pair", "s Continue saved work"}},
		{"task", []tea.Msg{press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab), press('/'), text("registration-1"), press(tea.KeyEnter), press(tea.KeyEnter)}, []string{"Task", "Destination: build-1", "Key: Laptop", "reconcile remote state before retry", "s Continue"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			screen := ansi.Strip(runSizedScreen(t, home, 100, 80, tc.journey...))
			for _, want := range append(tc.wants, "Saved status", "Next actions", "not a live SSH check", "v technical") {
				if !strings.Contains(screen, want) {
					t.Fatalf("missing %q:\n%s", want, screen)
				}
			}
			for _, hidden := range []string{"Destination ID:", "Key ID:", "Source IDs:", "Private path:", "Public path:", "Saved digest:"} {
				if strings.Contains(screen, hidden) {
					t.Fatalf("technical metadata %q leaked into overview:\n%s", hidden, screen)
				}
			}
			if strings.Index(screen, "Saved status") > strings.Index(screen, "Next actions") {
				t.Fatal("actions precede saved status")
			}
			if tc.name == "destination" && (strings.Count(screen, "Laptop · SHA256:") != 1 || strings.Count(screen, "Desktop · SHA256:") != 1 || strings.Contains(screen, "d delete")) {
				t.Fatalf("repeated relationship or irrelevant deletion:\n%s", screen)
			}
			for _, size := range [][2]int{{80, 24}, {60, 18}, {32, 12}} {
				for _, journey := range [][]tea.Msg{tc.journey, append(append([]tea.Msg{}, tc.journey...), press('v'))} {
					frame := runSizedScreen(t, home, size[0], size[1], journey...)
					if lipgloss.Width(frame) > size[0] || lipgloss.Height(frame) > size[1] {
						t.Fatalf("entry overflow at %v:\n%s", size, frame)
					}
					if tc.name == "task" && size[0] == 32 && len(journey) == len(tc.journey) && !strings.Contains(frame, "s Continue · v technical\n↑/↓ scroll · ? help") {
						t.Fatalf("compact task controls split across lines:\n%s", frame)
					}
				}
			}
		})
	}
}

func TestDetailConnectionStatesRemainDestinationSpecific(t *testing.T) {
	for _, tc := range []struct{ saved, want, color string }{
		{"verified", "verified previously", "32"}, {"failed", "failed at last check", "31"},
		{"unverified", "not checked", "33"}, {"unknown", "outcome unknown", "33"},
	} {
		t.Run(tc.saved, func(t *testing.T) {
			home, document := navigationHome(t)
			// The key's legacy aggregate says verified; only the pair is authoritative.
			document.Keys[0].PrivatePath = filepath.Join(home, "private-key")
			document.Keys[0].PrivateSHA256 = strings.Repeat("a", 64)
			document.Registrations[0].ConnectionCheck = tc.saved
			saveNavigationState(t, home, document)
			styled := runScreen(t, home, press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyEnter))
			screen := ansi.Strip(styled)
			if !strings.Contains(styled, "\x1b["+tc.color+"mConnection: "+tc.want) {
				t.Fatalf("connection evidence lost color:\n%q", styled)
			}
			if !strings.Contains(screen, "Registration: registered (recorded)") ||
				!strings.Contains(screen, "Connection: "+tc.want+" (recorded)") ||
				!strings.Contains(screen, "not a live SSH check") {
				t.Fatalf("misleading evidence:\n%s", screen)
			}
			if tc.saved != "verified" && strings.Contains(screen, "verified previously") {
				t.Fatalf("key-wide verification replaced pair evidence:\n%s", screen)
			}
		})
	}
}

func TestDetailConfigurationBindingAndSharedEvidenceAreSeparate(t *testing.T) {
	home, document := navigationHome(t)
	document.Keys[1].Name = "Other key"
	document.Hosts[0].ConfigKeyID = document.Keys[1].ID
	document.Registrations = append(document.Registrations, state.Registration{
		ID: "registration-bound", HostID: "host-1", KeyID: "key-2",
		Status: "registered", Operation: "register", Stage: "complete", ConnectionCheck: "unverified",
	})
	document.Configs = []state.Config{{Path: filepath.Join(state.Directory(home), "config"), SHA256: strings.Repeat("a", 64), PendingSHA256: strings.Repeat("b", 64)}}
	saveNavigationState(t, home, document)
	overview := runSizedScreen(t, home, 100, 80, press(tea.KeyTab), press(tea.KeyEnter))
	for _, want := range []string{"Configuration: alias bound to Other key", "Include activation is not established"} {
		if !strings.Contains(overview, want) {
			t.Fatalf("missing config distinction %q:\n%s", want, overview)
		}
	}
	if strings.Contains(overview, document.Configs[0].SHA256) || strings.Contains(overview, document.Configs[0].Path) {
		t.Fatal("global configuration metadata leaked into overview")
	}
	technical := runSizedScreen(t, home, 100, 100, press(tea.KeyTab), press(tea.KeyEnter), press('v'))
	compact := func(value string) string { return strings.Join(strings.Fields(ansi.Strip(value)), "") }
	for _, want := range []string{"Technical details", "global; not proof of alias activation", document.Configs[0].Path, document.Configs[0].SHA256, document.Configs[0].PendingSHA256, document.Keys[0].Fingerprint, "Key ID: key-1"} {
		if !strings.Contains(compact(technical), compact(want)) {
			t.Fatalf("technical evidence lost %q:\n%s", want, technical)
		}
	}
}

func TestDetailTechnicalHelpAndBackRestoreViewportWithoutMutation(t *testing.T) {
	home, _ := navigationHome(t)
	path := filepath.Join(state.Directory(home), "state.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range [][2]int{{80, 24}, {60, 18}, {32, 12}} {
		origin := []tea.Msg{press(tea.KeyTab), press(tea.KeyEnter), press(tea.KeyPgDown)}
		expected := runSizedScreen(t, home, size[0], size[1], origin...)
		back := runSizedScreen(t, home, size[0], size[1], append(origin, press('v'), press(tea.KeyPgDown), press('?'), press(tea.KeyEscape), press('r'), press('u'), press('d'), press('s'), press(tea.KeyEscape))...)
		// Blocked action notices occupy one reserved footer line on return.
		cleanBack := runSizedScreen(t, home, size[0], size[1], append(origin, press('v'), press(tea.KeyPgDown), press('?'), press(tea.KeyEscape), press(tea.KeyEscape))...)
		if expected != cleanBack {
			t.Fatalf("overview viewport lost at %v:\nbefore=%s\nafter=%s", size, expected, cleanBack)
		}
		if strings.Contains(back, "Review ") || strings.Contains(back, "Choose related") || !strings.Contains(back, "read-only") {
			t.Fatalf("technical document borrowed a hidden target:\n%s", back)
		}
		technicalOrigin := append(origin, press('v'), press(tea.KeyPgDown))
		technical := runSizedScreen(t, home, size[0], size[1], technicalOrigin...)
		restored := runSizedScreen(t, home, size[0], size[1], append(technicalOrigin, press('?'), press(tea.KeyEscape))...)
		if technical != restored {
			t.Fatalf("technical Help lost scroll at %v", size)
		}
		if lipgloss.Width(technical) > size[0] || lipgloss.Height(technical) > size[1] {
			t.Fatalf("technical overflow at %v:\n%s", size, technical)
		}
	}
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatal("detail navigation rewrote saved state")
	}
}

func TestDetailEmptyRelationshipsAndMissingChoicesExplainNextStep(t *testing.T) {
	home, document := navigationHome(t)
	document.Keys = []state.Key{}
	document.Registrations = []state.Registration{}
	saveNavigationState(t, home, document)
	screen := runScreen(t, home, press(tea.KeyEnter))
	for _, want := range []string{"Registration: none recorded", "Connection: not recorded", "c Create key / i Import key"} {
		if !strings.Contains(screen, want) {
			t.Fatalf("missing empty-state guidance %q:\n%s", want, screen)
		}
	}
	home, document = navigationHome(t)
	document.Hosts = []state.Host{}
	document.Registrations = []state.Registration{}
	saveNavigationState(t, home, document)
	screen = runScreen(t, home, press(tea.KeyTab), press(tea.KeyEnter))
	for _, want := range []string{"a Add destination", "d Delete local key files"} {
		if !strings.Contains(screen, want) {
			t.Fatalf("missing key action %q:\n%s", want, screen)
		}
	}
}

func TestDetailReadOnlyTaskDoesNotAdvertiseContinuation(t *testing.T) {
	home, _ := navigationHome(t)
	// Absent fixture files produce an integrity review task without a resume ID.
	journey := []tea.Msg{press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyEnter)}
	for _, size := range [][2]int{{80, 24}, {60, 18}, {32, 12}} {
		screen := runSizedScreen(t, home, size[0], size[1], journey...)
		if strings.Contains(screen, "s Continue") || !strings.Contains(screen, "v technical") {
			t.Fatalf("read-only task advertised execution at %v:\n%s", size, screen)
		}
	}
	technical := runSizedScreen(t, home, 100, 80, append(journey, press('v'))...)
	if !strings.Contains(technical, "Source IDs: state.json") || !strings.Contains(technical, "No Mark done") {
		t.Fatalf("task diagnostics unavailable:\n%s", technical)
	}
}

func TestDetailConfigurationTaskKeepsPathInTechnicalDocument(t *testing.T) {
	home, document := navigationHome(t)
	path := filepath.Join(state.Directory(home), "config")
	document.Configs = []state.Config{{Path: path, SHA256: strings.Repeat("a", 64), PendingSHA256: strings.Repeat("b", 64)}}
	saveNavigationState(t, home, document)
	journey := []tea.Msg{press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab), press('/'), text("config:"), press(tea.KeyEnter), press(tea.KeyEnter)}
	overview := runSizedScreen(t, home, 100, 80, journey...)
	if !strings.Contains(overview, "Requires review: configuration file") || strings.Contains(overview, path) {
		t.Fatalf("configuration path obscures task overview:\n%s", overview)
	}
	technical := runSizedScreen(t, home, 200, 100, append(journey, press('v'))...)
	if !strings.Contains(technical, path) || !strings.Contains(technical, "No standalone config/Include resume") {
		t.Fatalf("configuration recovery evidence lost:\n%s", technical)
	}
}
