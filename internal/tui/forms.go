package tui

import (
	"context"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"ssh-keyflow/internal/app"
)

type field struct {
	name, label string
	input       textinput.Model
}

func (m *model) beginForm(action string) tea.Cmd {
	if m.stale {
		m.status = "Refresh saved state before changes"
		return nil
	}
	if action == "add" && len(m.view.State.Keys) == 0 {
		m.status = "Create or import an existing key first"
		return nil
	}
	m.action, m.focus, m.fields, m.formError = action, 0, nil, ""
	m.request = app.Request{Operation: action}
	add := func(name, label, value string) {
		input := textinput.New()
		input.SetVirtualCursor(false)
		input.Prompt = "  "
		input.CharLimit = 4096
		input.SetValue(value)
		m.fields = append(m.fields, field{name, label, input})
	}
	switch action {
	case "create":
		add("name", "Management name", "")
	case "import":
		add("name", "Management name", "")
		add("public", "Public key path", "")
		add("private", "Private key path (optional)", "")
	case "add":
		add("alias", "Destination alias", "")
		add("hostname", "Hostname / IP", "")
		add("user", "Login user", "")
		add("port", "Port", "22")
		add("os", "OS: linux / macos / windows", "")
		add("account", "Windows only: standard / administrator", "")
	}
	m.open("form", entry{}, "", "")
	command := m.fields[0].input.Focus()
	m.renderForm()
	return m.inputCommand(command)
}
func (m model) updateForm(message tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := message.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "ctrl+s":
			command := m.submitForm()
			return m, command
		case "tab", "shift+tab", "enter":
			if key.String() == "enter" && m.focus >= len(m.fields) {
				switch m.focus - len(m.fields) {
				case 0:
					command := m.submitForm()
					return m, command
				case 1:
					m.status = "Draft cancelled"
					m.back()
					return m, nil
				case 2:
					m.open("help", entry{}, helpText, "")
					return m, nil
				}
			}
			count := len(m.fields) + 3
			step := 1
			if key.String() == "shift+tab" {
				step = count - 1
			}
			if m.focus < len(m.fields) {
				m.fields[m.focus].input.Blur()
			}
			m.focus = (m.focus + step) % count
			m.inputGeneration++
			var command tea.Cmd
			if m.focus < len(m.fields) {
				command = m.fields[m.focus].input.Focus()
			}
			m.renderForm()
			return m, m.inputCommand(command)
		case "pgup", "pgdown":
			var command tea.Cmd
			m.content, command = m.content.Update(message)
			return m, command
		}
	}
	return m.updateInput(message)
}
func (m *model) submitForm() tea.Cmd {
	m.formError = ""
	values := map[string]string{}
	for _, field := range m.fields {
		values[field.name] = field.input.Value()
	}
	require := func(name string) bool {
		if strings.TrimSpace(values[name]) != "" && !strings.ContainsAny(values[name], "\x00\r\n") {
			return true
		}
		m.formError = "Enter a valid " + name
		for i := range m.fields {
			m.fields[i].input.Blur()
			if m.fields[i].name == name {
				m.focus = i
			}
		}
		m.inputGeneration++
		m.fields[m.focus].input.Focus()
		m.renderForm()
		return false
	}
	request := app.Request{Operation: m.action}
	switch m.action {
	case "create", "import":
		if !require("name") {
			return nil
		}
		request.Name = values["name"]
		if m.action == "import" {
			if !require("public") {
				return nil
			}
			request.PublicPath, request.PrivatePath = values["public"], values["private"]
		}
	case "add":
		for _, name := range []string{"alias", "hostname", "user", "port", "os"} {
			if !require(name) {
				return nil
			}
		}
		if values["os"] != "linux" && values["os"] != "macos" && values["os"] != "windows" {
			m.formError = "OS: select linux, macos or windows"
			m.renderForm()
			return nil
		}
		port, err := strconv.Atoi(values["port"])
		if err != nil {
			m.formError = "Port must be between 1 and 65535"
			m.renderForm()
			return nil
		}
		request.Destination = app.Destination{Alias: values["alias"], Hostname: values["hostname"], User: values["user"], Port: port, OS: values["os"]}
		if values["os"] == "windows" {
			if values["account"] != "standard" && values["account"] != "administrator" {
				m.formError = "Windows account: select standard or administrator"
				m.renderForm()
				return nil
			}
			request.Destination.WindowsAccount = values["account"]
		}
		if err := request.Destination.Validate(); err != nil {
			m.formError = err.Error()
			m.renderForm()
			return nil
		}
		m.request = request
		m.picker("add", "key")
		return nil
	}
	return m.execute(request)
}
func (m *model) renderForm() {
	var lines []string
	focusStart, focusEnd, inputRow := 0, 0, 0
	for i, field := range m.fields {
		prefix := "  "
		if i == m.focus {
			prefix = "> "
		}
		start := len(lines)
		lines = append(lines, strings.Split(ansi.Hardwrap(prefix+safeText(field.label), max(1, m.width), true), "\n")...)
		if i == m.focus {
			inputRow = len(lines)
		}
		value := safeText(field.input.View())
		lines = append(lines, value)
		if i == m.focus {
			focusStart, focusEnd = start, len(lines)
		}
	}
	for i, label := range []string{"Continue", "Cancel", "Help"} {
		prefix := "  "
		if m.focus == len(m.fields)+i {
			prefix = "> "
			focusStart, focusEnd = len(lines), len(lines)+1
		}
		lines = append(lines, prefix+"["+label+"]")
	}
	if m.formError != "" {
		// Keep validation beside the focused field on a small terminal.
		notice := strings.Split(ansi.Hardwrap("Error: "+safeText(m.formError), max(1, m.width), true), "\n")
		withError := append([]string{}, lines[:focusEnd]...)
		withError = append(withError, notice...)
		lines = append(withError, lines[focusEnd:]...)
	}
	m.document = strings.Join(lines, "\n")
	m.resize()
	if focusStart < m.content.YOffset() || focusEnd > m.content.YOffset()+m.content.Height() {
		m.content.SetYOffset(focusStart)
	}
	m.inputRow = inputRow
}
func (m *model) execute(request app.Request) tea.Cmd {
	if m.running || m.stale {
		m.status = "Refresh saved state before continuing"
		return nil
	}
	m.request = request
	m.running = true
	m.searching = false
	m.inputGeneration++
	ctx, cancel := context.WithCancel(m.ctx)
	m.signals.start(cancel)
	operation := &execution{ctx: ctx, options: m.options, request: request}
	return tea.Exec(operation, func(err error) tea.Msg { return operation.complete(err) })
}

func (m model) formCursor() *tea.Cursor {
	if m.focus >= len(m.fields) {
		return nil
	}
	cursor := m.fields[m.focus].input.Cursor()
	if cursor == nil {
		return nil
	}
	cursor.Y += 2 + m.inputRow - m.content.YOffset()
	if cursor.Y < 2 || cursor.Y >= 2+m.content.Height() {
		return nil
	}
	return cursor
}
