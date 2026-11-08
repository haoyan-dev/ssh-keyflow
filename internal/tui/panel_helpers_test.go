package tui_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// Panel borders separate terminal cells, not characters in a saved value.
func panelText(screen string) string {
	return strings.ReplaceAll(ansi.Strip(screen), "│", "")
}

func compactPanelText(screen string) string {
	return strings.Join(strings.Fields(panelText(screen)), "")
}

// Results now place checkpoints below the outcome and next step. Inspect that
// evidence by scrolling the public program, then restore the initial viewport.
func (s handoffSession) document(t *testing.T) string {
	t.Helper()
	s.send(press(tea.KeyHome))
	var seen strings.Builder
	for page := 0; page < 40; page++ {
		frame := s.screen(t)
		seen.WriteString(frame)
		if strings.Contains(frame, "scroll end") || strings.Contains(frame, "scroll all") ||
			strings.Contains(frame, "↑/↓ end") || strings.Contains(frame, "↑/↓ all") {
			s.send(press(tea.KeyHome))
			return seen.String()
		}
		s.send(press(tea.KeyPgDown))
	}
	t.Fatal("document did not reach its end")
	return ""
}
