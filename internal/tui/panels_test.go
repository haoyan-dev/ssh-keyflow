package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestPanelPreservesCellWidthAndCompleteValues(t *testing.T) {
	value := strings.Repeat("東京/鍵/", 20) + "fingerprint-tail"
	for _, width := range []int{30, 58, 78} {
		framed := panel("Saved evidence", detailEvidence(value, "blocked"), width)
		for _, line := range strings.Split(framed, "\n") {
			if ansi.StringWidth(line) != width {
				t.Fatalf("width %d: got %d cells in %q", width, ansi.StringWidth(line), line)
			}
		}
		plain := strings.ReplaceAll(ansi.Strip(framed), "│", "")
		plain = strings.Join(strings.Fields(plain), "")
		if !strings.Contains(plain, value) {
			t.Fatalf("width %d: full value lost:\n%s", width, framed)
		}
	}
}

func TestPanelPairUsesTallerColumnAndNoOverflow(t *testing.T) {
	left := panel("Saved status", "Registration: registered\nConnection: failed\nPublic only\nSaved work unfinished", 41)
	right := panel("Next step", "s Continue saved work", 35)
	pair := panelPair(left, right)
	if lipgloss.Width(pair) != 78 || lipgloss.Height(pair) != lipgloss.Height(left) {
		t.Fatalf("columns lost their shared geometry:\n%s", pair)
	}
}

func TestDocumentsReflowFromSourceWithoutNestedBorders(t *testing.T) {
	source := detailHeading("Technical details") + "\n\nKey ID: fixed-target\nFull path: " + strings.Repeat("directory/", 25) + "tail"
	wide := documentPanels("technical", source, 78)
	narrow := documentPanels("technical", source, 30)
	if strings.Count(wide, "┌") != 1 || strings.Count(narrow, "┌") != 1 {
		t.Fatal("reflow nested or duplicated the original panel")
	}
	if lipgloss.Width(narrow) != 30 || lipgloss.Height(narrow) <= lipgloss.Height(wide) {
		t.Fatal("document did not reflow to the new interior width")
	}
	for _, want := range []string{"Technical details", "fixed-target", "tail"} {
		if !strings.Contains(narrow, want) {
			t.Fatalf("reflow lost %q", want)
		}
	}
}
