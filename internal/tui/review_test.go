package tui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func interruptKey() tea.Msg { return tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl}) }

func TestReviewDirtyCancelKeepsEditingByDefault(t *testing.T) {
	home := t.TempDir()
	journey := []tea.Msg{press('c'), text("ac"), press(tea.KeyLeft), interruptKey()}
	screen := runScreen(t, home, journey...)
	if !strings.Contains(screen, "> Keep editing") || !strings.Contains(screen, "Discard draft") {
		t.Fatalf("dirty cancel lost its safe default:\n%s", screen)
	}
	journey = append(journey, press(tea.KeyEnter), text("b"), reviewKey())
	screen = runScreen(t, home, journey...)
	if !strings.Contains(screen, "Management name: abc") {
		t.Fatalf("keep editing lost caret/value:\n%s", screen)
	}
	screen = runScreen(t, home, append(journey, interruptKey(), press(tea.KeyTab), press(tea.KeyEnter))...)
	if !strings.Contains(screen, "Changes cancelled") || strings.Contains(screen, "Review create") {
		t.Fatalf("explicit discard did not return:\n%s", screen)
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh")); !os.IsNotExist(err) {
		t.Fatal("draft cancel wrote storage")
	}
}

func TestReviewControlsAndHelpRestoreExactEditingPosition(t *testing.T) {
	home := t.TempDir()
	journey := []tea.Msg{press('i'), text("import label"), press(tea.KeyTab), text("C:/begin/" + strings.Repeat("long-directory/", 30) + "public.pub"), press(tea.KeyLeft), press(tea.KeyPgDown)}
	before := runSizedScreen(t, home, 32, 12, journey...)
	review := append(append([]tea.Msg{}, journey...), reviewKey())
	screen := runSizedScreen(t, home, 32, 12, review...)
	for _, label := range []string{"Continue", "Edit", "Cancel", "Help"} {
		if !strings.Contains(screen, label) {
			t.Fatalf("review control %q inaccessible:\n%s", label, screen)
		}
	}
	// Tab selects Edit, rather than cycling categories or restarting the form.
	after := runSizedScreen(t, home, 32, 12, append(review, press(tea.KeyTab), press(tea.KeyEnter))...)
	if before != after {
		t.Fatalf("review Edit lost exact scroll/focus:\nbefore=%s\nafter=%s", before, after)
	}
	// Help is available on review and restores its own scrolled origin.
	scrolled := append(append([]tea.Msg{}, review...), press(tea.KeyPgDown))
	before = runSizedScreen(t, home, 32, 12, scrolled...)
	after = runSizedScreen(t, home, 32, 12, append(scrolled, press('?'), press(tea.KeyEscape))...)
	if before != after {
		t.Fatalf("review Help lost origin:\nbefore=%s\nafter=%s", before, after)
	}
	help := runScreen(t, home, append(review, press('?'))...)
	if !strings.Contains(help, "Review help") {
		t.Fatalf("wrong review help context:\n%s", help)
	}
}

func TestReviewRepeatedEditingLiteralKeysAndCleanCancel(t *testing.T) {
	home := t.TempDir()
	journey := []tea.Msg{press('c'), text("q?cirauds"), press(tea.KeyLeft), reviewKey()}
	for i := 0; i < 3; i++ {
		journey = append(journey, press(tea.KeyEscape), text("X"), reviewKey())
	}
	screen := runScreen(t, home, journey...)
	if !strings.Contains(screen, "Management name: q?ciraudXXXs") {
		t.Fatalf("repeated edit lost literal keys/caret:\n%s", screen)
	}
	// Explicit Cancel is the last keyboard-focusable form control.
	screen = runScreen(t, home, press('c'), press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyEnter))
	if !strings.Contains(screen, "Changes cancelled") || strings.Contains(screen, "Keep editing") {
		t.Fatalf("clean Cancel did not return directly:\n%s", screen)
	}
}
func TestReviewPickerDirtyCancelRestoresSearchAndCaller(t *testing.T) {
	home, _ := navigationHome(t)
	caller := []tea.Msg{press(tea.KeyTab), press('/'), text("key-7"), press(tea.KeyEscape)}
	journey := append(append([]tea.Msg{}, caller...), destinationDraft("linux")...)
	journey = append(journey, reviewKey(), press(tea.KeyEnter), press('/'), text("key-8"))
	before := runScreen(t, home, journey...)
	after := runScreen(t, home, append(append([]tea.Msg{}, journey...), interruptKey(), press('?'), press(tea.KeyEscape), press(tea.KeyEnter))...)
	if before != after {
		t.Fatalf("Keep/Help lost picker search context:\nbefore=%s\nafter=%s", before, after)
	}
	// Search remains in editing mode: q/?/action letters remain in the filter.
	screen := runScreen(t, home, append(append([]tea.Msg{}, journey...), interruptKey(), press(tea.KeyEnter), text("q?c"))...)
	if !strings.Contains(screen, "key-8q?c") {
		t.Fatalf("search swallowed literal keys:\n%s", screen)
	}
	screen = runScreen(t, home, append(journey, interruptKey(), press(tea.KeyTab), press(tea.KeyEnter))...)
	for _, want := range []string{"[Keys]", "key-7", "Changes cancelled"} {
		if !strings.Contains(screen, want) {
			t.Fatalf("discard lost caller %q:\n%s", want, screen)
		}
	}
}
func TestReviewResizeRetainsDraftAndPreventsContinueBelowMinimum(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {60, 18}, {32, 12}} {
		home := t.TempDir()
		journey := []tea.Msg{press('c'), text("ac"), press(tea.KeyLeft), reviewKey()}
		before := runSizedScreen(t, home, size[0], size[1], journey...)
		journey = append(journey, tea.WindowSizeMsg{Width: 20, Height: 8}, press(tea.KeyEnter), tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		after := runSizedScreen(t, home, size[0], size[1], journey...)
		if before != after {
			t.Fatalf("resize lost review at %v:\nbefore=%s\nafter=%s", size, before, after)
		}
		screen := runSizedScreen(t, home, size[0], size[1], append(journey, press(tea.KeyEscape), text("b"), reviewKey())...)
		if size[1] > 12 && !strings.Contains(screen, "Management name: abc") {
			t.Fatalf("resize lost edit caret:\n%s", screen)
		}
		if _, err := os.Stat(filepath.Join(home, ".ssh")); !os.IsNotExist(err) {
			t.Fatal("undersize Continue wrote storage")
		}
	}
}

func TestReviewDraftIsRetainedThroughPickerDiscoveryOverlay(t *testing.T) {
	home, _ := navigationHome(t)
	journey := append(destinationDraft("linux"), reviewKey(), press(tea.KeyEnter), press('g'), press('q'), interruptKey())
	screen := runScreen(t, home, journey...)
	if !strings.Contains(screen, "> Keep editing") {
		t.Fatalf("overlay allowed the draft to be abandoned:\n%s", screen)
	}
}
