package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/state"
)

// returnContext is a read-only page origin. Category lists keep their own cursor
// and pagination; overlays retain the exact viewport they cover.
type returnContext struct {
	page      string
	target    entry
	content   viewport.Model
	searching bool
}

func (m *model) openDocument(page string, target entry, text string) {
	m.inputGeneration++
	m.origins = append(m.origins, returnContext{m.page, m.target, m.content, m.searching})
	m.page, m.target, m.searching = page, target, false
	m.content.SetContent(text)
	m.content.GotoTop()
}

func (m *model) back() {
	m.inputGeneration++
	if len(m.origins) == 0 {
		m.page, m.target = "list", entry{}
		return
	}
	origin := m.origins[len(m.origins)-1]
	m.origins = m.origins[:len(m.origins)-1]
	m.page, m.target, m.content = origin.page, origin.target, origin.content
	m.searching = origin.searching
}

// visibleTarget never falls through an overlay/result to a hidden list row.
func (m model) visibleTarget() entry {
	if m.page == "details" {
		return m.target
	}
	if m.page == "list" {
		item, _ := m.list.SelectedItem().(entry)
		return item
	}
	return entry{}
}

func (m model) host(id string) (state.Host, bool) {
	for _, host := range m.view.State.Hosts {
		if host.ID == id {
			return host, true
		}
	}
	return state.Host{}, false
}

func (m model) key(id string) (state.Key, bool) {
	for _, key := range m.view.State.Keys {
		if key.ID == id {
			return key, true
		}
	}
	return state.Key{}, false
}

func (m model) recordedKey(id string) (state.Key, bool) {
	if key, ok := m.key(id); ok {
		return key, true
	}
	for _, deleted := range m.view.State.DeletedKeys {
		if deleted.Key.ID == id {
			return deleted.Key, true
		}
	}
	return state.Key{}, false
}

func login(host state.Host) string {
	return fmt.Sprintf("%s@%s:%d", host.User, host.Hostname, host.Port)
}

func (m model) registrationLabel(reg state.Registration) string {
	host, hostOK := m.host(reg.HostID)
	key, keyOK := m.recordedKey(reg.KeyID)
	if !hostOK || !keyOK {
		return "Requires review: missing destination/key reference · " + reg.ID
	}
	return host.Alias + " / " + key.Name + " · " + login(host)
}

func (m model) categoryEntries() []entry {
	var entries []entry
	switch m.tab {
	case 0:
		for _, host := range m.view.State.Hosts {
			entries = append(entries, entry{kind: "host", id: host.ID, title: host.Alias, description: login(host) + " · " + host.OS + " · " + host.ID})
		}
	case 1:
		for _, key := range m.view.State.Keys {
			availability := "public only"
			if key.PrivatePath != "" {
				availability = "private key available"
			}
			entries = append(entries, entry{kind: "key", id: key.ID, title: key.Name, description: key.Fingerprint + " · " + availability + " · " + key.ID})
		}
	case 3:
		for _, task := range ProjectTasks(m.view) {
			entries = append(entries, m.taskEntry(task))
		}
	case 2:
		for _, reg := range m.view.State.Registrations {
			_, active := m.key(reg.KeyID)
			key, _ := m.recordedKey(reg.KeyID)
			entries = append(entries, entry{kind: "registration", id: reg.ID, title: m.registrationLabel(reg), description: reg.Status + " · connection=" + reg.ConnectionCheck + " · " + reg.ID, readOnly: !active, searchText: reg.KeyID + " " + reg.HostID + " " + key.Fingerprint})
		}
	}
	return entries
}

func (m *model) refreshList() bool {
	m.list.Title = []string{"Managed destinations", "Managed keys", "Current registrations", "Tasks"}[m.tab]
	selected, _ := m.list.SelectedItem().(entry)
	index := m.list.Index()
	var filtered []list.Item
	for _, item := range m.categoryEntries() {
		if strings.Contains(strings.ToLower(item.FilterValue()), strings.ToLower(m.queries[m.tab])) {
			filtered = append(filtered, item)
		}
	}
	m.list.SetItems(filtered)
	found := false
	for i, item := range filtered {
		if item.(entry).id == selected.id {
			index = i
			found = true
			break
		}
	}
	if len(filtered) > 0 {
		m.list.Select(min(index, len(filtered)-1))
	}
	return selected.id != "" && !found
}

func (m model) registrationDetails(reg state.Registration) string {
	var text strings.Builder
	host, hostOK := m.host(reg.HostID)
	key, keyOK := m.recordedKey(reg.KeyID)
	fmt.Fprintf(&text, "%s\nRegistration: %s (recorded)\nConnection: %s for this destination (recorded)\n", m.registrationLabel(reg), reg.Status, reg.ConnectionCheck)
	if keyOK && key.PrivatePath == "" {
		text.WriteString("Private-key verification unavailable: public key only\n")
	}
	if !hostOK || !keyOK {
		text.WriteString("Requires review: missing references; no relationship inferred\n")
	}
	text.WriteString(m.configurationDetails(host))
	count := 0
	for _, event := range m.view.State.History {
		if event.RegistrationID == reg.ID {
			count++
		}
	}
	fmt.Fprintf(&text, "History: %d recorded events (h: read-only history)\nSaved operation: %s / %s\nRegistration ID: %s\nDestination ID: %s\nKey ID: %s\n", count, reg.Operation, reg.Stage, reg.ID, reg.HostID, reg.KeyID)
	if hostOK {
		fmt.Fprintf(&text, "Destination: %s · %s · %s\n", host.Alias, login(host), host.OS)
	}
	if keyOK {
		fmt.Fprintf(&text, "Key: %s\nFull fingerprint: %s\n", key.Name, key.Fingerprint)
	}
	return text.String()
}

func (m model) configurationDetails(host state.Host) string {
	var text strings.Builder
	text.WriteString("Configuration: ")
	if host.ConfigKeyID == "" {
		text.WriteString("binding not recorded\n")
	} else if key, ok := m.key(host.ConfigKeyID); ok {
		fmt.Fprintf(&text, "saved binding to %s [%s]\n", key.Name, key.ID)
	} else {
		fmt.Fprintf(&text, "Requires review: missing key %s\n", host.ConfigKeyID)
	}
	return text.String()
}

func (m model) technicalDetails(item entry) string {
	var text strings.Builder
	switch item.kind {
	case "task":
		for _, task := range ProjectTasks(m.view) {
			if task.ID == item.id {
				return m.taskTechnicalDetails(task)
			}
		}
	case "registration":
		for _, reg := range m.view.State.Registrations {
			if reg.ID == item.id {
				text.WriteString(m.registrationDetails(reg))
				break
			}
		}
	case "host":
		if host, ok := m.host(item.id); ok {
			fmt.Fprintf(&text, "%s\n%s · %s\nDestination ID: %s\n", host.Alias, login(host), host.OS, host.ID)
			text.WriteString(m.configurationDetails(host))
			text.WriteString("Registration and connection evidence (recorded, not a live probe):\n")
			found := false
			for _, reg := range m.view.State.Registrations {
				if reg.HostID == host.ID {
					found = true
					text.WriteString("\n" + m.registrationDetails(reg))
				}
			}
			if !found {
				text.WriteString("No registrations recorded. Connection: not recorded\n")
			}
			if host.WindowsAccount != "" {
				fmt.Fprintf(&text, "Windows account: %s\n", host.WindowsAccount)
			}
		}
	case "key":
		if key, ok := m.key(item.id); ok {
			fmt.Fprintf(&text, "%s\nFull fingerprint: %s\nKey ID: %s\nOrigin: %s\n", key.Name, key.Fingerprint, key.ID, key.Origin)
			if key.PrivatePath == "" {
				text.WriteString("Private-key verification unavailable: public key only\n")
			} else {
				fmt.Fprintf(&text, "Private path: %s\n", key.PrivatePath)
			}
			fmt.Fprintf(&text, "Public path: %s\nCreated: %s\nImported: %s\n", key.PublicPath, key.CreatedAt, key.ImportedAt)
			text.WriteString("Registration and connection evidence per destination:\n")
			found := false
			for _, reg := range m.view.State.Registrations {
				if reg.KeyID == key.ID {
					found = true
					text.WriteString("\n" + m.registrationDetails(reg))
				}
			}
			if !found {
				text.WriteString("No registrations recorded. Connection: not recorded\n")
			}
		}
	}
	if text.Len() == 0 {
		return "Requires review: target no longer available. Refresh before continuing."
	}
	text.WriteString("\nConfiguration file evidence (global; not proof of alias activation):\n")
	if len(m.view.State.Configs) == 0 && len(m.view.State.ConfigEdits) == 0 {
		text.WriteString("No shared configuration file evidence recorded.\n")
	}
	for _, config := range m.view.State.Configs {
		fmt.Fprintf(&text, "Path: %s\nSaved digest: %s\nPending digest: %s\n", config.Path, config.SHA256, config.PendingSHA256)
	}
	for _, edit := range m.view.State.ConfigEdits {
		fmt.Fprintf(&text, "Include edit: %s\nComplete: %t\nBackup: %s\n", edit.Path, edit.Complete, edit.BackupPath)
	}
	text.WriteString("Include activation: not established by a saved binding alone\nHistory: h opens recorded events and completed local work\n")
	return detailHeading("Technical details") + "\n\n" + text.String()
}

func (m model) historyDetails(target entry) string {
	var text strings.Builder
	text.WriteString("History (read-only)\nRecorded evidence, not current remote state.\n\n")
	for _, event := range m.view.State.History {
		for _, reg := range m.view.State.Registrations {
			if reg.ID != event.RegistrationID {
				continue
			}
			if target.id != "" && !((target.kind == "host" && target.id == reg.HostID) || (target.kind == "key" && target.id == reg.KeyID) || (target.kind == "registration" && target.id == reg.ID)) {
				continue
			}
			fmt.Fprintf(&text, "%s\n%s\nRegistration ID: %s\nOperation: %s / %s\nRegistration: %s\nConnection: %s (recorded at that time)\n\n", event.At, m.registrationLabel(reg), reg.ID, event.Operation, event.Stage, event.Status, event.ConnectionCheck)
		}
	}
	if target.kind == "key" || (target.id == "" && m.tab == 1) {
		for _, deleted := range m.view.State.DeletedKeys {
			if target.id != "" && target.id != deleted.Key.ID {
				continue
			}
			fmt.Fprintf(&text, "Deleted local key: %s [%s]\nDeleted: %s\nDeletion ID: %s\nFull fingerprint: %s\nPublic path: %s\nPrivate path: %s\nRegistration history retained.\n\n", deleted.Key.Name, deleted.Key.ID, deleted.DeletedAt, deleted.DeletionID, deleted.Key.Fingerprint, deleted.Key.PublicPath, deleted.Key.PrivatePath)
		}
		for _, creation := range m.view.State.Creations {
			if creation.Stage == "complete" && (target.id == "" || creation.KeyID == target.id) {
				fmt.Fprintf(&text, "Completed key creation: %s\nCreation ID: %s\nKey ID: %s\nCreated: %s\nPublic path: %s\nPrivate path: %s\nFull fingerprint: %s\n\n", creation.Name, creation.ID, creation.KeyID, creation.CreatedAt, creation.PublicPath, creation.PrivatePath, creation.Fingerprint)
			}
		}
		for _, deletion := range m.view.State.Deletions {
			if deletion.Stage == "complete" && (target.id == "" || deletion.KeyID == target.id) {
				fmt.Fprintf(&text, "Completed local deletion: %s\nKey ID: %s\nCreated: %s\nPublic removed: %t\nPrivate removed: %t\n\n", deletion.ID, deletion.KeyID, deletion.CreatedAt, deletion.PublicRemoved, deletion.PrivateRemoved)
			}
		}
	}
	text.WriteString("End of recorded history. No actions run from history.\n")
	return text.String()
}

// refreshSnapshot retries only the read-only boundary, never the last operation.
func (m *model) refreshSnapshot() {
	view, err := app.Snapshot(m.options)
	if m.page == "result" {
		offset := m.content.YOffset()
		m.result.reloadErr, m.stale = err, err != nil
		if err == nil {
			m.view = view
			m.refreshList()
		}
		m.renderOperationResult()
		m.content.SetYOffset(offset)
		return
	}
	if err != nil {
		m.stale = true
		m.status = "Saved state stale; Refresh failed. F5: retry / q: quit"
		text := "Refresh failed: " + err.Error() + "\nPrevious saved state is stale. Mutations are disabled until Refresh succeeds."
		if m.page == "reload-error" {
			m.content.SetContent(text)
			m.content.GotoTop()
		} else {
			m.openDocument("reload-error", entry{}, text)
		}
		return
	}
	m.view, m.stale = view, false
	m.status = "Saved state refreshed; no SSH probe"
	active := m.tab
	m.lists[active] = m.list
	for index := range m.lists {
		m.tab, m.list = index, m.lists[index]
		lost := m.refreshList()
		if index == active && lost {
			m.status = "Selection changed: target removed; nearest remaining row selected"
			if len(m.list.Items()) == 0 {
				m.status = "Selection changed: no matching targets remain"
			}
		}
		m.lists[index] = m.list
	}
	m.tab, m.list = active, m.lists[active]
	if m.page == "reload-error" {
		m.back()
	}
	if m.page == "details" {
		found := false
		lookup := *m
		switch m.target.kind {
		case "host":
			lookup.tab = 0
		case "key":
			lookup.tab = 1
		case "registration":
			lookup.tab = 2
		case "task":
			lookup.tab = 3
		}
		for _, item := range lookup.categoryEntries() {
			if item.kind == m.target.kind && item.id == m.target.id {
				m.target = item
				found = true
				break
			}
		}
		if found {
			offset := m.content.YOffset()
			m.content.SetContent(m.entityDetails(m.target))
			m.content.SetYOffset(offset)
		} else {
			m.target.readOnly = true
			m.status = "Target removed; details are read-only"
			m.content.SetContent("Requires review: target no longer available. " + m.target.kind + " ID: " + m.target.id + "\nEsc returns to the previous view; Refresh reloads saved evidence only.")
			m.content.GotoTop()
		}
	}
}

func (m model) helpText() string {
	if m.page == "technical" {
		return "Technical details help\n\nThis document shows the displayed entry's full saved IDs, fingerprints, paths and checkpoints. Configuration file records are shared evidence, not proof of alias activation or current remote access. It is read-only; return to the overview for actions or F5 Refresh.\n\nArrows/PgUp/PgDn and Home/End scroll full values. Esc returns to the same overview position; ?/Esc closes Help and restores this document."
	}
	if m.page == "result" {
		return "Result help\n\nTab/Shift+Tab select the labelled result controls; Enter opens the focused action. s Continue saved work, t View task and v View entity use the exact result IDs. r Register again is offered only for manual registration with a saved destination and key. Multiple eligible continuations require a picker. No navigation executes an operation.\n\nArrows/PgUp/PgDn and Home/End scroll all facts, paths and diagnostics. ?/Esc closes Help and restores the result viewport. Esc leaves the result; q exits. F5 Refresh reloads Snapshot only; it never replays an operation or confirms remote state. A stale result permits Refresh/Help/Exit only.\n\nSaved checkpoints are independent facts, not completion. Saved evidence may predate this operation. Uncertain remote effects require reconciliation before retry. Manual instructions and Include follow-up may have no saved recovery task."
	}
	if m.visibleTarget().kind == "task" {
		return "Tasks help\n\ns Continue opens the supported saved operation with its exact ID; approval and execution remain in the terminal. If a creation belongs to destination setup, continuation can proceed into registration. Continue registration includes reconciliation/verification; there is no Verify-only or config-only operation. Local creation/deletion recovery has no remote authentication.\n\n/ searches readable labels, reasons and saved source IDs on the Tasks list. Arrows/PgUp/PgDn scroll details; ?/Esc returns to the exact viewport. F5 Refresh reloads saved evidence only. Review entries and history are read-only. No Mark done, dismissal, automatic retry or guessed continuation is available."
	}
	if m.page == "review" {
		return "Review help\n\nEnter activates the focused control; Continue to terminal starts the native handoff once. Tab/Shift+Tab choose Continue, Edit, Cancel or Help. e/Esc edits at the retained caret and scroll. Cancel or Ctrl+C asks before discarding changed values; Keep editing is the default. q does not abandon a draft.\n\nThis is orientation only. Exact paths, fingerprints, remote targets and approvals are checked in the terminal. Passwords/passphrases stay in OpenSSH. A stopped operation may have saved effects; no rollback is implied."
	}
	if m.page == "discard" {
		return "Draft cancellation help\n\nKeep editing is selected by default. Tab or arrows selects Discard draft; Enter confirms that choice. Esc/Ctrl+C returns to editing without discarding. Discard never executes an operation."
	}
	if m.page == "form" {
		return "Form help\n\nTab/Shift+Tab move fields and controls. Enter advances; Ctrl+S validates all steps. Esc returns one step; from the first step it invokes Cancel. Cancel or Ctrl+C returns clean drafts directly and asks before discarding changed drafts, with Keep editing selected. PgUp/PgDn scroll full values and errors. q, ? and action letters remain text in inputs. Esc closes Help and retains draft/focus.\n\nExisting and new keys are exclusive. Import existing files separately before destination setup. Manual instructions and configuration=no are defaults. Bootstrap credentials register the selected public key; passwords/passphrases stay in OpenSSH. Configuration is only requested: backend eligibility and terminal approvals decide alias/Include setup. Public-only keys cannot verify with a specified private key.\n\nUnregister uses fresh alternate authentication excluding the target fingerprint, or manual instructions. Local key files are preserved. Local creation/deletion resume has no remote authentication or configuration options. Native checks and approvals remain authoritative."
	}
	var text strings.Builder
	text.WriteString("Keyboard help\n\n/ Search names, login, fingerprints and IDs\nSearch: enter accepts; esc keeps query\nx Clear search\ntab/shift+tab Category (lists/entities)\nenter Overview / choose related target\nv Technical details from an overview (read-only)\nF5 Refresh saved state only (no SSH probe)\nh History (read-only)\nesc Back\n? Help / return to origin\nctrl+c Closes help first\nq Quit outside help and text input\n\n")
	text.WriteString("Actions (advisory; locked backend checks still decide):\n")
	if m.page != "list" && m.page != "details" {
		text.WriteString("This view is read-only. Go back to a current entity for actions.\n")
	} else if target := m.visibleTarget(); target.readOnly {
		text.WriteString("Historical target: read-only. Select a current managed entity.\n")
	} else {
		text.WriteString("c Create key · i Import key · a Add destination\n")
		switch target.kind {
		case "host":
			text.WriteString("r Register: choose a managed key\nu Unregister: choose a current eligible registration\nd Delete: requires a managed key in Keys\n")
		case "key":
			text.WriteString("r Register: choose a destination\nu Unregister: choose a current eligible registration\nd Delete local key: all recorded registrations must be confirmed removed; files/configuration are rechecked in the terminal\n")
		case "registration":
			text.WriteString("r Register: use this displayed destination/key pair\nu Unregister: current eligible registration only\nd Delete: requires a managed key in Keys\n")
		default:
			text.WriteString("Select a current destination/key/registration before r/u/d. No implicit target.\n")
		}
		text.WriteString("s Continue supported saved registration intent; otherwise inspect Tasks. Tasks uses saved evidence only.\n")
	}
	if m.status != "" {
		text.WriteString("\nCurrent notice:\n" + m.status + "\n")
	}
	text.WriteString("\ng Discovery (read-only; unmanaged files are not imported)\n")
	return text.String()
}
