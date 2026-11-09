package tui

import (
	"context"
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"ssh-keyflow/internal/app"
)

type entry struct{ kind, id, title, description string }
type screen struct {
	page, document, copyText string
	target                   entry
	offset                   int
	guideSteps               []app.GuideStep
	guideIndex               int
}
type model struct {
	ctx                context.Context
	options            app.Options
	view               app.View
	signals            *handoffSignals
	width, height, tab int
	selected           [2]int
	queries            [2]string
	page, status       string
	target             entry
	origins            []screen
	content            viewport.Model
	document, copyText string
	guideSteps         []app.GuideStep
	guideIndex         int
	search             textinput.Model
	searching          bool
	inputGeneration    uint64
	inputRow           int
	fields             []field
	focus              int
	action, formError  string
	request            app.Request
	choices            []entry
	choiceIndex        int
	choiceQuery        string
	pickerAction       string
	running, stale     bool
	result             operationResult
}

func Run(ctx context.Context, options app.Options) error {
	p, err := NewProgram(ctx, options)
	if err != nil {
		return err
	}
	_, err = p.Run()
	return err
}

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
	initial := newModel(ctx, options, view)
	defaults := []tea.ProgramOption{tea.WithContext(ctx), tea.WithInput(options.Input), tea.WithOutput(options.Output), tea.WithWindowSize(80, 24)}
	p := tea.NewProgram(initial, append(append(defaults, programOptions...), tea.WithoutSignalHandler())...)
	initial.signals.program = p
	return p, nil
}

func newModel(ctx context.Context, options app.Options, view app.View) model {
	m := model{ctx: ctx, options: options, view: view, width: 80, height: 24, page: "list", signals: &handoffSignals{},
		content: viewport.New(viewport.WithWidth(80), viewport.WithHeight(18))}
	m.content.SoftWrap = false
	m.search = textinput.New()
	m.search.SetVirtualCursor(false)
	m.search.Prompt = "Search: "
	m.search.CharLimit = 4096
	m.resize()
	return m
}

func (m model) Init() tea.Cmd { return m.signals.listen }

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	m.resize()
	if response, ok := message.(inputResponse); ok {
		if response.generation != m.inputGeneration {
			return m, nil
		}
		message = response.message
	}
	if m.running {
		switch message.(type) {
		case tea.WindowSizeMsg, completed:
		default:
			return m, nil
		}
	}
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resize()
		if m.page == "form" {
			m.renderForm()
		}
		return m, nil
	case completed:
		m.signals.finish()
		m.running = false
		m.result = msg.result
		m.origins = nil
		m.page, m.target, m.copyText = "result", entry{}, ""
		m.reload()
		m.setDocument(m.result.text())
		return m, nil
	case tea.KeyPressMsg:
		key := msg.String()
		if m.running {
			return m, nil
		}
		if m.width < 32 || m.height < 12 {
			if key == "q" || key == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil
		}
		if m.searching {
			if key == "enter" || key == "esc" {
				m.searching = false
				m.search.Blur()
				m.inputGeneration++
				return m, nil
			}
			return m.updateInput(message)
		}
		if m.page == "form" {
			if key == "ctrl+c" || key == "esc" {
				m.status = "Draft cancelled"
				m.back()
				return m, nil
			}
			if key == "?" && m.focus >= len(m.fields) {
				m.open("help", entry{}, helpText, "")
				return m, nil
			}
			return m.updateForm(message)
		}
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		if key == "esc" || (m.page == "help" && key == "?") {
			m.back()
			return m, nil
		}
		if key == "q" {
			return m, tea.Quit
		}
		if key == "?" {
			m.open("help", entry{}, helpText, "")
			return m, nil
		}
		if key == "f5" && m.page != "picker" {
			m.reloadPage()
			return m, nil
		}
		if (m.page == "list" || m.page == "details") && (key == "tab" || key == "shift+tab") {
			m.tab = 1 - m.tab
			m.page, m.target, m.origins, m.status = "list", entry{}, nil, ""
			return m, nil
		}
		if (m.page == "list" || m.page == "picker") && key == "/" {
			m.searching = true
			m.inputGeneration++
			value := m.queries[m.tab]
			if m.page == "picker" {
				value = m.choiceQuery
			}
			m.search.SetValue(value)
			return m, m.inputCommand(m.search.Focus())
		}
		if (m.page == "list" || m.page == "picker") && key == "x" {
			if m.page == "picker" {
				m.choiceQuery = ""
				m.choiceIndex = 0
			} else {
				m.queries[m.tab] = ""
				m.selected[m.tab] = 0
			}
			return m, nil
		}
		if m.page == "picker" {
			return m.updateSelection(key)
		}
		if m.page == "list" {
			switch key {
			case "up", "k", "down", "j", "home", "end", "pgup", "pgdown":
				return m.updateSelection(key)
			case "enter":
				if item := m.visibleTarget(); item.id != "" {
					m.open("details", item, m.details(item), "")
				}
				return m, nil
			}
		}
		if m.page == "document" && len(m.guideSteps) > 0 && !m.stale {
			switch key {
			case "right", "n", "left", "b":
				if key == "right" || key == "n" {
					m.guideIndex = min(m.guideIndex+1, len(m.guideSteps)-1)
				} else {
					m.guideIndex = max(0, m.guideIndex-1)
				}
				m.showGuideStep()
				return m, nil
			}
		}
		if m.page == "document" && key == "y" && m.copyText != "" && !m.stale {
			m.status = "Copy requested (OSC52); remote state unchanged"
			return m, tea.SetClipboard(m.copyText)
		}
		if m.page == "list" || m.page == "details" {
			switch key {
			case "c", "i", "a":
				action := map[string]string{"c": "create", "i": "import", "a": "add"}[key]
				if (m.tab == 0 && key != "a") || (m.tab == 1 && key == "a") {
					command := m.beginForm(action)
					return m, command
				}
			case "l":
				command := m.beginPair("associate")
				return m, command
			case "u":
				command := m.beginPair("dissociate")
				return m, command
			case "d":
				target := m.visibleTarget()
				request := app.Request{}
				switch target.kind {
				case "key":
					request.Operation, request.Key = "remove-key", target.id
				case "host":
					request.Operation, request.Host = "remove-host", target.id
				default:
					m.status = "Select a managed key or destination to remove"
					return m, nil
				}
				command := m.execute(request)
				return m, command
			case "g":
				command := m.beginPair("guide")
				return m, command
			case "p":
				target := m.visibleTarget()
				if target.kind == "key" {
					m.showPublic(target.id)
				} else if target.kind == "host" {
					command := m.beginPair("public")
					return m, command
				}
				return m, nil
			case "s":
				if target := m.visibleTarget(); target.kind == "creation" {
					command := m.execute(app.Request{Operation: "resume", Resume: target.id})
					return m, command
				}
			case "v":
				if target := m.visibleTarget(); target.id != "" {
					m.open("technical", target, m.technical(target), "")
				} else if m.view.Review != "" {
					m.open("technical", entry{}, m.view.Review, "")
				}
				return m, nil
			}
		}
		if key == "home" {
			m.content.GotoTop()
			return m, nil
		}
		if key == "end" {
			m.content.GotoBottom()
			return m, nil
		}
	}
	if m.searching || m.page == "form" {
		return m.updateInput(message)
	}
	if m.page != "list" && m.page != "picker" && !m.running {
		var command tea.Cmd
		m.content, command = m.content.Update(message)
		return m, command
	}
	return m, nil
}

func (m model) chrome() (string, string) {
	tabs := "[Keys] | Destinations"
	if m.tab == 1 {
		tabs = "Keys | [Destinations]"
	}
	title := "SSH KeyFlow\n" + tabs
	switch m.page {
	case "form":
		title = "SSH KeyFlow\n" + actionTitle(m.action)
	case "picker":
		title = "SSH KeyFlow\nChoose " + m.choiceKind()
	case "document":
		title = "SSH KeyFlow\nCopy-paste instructions"
	case "technical":
		title = "SSH KeyFlow\nTechnical details (historical)"
	case "help":
		title = "SSH KeyFlow\nHelp"
	case "result":
		title = "SSH KeyFlow\nLocal operation result"
	}
	footer := "↑/↓ select · Enter details · / search\nTab switch · ? help · q quit"
	switch m.page {
	case "list":
		if m.tab == 0 {
			footer = "c create · i import · d remove · u unlink\n" + footer
		} else {
			footer = "a add · l link · u unlink · d remove\n" + footer
		}
	case "details":
		footer = "p public · g guide · d remove · u unlink\nv details · ↑/↓ scroll · Esc back"
		if m.target.kind == "creation" {
			footer = "s resume local creation\n↑/↓ scroll · Esc back"
		}
	case "picker":
		footer = "↑/↓ select · Enter choose\n/ search · x clear · Esc back"
	case "form":
		footer = "Tab move · Ctrl+S continue\nPgUp/PgDn scroll · Esc cancel"
	case "document":
		footer = "y copy (OSC52) · ↑/↓ scroll\nHome/End · Esc back · ? help"
		if len(m.guideSteps) > 0 {
			footer = "y copy script · ←/→ steps (b/n)\n↑/↓ scroll · Esc back · ? help"
		}
	default:
		footer = "↑/↓ scroll · Home/End\nEsc back · F5 refresh · ? help"
	}
	if m.searching {
		footer = "Enter apply · Esc keep search"
	}
	if m.running {
		footer = "Native terminal operation in progress"
	}
	status := m.status
	if status == "" && m.view.Review != "" && m.page == "list" {
		status = "Manual review notice: v details"
	}
	if m.stale {
		status = "State stale; F5 refresh required"
	}
	if status != "" {
		footer = ansi.Truncate(safeText(status), max(1, m.width), "…") + "\n" + footer
	}
	return title, footer
}

func (m *model) resize() {
	header, footer := m.chrome()
	extra := 0
	if (m.page == "list" && (m.searching || m.queries[m.tab] != "")) || (m.page == "picker" && (m.searching || m.choiceQuery != "")) {
		extra = 1
	}
	height := max(1, m.height-strings.Count(header, "\n")-strings.Count(footer, "\n")-4-extra)
	width := max(1, m.width)
	offset := m.content.YOffset()
	m.content.SetWidth(width)
	m.content.SetHeight(height)
	m.search.SetWidth(max(1, width-10))
	m.content.SetContent(ansi.Hardwrap(safeText(m.document), width, true))
	m.content.SetYOffset(offset)
	for i := range m.fields {
		m.fields[i].input.SetWidth(max(1, width-4))
	}
}

func (m model) View() tea.View {
	if m.width < 32 || m.height < 12 {
		return terminalView(fit("Resize: minimum 32x12\nq quit", m.width, m.height))
	}
	m.resize()
	header, footer := m.chrome()
	body := m.content.View()
	if m.page == "list" || m.page == "picker" {
		items, index := m.visibleEntries(), m.selected[m.tab]
		if m.page == "picker" {
			index = m.choiceIndex
		}
		body = m.listText(items, index)
		query := m.queries[m.tab]
		if m.page == "picker" {
			query = m.choiceQuery
		}
		if m.searching || query != "" {
			m.search.SetValue(query)
			body = m.search.View() + "\n" + body
		}
	}
	view := terminalView(fit(header+"\n"+body+"\n"+footer, m.width, m.height))
	if m.page == "form" {
		view.Cursor = m.formCursor()
	}
	if m.searching {
		view.Cursor = m.search.Cursor()
		if view.Cursor != nil {
			view.Cursor.Y += 2
		}
	}
	return view
}

func terminalView(text string) tea.View {
	view := tea.NewView(text)
	view.AltScreen = true
	return view
}
func fit(text string, width, height int) string {
	lines := strings.Split(text, "\n")
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], max(1, width), "…")
	}
	return strings.Join(lines[:min(len(lines), max(1, height))], "\n")
}
func safeText(text string) string {
	text = ansi.Strip(text)
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if r < 32 || (r >= 0x7f && r < 0xa0) {
			return -1
		}
		return r
	}, text)
}
func (m *model) setDocument(text string) { m.document = text; m.resize(); m.content.GotoTop() }
func (m *model) open(page string, target entry, text, copyText string) {
	m.origins = append(m.origins, screen{m.page, m.document, m.copyText, m.target, m.content.YOffset(), m.guideSteps, m.guideIndex})
	m.page, m.target, m.copyText, m.searching = page, target, copyText, false
	m.guideSteps, m.guideIndex = nil, 0
	m.inputGeneration++
	m.status = ""
	m.setDocument(text)
}
func (m *model) back() {
	m.inputGeneration++
	m.searching = false
	if len(m.origins) == 0 {
		m.page, m.target = "list", entry{}
		return
	}
	previous := m.origins[len(m.origins)-1]
	m.origins = m.origins[:len(m.origins)-1]
	m.page, m.target, m.copyText = previous.page, previous.target, previous.copyText
	m.guideSteps, m.guideIndex = previous.guideSteps, previous.guideIndex
	m.setDocument(previous.document)
	m.content.SetYOffset(previous.offset)
	if m.page == "form" {
		m.renderForm()
	}
}
func (m *model) reload() {
	// Returning from Help must not restore copy data from before this reload.
	m.copyText = ""
	m.guideSteps = nil
	for i := range m.origins {
		m.origins[i].copyText = ""
		m.origins[i].guideSteps = nil
	}
	view, err := app.Snapshot(m.options)
	if err != nil {
		m.stale = true
		m.status = "Reload failed: " + err.Error()
		m.result.reloadErr = err
		return
	}
	m.view, m.stale = view, false
	m.result.reloadErr = nil
	m.status = "State refreshed"
}
func (m *model) reloadPage() {
	m.reload()
	switch m.page {
	case "result":
		m.setDocument(m.result.text())
	case "details":
		m.setDocument(m.details(m.target))
	case "technical":
		m.setDocument(m.technical(m.target))
	case "document":
		m.status = "Reopen public key or guide to copy refreshed text"
	}
}

const helpText = `Keys / Destinations
Tab / Shift+Tab: switch category
↑/↓ or j/k: select; Enter: details
/ search; x clear; Home/End; PgUp/PgDn
Esc: back; F5: refresh; q / Ctrl+C: quit

Keys
c create Ed25519 key; i import existing key
p display public key; y copies displayed text
s resumes the selected unfinished local creation
d removes the selected managed key and its local associations
Passphrases are entered only in native ssh-keygen.

Destinations
a add using an existing key; l associate a key
d removes the selected destination and its local associations
u disconnects an existing association from either category
Removal and disconnect require confirmation; key files and SSH access stay unchanged.
g display terminal scripts for destination setup, local config and connection
p display a selected public key
Before the guide, choose the local and destination OS: linux / macos / windows.
Defaults: running app OS locally; saved destination OS/account remotely.
If local OS differs, supply the matching private-key path in that terminal.
Platform/path choices apply only to this guide; saved records stay the same.
In a guide: ←/→ or b/n select step; y copies only that step's script.
Each step names the terminal and platform where you should paste it.

Copy uses terminal OSC52 support. If unavailable, run:
ssh-keyflow keys public --key ID
ssh-keyflow hosts guide --host ID --key ID
Use the CLI output with the system clipboard or an editor, preserving the public key as a single line. Do not select wrapped TUI text.
Copying never confirms remote registration or a working connection.
Scripts change files only when you paste and run them in the named terminal.

Forms
Tab / Shift+Tab move; Enter next or activate control
Ctrl+S continue; Esc / Ctrl+C cancel draft
Continue prepares the guide or hands off a local change for confirmation.

v technical details: historical records are read-only.
Legacy remote operations cannot be resumed. Only local key creation can resume.
Paths are shown for manual local cleanup; this utility does not delete key files.`

func actionTitle(action string) string {
	return map[string]string{"create": "Create Ed25519 key", "import": "Import existing key", "add": "Add destination", "associate": "Associate key", "dissociate": "Disconnect association", "remove-key": "Remove managed key", "remove-host": "Remove destination", "resume": "Resume local creation", "guide": "Guide platforms (this guide only)"}[action]
}
func (m model) choiceKind() string {
	if len(m.choices) > 0 && m.choices[0].kind == "host" {
		return "destination"
	}
	return "key"
}
func (m model) listText(items []entry, index int) string {
	height := m.content.Height()
	if len(items) == 0 {
		text := "No matching records. x: clear search"
		if m.page == "list" && m.queries[m.tab] == "" {
			if m.tab == 0 {
				text = "No keys yet. c: create / i: import"
			} else {
				text = "No destinations yet. a: add\nCreate or import a key first."
			}
		}
		return padLines(fit(text, m.width, height), height)
	}
	index = max(0, min(index, len(items)-1))
	perPage := max(1, (height-1)/2)
	start := (index / perPage) * perPage
	var lines []string
	lines = append(lines, fmt.Sprintf("%d/%d", index+1, len(items)))
	for i := start; i < min(len(items), start+perPage); i++ {
		prefix := "  "
		if i == index {
			prefix = "> "
		}
		lines = append(lines, prefix+safeText(items[i].title), "  "+safeText(items[i].description))
	}
	return padLines(fit(strings.Join(lines, "\n"), m.width, height), height)
}
func padLines(text string, height int) string {
	return text + strings.Repeat("\n", max(0, height-strings.Count(text, "\n")-1))
}
