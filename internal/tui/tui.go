package tui

import (
	"context"
	"os"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"ssh-keyflow/internal/app"
)

type model struct {
	inputGeneration uint64
	signals         *handoffSignals
	result          operationResult
	resultControl   int
	initialDraft    map[string]string
	initialTarget   app.Request
	discardSelected bool
	editContent     viewport.Model
	reviewControl   int
	stale           bool
	formTarget      app.Request
	choices         list.Model
	choiceEntries   []entry
	choiceQuery     string
	pickerContext   string
	target          entry
	origins         []returnContext
	ctx             context.Context
	options         app.Options
	view            app.View
	list            list.Model
	lists           [4]list.Model
	queries         [4]string
	search          textinput.Model
	searching       bool
	tab             int
	width           int
	height          int
	page            string
	status          string
	content         viewport.Model
	document        string
	draft           map[string]string
	step            int
	formError       string
	errorField      string
	fields          []field
	focus           int
	action          string
	request         app.Request
	running         bool
}

type entry struct {
	title, description, kind, id, searchText string
	reference                                string
	readOnly                                 bool
}

func (item entry) Title() string       { return item.title }
func (item entry) Description() string { return item.description }
func (item entry) FilterValue() string {
	return item.title + " " + item.description + " " + item.id + " " + item.searchText
}

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
	initial := model{ctx: ctx, options: options, view: view, width: 80, height: 24, page: "list",
		content: viewport.New(viewport.WithWidth(76), viewport.WithHeight(15))}
	// Pre-wrap documents using terminal cell width so the viewport's logical
	// line count matches its rendered height, including long CJK values.
	initial.content.SoftWrap = false
	initial.search = textinput.New()
	initial.search.CharLimit = 4096
	initial.search.Prompt = "Search: "
	for index := range initial.lists {
		initial.lists[index] = list.New(nil, list.NewDefaultDelegate(), 76, 16)
		initial.lists[index].SetFilteringEnabled(false)
		initial.lists[index].SetShowHelp(false)
	}
	initial.list = initial.lists[0]
	initial.refreshList()
	defaults := []tea.ProgramOption{tea.WithContext(ctx), tea.WithInput(options.Input), tea.WithOutput(options.Output), tea.WithWindowSize(80, 24)}
	initial.signals = &handoffSignals{}
	program := tea.NewProgram(initial, append(append(defaults, programOptions...), tea.WithoutSignalHandler())...)
	initial.signals.program = program
	return program, nil
}

func (m model) Init() tea.Cmd { return m.signals.listen }

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	m.resizeBody()
	if response, ok := message.(inputResponse); ok {
		if response.generation != m.inputGeneration {
			return m, nil
		}
		message = response.message
	}
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = message.Width, message.Height
		m.resizeBody()
		for index := range m.fields {
			m.fields[index].input.SetWidth(max(1, m.width-7))
		}
		if m.page == "form" {
			m.revealFormFocus()
		}
	case tea.KeyPressMsg:
		pressed := message.String()
		if m.running {
			return m, nil
		}
		if m.width < 32 || m.height < 12 {
			if pressed == "q" || pressed == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil
		}
		if m.page == "help" && (pressed == "esc" || pressed == "?" || pressed == "ctrl+c") {
			m.back()
			return m, nil
		}
		if m.page == "discard" {
			return m.updateDiscard(message)
		}
		if pressed == "ctrl+c" && m.draftActive() {
			m.cancelDraft()
			return m, nil
		}
		if pressed == "ctrl+c" {
			return m, tea.Quit
		}
		if m.searching {
			switch pressed {
			case "enter", "esc":
				m.searching = false
				m.inputGeneration++
				m.search.Blur()
			default:
				return m.updateSearch(message)
			}
			return m, nil
		}
		if (m.page == "list" || m.page == "picker") && pressed == "/" {
			m.searching = true
			m.inputGeneration++
			if m.page == "picker" {
				m.search.SetValue(m.choiceQuery)
			} else {
				m.search.SetValue(m.queries[m.tab])
			}
			return m, m.inputCommand(m.search.Focus())
		}
		if (m.page == "list" || m.page == "picker") && pressed == "x" {
			if m.page == "picker" {
				m.choiceQuery = ""
				m.refreshChoices()
			} else {
				m.queries[m.tab] = ""
				m.refreshList()
			}
			return m, nil
		}
		if pressed == "q" && !m.draftActive() && m.page != "help" {
			return m, tea.Quit
		}
		if m.page == "form" {
			next, command := m.updateForm(message)
			updated := next.(model)
			return updated, updated.inputCommand(command)
		}
		if (pressed == "home" || pressed == "end") && m.page != "list" && m.page != "picker" {
			if pressed == "home" {
				m.content.GotoTop()
			} else {
				m.content.GotoBottom()
			}
			return m, nil
		}
		if m.page == "review" && (pressed == "esc" || pressed == "e") {
			m.editReview()
			return m, nil
		}
		if pressed == "esc" {
			m.back()
			return m, nil
		}
		if m.page == "review" {
			switch pressed {
			case "tab", "shift+tab":
				direction := 1
				if pressed == "shift+tab" {
					direction = 3
				}
				m.reviewControl = (m.reviewControl + direction) % 4
			case "?":
				m.openDocument("help", entry{}, m.helpText())
			case "c":
				m.cancelDraft()
			case "enter":
				switch m.reviewControl {
				case 1:
					m.editReview()
				case 2:
					m.cancelDraft()
				case 3:
					m.openDocument("help", entry{}, m.helpText())
				default:
					if err := m.resolveFormTargets(); err != nil {
						m.targetError(err)
						return m, nil
					}
					m.running = true
					ctx, cancel := context.WithCancel(m.ctx)
					m.signals.start(cancel)
					operation := &execution{ctx: ctx, options: m.options, request: m.request, target: m.handoffTarget()}
					return m, tea.Exec(operation, func(err error) tea.Msg {
						return operation.complete(err)
					})
				}
			default:
				var command tea.Cmd
				m.content, command = m.content.Update(message)
				return m, command
			}
			return m, nil
		}
		if (m.page == "list" || m.page == "details") && (pressed == "tab" || pressed == "shift+tab") {
			m.lists[m.tab] = m.list
			if pressed == "tab" {
				m.tab = (m.tab + 1) % 4
			} else {
				m.tab = (m.tab + 3) % 4
			}
			m.page, m.target, m.origins = "list", entry{}, nil
			m.list = m.lists[m.tab]
			m.refreshList()
			return m, nil
		}
		if pressed == "f5" && (m.page == "list" || m.page == "details" || m.page == "result" || m.page == "reload-error") {
			m.refreshSnapshot()
			return m, nil
		}
		if pressed == "h" && (m.page == "list" || m.page == "details") {
			target := m.visibleTarget()
			if m.page == "list" {
				target = entry{}
			}
			m.openDocument("history", entry{}, m.historyDetails(target))
			return m, nil
		}
		if pressed == "g" && m.page != "help" {
			m.openDocument("discovery", entry{}, m.discoveryDetails())
			return m, nil
		}
		if pressed == "?" {
			m.openDocument("help", entry{}, m.helpText())
			return m, nil
		}
		if pressed == "v" && m.page == "details" {
			m.openDocument("technical", m.target, m.technicalDetails(m.target))
			return m, nil
		}
		if m.page == "result" {
			return m.updateResult(message)
		}
		if m.page == "help" {
			var command tea.Cmd
			m.content, command = m.content.Update(message)
			return m, command
		}
		if pressed == "enter" && m.page == "picker" {
			m.chooseTarget()
			return m, nil
		}
		if pressed == "enter" && m.page == "list" {
			if item, ok := m.list.SelectedItem().(entry); ok {
				m.openDocument("details", item, m.entityDetails(item))
			}
			return m, nil
		}
		actions := map[string]string{"c": "create", "i": "import", "a": "add", "r": "register", "u": "unregister", "d": "delete", "s": "resume"}
		if action := actions[pressed]; action != "" {
			m.beginAction(action)
			return m, nil
		}
	case completed:
		m.signals.finish()
		m.running = false
		m.result = message.result
		m.resultControl = 0
		m.page, m.target, m.origins = "result", entry{}, nil
		view, err := app.Snapshot(m.options)
		m.result.reloadErr = err
		if err != nil {
			m.stale = true
		} else {
			m.view, m.stale = view, false
			m.refreshList()
		}
		m.renderOperationResult()
		m.content.GotoTop()
		return m, nil
	}
	if !m.running && m.width >= 32 && m.height >= 12 {
		if m.searching && (m.page == "list" || m.page == "picker") {
			return m.updateSearch(message)
		}
		if m.page == "form" {
			next, command := m.updateForm(message)
			updated := next.(model)
			return updated, updated.inputCommand(command)
		}
	}
	var command tea.Cmd
	if m.page == "list" {
		m.list, command = m.list.Update(message)
	} else if m.page == "picker" {
		m.choices, command = m.choices.Update(message)
	} else {
		m.content, command = m.content.Update(message)
	}
	return m, command
}

func (m model) chrome() (string, string) {
	tabs := []string{"Destinations", "Keys", "Registrations", "Tasks"}
	tabs[m.tab] = "[" + tabs[m.tab] + "]"
	categories := strings.Join(tabs, " | ")
	if m.width < 60 {
		categories = tabs[m.tab] + " (tab: switch)"
	}
	footer := "↑/↓ select · enter details · / search · x Clear search\n? help · h history · g discovery · F5 refresh · q quit"
	switch m.page {
	case "details":
		footer = "v technical · h history · F5 refresh\n↑/↓ scroll · ? help · esc back · q quit"
	case "technical":
		footer = "↑/↓ scroll · Home/End · ? help\nesc overview · q quit"
	case "history", "discovery", "result", "reload-error", "instructions":
		footer = "↑/↓ scroll · ? help · esc back · q quit"
	case "help":
		footer = "↑/↓ scroll · esc/? back · ctrl+c back"
	case "picker":
		footer = "↑/↓ select · enter choose · / search\nx Clear search · ? help · esc back"
	case "form":
		footer = "tab/shift+tab move · enter next · ctrl+s review · esc back\nPgUp/PgDn scroll · Help control"
	case "discard":
		footer = "tab/↑/↓ choose · enter select · esc keep editing\n? help"
	case "review":
		footer = m.reviewControls()
	}
	if m.width < 60 {
		switch m.page {
		case "list":
			footer = "/ search · enter details\n? help · q quit"
		case "details":
			footer = "v technical · F5 refresh\n↑/↓ scroll · ? help\nesc back · q quit"
		case "technical":
			footer = "↑/↓ scroll · ? help\nesc overview · q quit"
		case "history", "discovery", "result", "reload-error", "instructions":
			footer = "↑/↓ scroll · ? help\nesc back · q quit"
		case "picker":
			footer = "enter choose · / search\n? help · esc back"
		case "help":
			footer = "↑/↓ scroll · esc/? back"
		case "form":
			footer = "tab move · ctrl+s review\nesc back · PgUp/PgDn scroll"
		case "review":
			footer = m.reviewControls()
		}
	}
	if m.page == "details" && m.target.kind == "task" {
		footer = "v technical · ↑/↓ scroll\n? help · esc back · q quit"
		if m.width < 60 {
			footer = "v technical\n↑/↓ scroll · ? help\nesc back · q quit"
		}
		if !m.target.readOnly && !m.stale {
			footer = "s Continue · " + footer
			if m.width < 60 {
				footer = "s Continue · v technical\n↑/↓ scroll · ? help\nesc back · q quit"
			}
		}
	}
	if m.page == "result" {
		footer = m.resultFooter()
	}
	if m.draftActive() && m.page == "discovery" {
		footer = "↑/↓ scroll · ? help · esc back\nctrl+c cancel draft"
	}
	if m.searching {
		footer = "enter filter · esc keep query"
	}
	if m.page == "list" && !m.searching {
		actions := m.listActions()
		footer = actions + "\n↑/↓ select · Enter details · / search · Tab category · ? help · q quit"
		if m.width < 60 {
			// Keep primary actions on one row; Help contains the rest.
			actions = ansi.Truncate(actions, m.width, "…")
			footer = "Enter details · / search\n" + actions + "\nTab category · ? help · q quit"
		}
	}
	if m.page != "list" && m.page != "picker" && m.page != "form" && m.page != "discard" {
		footer = strings.Replace(footer, "↑/↓ scroll", "↑/↓ scroll "+m.scrollPosition(), 1)
	}
	status := m.status
	if m.stale {
		status = "Saved state stale; F5 Refresh / q Exit"
	}
	if status == "" && m.view.Review != "" && m.page == "list" {
		status = "Requires review (g: details)"
	}
	if status == "" && m.view.Review != "" && m.page == "details" {
		if m.width < 60 {
			if m.target.kind == "task" && m.target.readOnly {
				footer = strings.Replace(footer, "v technical\n", "v technical · g review\n", 1)
			} else {
				footer = strings.Replace(footer, "esc back · q quit", "g review · esc back · q quit", 1)
			}
		} else {
			footer = strings.Replace(footer, "F5 refresh", "g review · F5 refresh", 1)
		}
	}
	if status != "" {
		footer = ansi.Truncate(status, max(1, m.width), "…") + "\n" + footer
	}
	return detailHeading("SSH KeyFlow") + "\n" + ansi.Truncate(m.location(categories), max(1, m.width), "…"), ansi.Hardwrap(footer, max(1, m.width), false)
}

func (m *model) resizeBody() {
	header, footer := m.chrome()
	extra := 0
	if (m.page == "list" && (m.searching || m.queries[m.tab] != "")) || (m.page == "picker" && (m.searching || m.choiceQuery != "")) {
		extra = 1
	}
	available := max(1, m.height-lipgloss.Height(header)-lipgloss.Height(footer)-extra)
	m.search.SetWidth(max(1, m.width-10))
	m.content.SetHeight(available)
	m.sizeSelection(&m.list)
	width := max(10, m.width-2)
	if m.content.Width() != width {
		// Restored document origins may have been composed at an older width.
		offset := m.content.YOffset()
		m.content.SetWidth(width)
		switch m.page {
		case "details":
			m.setContent(m.entityDetails(m.target))
		case "form":
			m.setContent(m.formBody())
		default:
			m.setContent(m.document)
		}
		m.content.SetYOffset(offset)
	}
	if m.page == "picker" {
		m.sizeSelection(&m.choices)
	}
}

func (m model) View() tea.View {
	if m.width < 32 || m.height < 12 {
		lines := strings.Split(ansi.Hardwrap("Resize: minimum 32x12\nq quit", max(1, m.width), false), "\n")
		view := tea.NewView(strings.Join(lines[:min(len(lines), max(1, m.height))], "\n"))
		view.AltScreen = true
		return view
	}
	m.resizeBody()
	header, footer := m.chrome()
	body := m.selectionView(m.list)
	if m.page == "list" && len(m.list.Items()) == 0 {
		body = ansi.Hardwrap(m.emptyList(), max(1, m.width-2), false)
	}
	if m.page == "list" && (m.searching || m.queries[m.tab] != "") {
		m.search.SetWidth(max(1, m.width-10))
		m.search.SetValue(m.queries[m.tab])
		body = m.search.View() + "\n" + body
	}
	if m.page == "form" {
		body = m.formView()
	} else if m.page == "picker" {
		body = m.selectionView(m.choices)
		if len(m.choices.Items()) == 0 {
			body = "No matching eligible targets.\nx: Clear search · esc: Back"
		}
		if m.searching || m.choiceQuery != "" {
			m.search.SetWidth(max(1, m.width-10))
			body = m.search.View() + "\n" + body
		}
	} else if m.page != "list" {
		body = m.content.View()
	}
	if m.page == "list" || m.page == "picker" {
		extra := 0
		if m.searching || (m.page == "list" && m.queries[m.tab] != "") || (m.page == "picker" && m.choiceQuery != "") {
			extra = 1
		}
		body = lipgloss.NewStyle().Height(m.content.Height() + extra).Render(body)
	}
	view := tea.NewView(header + "\n" + body + "\n" + footer)
	view.AltScreen = true
	return view
}

func (m model) emptyList() string {
	if m.queries[m.tab] != "" {
		return "No matching records.\nx: Clear search"
	}
	switch m.tab {
	case 0:
		return "No destinations yet.\na: Add destination\nc: Create key / i: Import key"
	case 1:
		return "Managed keys\nNo managed keys yet.\nc: Create key / i: Import key"
	case 2:
		return "No registrations recorded.\nSelect a destination or key to register."
	default:
		return "No recorded follow-up.\nRemote access is not verified."
	}
}
