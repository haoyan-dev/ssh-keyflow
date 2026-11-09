package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/state"
)

func (m model) entries() []entry {
	var items []entry
	if m.tab == 0 {
		for _, key := range m.view.State.Keys {
			mode := "Key pair"
			if key.PrivatePath == "" {
				mode = "Public only"
			}
			for _, deletion := range m.view.State.Deletions {
				if deletion.KeyID == key.ID && deletion.Stage != "complete" {
					mode = "Manual review; key unavailable"
				}
			}
			items = append(items, entry{"key", key.ID, key.Name + " [" + key.ID + "]", mode + " · " + key.Fingerprint})
		}
		for _, creation := range m.view.State.Creations {
			if creation.Stage != "complete" {
				items = append(items, entry{"creation", creation.ID, creation.Name + " (unfinished local creation)", "s: resume · " + creation.Stage})
			}
		}
	} else {
		for _, host := range m.view.State.Hosts {
			items = append(items, entry{"host", host.ID, host.Alias + " [" + host.ID + "]", fmt.Sprintf("%s@%s:%d · %s", host.User, host.Hostname, host.Port, host.OS)})
		}
	}
	return items
}
func filterEntries(items []entry, query string) []entry {
	if query == "" {
		return items
	}
	var matches []entry
	for _, item := range items {
		haystack := strings.ToLower(item.title + " " + item.description + " " + item.id)
		if strings.Contains(haystack, strings.ToLower(query)) {
			matches = append(matches, item)
		}
	}
	return matches
}
func (m model) visibleEntries() []entry {
	if m.page == "picker" {
		return filterEntries(m.choices, m.choiceQuery)
	}
	return filterEntries(m.entries(), m.queries[m.tab])
}
func (m model) visibleTarget() entry {
	if m.page == "details" {
		return m.target
	}
	if m.page != "list" {
		return entry{}
	}
	items := m.visibleEntries()
	if len(items) == 0 {
		return entry{}
	}
	return items[max(0, min(m.selected[m.tab], len(items)-1))]
}
func (m model) updateSelection(key string) (tea.Model, tea.Cmd) {
	items, index := m.visibleEntries(), m.selected[m.tab]
	if m.page == "picker" {
		index = m.choiceIndex
	}
	if len(items) == 0 {
		return m, nil
	}
	switch key {
	case "up", "k":
		index--
	case "down", "j":
		index++
	case "home":
		index = 0
	case "end":
		index = len(items) - 1
	case "pgup":
		index -= max(1, (m.content.Height()-1)/2)
	case "pgdown":
		index += max(1, (m.content.Height()-1)/2)
	case "enter":
		if m.page == "picker" {
			item := items[max(0, min(index, len(items)-1))]
			if item.kind == "host" {
				m.request.Host = item.id
			} else {
				m.request.Key = item.id
			}
			command := m.finishPair()
			return m, command
		}
	}
	index = max(0, min(index, len(items)-1))
	if m.page == "picker" {
		m.choiceIndex = index
	} else {
		m.selected[m.tab] = index
	}
	return m, nil
}
func (m *model) picker(action, kind string) {
	m.pickerAction = action
	m.choices = nil
	m.choiceIndex, m.choiceQuery = 0, ""
	if kind == "host" {
		for _, host := range m.view.State.Hosts {
			if action == "dissociate" && !m.associated(host.ID, m.request.Key) {
				continue
			}
			m.choices = append(m.choices, entry{"host", host.ID, host.Alias, fmt.Sprintf("%s@%s:%d", host.User, host.Hostname, host.Port)})
		}
	} else {
		for _, key := range m.view.State.Keys {
			if action == "dissociate" && !m.associated(m.request.Host, key.ID) {
				continue
			}
			suffix := ""
			for _, association := range m.view.State.Associations {
				if association.HostID == m.request.Host && association.KeyID == key.ID {
					suffix = " · associated (remote state unverified)"
				}
			}
			m.choices = append(m.choices, entry{"key", key.ID, key.Name + " [" + key.ID + "]", key.Fingerprint + suffix})
		}
	}
	if action == "dissociate" && len(m.choices) == 0 {
		m.status = "No local associations to disconnect"
		return
	}
	m.open("picker", entry{}, "", "")
}
func (m model) associated(hostID, keyID string) bool {
	for _, association := range m.view.State.Associations {
		if association.HostID == hostID && association.KeyID == keyID {
			return true
		}
	}
	return false
}
func (m *model) beginPair(action string) tea.Cmd {
	if m.stale {
		m.status = "Refresh state before selecting identities"
		return nil
	}
	target := m.visibleTarget()
	if target.kind != "host" && target.kind != "key" {
		m.status = "Select a destination or key first"
		return nil
	}
	m.request = app.Request{Operation: action}
	if target.kind == "host" {
		m.request.Host = target.id
	} else {
		m.request.Key = target.id
	}
	if action == "public" || m.request.Key == "" {
		m.picker(action, "key")
	} else {
		m.picker(action, "host")
	}
	return nil
}
func (m *model) finishPair() tea.Cmd {
	action := m.pickerAction
	// Returning to the form/list first keeps help/back navigation independent of execution.
	m.back()
	switch action {
	case "add", "associate", "dissociate":
		return m.execute(m.request)
	case "public":
		m.showPublic(m.request.Key)
	case "guide":
		return m.beginForm("guide")
	}
	return nil
}
func (m *model) showGuideStep() {
	step := m.guideSteps[m.guideIndex]
	m.copyText = step.Script
	m.status = ""
	m.setDocument(fmt.Sprintf("Step %d/%d: %s\nLocal platform: %s · Destination platform: %s\n\nCopy this script with y, then paste it into the terminal named above:\n\n%s\n\n%s\n\nRemote registration and connection: not checked. Copying records no success.",
		m.guideIndex+1, len(m.guideSteps), step.Title, m.request.GuideOptions.LocalOS, m.request.GuideOptions.RemoteOS, step.Script, step.Instructions))
}

func (m *model) showPublic(key string) {
	text, err := app.PublicKey(m.options.Home, key)
	if err != nil {
		m.status = err.Error()
		return
	}
	m.open("document", entry{}, "Public key\nCopy/paste this text. Remote placement is not verified.\n\n"+text, text)
}
func (m model) details(item entry) string {
	if item.kind == "key" {
		if key, ok := m.key(item.id); ok {
			mode := "Key pair"
			if key.PrivatePath == "" {
				mode = "Public only; private-key use unavailable"
			}
			var hosts []string
			for _, a := range m.view.State.Associations {
				if a.KeyID == key.ID {
					hosts = append(hosts, m.hostName(a.HostID))
				}
			}
			if len(hosts) == 0 {
				hosts = []string{"None"}
			}
			return fmt.Sprintf("Key: %s\n%s\nFingerprint: %s\n\nPublic path: %s\nPrivate path: %s\n\nAssociated destinations:\n%s\n\nAssociation is a local note. Remote placement and connection are unverified.\n\np public key · g placement guide · l associate\nv historical details\nFor local cleanup, use the paths above manually.", key.Name, mode, key.Fingerprint, key.PublicPath, key.PrivatePath, strings.Join(hosts, "\n"))
		}
	}
	if item.kind == "host" {
		if host, ok := m.host(item.id); ok {
			var keys []string
			for _, a := range m.view.State.Associations {
				if a.HostID == host.ID {
					if key, ok := m.key(a.KeyID); ok {
						keys = append(keys, key.Name+" ["+key.ID+"]")
					}
				}
			}
			if len(keys) == 0 {
				keys = []string{"None"}
			}
			return fmt.Sprintf("Destination: %s\n%s@%s:%d\nOS: %s\nWindows account: %s\n\nAssociated keys:\n%s\n\nAssociation is a local note. Remote placement and connection are unverified.\n\ng terminal scripts: destination setup, local config and connection\np public key · l associate key · v historical details\nScripts change files only when you paste and run them.", host.Alias, host.User, host.Hostname, host.Port, host.OS, host.WindowsAccount, strings.Join(keys, "\n"))
		}
	}
	if item.kind == "creation" {
		for _, creation := range m.view.State.Creations {
			if creation.ID == item.id {
				return fmt.Sprintf("Unfinished local creation\nName: %s\nCreation ID: %s\nKey ID: %s\nStage: %s\nPrivate path: %s\nPublic path: %s\n\ns: resume this local creation\nNative ssh-keygen receives passphrase input. Existing files are preserved and checked before resuming.", creation.Name, creation.ID, creation.KeyID, creation.Stage, creation.PrivatePath, creation.PublicPath)
			}
		}
	}
	return "Record is no longer present. F5 refresh."
}
func (m model) technical(item entry) string {
	var text strings.Builder
	text.WriteString(m.details(item) + "\n\nHistorical records (read-only)\nThese records are not current registration or connection evidence.\nLegacy remote operations cannot resume.\n")
	for _, reg := range m.view.State.Registrations {
		if (item.kind == "key" && reg.KeyID == item.id) || (item.kind == "host" && reg.HostID == item.id) {
			fmt.Fprintf(&text, "\nLegacy registration %s\nRecorded status: %s\nRecorded connection check: %s\nRecorded operation: %s\nRecorded stage: %s\n", reg.ID, reg.Status, reg.ConnectionCheck, reg.Operation, reg.Stage)
			for _, event := range m.view.State.History {
				if event.RegistrationID == reg.ID {
					fmt.Fprintf(&text, "  %s: %s, %s, %s\n", event.At, event.Operation, event.Status, event.Stage)
				}
			}
		}
	}
	for _, job := range m.view.State.Onboardings {
		if (item.kind == "key" && job.KeyID == item.id) || (item.kind == "host" && job.HostID == item.id) {
			fmt.Fprintf(&text, "\nLegacy onboarding %s: %s (read-only)\n", job.ID, job.Stage)
		}
	}
	for _, deletion := range m.view.State.Deletions {
		if item.kind == "key" && deletion.KeyID == item.id {
			fmt.Fprintf(&text, "\nLegacy local deletion %s: %s (manual review; not resumable)\n", deletion.ID, deletion.Stage)
		}
	}
	if m.view.Review != "" {
		text.WriteString("\nSaved-state review notice:\n" + m.view.Review)
	}
	return text.String()
}
func (m model) key(id string) (state.Key, bool) {
	for _, key := range m.view.State.Keys {
		if key.ID == id {
			return key, true
		}
	}
	return state.Key{}, false
}
func (m model) host(id string) (state.Host, bool) {
	for _, host := range m.view.State.Hosts {
		if host.ID == id {
			return host, true
		}
	}
	return state.Host{}, false
}
func (m model) hostName(id string) string {
	if host, ok := m.host(id); ok {
		return host.Alias + " [" + id + "]"
	}
	return id
}
