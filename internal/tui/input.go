package tui

import tea "charm.land/bubbletea/v2"

// Ignore delayed clipboard/cursor responses after focus or page changes.
type inputResponse struct {
	generation uint64
	message    tea.Msg
}

func (m model) inputCommand(command tea.Cmd) tea.Cmd {
	if command == nil {
		return nil
	}
	return func() tea.Msg {
		message := command()
		if batch, ok := message.(tea.BatchMsg); ok {
			for i, cmd := range batch {
				batch[i] = m.inputCommand(cmd)
			}
			return batch
		}
		return inputResponse{m.inputGeneration, message}
	}
}
func (m model) updateInput(message tea.Msg) (tea.Model, tea.Cmd) {
	var command tea.Cmd
	if m.searching {
		m.search, command = m.search.Update(message)
		if m.page == "picker" {
			m.choiceQuery = m.search.Value()
			m.choiceIndex = 0
		} else {
			m.queries[m.tab] = m.search.Value()
			m.selected[m.tab] = 0
		}
	} else if m.page == "form" && m.focus < len(m.fields) {
		m.fields[m.focus].input, command = m.fields[m.focus].input.Update(message)
		m.renderForm()
	}
	return m, m.inputCommand(command)
}
