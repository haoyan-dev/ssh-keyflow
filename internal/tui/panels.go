package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// A panel's width includes its border and one cell of inner padding per side.
// Wrap before framing so viewport lines, scroll offsets and terminal cells agree.
func panel(title, body string, width int) string {
	width = max(6, width)
	inner := width - 4
	label := ansi.Truncate(title, width-6, "…")
	top := "┌ " + detailHeading(label) + " " + strings.Repeat("─", max(0, width-4-ansi.StringWidth(label))) + "┐"
	lines := []string{top}
	for _, line := range strings.Split(ansi.Wrap(strings.TrimRight(body, "\n"), inner, ""), "\n") {
		lines = append(lines, "│ "+line+strings.Repeat(" ", max(0, inner-ansi.StringWidth(line)))+" │")
	}
	return strings.Join(append(lines, "└"+strings.Repeat("─", width-2)+"┘"), "\n")
}

func panelPair(left, right string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
}

// Document sources stay unframed in model.document. Rebuild their panels after
// every width change, including when returning from an overlay at a new size.
// Existing explicit headings define the groups; blank paragraphs define records.
func documentPanels(page, text string, width int) string {
	fallback := map[string]string{
		"help": "Guidance", "history": "Recorded event", "technical": "Saved record",
		"discovery": "Discovery report", "instructions": "Session output",
		"reload-error": "Refresh failed", "review": "Review", "result": "Result",
		"discard": "Decision",
	}[page]
	if fallback == "" {
		return text // Forms and overviews already compose their own geometry.
	}
	var groups []string
	title := fallback
	var body []string
	flush := func() {
		if len(body) > 0 {
			groups = append(groups, panel(title, strings.Join(body, "\n"), width))
			body = nil
			title = fallback
		}
	}
	for _, block := range strings.Split(strings.TrimSpace(text), "\n\n") {
		for i, line := range strings.Split(block, "\n") {
			plain := ansi.Strip(line)
			heading := line != plain && strings.HasPrefix(line, "\x1b[1;")
			// Plain headings from context-specific help do not depend on color.
			heading = heading || (i == 0 && (strings.HasSuffix(plain, " help") ||
				strings.HasSuffix(plain, " help: read-only review")))
			if heading {
				if len(body) == 0 && title != fallback {
					// Preserve a document title followed by a subheading, e.g.
					// Help: Destinations → Actions here.
					body = append(body, plain)
					continue
				}
				flush()
				title = strings.TrimSuffix(plain, ":")
				switch {
				case strings.HasPrefix(title, "Result: "):
					body = append(body, title)
					title = "Result"
				}
			} else {
				body = append(body, line)
			}
		}
		flush()
	}
	flush()
	return strings.Join(groups, "\n\n")
}
