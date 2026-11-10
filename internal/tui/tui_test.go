package tui

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/state"
)

func press(code rune) tea.KeyPressMsg {
	key := tea.Key{Code: code}
	if code >= 32 && code <= 0x10ffff {
		key.Text = string(code)
	}
	return tea.KeyPressMsg(key)
}
func text(value string) tea.PasteMsg { return tea.PasteMsg{Content: value} }
func update(m model, messages ...tea.Msg) model {
	for _, message := range messages {
		next, _ := m.Update(message)
		m = next.(model)
	}
	return m
}
func emptyModel(t *testing.T) model {
	t.Helper()
	options := app.Options{Home: t.TempDir(), Input: strings.NewReader(""), Output: io.Discard}
	view, err := app.Snapshot(options)
	if err != nil {
		t.Fatal(err)
	}
	return newModel(context.Background(), options, view)
}
func fixture(t *testing.T) model {
	t.Helper()
	m := emptyModel(t)
	blob := make([]byte, 4+len("ssh-ed25519")+4+32)
	binary.BigEndian.PutUint32(blob, uint32(len("ssh-ed25519")))
	copy(blob[4:], "ssh-ed25519")
	binary.BigEndian.PutUint32(blob[4+len("ssh-ed25519"):], 32)
	public := "ssh-ed25519 " + base64.StdEncoding.EncodeToString(blob)
	digest := sha256.Sum256(blob)
	fingerprint := "SHA256:" + base64.RawStdEncoding.EncodeToString(digest[:])
	path := filepath.Join(m.options.Home, "key.pub")
	if err := os.WriteFile(path, []byte(public+" comment with controls omitted"), 0600); err != nil {
		t.Fatal(err)
	}
	m.view.State.Keys = []state.Key{{ID: "key-one", Name: "日本語 key", Origin: "imported", PublicPath: path, Fingerprint: fingerprint, ImportedAt: "2026-11-08T00:00:00Z", ConnectionCheck: "unverified"}}
	m.view.State.Hosts = []state.Host{{ID: "host-one", Alias: "laptop", Hostname: "example.test", User: "me", Port: 2222, OS: "linux"}}
	m.view.State.Associations = []state.Association{{HostID: "host-one", KeyID: "key-one"}}
	saveFixture(t, m)
	return m
}
func saveFixture(t *testing.T, m model) {
	t.Helper()
	data, err := json.Marshal(m.view.State)
	if err != nil {
		t.Fatal(err)
	}
	directory := state.Directory(m.options.Home)
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "state.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Snapshot(m.options); err != nil {
		t.Fatal(err)
	}
}
func assertFits(t *testing.T, m model) {
	t.Helper()
	view := m.View()
	lines := strings.Split(view.Content, "\n")
	if len(lines) > m.height {
		t.Fatalf("height %d > %d:\n%s", len(lines), m.height, view.Content)
	}
	for _, line := range lines {
		if ansi.StringWidth(line) > m.width {
			t.Fatalf("width %d > %d: %q", ansi.StringWidth(line), m.width, line)
		}
	}
}
func TestEssentialTabsAndNavigationNeverInitializeStorage(t *testing.T) {
	m := emptyModel(t)
	for i := 0; i < 4; i++ {
		screen := m.View().Content
		for _, want := range []string{"Keys", "Destinations"} {
			if !strings.Contains(screen, want) {
				t.Fatalf("missing %s:\n%s", want, screen)
			}
		}
		for _, removed := range []string{"Registrations", "Tasks", "register", "delete", "configure", "auth"} {
			if strings.Contains(screen, removed) {
				t.Fatalf("obsolete action %s:\n%s", removed, screen)
			}
		}
		m = update(m, press(tea.KeyTab))
	}
	m = update(m, press('c'), text("cancelled draft"), press(tea.KeyEscape), press('?'), press(tea.KeyEscape))
	if _, err := os.Stat(filepath.Join(m.options.Home, ".ssh")); !os.IsNotExist(err) {
		t.Fatal("read-only navigation/cancel created storage")
	}
}
func TestMinimumSizeAndAllPagesFitCJK(t *testing.T) {
	m := fixture(t)
	for _, size := range [][2]int{{80, 24}, {32, 12}, {31, 11}, {1, 1}, {32, 12}, {100, 30}} {
		m = update(m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		assertFits(t, m)
	}
	m = update(m, tea.WindowSizeMsg{Width: 32, Height: 12})
	for _, key := range []rune{tea.KeyEnter, 'v', '?', tea.KeyEscape, tea.KeyEscape, tea.KeyEscape, 'i'} {
		m = update(m, press(key))
		assertFits(t, m)
	}
	m = update(m, text(strings.Repeat("日本語", 40)))
	assertFits(t, m)
	if !strings.Contains(m.View().Content, "Ctrl+S") {
		t.Fatal("narrow form lost continue/cancel controls")
	}
}
func TestSearchTreatsActionKeysAsTextAndKeepsIndependentQueries(t *testing.T) {
	m := fixture(t)
	m = update(m, press('/'), text("日本語"), press('q'))
	if m.queries[0] != "日本語q" || !m.searching {
		t.Fatalf("search hotkey escaped input: query=%q searching=%v", m.queries[0], m.searching)
	}
	m = update(m, press(tea.KeyEscape), press('x'), press(tea.KeyTab), press('/'), text("laptop"), press(tea.KeyEnter), press(tea.KeyTab))
	if m.queries[0] != "" || m.queries[1] != "laptop" || m.tab != 0 {
		t.Fatal("tab searches leaked")
	}
	m = update(m, inputResponse{m.inputGeneration - 1, text("late")})
	if m.queries[0] != "" {
		t.Fatal("delayed input reached another page")
	}
}
func TestPublicCopyUsesValidatedRawTextWithoutMutation(t *testing.T) {
	m := fixture(t)
	statePath := filepath.Join(state.Directory(m.options.Home), "state.json")
	before, _ := os.ReadFile(statePath)
	m = update(m, press('p'))
	if m.page != "document" || !strings.Contains(m.document, "Remote placement is not verified") {
		t.Fatal(m.document)
	}
	expected, err := app.PublicKey(m.options.Home, "key-one")
	if err != nil {
		t.Fatal(err)
	}
	next, command := m.Update(press('y'))
	m = next.(model)
	if command == nil || fmt.Sprint(command()) != expected || m.copyText != expected {
		t.Fatal("copy changed the public key text")
	}
	if strings.Contains(m.status, "success") || !strings.Contains(m.status, "requested") {
		t.Fatal("OSC52 copy claimed success")
	}
	after, _ := os.ReadFile(statePath)
	if string(after) != string(before) {
		t.Fatal("public copy changed state")
	}
	m = update(m, press(tea.KeyEscape))
	if m.page != "list" {
		t.Fatal("copy back lost origin")
	}
}
func TestGuidePickerUsesFixedIDsAndCopyIsReadOnly(t *testing.T) {
	m := fixture(t)
	before, _ := os.ReadFile(filepath.Join(state.Directory(m.options.Home), "state.json"))
	m = update(m, press(tea.KeyTab), press('g'))
	if m.page != "picker" || m.request.Host != "host-one" || len(m.choices) != 1 {
		t.Fatal("guide did not bind destination")
	}
	m = update(m, press(tea.KeyEnter))
	if m.page != "form" || m.fields[0].input.Value() != app.LocalPlatform() || m.fields[1].input.Value() != "linux" {
		t.Fatal("guide platform defaults are not visible")
	}
	m = update(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if m.page != "document" || m.request.Key != "key-one" {
		t.Fatal("guide did not bind key")
	}
	for _, want := range []string{"authorized_keys", "Manual removal", "Public-only", "not checked", "no success"} {
		if !strings.Contains(m.document, want) {
			t.Fatalf("missing %s: %s", want, m.document)
		}
	}
	next, command := m.Update(press('y'))
	if command == nil || fmt.Sprint(command()) != next.(model).copyText {
		t.Fatal("guide copy lost original text")
	}
	after, _ := os.ReadFile(filepath.Join(state.Directory(m.options.Home), "state.json"))
	if string(before) != string(after) {
		t.Fatal("guide/copy recorded remote success")
	}
}
func TestCopyNeverReleasesPrivateMaterialOrCorruptPublicKey(t *testing.T) {
	m := fixture(t)
	if err := os.WriteFile(m.view.State.Keys[0].PublicPath, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	m = update(m, press('p'))
	if m.page == "document" || m.copyText != "" || m.status == "" {
		t.Fatal("corrupt public key became copyable")
	}
}
func TestReloadInvalidatesDocumentCopyIncludingHelpOrigins(t *testing.T) {
	for _, document := range []string{"public", "guide"} {
		for _, help := range []bool{false, true} {
			for _, failed := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/help=%t/failed=%t", document, help, failed), func(t *testing.T) {
					m := fixture(t)
					if document == "public" {
						m = update(m, press('p'))
					} else {
						m = update(m, press(tea.KeyTab), press('g'), press(tea.KeyEnter), tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
					}
					if m.page != "document" || m.copyText == "" {
						t.Fatal("missing copyable document")
					}
					if help {
						m = update(m, press('?'))
					}
					if failed {
						path := filepath.Join(state.Directory(m.options.Home), "state.json")
						if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
							t.Fatal(err)
						}
					}
					m = update(m, press(tea.KeyF5))
					if m.stale != failed {
						t.Fatalf("stale=%t, want %t", m.stale, failed)
					}
					if m.copyText != "" {
						t.Fatal("reload retained current clipboard data")
					}
					for _, origin := range m.origins {
						if origin.copyText != "" {
							t.Fatal("reload retained clipboard data in back history")
						}
					}
					if help {
						m = update(m, press(tea.KeyEscape))
					}
					next, command := m.Update(press('y'))
					m = next.(model)
					if m.page != "document" || m.copyText != "" || command != nil {
						t.Fatal("restored document emitted obsolete OSC52 data")
					}
					if failed {
						saveFixture(t, m)
						m = update(m, press(tea.KeyF5))
						next, command = m.Update(press('y'))
						if next.(model).stale || next.(model).copyText != "" || command != nil {
							t.Fatal("reload recovery re-enabled obsolete OSC52 data")
						}
					}
				})
			}
		}
	}
}
func TestStaleDocumentCannotCopyEvenWithRetainedText(t *testing.T) {
	m := fixture(t)
	m = update(m, press('p'))
	if m.copyText == "" {
		t.Fatal("missing copyable document")
	}
	m.stale = true
	_, command := m.Update(press('y'))
	if command != nil {
		t.Fatal("stale document emitted OSC52 data")
	}
}
func TestLegacyRecordsAreReadOnlyAndOnlyLocalCreationCanResume(t *testing.T) {
	m := fixture(t)
	m.view.State.Registrations = []state.Registration{{ID: "reg-old", HostID: "host-one", KeyID: "key-one", Status: "registered", Stage: "pending", Operation: "register", ConnectionCheck: "verified"}}
	m.view.State.Onboardings = []state.Onboarding{{ID: "onboarding-old", KeyID: "key-one", HostID: "host-one", Stage: "pending"}}
	m.view.State.Creations = []state.Creation{{ID: "creation-local", KeyID: "key-new", Name: "recover", Stage: "pending"}, {ID: "creation-complete", Stage: "complete"}}
	items := m.entries()
	if len(items) != 2 || items[1].kind != "creation" {
		t.Fatalf("legacy operation appeared resumable: %+v", items)
	}
	m = update(m, press(tea.KeyEnter), press('v'))
	if m.page != "technical" || !strings.Contains(m.document, "Legacy remote operations cannot resume") || !strings.Contains(m.document, "Recorded status: registered") {
		t.Fatal(m.document)
	}
	next, command := m.Update(press('s'))
	if command != nil || next.(model).running {
		t.Fatal("historical remote record launched execution")
	}
	m = update(m, press(tea.KeyEscape), press(tea.KeyEscape), press('r'))
	if m.page != "list" || m.running {
		t.Fatal("removed remote registration hotkey still active")
	}
}
func TestHelpReturnsToScrolledDocumentAfterResize(t *testing.T) {
	m := fixture(t)
	m.open("document", entry{}, strings.Repeat("日本語 long line\n", 80), "copy")
	m = update(m, press(tea.KeyEnd))
	if !m.content.AtBottom() {
		t.Fatal("End did not scroll")
	}
	m = update(m, press('?'), tea.WindowSizeMsg{Width: 32, Height: 12}, press(tea.KeyEscape))
	if m.page != "document" || m.copyText != "copy" || m.content.YOffset() == 0 {
		t.Fatal("help lost document origin")
	}
	assertFits(t, m)
	m = update(m, press(tea.KeyHome))
	if m.content.YOffset() != 0 {
		t.Fatal("Home did not scroll")
	}
}
func TestFormsHaveOnlyEssentialFieldsAndNoMutationBeforeContinue(t *testing.T) {
	m := emptyModel(t)
	for _, action := range []string{"create", "import"} {
		m.beginForm(action)
		for _, field := range m.fields {
			for _, secret := range []string{"auth", "identity", "configure", "passphrase", "password", "keymode"} {
				if field.name == secret {
					t.Fatal("obsolete/sensitive form field")
				}
			}
		}
		m = update(m, text("draft"))
		for i := 0; i < len(m.fields)+2; i++ {
			m = update(m, press(tea.KeyTab))
		}
		m = update(m, press(tea.KeyEnter), press(tea.KeyEscape))
		if m.fields[0].input.Value() != "draft" {
			t.Fatal("Help discarded draft")
		}
		m = update(m, press(tea.KeyEscape))
	}
	m = update(m, press(tea.KeyTab), press('a'))
	if m.page != "list" || !strings.Contains(m.status, "key first") {
		t.Fatal("destination allocated a new key implicitly")
	}
	if _, err := os.Stat(filepath.Join(m.options.Home, ".ssh")); !os.IsNotExist(err) {
		t.Fatal("form changed files")
	}
}
func TestQuestionMarkIsTextInFocusedFormFieldsAndHelpControlStillWorks(t *testing.T) {
	for _, action := range []string{"create", "import"} {
		t.Run(action, func(t *testing.T) {
			m := emptyModel(t)
			m.beginForm(action)
			values := []string{"name?draft", "/tmp/key?.pub", "/tmp/private?key"}
			for i := range m.fields {
				before, after, _ := strings.Cut(values[i], "?")
				m = update(m, text(before), press('?'), text(after))
				if m.page != "form" || m.fields[i].input.Value() != values[i] {
					t.Fatalf("question mark escaped field %s: page=%s value=%q", m.fields[i].name, m.page, m.fields[i].input.Value())
				}
				m = update(m, press(tea.KeyTab))
			}
			m = update(m, press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyEnter))
			if m.page != "help" {
				t.Fatal("Help control did not open help")
			}
			m = update(m, press(tea.KeyEscape))
			if m.page != "form" {
				t.Fatal("Help did not restore form")
			}
			for i := range m.fields {
				if m.fields[i].input.Value() != values[i] {
					t.Fatal("Help discarded question marks from draft")
				}
			}
		})
	}
}
func fillFields(m *model, values ...string) {
	for i, value := range values {
		m.fields[i].input.SetValue(value)
	}
}
func TestAddDestinationRequiresExplicitOSAndWindowsAccountThenExistingKey(t *testing.T) {
	m := fixture(t)
	m.tab = 1
	m.beginForm("add")
	fillFields(&m, "new-host", "example.test", "me", "22", "", "")
	if command := m.submitForm(); command != nil || m.page != "form" || m.formError == "" {
		t.Fatal("unconfirmed OS accepted")
	}
	fillFields(&m, "new-host", "example.test", "me", "22", "windows", "")
	m.submitForm()
	if !strings.Contains(m.formError, "Windows account") {
		t.Fatal("Windows account was assumed")
	}
	fillFields(&m, "new-host", "example.test", "me", "2222", "windows", "administrator")
	m.submitForm()
	if m.page != "picker" || m.request.Destination.WindowsAccount != "administrator" || m.request.Name != "" {
		t.Fatal("destination did not choose existing key")
	}
	m = update(m, press(tea.KeyEscape))
	if m.page != "form" || m.fields[5].input.Value() != "administrator" {
		t.Fatal("picker back lost fields")
	}
}
func TestAssociationUsesSavedIdentityAndNativeConfirmation(t *testing.T) {
	m := fixture(t)
	m.tab = 1
	m.beginPair("associate")
	if m.page != "picker" || m.request.Host != "host-one" {
		t.Fatal("association lacks destination")
	}
	next, command := m.Update(press(tea.KeyEnter))
	m = next.(model)
	if command == nil || !m.running || m.request.Operation != "associate" || m.request.Key != "key-one" {
		t.Fatal("association did not use explicit IDs")
	}
	_, second := m.Update(press(tea.KeyEnter))
	if second != nil {
		t.Fatal("double Enter launched duplicate native operation")
	}
	m.signals.finish()
}
func TestStaleReloadBlocksMutationAndDoesNotReplayOperation(t *testing.T) {
	m := fixture(t)
	path := filepath.Join(state.Directory(m.options.Home), "state.json")
	before, _ := os.ReadFile(path)
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	m = update(m, completed{operationResult{request: app.Request{Operation: "create"}, err: context.Canceled, diagnostics: "original output"}})
	if !m.stale || !strings.Contains(m.document, "Reload failed") {
		t.Fatal("reload failure hidden")
	}
	m = update(m, press(tea.KeyEscape), press('c'))
	if m.page == "form" || m.running {
		t.Fatal("stale mutation enabled")
	}
	if err := os.WriteFile(path, before, 0600); err != nil {
		t.Fatal(err)
	}
	m = update(m, press(tea.KeyF5))
	if m.stale || m.running {
		t.Fatal("refresh did not recover read-only")
	}
}
func TestDiagnosticsAndSavedStateCannotInjectTerminalControls(t *testing.T) {
	m := fixture(t)
	m.view.State.Keys[0].Name = "safe\x1b[2J\x1b]52;c;c2VjcmV0\a\x7f"
	m = update(m, press(tea.KeyEnter))
	rendered := m.View().Content
	// Only presentation SGR is allowed; saved-state OSC/cursor controls are not.
	if strings.Contains(ansi.Strip(rendered), "\x7f") || strings.Contains(rendered, "\x1b[2J") || strings.Contains(rendered, "\x1b]52") {
		t.Fatal("state injected terminal controls")
	}
	var buffer diagnosticBuffer
	payload := strings.Repeat("x", 40000) + "\x1b[2J"
	if count, err := buffer.Write([]byte(payload)); err != nil || count != len(payload) {
		t.Fatal("limited capture broke native output writer")
	}
	if buffer.Len() != 32768 || !strings.Contains(buffer.String(), "truncated") {
		t.Fatal("native capture unbounded")
	}
	buffer = diagnosticBuffer{}
	buffer.Write([]byte("native\x1b[2J\x1b]52;c;c2VjcmV0\a\x00"))
	if buffer.String() != "native" {
		t.Fatalf("unsafe diagnostics: %q", buffer.String())
	}
}

func TestNativeHandoffIgnoresQueuedPasteAndClipboardInput(t *testing.T) {
	m := emptyModel(t)
	m.beginForm("create")
	m.fields[0].input.SetValue("intended name")
	m.running = true
	m = update(m, text("queued secret"), tea.ClipboardMsg{Content: "clipboard secret"}, press(tea.KeyEnter))
	if m.fields[0].input.Value() != "intended name" {
		t.Fatal("native input was captured by TUI fields")
	}
}
func TestFormsExposeCaretAndPreserveKeyboardEditing(t *testing.T) {
	m := emptyModel(t)
	m = update(m, press('c'), text("name"))
	view := m.View()
	if view.Cursor == nil || view.Cursor.Y < 2 || view.Cursor.Y >= m.height-2 {
		t.Fatal("focused input caret missing or outside form")
	}
	m = update(m, press(tea.KeyLeft), press('x'))
	if m.fields[0].input.Value() != "namxe" {
		t.Fatal("caret did not track editing")
	}
	m = update(m, press(tea.KeyEscape), press('/'), text("日本語"))
	if m.View().Cursor == nil {
		t.Fatal("search caret missing")
	}
}
func TestLegacyReviewNoticeDoesNotBlockUnrelatedLocalForms(t *testing.T) {
	m := fixture(t)
	m.view.Review = "Legacy operation requires manual review"
	if !strings.Contains(m.View().Content, "v details") {
		t.Fatal("review notice inaccessible")
	}
	m = update(m, press('c'))
	if m.page != "form" {
		t.Fatal("legacy review blocked unrelated local creation form")
	}
}
func TestGuideWithPairIncludesCopyableSnippetAndConnectionCommand(t *testing.T) {
	m := fixture(t)
	path := filepath.Join(m.options.Home, "private")
	private := []byte("private fixture material - never display this")
	if err := os.WriteFile(path, private, 0600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(private)
	m.view.State.Keys[0].PrivatePath = path
	m.view.State.Keys[0].PrivateSHA256 = fmt.Sprintf("%x", digest)
	saveFixture(t, m)
	m = update(m, press(tea.KeyTab), press('g'), press(tea.KeyEnter), tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if len(m.guideSteps) != 3 || !strings.Contains(m.document, "Step 1/3: DESTINATION") {
		t.Fatal("guide did not start at destination setup", m.document)
	}
	for _, want := range []string{"authorized_keys", "ssh-ed25519"} {
		if !strings.Contains(m.copyText, want) {
			t.Fatalf("guide lost copyable %s", want)
		}
	}
	for index := 0; index < 3; index++ {
		if index > 0 {
			m = update(m, press('n'))
		}
		_, command := m.Update(press('y'))
		if command == nil || fmt.Sprint(command()) != m.guideSteps[index].Script || m.copyText != m.guideSteps[index].Script {
			t.Fatal("copy included instructions or the wrong step")
		}
		if !strings.Contains(m.document, fmt.Sprintf("Step %d/3", index+1)) {
			t.Fatal("step label missing")
		}
	}
	if m.copyText != "ssh laptop\n" {
		t.Fatal("connection step included another script", m.copyText)
	}
	m = update(m, press('b'), press('?'), press(tea.KeyEscape))
	if m.guideIndex != 1 || !strings.Contains(m.copyText, "Host laptop") {
		t.Fatal("help/back lost selected step")
	}
	m = update(m, press(tea.KeyF5), press('n'))
	if m.copyText != "" || len(m.guideSteps) != 0 {
		t.Fatal("step navigation restored stale scripts")
	}
	if strings.Contains(m.document, string(private)) {
		t.Fatal("guide disclosed private key material")
	}
}

func TestGuidePlatformFormValidationEditingAndCancel(t *testing.T) {
	m := fixture(t)
	private := []byte("private fixture")
	path := filepath.Join(m.options.Home, "private")
	if err := os.WriteFile(path, private, 0600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(private)
	m.view.State.Keys[0].PrivatePath = path
	m.view.State.Keys[0].PrivateSHA256 = fmt.Sprintf("%x", digest)
	saveFixture(t, m)
	before, _ := os.ReadFile(filepath.Join(state.Directory(m.options.Home), "state.json"))
	m = update(m, press(tea.KeyTab), press('g'), press(tea.KeyEnter))
	m.fields[0].input.SetValue("invalid")
	m = update(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if m.page != "form" || m.formError == "" || m.copyText != "" || len(m.guideSteps) != 0 {
		t.Fatal("invalid platforms produced a copyable guide")
	}
	local, pathInTerminal := "linux", "/home/alice/.ssh/key"
	if app.LocalPlatform() != "windows" {
		local, pathInTerminal = "windows", `C:\Users\alice\.ssh\key`
	}
	m.fields[0].input.SetValue(local)
	m.fields[1].input.SetValue("windows")
	m = update(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if m.page != "form" || !strings.Contains(m.formError, "account") {
		t.Fatal("missing destination account was accepted")
	}
	m.fields[2].input.SetValue("standard")
	m = update(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if m.page != "form" || !strings.Contains(m.formError, "identity-file") {
		t.Fatal("foreign platform inherited managed private-key path")
	}
	m.fields[3].input.SetValue(pathInTerminal)
	m = update(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if m.page != "document" || !strings.Contains(m.copyText, "$env:USERPROFILE") ||
		!strings.Contains(m.document, "Local platform: "+local+" · Destination platform: windows") {
		t.Fatal("destination override or platform indicators lost", m.document)
	}
	m = update(m, press('n'))
	if strings.HasPrefix(m.copyText, "& {") != (local == "windows") || !strings.Contains(m.copyText, strings.ReplaceAll(pathInTerminal, "\\", "/")) {
		t.Fatal("local override lost", m.copyText)
	}
	m = update(m, press('?'), press(tea.KeyEscape), press(tea.KeyEscape))
	if m.page != "form" || m.fields[0].input.Value() != local {
		t.Fatal("back did not restore platform choices")
	}
	m.fields[0].input.SetValue("invalid")
	m = update(m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if m.copyText != "" || len(m.guideSteps) != 0 {
		t.Fatal("failed regeneration retained old scripts")
	}
	m = update(m, press(tea.KeyEscape))
	if m.page != "list" {
		t.Fatal("cancel did not restore guide origin")
	}
	after, _ := os.ReadFile(filepath.Join(state.Directory(m.options.Home), "state.json"))
	if string(after) != string(before) {
		t.Fatal("guide platform form modified saved records")
	}
}
