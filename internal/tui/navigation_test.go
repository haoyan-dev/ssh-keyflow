package tui_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/tui"
	"strings"
	"testing"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"ssh-keyflow/internal/state"
)

func text(value string) tea.Msg { return tea.KeyPressMsg(tea.Key{Code: []rune(value)[0], Text: value}) }

// Fixtures enter at the filesystem boundary; all journeys use NewProgram.
func navigationHome(t *testing.T) (string, state.State) {
	t.Helper()
	home := t.TempDir()
	document := state.State{Version: 1, Keys: []state.Key{}, Hosts: []state.Host{}, Registrations: []state.Registration{}}
	for i := 1; i <= 9; i++ {
		digest := make([]byte, 32)
		digest[0] = byte(i)
		document.Keys = append(document.Keys, state.Key{ID: fmt.Sprintf("key-%d", i), Name: "Laptop", Origin: "imported", ImportedAt: "2026-11-07T00:00:00Z", Fingerprint: "SHA256:" + base64.RawStdEncoding.EncodeToString(digest), PublicPath: filepath.Join(home, fmt.Sprintf("key-%d.pub", i)), ConnectionCheck: "verified"})
		document.Hosts = append(document.Hosts, state.Host{ID: fmt.Sprintf("host-%d", i), Alias: fmt.Sprintf("build-%d", i), Hostname: fmt.Sprintf("server-%d.example", i), User: "alice", Port: 22, OS: "linux"})
		document.Registrations = append(document.Registrations, state.Registration{ID: fmt.Sprintf("registration-%d", i), KeyID: fmt.Sprintf("key-%d", i), HostID: fmt.Sprintf("host-%d", i), Operation: "register", Status: "registered", Stage: "complete", ConnectionCheck: "failed"})
	}
	saveNavigationState(t, home, document)
	return home, document
}

func saveNavigationState(t *testing.T, home string, document state.State) {
	t.Helper()
	if err := document.Validate(); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(state.Directory(home), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(state.Directory(home), "state.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestNavigationSearchKeepsQueryAndCategoryIdentity(t *testing.T) {
	home, _ := navigationHome(t)
	screen := runScreen(t, home, press('/'), text("BUILD-7"), press(tea.KeyEnter), press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyEnter), press('v'))
	for _, want := range []string{"Destinations", "build-7", "alice@server-7.example:22", "host-7"} {
		if !strings.Contains(screen, want) {
			t.Fatalf("missing %q:\n%s", want, screen)
		}
	}
	filtered := runScreen(t, home, press('/'), text("not-present?q"), press(tea.KeyEscape))
	for _, want := range []string{"No matching", "not-present?q", "Clear search"} {
		if !strings.Contains(filtered, want) {
			t.Fatalf("missing %q:\n%s", want, filtered)
		}
	}
	cleared := runScreen(t, home, press('/'), text("not-present"), press(tea.KeyEnter), press('x'))
	if !strings.Contains(cleared, "build-1") {
		t.Fatalf("clear did not restore rows:\n%s", cleared)
	}
}

func TestNavigationDetailsJoinRecordedEvidenceAndHelpReturnsToOrigin(t *testing.T) {
	home, _ := navigationHome(t)
	journey := []tea.Msg{press(tea.KeyTab), press(tea.KeyTab), press('/'), text("registration-7"), press(tea.KeyEnter), press(tea.KeyEnter)}
	screen := runScreen(t, home, journey...)
	for _, want := range []string{"build-7", "Laptop", "alice@server-7.example:22", "Registration: registered", "Connection: failed", "verification unavailable", "Configuration:", "History:"} {
		if !strings.Contains(screen, want) {
			t.Fatalf("missing %q:\n%s", want, screen)
		}
	}
	if strings.Contains(screen, `"key_id"`) {
		t.Fatalf("details are still raw JSON:\n%s", screen)
	}
	origin := runScreen(t, home, append(journey, press(tea.KeyPgDown))...)
	restored := runScreen(t, home, append(journey, press(tea.KeyPgDown), press('?'), press(tea.KeyEscape))...)
	if origin != restored {
		t.Fatalf("help lost detail scroll/origin:\nbefore=%s\nafter=%s", origin, restored)
	}
	back := runScreen(t, home, append(journey, press('?'), tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl}), press(tea.KeyEscape))...)
	if !strings.Contains(back, "registration-7") || !strings.Contains(back, "Search:") {
		t.Fatalf("help/back lost list filter:\n%s", back)
	}
}

func TestNavigationActionsUseVisibleTargetAndRelatedSelection(t *testing.T) {
	home, _ := navigationHome(t)
	selectHost := []tea.Msg{press('/'), text("host-7"), press(tea.KeyEnter), press(tea.KeyEnter)}
	blocked := runScreen(t, home, append(selectHost, press('d'))...)
	if !strings.Contains(blocked, "managed key") || strings.Contains(blocked, "Delete local key\n") {
		t.Fatalf("destination opened blank key deletion:\n%s", blocked)
	}
	journey := append(selectHost, press('r'), press('/'), text("key-8"), press(tea.KeyEnter), press(tea.KeyEnter), tea.KeyPressMsg(tea.Key{Code: 's', Mod: tea.ModCtrl}))
	review := runScreen(t, home, journey...)
	for _, want := range []string{"Review register", "host-7", "key-8", "build-7", "Laptop"} {
		if !strings.Contains(review, want) {
			t.Fatalf("related selection missing %q:\n%s", want, review)
		}
	}
	discovery := runScreen(t, home, append(selectHost, press('g'), press('u'))...)
	if !strings.Contains(discovery, "read-only") || strings.Contains(discovery, "Registration ID to remove") {
		t.Fatalf("discovery borrowed hidden target:\n%s", discovery)
	}
	history := runScreen(t, home, append(selectHost, press('h'), press('r'))...)
	if !strings.Contains(history, "History (read-only)") || strings.Contains(history, "Destination ID or alias") {
		t.Fatalf("history borrowed hidden target:\n%s", history)
	}
}

// A filter injects an external filesystem change immediately before the user's
// Refresh event, without inspecting the model or mocking app internals.
type replaceSavedState struct{ document state.State }

func TestNavigationRefreshPreservesFixedIDAndAnnouncesRemoval(t *testing.T) {
	home, document := navigationHome(t)
	reordered := document
	reordered.Hosts = append([]state.Host{}, document.Hosts[8], document.Hosts[6], document.Hosts[0])
	// Preserve unrelated hosts so all registrations remain valid.
	for i, host := range document.Hosts {
		if i != 8 && i != 6 && i != 0 {
			reordered.Hosts = append(reordered.Hosts, host)
		}
	}
	options := []tea.ProgramOption{tea.WithFilter(func(_ tea.Model, message tea.Msg) tea.Msg {
		if change, ok := message.(replaceSavedState); ok {
			saveNavigationState(t, home, change.document)
			return press(tea.KeyF5)
		}
		return message
	})}
	selection := []tea.Msg{press(tea.KeyDown), press(tea.KeyDown), press(tea.KeyDown), press(tea.KeyDown), press(tea.KeyDown), press(tea.KeyDown)}
	screen := runSizedScreenOptions(t, home, 80, 24, options, append(selection, replaceSavedState{reordered}, press(tea.KeyEnter), press('v'))...)
	if !strings.Contains(screen, "Destination ID: host-7") {
		t.Fatalf("refresh retargeted selected entity:\n%s", screen)
	}
	removed := reordered
	removed.Hosts = append([]state.Host{}, reordered.Hosts[:1]...)
	removed.Hosts = append(removed.Hosts, reordered.Hosts[2:]...)
	removed.Registrations = append([]state.Registration{}, document.Registrations[:6]...)
	removed.Registrations = append(removed.Registrations, document.Registrations[7:]...)
	saveNavigationState(t, home, document)
	screen = runSizedScreenOptions(t, home, 80, 24, options, append(selection, replaceSavedState{reordered}, replaceSavedState{removed}, press(tea.KeyEnter), press('v'))...)
	if !strings.Contains(screen, "Selection changed") || !strings.Contains(screen, "Destination ID: host-1") {
		t.Fatalf("removed selection not announced/nearest:\n%s", screen)
	}
}

func TestNavigationLayoutsAndSmallWindowMutationGuard(t *testing.T) {
	home, document := navigationHome(t)
	document.Hosts[0].Alias = strings.Repeat("東京", 20)
	saveNavigationState(t, home, document)
	for _, size := range [][2]int{{80, 24}, {60, 18}, {32, 12}} {
		for _, journey := range [][]tea.Msg{
			nil, {press(tea.KeyEnter)}, {press(tea.KeyTab), press(tea.KeyEnter)}, {press('?')}, {press('h')}, {press('g')},
			{press('/'), text(strings.Repeat("q?東京", 20))}, {press('r')}, {press('r'), press('/'), text("missing")},
			{press(tea.KeyEnter), tea.WindowSizeMsg{Width: 32, Height: 12}, press('?'), press(tea.KeyEscape)},
		} {
			screen := runSizedScreen(t, home, size[0], size[1], journey...)
			if lipgloss.Width(screen) > size[0] || lipgloss.Height(screen) > size[1] {
				t.Fatalf("overflow %dx%d at %v:\n%s", size[0], size[1], journey, screen)
			}
		}
	}
	small := runSizedScreen(t, t.TempDir(), 24, 8, press('c'))
	if !strings.Contains(small, "32x12") || strings.Contains(small, "Management name") {
		t.Fatalf("small terminal allowed mutation:\n%s", small)
	}
}

func TestNavigationHistoryKeepsTombstonesOutOfCurrentKeys(t *testing.T) {
	home, document := navigationHome(t)
	key := document.Keys[8]
	document.Keys = document.Keys[:8]
	document.Registrations[8].Status, document.Registrations[8].Operation, document.Registrations[8].Stage = "removed", "unregister", "complete"
	document.DeletedKeys = []state.DeletedKey{{Key: key, DeletedAt: "2026-11-07T01:00:00Z", DeletionID: "deletion-9"}}
	document.Deletions = []state.LocalDeletion{{ID: "deletion-9", KeyID: key.ID, CreatedAt: "2026-11-07T00:00:00Z", Stage: "complete", PublicSHA256: strings.Repeat("a", 64), PublicRemoved: true, PrivateRemoved: true}}
	document.History = []state.RegistrationEvent{{RegistrationID: "registration-9", At: "2026-11-07T01:00:00Z", Operation: "unregister", Status: "removed", Stage: "complete", ConnectionCheck: "unverified"}}
	saveNavigationState(t, home, document)
	keys := runScreen(t, home, press(tea.KeyTab), press('/'), text("key-9"), press(tea.KeyEnter))
	if !strings.Contains(keys, "No matching") {
		t.Fatalf("tombstone still a current key:\n%s", keys)
	}
	history := runScreen(t, home, press(tea.KeyTab), press('h'), press('d'))
	for _, want := range []string{"History (read-only)", "Deleted local key: Laptop [key-9]", "deletion-9"} {
		if !strings.Contains(history, want) {
			t.Fatalf("missing %q:\n%s", want, history)
		}
	}
}

func TestNavigationReadOnlyJourneysKeepStorageAndCategoryScroll(t *testing.T) {
	home, _ := navigationHome(t)
	before, err := os.ReadFile(filepath.Join(state.Directory(home), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	selection := []tea.Msg{press(tea.KeyDown), press(tea.KeyDown), press(tea.KeyDown), press(tea.KeyDown), press(tea.KeyDown), press(tea.KeyDown)}
	original := runSizedScreen(t, home, 32, 12, selection...)
	restored := runSizedScreen(t, home, 32, 12, append(selection, press(tea.KeyEnter), press('?'), press(tea.KeyEscape), press(tea.KeyEscape), press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab))...)
	if original != restored {
		t.Fatalf("category/detail/help lost cursor or pagination:\nbefore=%s\nafter=%s", original, restored)
	}
	after, err := os.ReadFile(filepath.Join(state.Directory(home), "state.json"))
	if err != nil || string(before) != string(after) {
		t.Fatal("navigation changed saved state")
	}
	fresh := t.TempDir()
	tasks := runScreen(t, fresh, press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab))
	if !strings.Contains(tasks, "No recorded follow-up") || !strings.Contains(tasks, "not verified") {
		t.Fatalf("dishonest empty Tasks:\n%s", tasks)
	}
	_ = runScreen(t, fresh, press('/'), text("q?rd"), press(tea.KeyEscape), press(tea.KeyEnter), press('d'), press('?'), press(tea.KeyEscape), press('g'), press(tea.KeyEscape), press('h'), press(tea.KeyEscape), press(tea.KeyF5))
	if _, err := os.Stat(filepath.Join(fresh, ".ssh")); !os.IsNotExist(err) {
		t.Fatal("read-only journeys created storage")
	}
	filtered := runScreen(t, home, press('/'), text("absent"), press(tea.KeyEnter), press('r'))
	if strings.Contains(filtered, "Destination ID or alias") || !strings.Contains(filtered, "Select a destination") {
		t.Fatalf("empty selection opened ID form:\n%s", filtered)
	}
}

func TestNavigationRegistrationSearchIncludesRelatedIdentity(t *testing.T) {
	home, document := navigationHome(t)
	query := strings.TrimPrefix(document.Keys[6].Fingerprint, "SHA256:")[:12]
	screen := runScreen(t, home, press(tea.KeyTab), press(tea.KeyTab), press('/'), text(query), press(tea.KeyEnter), press(tea.KeyEnter), press('v'))
	if !strings.Contains(screen, "Registration ID: registration-7") {
		t.Fatalf("related fingerprint not searchable:\n%s", screen)
	}
}

func TestNavigationFullValuesRemainReachableByVerticalScroll(t *testing.T) {
	home, document := navigationHome(t)
	document.Keys[0].PublicPath = filepath.Join(home, strings.Repeat("長い-path-", 20)+".pub")
	saveNavigationState(t, home, document)
	var seen strings.Builder
	journey := []tea.Msg{press(tea.KeyTab), press(tea.KeyEnter), press('v')}
	for i := 0; i < 20; i++ {
		frame := strings.Split(ansi.Strip(runSizedScreen(t, home, 32, 12, journey...)), "\n")
		// Inspect document text only, excluding the repeated category and controls.
		seen.WriteString(strings.Join(frame[2:len(frame)-3], "\n"))
		journey = append(journey, press(tea.KeyPgDown))
	}
	compact := func(s string) string {
		return strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, ansi.Strip(s))
	}
	for _, want := range []string{document.Keys[0].Fingerprint, document.Keys[0].PublicPath, "key-1"} {
		if !strings.Contains(compact(seen.String()), compact(want)) {
			t.Fatalf("full value inaccessible: %s", want)
		}
	}
}

func TestNavigationMissingJoinFailsReadOnlyInsteadOfInventingRelationship(t *testing.T) {
	home, document := navigationHome(t)
	document.Hosts = document.Hosts[1:]
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(state.Directory(home), "state.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	_, err = tui.NewProgram(context.Background(), app.Options{Home: home, Input: strings.NewReader(""), Output: io.Discard})
	if err == nil || !strings.Contains(err.Error(), "relationship") {
		t.Fatalf("missing join became empty/invented data: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(data) {
		t.Fatal("invalid saved data was modified")
	}
}

func TestNavigationSelectedTargetCannotBeReplacedByLegacyIDInput(t *testing.T) {
	home, _ := navigationHome(t)
	selectAll := tea.KeyPressMsg(tea.Key{Code: 'a', Mod: tea.ModCtrl})
	review := runScreen(t, home, press('r'), press(tea.KeyEnter), selectAll, text("host-9"), press(tea.KeyTab), selectAll, text("key-9"), tea.KeyPressMsg(tea.Key{Code: 's', Mod: tea.ModCtrl}))
	for _, want := range []string{"Review register", "Destination ID or alias: host-1", "Managed key ID / unique name: key-1"} {
		if !strings.Contains(review, want) {
			t.Fatalf("selected target changed or review missing %q:\n%s", want, review)
		}
	}
	if strings.Contains(review, "host-9") || strings.Contains(review, "key-9") {
		t.Fatalf("editable ID retargeted selected identity:\n%s", review)
	}
}

func TestNavigationRepeatedRefreshFailureReturnsToDetailsAfterRecovery(t *testing.T) {
	home, document := navigationHome(t)
	type corruptSavedState struct{}
	options := []tea.ProgramOption{tea.WithFilter(func(_ tea.Model, message tea.Msg) tea.Msg {
		switch message := message.(type) {
		case corruptSavedState:
			if err := os.WriteFile(filepath.Join(state.Directory(home), "state.json"), []byte("invalid saved document"), 0600); err != nil {
				t.Fatal(err)
			}
			return press(tea.KeyF5)
		case replaceSavedState:
			saveNavigationState(t, home, message.document)
			return press(tea.KeyF5)
		}
		return message
	})}
	journey := []tea.Msg{press(tea.KeyEnter), corruptSavedState{}, press(tea.KeyF5), replaceSavedState{document}, press('v')}
	screen := runSizedScreenOptions(t, home, 80, 24, options, journey...)
	if !strings.Contains(screen, "Destination ID: host-1") || strings.Contains(screen, "Refresh failed:") {
		t.Fatalf("refresh recovery lost detail origin:\n%s", screen)
	}
}

func TestNavigationRelatedUnregisterPickerSearchesFingerprint(t *testing.T) {
	home, document := navigationHome(t)
	document.Registrations = append(document.Registrations, state.Registration{ID: "registration-10", HostID: "host-1", KeyID: "key-2", Operation: "register", Stage: "complete", Status: "registered", ConnectionCheck: "failed"})
	saveNavigationState(t, home, document)
	query := strings.TrimPrefix(document.Keys[1].Fingerprint, "SHA256:")[:12]
	screen := runScreen(t, home, press('u'), press('/'), text(query), press(tea.KeyEnter), press(tea.KeyEnter), tea.KeyPressMsg(tea.Key{Code: 's', Mod: tea.ModCtrl}))
	for _, want := range []string{"Review unregister", "Registration ID to remove: registration-10", "key-2"} {
		if !strings.Contains(screen, want) {
			t.Fatalf("related registration missing %q:\n%s", want, screen)
		}
	}
}
