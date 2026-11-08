package tui

import (
	"fmt"
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
	fixed       bool
}

// draft contains only form inputs; selected entities remain fixed IDs in formTarget.
// fields is the visible step. Rebuilding it never discards applicable draft values.
func (m *model) openForm(action string) {
	m.origins = append(m.origins, returnContext{m.page, m.target, m.content, m.document, m.searching})
	m.action, m.page, m.focus, m.step, m.status = action, "form", 0, 0, ""
	m.draft = map[string]string{"port": "22", "auth": "manual", "configure": "no", "keymode": "existing"}
	if len(m.view.State.Keys) == 0 {
		m.draft["keymode"] = "new"
	}
	m.formError, m.errorField = "", ""
	m.buildFields("")
	m.rememberDraft()
}

func (m model) localResume() bool {
	if m.action != "resume" {
		return false
	}
	for _, c := range m.view.State.Creations {
		if c.ID == m.formTarget.Resume {
			return true
		}
	}
	for _, d := range m.view.State.Deletions {
		if d.ID == m.formTarget.Resume {
			return true
		}
	}
	return false
}

func (m model) usesRemoteOptions() bool {
	return m.action == "add" || m.action == "register" || m.action == "unregister" || (m.action == "resume" && !m.localResume())
}

func (m model) removing() bool {
	if m.action == "unregister" {
		return true
	}
	if m.action == "resume" {
		for _, r := range m.view.State.Registrations {
			if r.ID == m.formTarget.Resume {
				return r.Operation == "unregister"
			}
		}
	}
	return false
}

func (m *model) buildFields(focused string) {
	m.inputGeneration++
	m.fields = nil
	add := func(name, label string, fixed bool) {
		input := textinput.New()
		input.SetWidth(max(1, m.width-8))
		input.Prompt = "  "
		input.CharLimit = 4096
		input.SetValue(m.draft[name])
		m.fields = append(m.fields, field{name: name, label: label, input: input, fixed: fixed})
	}
	switch m.action {
	case "create":
		add("name", "Management name", false)
	case "import":
		add("name", "Management name", false)
		add("public", "Existing public key path", false)
		add("private", "Private key path (optional)", false)
	case "add":
		switch m.step {
		case 0:
			add("alias", "Destination alias", false)
			add("hostname", "Host / IP", false)
			add("user", "Login user", false)
			add("port", "Port", false)
			add("os", "OS: linux/macos/windows", false)
			if m.draft["os"] == "windows" {
				add("account", "Windows account: standard/administrator", false)
			}
		case 1:
			add("keymode", "Key choice: existing/new (←/→)", false)
			if m.draft["keymode"] == "new" {
				add("name", "Management name", false)
			} else {
				add("key", "Managed key: Enter to choose", true)
			}
		}
	case "register":
		m.draft["host"], m.draft["key"] = m.formTarget.Host, m.formTarget.Key
		add("host", "Destination ID or alias", true)
		add("key", "Managed key ID / unique name", true)
	case "unregister":
		m.draft["registration"] = m.formTarget.Registration
		add("registration", "Registration ID to remove", true)
	case "delete":
		m.draft["key"] = m.formTarget.Key
		add("key", "Managed key ID / unique name", true)
	case "resume":
		m.draft["resume"] = m.formTarget.Resume
		add("resume", "Saved operation ID", true)
	}
	if m.usesRemoteOptions() && (m.action != "add" || m.step == 2) {
		add("auth", "Auth: manual/password/key (←/→)", false)
		if m.draft["auth"] == "key" {
			label := "Authentication key path"
			if m.removing() {
				label = "Alternate private key path"
			}
			add("identity", label, false)
		}
		if !m.removing() {
			add("configure", "SSH configuration: yes/no (←/→)", false)
		}
	}
	add("continue", "Continue", true)
	add("help", "Help", true)
	add("cancel", "Cancel", true)
	m.focus = 0
	for i := range m.fields {
		if m.fields[i].name == focused {
			m.focus = i
		}
	}
	m.fields[m.focus].input.Focus()
	m.content.GotoTop()
	m.revealFormFocus()
}

func (m *model) saveFields() {
	for _, f := range m.fields {
		if !f.fixed {
			value := f.input.Value()
			switch f.name {
			case "os", "account", "auth", "configure", "keymode":
				value = strings.TrimSpace(value)
			}
			m.draft[f.name] = value
		}
	}
}

func (m *model) clearHidden() {
	if m.draft["os"] != "windows" {
		m.draft["account"] = ""
	}
	if m.draft["auth"] != "key" {
		m.draft["identity"] = ""
	}
	if m.action == "add" {
		if m.draft["keymode"] == "new" {
			m.formTarget.Key = ""
			m.draft["key"] = ""
		} else {
			m.draft["name"] = ""
		}
	}
}

func (m model) updateForm(message tea.Msg) (tea.Model, tea.Cmd) {
	pressed := ""
	if key, ok := message.(tea.KeyPressMsg); ok {
		pressed = key.String()
	}
	switch pressed {
	case "pgup", "pgdown":
		m.setContent(m.formBody())
		var command tea.Cmd
		m.content, command = m.content.Update(message)
		return m, command
	case "esc":
		m.saveFields()
		if m.action == "add" && m.step > 0 {
			m.step--
			m.buildFields("")
		} else {
			m.cancelDraft()
		}
		return m, nil
	case "ctrl+s":
		return m.reviewForm()
	case "tab", "shift+tab", "enter":
		f := m.fields[m.focus]
		if pressed == "enter" {
			switch f.name {
			case "cancel":
				m.cancelDraft()
				return m, nil
			case "help":
				m.openDocument("help", entry{}, m.helpText())
				return m, nil
			case "key":
				if m.action == "add" {
					m.saveFields()
					m.pickDraftKey()
					return m, nil
				}
			case "continue":
				return m.continueForm()
			}
		}
		m.content.GotoTop()
		m.inputGeneration++
		m.fields[m.focus].input.Blur()
		direction := 1
		if pressed == "shift+tab" {
			direction = len(m.fields) - 1
		}
		m.focus = (m.focus + direction) % len(m.fields)
		command := m.fields[m.focus].input.Focus()
		m.revealFormFocus()
		return m, command
	}
	f := &m.fields[m.focus]
	if f.fixed {
		return m, nil
	}
	value, position := f.input.Value(), f.input.Position()
	choices := map[string][]string{"keymode": {"existing", "new"}, "auth": {"manual", "password", "key"}, "configure": {"no", "yes"}, "os": {"linux", "macos", "windows"}, "account": {"standard", "administrator"}}
	var command tea.Cmd
	if values := choices[f.name]; len(values) > 0 && (pressed == "left" || pressed == "right" || pressed == "space") {
		index := -1
		for i, v := range values {
			if v == f.input.Value() {
				index = i
			}
		}
		if pressed == "left" {
			index = (index + len(values) - 1) % len(values)
		} else {
			index = (index + 1) % len(values)
		}
		f.input.SetValue(values[index])
	} else {
		f.input, command = f.input.Update(message)
	}
	changed := f.input.Value() != value || f.input.Position() != position
	if changed {
		m.content.GotoTop()
	}
	name, controlling := f.name, false
	switch name {
	case "os", "auth", "keymode":
		controlling = m.draft[name] != f.input.Value()
	}
	m.saveFields()
	m.clearHidden()
	if controlling {
		m.buildFields(name)
	} else if changed {
		m.revealFormFocus()
	}
	return m, command
}

func (m model) continueForm() (tea.Model, tea.Cmd) {
	m.saveFields()
	m.clearHidden()
	if m.action == "add" && m.step < 2 {
		if name, message := m.validateInputs(m.step); message != "" {
			m.invalid(name, message)
			return m, nil
		}
		m.step++
		m.formError, m.errorField = "", ""
		m.buildFields("")
		return m, nil
	}
	return m.reviewForm()
}

func (m *model) invalid(name, message string) {
	m.page, m.errorField, m.formError, m.status = "form", name, message, "Correct the highlighted field"
	if m.action == "add" {
		switch name {
		case "alias", "hostname", "user", "port", "os", "account":
			m.step = 0
		case "keymode", "key", "name":
			m.step = 1
		default:
			m.step = 2
		}
	}
	m.buildFields(name)
}

// Check each destination field through the shared Host validator using a valid
// surrounding destination; this locates errors without duplicating its grammar.
func (m model) validateInputs(through int) (string, string) {
	value := func(name string) string { return strings.TrimSpace(m.draft[name]) }
	if m.action == "add" {
		baseline := app.Destination{Alias: "destination", Hostname: "example.test", User: "user", Port: 22, OS: "linux"}
		for _, name := range []string{"alias", "hostname", "user", "port", "os", "account"} {
			host := baseline
			guidance := ""
			switch name {
			case "alias":
				host.Alias = value(name)
				guidance = "Destination alias: enter a supported nonempty alias"
			case "hostname":
				host.Hostname = value(name)
				guidance = "Host / IP: enter a hostname or IP (IPv6 without brackets)"
			case "user":
				host.User = value(name)
				guidance = "Login user: enter the destination login user"
			case "port":
				port, err := strconv.Atoi(value(name))
				if err != nil || port < 1 || port > 65535 {
					return name, "Port: use 1..65535"
				}
				host.Port = port
			case "os":
				host.OS = value(name)
				guidance = "OS: select linux, macos or windows"
				if host.OS == "unknown" {
					return name, guidance
				}
			case "account":
				if value("os") != "windows" {
					continue
				}
				host.OS = "windows"
				host.WindowsAccount = value(name)
				guidance = "Windows account: select standard or administrator"
				if host.WindowsAccount == "" {
					return name, guidance
				}
			}
			if host.Validate() != nil {
				return name, guidance
			}
		}
		if through == 0 {
			return "", ""
		}
		if value("keymode") != "new" && value("keymode") != "existing" {
			return "keymode", "Key choice: select existing or new"
		}
		if value("keymode") == "existing" && m.formTarget.Key == "" {
			return "key", "Managed key: choose a key; create/import one first if none are available"
		}
	}
	if m.action == "create" || m.action == "import" || (m.action == "add" && value("keymode") == "new") {
		if err := (app.Request{Operation: "create", Name: value("name")}).Validate(); err != nil {
			return "name", "Management name: enter a nonempty name without line breaks"
		}
	}
	if m.action == "import" && value("public") == "" {
		return "public", "Existing public key path: enter the path to an existing public key file"
	}
	if m.action == "add" && through == 1 {
		return "", ""
	}
	if m.usesRemoteOptions() {
		if value("auth") == "" {
			return "auth", "Auth: select manual, password or key"
		}
		// Delegate authentication vocabulary and identity requirements to Request.
		r := app.Request{Operation: "resume", Resume: "selected", Bootstrap: value("auth"), Identity: value("identity")}
		if err := r.Validate(); err != nil {
			if value("auth") == "key" {
				return "identity", "Authentication key path: enter a private key path; secrets stay in OpenSSH"
			}
			return "auth", "Auth: select manual, password or key"
		}
		if !m.removing() && value("configure") != "yes" && value("configure") != "no" {
			return "configure", "SSH configuration: select yes or no"
		}
	}
	return "", ""
}

func (m model) draftRequest() app.Request {
	value := func(name string) string { return strings.TrimSpace(m.draft[name]) }
	r := m.formTarget
	r.Operation = m.action
	switch m.action {
	case "create", "import":
		r.Name = value("name")
		if m.action == "import" {
			r.PublicPath = value("public")
			r.PrivatePath = value("private")
		}
	case "add":
		port, _ := strconv.Atoi(value("port"))
		r.Destination = app.Destination{Alias: value("alias"), Hostname: value("hostname"), User: value("user"), Port: port, OS: value("os")}
		if r.Destination.OS == "windows" {
			r.Destination.WindowsAccount = value("account")
		}
		if value("keymode") == "new" {
			r.Key = ""
			r.Name = value("name")
		}
	}
	if m.usesRemoteOptions() {
		r.Bootstrap = value("auth")
		if r.Bootstrap == "key" {
			r.Identity = value("identity")
		}
		if !m.removing() {
			r.Configure = value("configure") == "yes"
		}
	}
	return r
}

func (m model) reviewForm() (tea.Model, tea.Cmd) {
	m.saveFields()
	m.clearHidden()
	if name, message := m.validateInputs(2); message != "" {
		m.invalid(name, message)
		return m, nil
	}
	if err := m.resolveFormTargets(); err != nil {
		m.targetError(err)
		return m, nil
	}
	request := m.draftRequest()
	if err := request.Validate(); err != nil {
		m.invalid(m.fields[0].name, err.Error())
		return m, nil
	}
	m.rememberEdit()
	m.reviewControl = 0
	m.request, m.page, m.status = request, "review", ""
	m.inputGeneration++
	var preview strings.Builder
	fmt.Fprintf(&preview, "%s\nNo changes yet.\n", detailHeading("Review "+m.action))
	if m.action == "add" {
		fmt.Fprintf(&preview, "Destination: %s · %s@%s:%d\nOS: linux/macos/windows: %s\n", request.Destination.Alias, request.Destination.User, request.Destination.Hostname, request.Destination.Port, request.Destination.OS)
		if request.Destination.WindowsAccount != "" {
			fmt.Fprintf(&preview, "Windows account: %s\n", request.Destination.WindowsAccount)
		}
	}
	if request.Host != "" {
		fmt.Fprintf(&preview, "%s\n", m.hostLabel(request.Host))
	}
	if request.Key != "" {
		fmt.Fprintf(&preview, "%s\n", m.keyLabel(request.Key))
		if key, ok := m.key(request.Key); ok && key.PrivatePath == "" {
			preview.WriteString("Public key only; verification unavailable.\n")
		}
	}
	if request.Name != "" {
		fmt.Fprintf(&preview, "Management name: %s\n", request.Name)
	}
	if request.PublicPath != "" {
		fmt.Fprintf(&preview, "Public path: %s\nPrivate path (optional): %s\n", request.PublicPath, request.PrivatePath)
	}
	if request.Registration != "" {
		label := strings.TrimSuffix(m.fixedOperationLabel(), " ["+request.Registration+"]")
		fmt.Fprintf(&preview, "%s\nRegistration ID to remove: %s\n", label, request.Registration)
		for _, registration := range m.view.State.Registrations {
			if registration.ID == request.Registration {
				fmt.Fprintf(&preview, "%s\n", m.keyLabel(registration.KeyID))
			}
		}
	}
	if request.Resume != "" {
		fmt.Fprintf(&preview, "%s\n", m.fixedOperationLabel())
		fmt.Fprintf(&preview, "Saved operation: %s\n", request.Resume)
	}
	if request.Bootstrap != "" {
		preview.WriteString("\n" + detailHeading("Options") + "\n")
		fmt.Fprintf(&preview, "Auth: %s\n", request.Bootstrap)
		if request.Identity != "" {
			fmt.Fprintf(&preview, "Authentication key path: %s\n", request.Identity)
		}
		if !m.removing() {
			fmt.Fprintf(&preview, "SSH configuration: %s\n", map[bool]string{false: "no", true: "yes"}[request.Configure])
		}
	}
	preview.WriteString("\n" + detailHeading("Expected changes") + "\n" + m.expectedEffects() +
		"\n\nContinue in the terminal for exact checks and approvals.\nAuthentication secrets are entered only in OpenSSH.")
	if label := m.formTargetLabel(); label != "" {
		preview.WriteString("\n\n" + detailHeading("References") + "\n" + label)
	}
	m.setContent(preview.String())
	m.content.GotoTop()
	return m, nil
}

func (m model) formView() string {
	m.setContent(m.formBody())
	return m.content.View()
}

func (m model) formBody() string {
	body, _, _ := m.formDocument()
	return body
}

// Measure the actual wrapped document rather than guessing a fixed field height.
// The active field has a textual focus marker; earlier fields stay in context.
func (m model) formDocument() (string, int, int) {
	title := actionTitle(m.action)
	if m.action == "add" {
		title = fmt.Sprintf("%s · step %d/3", []string{"Destination", "Key", "Access method/options"}[m.step], m.step+1)
	}
	width := max(1, m.width-6)
	wrap := func(text string) string { return ansi.Wrap(text, width, "") }
	var body strings.Builder
	if m.removing() {
		body.WriteString(wrap("Use alternate authentication or manual instructions; local key files remain.") + "\n\n")
	}
	focusStart, focusEnd := 0, 0
	for i := range m.fields {
		f := m.fields[i]
		label := f.label
		if f.fixed {
			switch f.name {
			case "host":
				label = "Destination"
			case "key":
				label = "Managed key"
			case "registration":
				label = "Registration to remove"
			case "resume":
				label = "Saved operation"
			}
		}
		start := strings.Count(body.String(), "\n")
		marker := "  "
		if i == m.focus {
			marker = "> "
		}
		if f.name == "continue" || f.name == "help" || f.name == "cancel" {
			control := marker + "[" + f.label + "]"
			if i == m.focus {
				control = detailHeading(control)
			}
			body.WriteString(control + "\n")
		} else {
			value := f.input.Value()
			switch {
			case f.name == "key":
				if k, ok := m.key(m.formTarget.Key); ok {
					value = k.Name + " [" + k.ID + "] · " + keyAvailability(k)
				} else if m.action == "add" {
					value = "Enter: Choose managed key"
				}
			case f.fixed && f.name == "host":
				value = m.hostLabel(m.formTarget.Host)
			case f.fixed && (f.name == "registration" || f.name == "resume"):
				value = m.fixedOperationLabel()
			}
			if i == m.focus {
				body.WriteString(detailHeading(wrap(marker+label)) + "\n")
				if f.fixed {
					body.WriteString(wrap("  "+value) + "\n")
				} else {
					body.WriteString(f.input.View() + "\n")
				}
			} else {
				if value == "" {
					value = "(empty)"
				}
				body.WriteString(wrap(label+": "+value) + "\n")
			}
			if m.formError != "" && m.errorField == f.name {
				body.WriteString(detailEvidence(wrap(m.formError), "blocked") + "\n")
			}
			if i == m.focus {
				if guidance := m.formGuidance(f.name); guidance != "" {
					body.WriteString(wrap(guidance) + "\n")
				}
				// Keep orientation and guidance beside the caret. A complete copy
				// follows them and remains part of the ordinary scrollable form.
				if !f.fixed && ansi.StringWidth(value) > f.input.Width() {
					focusStart, focusEnd = start, strings.Count(body.String(), "\n")
					body.WriteString("Full value:\n" + ansi.Hardwrap(value, width, false) + "\n\n")
					continue
				}
				body.WriteString("\n")
			}
		}
		if i == m.focus {
			focusStart, focusEnd = start, strings.Count(body.String(), "\n")
		}
	}
	body.WriteString(fmt.Sprintf("\nField %d/%d", m.focus+1, len(m.fields)))
	return panel(title, body.String(), max(10, m.width-2)), focusStart + 1, focusEnd + 1
}

func (m *model) revealFormFocus() {
	body, start, end := m.formDocument()
	m.setContent(body)
	height := m.content.Height()
	if start < m.content.YOffset() || end-start >= height {
		m.content.SetYOffset(start)
	} else if end > m.content.YOffset()+height {
		m.content.SetYOffset(max(0, end-height))
	}
}

func (m model) formGuidance(name string) string {
	switch name {
	case "auth", "identity":
		if m.removing() {
			return "Use fresh alternate authentication excluding the target fingerprint, or manual instructions. Local key files are preserved."
		}
		return "Bootstrap credentials register this key. Secrets are entered only in OpenSSH."
	case "configure":
		return "Configuration is a request: backend eligibility and terminal approvals decide alias/Include setup. Public-only verification is unavailable."
	case "key":
		if k, ok := m.key(m.formTarget.Key); ok && k.PrivatePath == "" {
			return "Public-only key: specified-private-key verification unavailable."
		}
	}
	return ""
}
