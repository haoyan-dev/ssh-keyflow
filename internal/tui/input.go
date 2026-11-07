package tui

import tea "charm.land/bubbletea/v2"

// Textinput commands (clipboard replies and cursor ticks) belong to the input
// that issued them, even if the user changes focus before the response arrives.
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

func (m model) updateSearch(message tea.Msg) (tea.Model, tea.Cmd) {
	var command tea.Cmd
	m.search, command = m.search.Update(message)
	if m.page == "picker" {
		m.choiceQuery = m.search.Value()
		m.refreshChoices()
	} else {
		m.queries[m.tab] = m.search.Value()
		m.refreshList()
	}
	return m, m.inputCommand(command)
}
