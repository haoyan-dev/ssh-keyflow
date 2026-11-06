package tui

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"ssh-keyflow/internal/app"
)

type field struct {
	name, label string
	input       textinput.Model
}

func (m *model) openForm(action string) {
	m.action, m.page, m.focus, m.status = action, "form", 0, ""
	m.fields = nil
	add := func(name, label, value string) {
		input := textinput.New()
		input.SetWidth(max(8, m.width-4))
		input.CharLimit = 4096
		input.SetValue(value)
		m.fields = append(m.fields, field{name, label, input})
	}
	selected, _ := m.list.SelectedItem().(entry)
	key, host, registration, resume := "", "", "", ""
	switch selected.kind {
	case "key":
		key = selected.id
	case "host":
		host = selected.id
	case "registration", "history":
		registration, resume = selected.id, selected.id
		for _, saved := range m.view.State.Registrations {
			if saved.ID == selected.id {
				host, key = saved.HostID, saved.KeyID
			}
		}
	case "creation", "onboarding", "deletion":
		resume = selected.id
	}
	switch action {
	case "create":
		add("name", "Management name", "")
	case "import":
		add("name", "Management name", "")
		add("public", "Existing public key path", "")
		add("private", "Private key path (optional)", "")
	case "add":
		add("alias", "Destination alias", "")
		add("hostname", "Host / IP", "")
		add("user", "Login user", "")
		add("port", "Port", "22")
		add("os", "OS: linux/macos/windows", "linux")
		add("account", "Windows account (or blank)", "")
		m.fields[len(m.fields)-1].input.Placeholder = "standard / administrator"
		add("key", "Key ID/name (blank = new)", key)
		add("name", "New key name (or blank)", "")
	case "register":
		add("host", "Destination ID or alias", host)
		add("key", "Managed key ID / unique name", key)
	case "unregister":
		add("registration", "Registration ID to remove", registration)
	case "delete":
		add("key", "Managed key ID / unique name", key)
	case "resume":
		add("resume", "Saved operation ID", resume)
	}
	if action == "add" || action == "register" || action == "unregister" || action == "resume" {
		add("auth", "Auth: manual/password/key", "manual")
		identityLabel := "Authentication key path"
		if action == "unregister" {
			identityLabel = "Alternate private key path"
		}
		add("identity", identityLabel, "")
		if action != "unregister" {
			add("configure", "SSH configuration: yes/no", "no")
		}
	}
	m.fields[0].input.Focus()
}

func (m model) updateForm(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch message.String() {
	case "esc":
		m.page, m.status = "list", "Changes cancelled"
		return m, nil
	case "tab", "shift+tab", "enter":
		if message.String() == "enter" && m.focus == len(m.fields)-1 {
			return m.reviewForm()
		}
		m.fields[m.focus].input.Blur()
		direction := 1
		if message.String() == "shift+tab" {
			direction = len(m.fields) - 1
		}
		m.focus = (m.focus + direction) % len(m.fields)
		return m, m.fields[m.focus].input.Focus()
	case "ctrl+s":
		return m.reviewForm()
	}
	var command tea.Cmd
	m.fields[m.focus].input, command = m.fields[m.focus].input.Update(message)
	return m, command
}

func (m model) reviewForm() (tea.Model, tea.Cmd) {
	values := map[string]string{}
	var preview strings.Builder
	fmt.Fprintf(&preview, "Review %s\n\n", m.action)
	for _, field := range m.fields {
		values[field.name] = strings.TrimSpace(field.input.Value())
		fmt.Fprintf(&preview, "%s: %s\n", field.label, values[field.name])
	}
	request := app.Request{Operation: m.action, Name: values["name"], PublicPath: values["public"],
		PrivatePath: values["private"], Key: values["key"], Host: values["host"], Registration: values["registration"],
		Resume: values["resume"], Bootstrap: values["auth"], Identity: values["identity"]}
	if values["configure"] != "" && values["configure"] != "yes" && values["configure"] != "no" {
		m.status = "Configuration choice must be yes or no"
		return m, nil
	}
	request.Configure = values["configure"] == "yes"
	if m.action == "add" {
		port, err := strconv.Atoi(values["port"])
		if err != nil {
			m.status = "Port must be a number"
			return m, nil
		}
		request.Destination = app.Destination{Alias: values["alias"], Hostname: values["hostname"], User: values["user"],
			Port: port, OS: values["os"], WindowsAccount: values["account"]}
	}
	if err := request.Validate(); err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.request, m.page, m.status = request, "review", ""
	preview.WriteString("\nNo changes yet. Continue to review exact paths, fingerprints and any remote target before confirming in the terminal.\nAuthentication secrets are entered only in OpenSSH.")
	m.content.SetContent(preview.String())
	m.content.GotoTop()
	return m, nil
}

func (m model) formView() string {
	titles := map[string]string{"create": "Create Ed25519 key", "import": "Import existing key", "add": "Add destination",
		"register": "Register public key", "unregister": "Unregister public key", "delete": "Delete local key", "resume": "Resume saved operation"}
	var body strings.Builder
	body.WriteString(titles[m.action] + "\n\n")
	count := max(1, (m.height-10)/3)
	start := max(0, m.focus-count+1)
	end := min(len(m.fields), start+count)
	for index := start; index < end; index++ {
		fmt.Fprintf(&body, "%s\n%s\n\n", m.fields[index].label, m.fields[index].input.View())
	}
	fmt.Fprintf(&body, "Field %d/%d", m.focus+1, len(m.fields))
	return body.String()
}
