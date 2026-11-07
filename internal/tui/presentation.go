package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func actionTitle(action string) string {
	return map[string]string{
		"create": "Create Ed25519 key", "import": "Import existing key",
		"add": "Add destination", "register": "Register public key",
		"unregister": "Unregister public key", "delete": "Delete local key",
		"resume": "Resume saved operation",
	}[action]
}

// Lists retain Bubbles' keyboard/selection model, but share our page hierarchy.
// Row summaries may be shortened; Enter always exposes the complete record.
func (m model) selectionLayout(selection list.Model) (string, int, int) {
	items := selection.Items()
	width := max(1, m.width-2)
	height := m.content.Height()
	title := selection.Title
	if m.page == "picker" && title == "Choose related destination/key" {
		title = "Destinations"
		if m.formTarget.Host != "" {
			title = "Managed keys"
		}
	}
	position := fmt.Sprintf("%d/%d", selection.Index()+1, len(items))
	if m.width < 60 && m.page == "list" {
		title = []string{"Destinations", "Managed keys", "Registrations", "Tasks"}[m.tab]
	}
	heading := ansi.Truncate(title, max(1, width-len(position)-2), "…") + "  " + position
	var text strings.Builder
	if m.page == "picker" {
		if m.pickerContext != "" {
			text.WriteString(ansi.Truncate(m.pickerContext, width, "…") + "\n")
		}
	}
	text.WriteString(detailHeading(heading))
	rowHeight := 3
	if height < 7 || m.width < 60 {
		rowHeight = 2
	} else {
		text.WriteString("\n ")
	}
	if m.page == "list" && m.tab == 2 && m.width < 60 {
		rowHeight = 3 // Independent evidence must not be truncated into "fai…".
	}
	if m.page == "list" && m.tab == 3 {
		rowHeight = 3 // Let the reason span two lines before opening details.
	}
	prefix := text.String()
	return prefix, rowHeight, max(1, (height-lipgloss.Height(prefix))/rowHeight)
}

func (m model) sizeSelection(selection *list.Model) {
	index := selection.Index()
	prefix, rowHeight, _ := m.selectionLayout(*selection)
	selection.SetShowTitle(false)
	selection.SetShowStatusBar(false)
	selection.SetShowPagination(false)
	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(rowHeight)
	delegate.SetSpacing(0)
	selection.SetDelegate(delegate)
	selection.SetSize(max(10, m.width-2), max(1, m.content.Height()-lipgloss.Height(prefix)))
	selection.Select(max(0, index))
}

func (m model) selectionView(selection list.Model) string {
	items := selection.Items()
	if len(items) == 0 {
		return ""
	}
	width := max(1, m.width-2)
	prefix, rowHeight, perPage := m.selectionLayout(selection)
	var text strings.Builder
	text.WriteString(prefix + "\n")
	index := selection.Index()
	start := (index / perPage) * perPage
	for i := start; i < min(len(items), start+perPage); i++ {
		item, ok := items[i].(entry)
		if !ok {
			continue
		}
		marker, title := "  ", item.title
		if item.reference != "" {
			suffix := " [" + item.reference + "]"
			title = ansi.Truncate(strings.TrimSuffix(title, suffix), max(1, width-2-ansi.StringWidth(suffix)), "…") + suffix
		}
		if i == index {
			marker, title = "> ", detailHeading(ansi.Truncate(title, max(1, width-2), "…"))
		} else {
			title = ansi.Truncate(title, max(1, width-2), "…")
		}
		text.WriteString(marker + title + "\n")
		description := item.description
		if m.page == "list" && m.tab == 2 && m.width < 60 {
			description = strings.Replace(description, " · Connection: ", "\nConnection: ", 1)
			for lineIndex, line := range strings.Split(description, "\n") {
				if lineIndex > 0 {
					text.WriteString("\n")
				}
				text.WriteString("  " + ansi.Truncate(strings.TrimSpace(line), max(1, width-2), "…"))
			}
		} else if m.page == "list" && m.tab == 3 {
			lines := strings.Split(ansi.Wrap(description, max(1, width-2), ""), "\n")
			for lineIndex, line := range lines[:min(2, len(lines))] {
				if lineIndex > 0 {
					text.WriteString("\n")
				}
				if lineIndex == 1 && len(lines) > 2 {
					line = ansi.Truncate(line+"…", max(1, width-2), "…")
				}
				text.WriteString("  " + line)
			}
			if len(lines) < 2 {
				text.WriteString("\n")
			}
		} else {
			text.WriteString("  " + ansi.Truncate(description, max(1, width-2), "…"))
		}
		if rowHeight == 3 && !(m.page == "list" && (m.tab == 3 || (m.tab == 2 && m.width < 60))) {
			text.WriteString("\n")
		}
		if i+1 < min(len(items), start+perPage) {
			text.WriteString("\n")
		}
	}
	return strings.TrimRight(text.String(), "\n")
}

func (m model) location(categories string) string {
	switch m.page {
	case "list", "details":
		return categories
	case "form":
		return "Prepare · " + actionTitle(m.action)
	case "review":
		return "Review · " + actionTitle(m.action)
	case "picker":
		switch m.choices.Title {
		case "Choose related destination/key":
			if m.formTarget.Host != "" {
				return "Register > Choose key"
			}
			return "Register > Choose destination"
		case "Choose managed key":
			return "Add destination > Choose key"
		case "Choose registration to unregister":
			return "Unregister > Choose registration"
		case "Choose saved operation to continue", "Choose result operation to continue":
			return "Continue > Choose saved work"
		default:
			return m.choices.Title
		}
	case "discard":
		return "Unsaved draft · esc keep editing"
	case "result":
		return "Operation result"
	case "technical":
		return "Technical details · read-only"
	case "history":
		return "History · read-only"
	case "discovery":
		return "Discovery · read-only"
	case "instructions":
		return "Instructions / diagnostics"
	case "help":
		return "Help · esc back"
	default:
		return "Saved state"
	}
}

func (m model) scrollPosition() string {
	if m.content.TotalLineCount() <= m.content.Height() {
		return "all"
	}
	if m.content.AtBottom() {
		return "end"
	}
	return fmt.Sprintf("%d%%", int(m.content.ScrollPercent()*100))
}

func (m model) listActions() string {
	target := m.visibleTarget()
	if m.stale {
		return "F5 refresh · read-only"
	}
	if target.readOnly {
		return "Enter details · read-only"
	}
	var actions []string
	switch m.tab {
	case 0:
		actions = []string{"a add"}
	case 1:
		actions = []string{"c create", "i import"}
	case 3:
		if target.id != "" && !target.readOnly {
			return "s continue · Enter details"
		}
		return "Enter details"
	}
	if target.id != "" && ((target.kind == "host" && len(m.view.State.Keys) > 0) || (target.kind == "key" && len(m.view.State.Hosts) > 0) || target.kind == "registration") {
		actions = append(actions, "r register")
	}
	if len(m.unregisterChoices(target)) > 0 {
		actions = append(actions, "u unregister")
	}
	if target.kind == "key" && m.deleteBlocker(target) == "" {
		actions = append(actions, "d delete")
	}
	if len(actions) == 0 {
		return "Enter details"
	}
	return strings.Join(actions, " · ")
}
