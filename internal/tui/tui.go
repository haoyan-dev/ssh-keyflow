package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"ssh-keyflow/internal/app"
)

type model struct {
	ctx     context.Context
	options app.Options
	view    app.View
	list    list.Model
	tab     int
	width   int
	height  int
	page    string
	status  string
	content viewport.Model
	fields  []field
	focus   int
	action  string
	request app.Request
	running bool
}

type entry struct{ title, description, detail, kind, id string }

func (item entry) Title() string       { return item.title }
func (item entry) Description() string { return item.description }
func (item entry) FilterValue() string { return item.title + " " + item.description }

func Run(ctx context.Context, options app.Options) error {
	program, err := NewProgram(ctx, options)
	if err != nil {
		return err
	}
	_, err = program.Run()
	return err
}

// NewProgram exposes the same TUI entry for embedding and event-driven tests.
func NewProgram(ctx context.Context, options app.Options, programOptions ...tea.ProgramOption) (*tea.Program, error) {
	if options.Input == nil {
		options.Input = os.Stdin
	}
	if options.Output == nil {
		options.Output = os.Stdout
	}
	view, err := app.Snapshot(options)
	if err != nil {
		return nil, err
	}
	items := list.New(nil, list.NewDefaultDelegate(), 76, 16)
	items.SetFilteringEnabled(false)
	items.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "category")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "details"))}
	}
	items.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "create key")),
			key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "import key")),
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add destination")),
			key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "register")),
			key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "unregister")),
			key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete local key")),
			key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "resume")),
			key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "discovery")),
		}
	}
	initial := model{ctx: ctx, options: options, view: view, list: items, width: 80, height: 24, page: "list",
		content: viewport.New(viewport.WithWidth(76), viewport.WithHeight(15))}
	initial.content.SoftWrap = true
	initial.refreshList()
	defaults := []tea.ProgramOption{tea.WithContext(ctx), tea.WithInput(options.Input), tea.WithOutput(options.Output), tea.WithWindowSize(80, 24)}
	return tea.NewProgram(initial, append(defaults, programOptions...)...), nil
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = message.Width, message.Height
		m.list.SetSize(max(10, m.width-2), max(3, m.height-8))
		m.content.SetWidth(max(10, m.width-2))
		m.content.SetHeight(max(3, m.height-8))
		for index := range m.fields {
			m.fields[index].input.SetWidth(max(8, m.width-4))
		}
	case tea.KeyPressMsg:
		pressed := message.String()
		if pressed == "ctrl+c" && m.page == "form" {
			m.page, m.status = "list", "Changes cancelled"
			return m, nil
		}
		if pressed == "ctrl+c" || (pressed == "q" && m.page != "form") {
			return m, tea.Quit
		}
		if m.page == "form" {
			return m.updateForm(message)
		}
		if pressed == "esc" {
			if m.page == "review" {
				m.status = "Changes cancelled"
			}
			m.page = "list"
			return m, nil
		}
		if m.page == "review" {
			if pressed == "enter" {
				m.running = true
				operation := &execution{ctx: m.ctx, options: m.options, request: m.request}
				return m, tea.Exec(operation, func(err error) tea.Msg { return completed{err, operation.output.String()} })
			}
			var command tea.Cmd
			m.content, command = m.content.Update(message)
			return m, command
		}
		if pressed == "tab" || pressed == "shift+tab" {
			if pressed == "tab" {
				m.tab = (m.tab + 1) % 3
			} else {
				m.tab = (m.tab + 2) % 3
			}
			m.page = "list"
			m.refreshList()
			return m, nil
		}
		if pressed == "g" {
			m.page = "details"
			notes := ""
			if m.view.Review != "" {
				notes = "Requires review: " + m.view.Review + "\n\n"
			}
			m.content.SetContent("Discovery (read-only; unmanaged files are not imported)\n\n" + notes + m.view.Discovery)
			m.content.GotoTop()
			return m, nil
		}
		if pressed == "?" && m.page == "details" {
			m.page = "list"
			m.list.Help.ShowAll = true
			return m, nil
		}
		if pressed == "enter" && m.page == "list" {
			if item, ok := m.list.SelectedItem().(entry); ok {
				m.page = "details"
				m.content.SetContent(item.title + "\n\n" + item.detail)
				m.content.GotoTop()
			}
			return m, nil
		}
		actions := map[string]string{"c": "create", "i": "import", "a": "add", "r": "register", "u": "unregister", "d": "delete", "s": "resume"}
		if action := actions[pressed]; action != "" {
			m.openForm(action)
			return m, nil
		}
	case completed:
		m.running = false
		m.page = "details"
		m.status = "Operation complete"
		result := m.status
		if message.err != nil {
			m.status = "Operation stopped; see details"
			result = "Operation stopped: " + message.err.Error()
		}
		m.content.SetContent(result + "\n\n" + message.output)
		m.content.GotoTop()
		view, err := app.Snapshot(m.options)
		if err != nil {
			m.status = "Reload failed; see details"
			m.content.SetContent(result + "\n\n" + message.output + "\nReload failed: " + err.Error())
		} else {
			m.view = view
			m.refreshList()
		}
		return m, nil
	}
	var command tea.Cmd
	if m.page == "list" {
		m.list, command = m.list.Update(message)
	} else {
		m.content, command = m.content.Update(message)
	}
	return m, command
}

func (m model) View() tea.View {
	header := lipgloss.NewStyle().Bold(true).Render("SSH KeyFlow")
	tabs := []string{"Hosts", "Keys", "Registrations"}
	tabs[m.tab] = "[" + tabs[m.tab] + "]"
	categories := strings.Join(tabs, " | ")
	if m.width < 45 {
		categories = tabs[m.tab] + " (tab: categories)"
	}
	body := m.list.View()
	if m.page == "form" {
		body = m.formView()
	} else if m.page != "list" {
		body = m.content.View()
	}
	footer := "c create · i import · a add host · r register · u unregister · d delete · s resume · g discovery · esc back · q quit"
	if m.page == "form" {
		footer = "tab/shift+tab move · enter next/review · ctrl+s review · esc cancel"
	} else if m.page == "review" {
		footer = "enter continue to explicit terminal confirmation · esc cancel · ↑/↓ scroll"
	}
	if m.width < 60 {
		footer = "? help · esc back · q quit"
		if m.page == "form" {
			footer = "tab move · ctrl+s review\nesc cancel"
		} else if m.page == "review" {
			footer = "enter continue · esc cancel\n↑/↓ scroll"
		}
	}
	status := m.status
	if m.view.Review != "" {
		status += "\nRequires review (g: details)"
	}
	body = fmt.Sprintf("%s\n%s\n%s\n%s\n%s", header, categories, body, status, footer)
	body = lipgloss.NewStyle().Width(max(10, m.width)).Render(body)
	view := tea.NewView(body)
	view.AltScreen = true
	return view
}

func (m *model) refreshList() {
	var entries []list.Item
	add := func(kind, id, title, description string, value any) {
		detail, _ := json.MarshalIndent(value, "", "  ")
		entries = append(entries, entry{title, description, string(detail), kind, id})
	}
	switch m.tab {
	case 0:
		m.list.Title = "Managed destinations"
		for _, host := range m.view.State.Hosts {
			add("host", host.ID, host.Alias, fmt.Sprintf("%s@%s:%d · %s", host.User, host.Hostname, host.Port, host.OS), host)
		}
		for _, job := range m.view.State.Onboardings {
			add("onboarding", job.ID, "Onboarding "+job.ID, "stage="+job.Stage+" · host="+job.HostID, job)
		}
	case 1:
		m.list.Title = "Managed keys"
		for _, item := range m.view.State.Keys {
			add("key", item.ID, item.Name, item.Fingerprint+" · connection="+item.ConnectionCheck, item)
		}
		for _, job := range m.view.State.Creations {
			if job.Stage != "complete" {
				add("creation", job.ID, "Creation "+job.Name, "stage="+job.Stage+" · "+job.ID, job)
			}
		}
		for _, job := range m.view.State.Deletions {
			if job.Stage != "complete" {
				add("deletion", job.ID, "Deletion "+job.ID, "stage="+job.Stage+" · key="+job.KeyID, job)
			}
		}
		for _, item := range m.view.State.DeletedKeys {
			add("deleted", item.Key.ID, "Deleted "+item.Key.Name, item.Key.Fingerprint+" · history retained", item)
		}
	case 2:
		m.list.Title = "Registrations and recorded history"
		for _, item := range m.view.State.Registrations {
			add("registration", item.ID, item.ID, item.Status+" · connection="+item.ConnectionCheck+" · "+item.Operation+"/"+item.Stage, item)
		}
		for _, item := range m.view.State.History {
			add("history", item.RegistrationID, item.At+" · "+item.Operation, item.Status+" · connection="+item.ConnectionCheck+" · "+item.Stage, item)
		}
	}
	m.list.SetItems(entries)
}
