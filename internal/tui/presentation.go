package tui

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Use the terminal's own palette and background so custom themes remain usable.
var (
	accentStyle   = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(6)).Bold(true)
	borderStyle   = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(6))
	selectedStyle = lipgloss.NewStyle().Reverse(true).Bold(true)
	infoStyle     = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(6))
	warningStyle  = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(3)).Bold(true)
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(1)).Bold(true)
)

func paint(style lipgloss.Style, text string) string {
	if os.Getenv("NO_COLOR") != "" {
		return text
	}
	return style.Render(text)
}

func safeLine(text string) string {
	return strings.NewReplacer("\n", " ", "\t", " ").Replace(safeText(text))
}

type paneLayout struct {
	bodyHeight       int
	search, messages bool
	menu, message    string
}

func (m model) query() string {
	if m.page == "picker" {
		return m.choiceQuery
	}
	return m.queries[m.tab]
}

func (m model) layout() paneLayout {
	_, menu := m.chrome()
	width := max(1, m.width-4)
	if m.width < 60 {
		switch m.page {
		case "list":
			menu = "↑/↓ select · Enter details\nTab tabs  / search  ?  q quit"
			if m.tab == 0 {
				menu = "c create  i import  d remove\n" + menu
			} else {
				menu = "a add · l link · d remove\n" + menu
			}
		case "details":
			menu = "p public · g guide · d remove\n↑/↓ scroll · Esc back · ? help"
			if m.target.kind == "creation" {
				menu = "s resume · ↑/↓ scroll\nEsc back · ? help"
			}
		case "document":
			menu = "y copy · ↑/↓ scroll\nEsc back · ? help"
			if len(m.guideSteps) > 0 {
				menu = "y copy · ←/→ steps\n↑/↓ scroll · Esc back"
			}
		}
	}
	if m.page == "form" {
		var buttons []string
		for i, label := range []string{"Continue", "Cancel", "Help"} {
			button := "[" + label + "]"
			if m.focus == len(m.fields)+i {
				button = paint(selectedStyle, ">"+button)
			}
			buttons = append(buttons, button)
		}
		menu = strings.Join(buttons, " ") + "\nTab move · Ctrl+S continue\nPgUp/PgDn · Esc cancel"
	}
	if m.searching {
		menu = "Enter apply · Esc keep search"
	}
	if m.running {
		menu = "Native terminal operation\nin progress"
	}
	message := safeText(m.status)
	if message == "" && m.view.Review != "" && m.page == "list" {
		message = "Manual review notice: v details"
	}
	if m.stale {
		message = "State stale; F5 refresh required"
	}
	if m.page == "form" && m.formError != "" {
		message = "Error: " + safeText(m.formError)
	}
	layout := paneLayout{
		search:   (m.page == "list" || m.page == "picker") && (m.searching || m.query() != ""),
		messages: message != "" || m.height >= 20,
		menu:     fit(menu, width, m.height),
	}
	menuRows := m.height - 6
	if layout.search {
		menuRows -= 2
	}
	if layout.messages {
		menuRows -= 2
	}
	layout.menu = fit(layout.menu, width, max(1, menuRows))
	// Shared borders keep separate panes usable even in a 32x12 terminal.
	overhead := 5 + strings.Count(layout.menu, "\n") + 1
	if layout.search {
		overhead += 2
	}
	if layout.messages {
		if message == "" {
			message = "No messages"
		}
		message = ansi.Hardwrap(message, width, true)
		messageRows := min(3, max(1, m.height-overhead-4))
		layout.message = fit(message, width, messageRows)
		if strings.Count(message, "\n") >= messageRows {
			lines := strings.Split(layout.message, "\n")
			lines[len(lines)-1] = ansi.Truncate(lines[len(lines)-1], width-1, "") + "…"
			layout.message = strings.Join(lines, "\n")
		}
		overhead += 2 + strings.Count(layout.message, "\n")
	}
	layout.bodyHeight = max(1, m.height-overhead)
	return layout
}

func (m model) navigation() string {
	if m.page == "list" || m.page == "details" {
		keys, hosts := "[Keys]", "Destinations"
		if m.tab == 1 {
			keys, hosts = "Keys", "[Destinations]"
			hosts = paint(accentStyle, hosts)
		} else {
			keys = paint(accentStyle, keys)
		}
		return keys + "  " + hosts
	}
	header, _ := m.chrome()
	_, title, _ := strings.Cut(header, "\n")
	return paint(accentStyle, safeText(title))
}

func (m model) paneTitle() string {
	switch m.page {
	case "list", "picker":
		title, index := "Keys", m.selected[m.tab]
		if m.tab == 1 {
			title = "Destinations"
		}
		if m.page == "picker" {
			title, index = "Choose "+m.choiceKind(), m.choiceIndex
		}
		items := m.visibleEntries()
		if len(items) == 0 {
			return title + " · 0 records"
		}
		return fmt.Sprintf("%s · %d/%d", title, max(0, min(index, len(items)-1))+1, len(items))
	case "form":
		return "Input"
	case "details":
		return "Details"
	case "document":
		return "Copy-paste instructions"
	case "technical":
		return "Historical details"
	case "result":
		return "Local operation result"
	default:
		return "Help"
	}
}

func (m model) rule(title, left, right string, style lipgloss.Style) string {
	label := ""
	if title != "" {
		label = " " + ansi.Truncate(safeText(title), max(1, m.width-6), "…") + " "
	}
	return paint(style, left+"─"+label+strings.Repeat("─", max(0, m.width-3-ansi.StringWidth(label)))+right)
}

func (m model) paneRow(text string) string {
	width := max(1, m.width-4)
	text = ansi.Truncate(text, width, "…")
	return paint(borderStyle, "│") + " " + text + strings.Repeat(" ", max(0, width-ansi.StringWidth(text))) + " " + paint(borderStyle, "│")
}

func (m model) entryHeight() int {
	return min(3, max(1, m.content.Height()))
}

func (m model) bodyTop() int {
	if m.layout().search {
		return 5
	}
	return 3
}
