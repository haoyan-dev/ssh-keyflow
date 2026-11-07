package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/state"
)

// beginAction resolves only the visible entity; formTarget contains fixed IDs.
func (m *model) beginAction(action string) {
	if m.stale {
		m.status = "Saved state stale; F5: Refresh before changing anything"
		return
	}
	if m.page != "list" && m.page != "details" {
		m.status = "This view is read-only; go back to a current entity"
		return
	}
	target := m.visibleTarget()
	if target.readOnly {
		m.status = "Read-only review/history; inspect details for supported recovery"
		return
	}
	if target.kind == "task" && action != "resume" {
		m.status = "Use s: Continue for this saved task; inspect its exact entity for other actions"
		return
	}
	m.formTarget = app.Request{Operation: action}
	switch action {
	case "create", "import", "add":
		m.openForm(action)
		return
	case "register":
		switch target.kind {
		case "host":
			m.formTarget.Host = target.id
		case "key":
			m.formTarget.Key = target.id
		case "registration":
			for _, reg := range m.view.State.Registrations {
				if reg.ID == target.id {
					m.formTarget.Host, m.formTarget.Key = reg.HostID, reg.KeyID
				}
			}
		default:
			m.status = "Select a destination or managed key to register"
			return
		}
		if m.formTarget.Host != "" && m.formTarget.Key != "" {
			m.openForm(action)
			return
		}
		var choices []entry
		for _, key := range m.view.State.Keys {
			if m.formTarget.Key == "" {
				choices = append(choices, m.keyChoiceEntry(key))
			}
		}
		if m.formTarget.Host == "" {
			for _, host := range m.view.State.Hosts {
				choices = append(choices, entry{kind: "host", id: host.ID, title: host.Alias, description: login(host)})
			}
		}
		if len(choices) == 0 {
			if m.formTarget.Key == "" {
				m.status = "No managed keys available; c: Create key or i: Import key"
			} else {
				m.status = "No destinations available; a: Add destination"
			}
			return
		}
		m.openPicker("Choose related destination/key", choices)
	case "unregister":
		choices := m.unregisterChoices(target)
		if len(choices) == 0 {
			m.status = "No current eligible registration; select a registered destination/key"
			return
		}
		if len(choices) == 1 {
			m.formTarget.Registration = choices[0].id
			m.openForm(action)
			return
		}
		m.openPicker("Choose registration to unregister", choices)
	case "delete":
		if blocker := m.deleteBlocker(target); blocker != "" {
			m.status = blocker
			return
		}
		m.formTarget.Key = target.id
		m.openForm(action)
	case "resume":
		choices := m.resumeChoices(target)
		if len(choices) == 0 {
			m.status = "No supported continuation here; inspect Tasks and saved operation details"
			return
		}
		if len(choices) == 1 {
			m.formTarget.Resume = choices[0].id
			m.openForm(action)
			return
		}
		m.openPicker("Choose saved operation to continue", choices)
	}
}

func (m model) unregisterChoices(target entry) []entry {
	var choices []entry
	for _, reg := range m.view.State.Registrations {
		if relatedRegistration(target, reg) {
			if key, active := m.key(reg.KeyID); active && !(reg.Status == "removed" && reg.Stage == "complete" && reg.Operation == "unregister") {
				choices = append(choices, entry{kind: "registration", id: reg.ID, title: m.registrationLabel(reg), description: reg.Status + " · " + fingerprintHint(key.Fingerprint) + " · " + shortReference(reg.ID), searchText: reg.HostID + " " + reg.KeyID + " " + key.Fingerprint})
			}
		}
	}
	return choices
}

func (m model) deleteBlocker(target entry) string {
	if target.kind != "key" {
		return "Delete requires a current managed key; open Keys"
	}
	for _, reg := range m.view.State.Registrations {
		if reg.KeyID == target.id && !(reg.Status == "removed" && reg.Stage == "complete" && reg.Operation == "unregister") {
			return "Unregister outstanding recorded registrations first; local files preserved"
		}
	}
	for _, deletion := range m.view.State.Deletions {
		if deletion.Stage == "pending" && deletion.KeyID != target.id {
			return "Requires review: resume unfinished deletion " + deletion.ID + " first"
		}
	}
	for _, edit := range m.view.State.ConfigEdits {
		if !edit.Complete {
			return "Requires review: unfinished Include edit; preserve configuration before deletion"
		}
	}
	return ""
}

func keyAvailability(key state.Key) string {
	if key.PrivatePath == "" {
		return "public only"
	}
	return "private path recorded"
}

func shortReference(id string) string {
	if len(id) > 24 {
		return id[:12] + "…" + id[len(id)-6:]
	}
	return id
}

func (m model) keyChoiceEntry(key state.Key) entry {
	title := key.Name
	reference := ""
	for _, other := range m.view.State.Keys {
		if other.ID != key.ID && other.Name == key.Name {
			reference = shortReference(key.ID)
			title += " [" + reference + "]"
			break
		}
	}
	return entry{kind: "key", id: key.ID, title: title,
		description: keyAvailability(key) + " · " + fingerprintHint(key.Fingerprint),
		searchText:  key.Name + " " + key.Fingerprint, reference: reference}
}

func (m *model) openPicker(title string, entries []entry) {
	m.pickerContext = ""
	target := m.visibleTarget()
	switch {
	case m.page == "form" && m.action == "add":
		m.pickerContext = "Destination: " + m.draft["alias"]
	case m.formTarget.Host != "":
		if host, ok := m.host(m.formTarget.Host); ok {
			m.pickerContext = "Destination: " + host.Alias
		}
	case m.formTarget.Key != "":
		if key, ok := m.key(m.formTarget.Key); ok {
			m.pickerContext = "Key: " + key.Name
		}
	case target.id != "":
		switch target.kind {
		case "host":
			m.pickerContext = "Destination: " + m.hostLabel(target.id)
		case "key":
			if key, ok := m.key(target.id); ok {
				m.pickerContext = "Key: " + key.Name
			}
		default:
			m.pickerContext = "From: " + target.title
		}
	case m.page == "result":
		m.pickerContext = "Result: " + m.result.request.Operation
		if host, ok := m.host(m.result.target.HostID); ok {
			m.pickerContext = "Result: " + host.Alias
		} else if key, ok := m.key(m.result.target.KeyID); ok {
			m.pickerContext = "Result: " + key.Name
		}
	}
	m.openDocument("picker", entry{}, "")
	m.choices = list.New(nil, list.NewDefaultDelegate(), max(10, m.width-2), 12)
	m.choices.DisableQuitKeybindings()
	m.choices.SetFilteringEnabled(false)
	m.choices.SetShowHelp(false)
	m.choices.Title = title
	m.choiceEntries, m.choiceQuery = entries, ""
	m.refreshChoices()
}

func (m *model) refreshChoices() {
	selected, _ := m.choices.SelectedItem().(entry)
	var items []list.Item
	index := 0
	for _, item := range m.choiceEntries {
		if strings.Contains(strings.ToLower(item.FilterValue()), strings.ToLower(m.choiceQuery)) {
			if selected.id == item.id {
				index = len(items)
			}
			items = append(items, item)
		}
	}
	m.choices.SetItems(items)
	if len(items) > 0 {
		m.choices.Select(index)
	}
}

func (m *model) chooseTarget() {
	item, ok := m.choices.SelectedItem().(entry)
	if !ok {
		m.status = "No matching eligible targets; x: Clear search or esc: Back"
		return
	}
	switch item.kind {
	case "task":
		m.back()
		m.openDocument("details", item, m.entityDetails(item))
		return
	case "key":
		m.formTarget.Key = item.id
	case "host":
		m.formTarget.Host = item.id
	case "registration":
		m.formTarget.Registration = item.id
	case "resume":
		m.formTarget.Resume = item.id
	}
	m.back()
	if m.page == "form" {
		m.draft["key"] = m.formTarget.Key
		m.buildFields("continue")
	} else {
		m.openForm(m.formTarget.Operation)
	}
}

func (m model) formTargetLabel() string {
	request := m.formTarget
	for _, reg := range m.view.State.Registrations {
		if reg.ID == request.Registration || reg.ID == request.Resume {
			request.Host, request.Key = reg.HostID, reg.KeyID
		}
	}
	creationID := ""
	for _, job := range m.view.State.Onboardings {
		if job.ID == request.Resume {
			request.Host, request.Key, creationID = job.HostID, job.KeyID, job.CreationID
		}
	}
	for _, deletion := range m.view.State.Deletions {
		if deletion.ID == request.Resume {
			request.Key = deletion.KeyID
		}
	}
	var parts []string
	if host, ok := m.host(request.Host); ok {
		parts = append(parts, fmt.Sprintf("%s · %s [%s]", host.Alias, login(host), host.ID))
	}
	if key, ok := m.recordedKey(request.Key); ok {
		availability := keyAvailability(key)
		if key.PrivatePath == "" {
			availability = "public only; verification unavailable"
		}
		parts = append(parts, fmt.Sprintf("%s [%s]\n%s\n%s", key.Name, key.ID, key.Fingerprint, availability))
	}
	for _, creation := range m.view.State.Creations {
		if creation.ID == request.Resume || (creation.ID == creationID && creation.Stage != "complete") {
			parts = append(parts, fmt.Sprintf("New Ed25519 key: %s [%s]\nCreation: %s · %s\nPrivate path: %s\nPublic path: %s\nReconcile saved paths; generate only if absent; preserve existing files. Native checks still apply.", creation.Name, creation.KeyID, creation.ID, creation.Stage, creation.PrivatePath, creation.PublicPath))
			if creation.Fingerprint != "" {
				parts = append(parts, "Saved fingerprint: "+creation.Fingerprint)
			}
		}
	}
	return strings.Join(parts, "\n")
}

func (m *model) pickDraftKey() {
	choices := m.keyChoices()
	if len(choices) == 0 {
		m.formError = "No managed keys: choose new, or cancel and c: Create / i: Import"
		m.errorField = "key"
		return
	}
	m.openPicker("Choose managed key", choices)
}
