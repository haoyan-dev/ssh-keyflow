package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"ssh-keyflow/internal/state"
)

func detailHeading(text string) string {
	return lipgloss.NewStyle().Bold(true).Render(text)
}

func relatedRegistration(target entry, reg state.Registration) bool {
	return (target.kind == "host" && target.id == reg.HostID) ||
		(target.kind == "key" && target.id == reg.KeyID) ||
		(target.kind == "registration" && target.id == reg.ID)
}

func (m model) entityDetails(item entry) string {
	if item.kind == "task" {
		for _, task := range ProjectTasks(m.view) {
			if task.ID == item.id {
				return m.taskDetails(task)
			}
		}
	}
	var text strings.Builder
	switch item.kind {
	case "host":
		if host, ok := m.host(item.id); ok {
			fmt.Fprintf(&text, "%s\n%s · %s\n", detailHeading("Destination: "+host.Alias), login(host), host.OS)
			if host.WindowsAccount != "" {
				fmt.Fprintf(&text, "Windows account: %s\n", host.WindowsAccount)
			}
			fmt.Fprintf(&text, "\n%s\n%s%s\n", detailHeading("Saved status"), m.relationshipSummary(item), m.configurationSummary(host))
		}
	case "key":
		if key, ok := m.key(item.id); ok {
			fmt.Fprintf(&text, "%s\n%s · %s\n\n%s\n", detailHeading("Key: "+key.Name), key.Origin, keyAvailability(key), detailHeading("Saved status"))
			text.WriteString(m.relationshipSummary(item))
			if key.PrivatePath == "" {
				text.WriteString("Public key only; private-key verification unavailable.\n")
			} else {
				text.WriteString("Private key available; connection checks are per destination.\n")
			}
		}
	case "registration":
		for _, reg := range m.view.State.Registrations {
			if reg.ID == item.id {
				host, hostOK := m.host(reg.HostID)
				key, keyOK := m.recordedKey(reg.KeyID)
				if !hostOK || !keyOK {
					return "Requires review: missing destination/key reference. v: Technical details."
				}
				fmt.Fprintf(&text, "%s\n%s\n\n%s\n%s%s\n", detailHeading("Registration: "+host.Alias+" / "+key.Name), login(host), detailHeading("Saved status"), m.registrationSummary(reg), m.configurationSummary(host))
			}
		}
	}
	if text.Len() == 0 {
		return "Requires review: target no longer available. Refresh before continuing."
	}
	text.WriteString("Saved evidence only; not a live SSH check.\n")
	fmt.Fprintf(&text, "\n%s\n%s", detailHeading("Next actions"), m.detailActions(item))
	if item.kind == "host" || item.kind == "key" {
		title := "Key registrations"
		if item.kind == "key" {
			title = "Destinations"
		}
		fmt.Fprintf(&text, "\n%s\n", detailHeading(title))
		found := false
		for _, reg := range m.view.State.Registrations {
			if !relatedRegistration(item, reg) {
				continue
			}
			found = true
			if item.kind == "host" {
				if key, ok := m.recordedKey(reg.KeyID); ok {
					// A fingerprint hint distinguishes duplicate names; v retains the full value.
					fmt.Fprintf(&text, "%s · %s\n", key.Name, fingerprintHint(key.Fingerprint))
				} else {
					text.WriteString("Requires review: key reference missing\n")
				}
			} else if host, ok := m.host(reg.HostID); ok {
				fmt.Fprintf(&text, "%s · %s\n", host.Alias, login(host))
				text.WriteString(m.configurationSummary(host) + "\n")
			} else {
				text.WriteString("Requires review: destination reference missing\n")
			}
			text.WriteString(m.registrationSummary(reg) + "\n")
		}
		if !found {
			text.WriteString("No registrations recorded. Connection: not recorded.\nUse r to register a public key with a destination.\n")
		}
	}
	history := 0
	for _, event := range m.view.State.History {
		for _, reg := range m.view.State.Registrations {
			if event.RegistrationID == reg.ID && relatedRegistration(item, reg) {
				history++
			}
		}
	}
	fmt.Fprintf(&text, "\nHistory: %d recorded event(s). h opens read-only history.\nv opens full IDs, fingerprints, paths and saved checkpoints.\n", history)
	return text.String()
}

func fingerprintHint(fingerprint string) string {
	if len(fingerprint) > 23 {
		return fingerprint[:23] + "…"
	}
	return fingerprint
}

func (m model) relationshipSummary(item entry) string {
	registrations, connections := map[string]int{}, map[string]int{}
	for _, reg := range m.view.State.Registrations {
		if relatedRegistration(item, reg) {
			registrations[registrationStatus(reg.Status)]++
			connections[connectionStatus(reg.ConnectionCheck)]++
		}
	}
	if len(registrations) == 0 {
		return "Registration: none recorded.\nConnection: not recorded.\n"
	}
	var registered, checked []string
	for _, status := range []string{"registered", "removed", "awaiting registration", "attempt failed", "outcome unknown"} {
		if count := registrations[status]; count > 0 {
			registered = append(registered, fmt.Sprintf("%d %s", count, status))
		}
	}
	for _, status := range []string{"verified previously", "failed at last check", "not checked", "outcome unknown", "not recorded"} {
		if count := connections[status]; count > 0 {
			checked = append(checked, fmt.Sprintf("%d %s", count, status))
		}
	}
	return "Registration: " + strings.Join(registered, ", ") + ".\nConnection: " + strings.Join(checked, ", ") + ".\n"
}

func registrationStatus(status string) string {
	switch status {
	case "registered", "removed":
		return status
	case "pending":
		return "awaiting registration"
	case "failed":
		return "attempt failed"
	default:
		return "outcome unknown"
	}
}

func connectionStatus(status string) string {
	switch status {
	case "verified":
		return "verified previously"
	case "failed":
		return "failed at last check"
	case "unverified":
		return "not checked"
	case "unknown":
		return "outcome unknown"
	default:
		return "not recorded"
	}
}

func (m model) registrationSummary(reg state.Registration) string {
	text := fmt.Sprintf("Registration: %s (recorded)\nConnection: %s (recorded)\n", registrationStatus(reg.Status), connectionStatus(reg.ConnectionCheck))
	if key, ok := m.recordedKey(reg.KeyID); ok && key.PrivatePath == "" {
		text += "Public key only; private-key verification unavailable.\n"
	}
	if reg.Stage != "complete" {
		text += "Saved work unfinished; inspect Tasks or continue saved work.\n"
	}
	return text
}

func (m model) configurationSummary(host state.Host) string {
	if host.ConfigKeyID == "" {
		return "Configuration: no SSH alias binding recorded."
	}
	if key, ok := m.key(host.ConfigKeyID); ok {
		return "Configuration: alias bound to " + key.Name + ".\nInclude activation is not established by this saved binding."
	}
	return "Configuration: requires review; bound key is missing."
}

func (m model) detailActions(item entry) string {
	// Overview content is composed before openDocument changes the current page.
	m.page = "details"
	if m.stale {
		return "F5 Refresh saved state before changing anything.\n"
	}
	if item.readOnly {
		return "Read-only entry. v: inspect technical details for recovery guidance.\n"
	}
	var text strings.Builder
	if item.kind == "task" {
		return "s Continue saved work; review before terminal approval.\n"
	}
	switch {
	case item.kind == "host" && len(m.view.State.Keys) == 0:
		text.WriteString("c Create key / i Import key before registering.\n")
	case item.kind == "key" && len(m.view.State.Hosts) == 0:
		text.WriteString("a Add destination to register this key.\n")
	case item.kind == "registration":
		text.WriteString("r Register this destination/key pair.\n")
	default:
		text.WriteString("r Register public key; choose the related destination/key.\n")
	}
	if len(m.unregisterChoices(item)) > 0 {
		text.WriteString("u Unregister public key from destination; keep local files.\n")
	}
	if len(m.resumeChoices(item)) > 0 {
		text.WriteString("s Continue saved work; review before terminal approval.\n")
	}
	if item.kind == "key" {
		if blocker := m.deleteBlocker(item); blocker != "" {
			// Exact operation identifiers and file diagnostics belong in technical details.
			if strings.Contains(blocker, "Requires review:") {
				blocker = "Requires review: unfinished local deletion or Include edit; inspect Tasks"
			}
			text.WriteString("Local deletion blocked: " + blocker + ".\n")
		} else {
			text.WriteString("d Delete local key files; terminal checks and approval required.\n")
		}
	}
	return text.String()
}

func (m model) taskDetails(task Task) string {
	var text strings.Builder
	title := task.Title
	if strings.HasPrefix(task.ID, "config:") {
		title = "Requires review: configuration file"
	}
	fmt.Fprintf(&text, "%s\n%s\n", detailHeading("Task"), title)
	if host, ok := m.host(task.HostID); ok {
		fmt.Fprintf(&text, "Destination: %s · %s\n", host.Alias, login(host))
	}
	if key, ok := m.recordedKey(task.KeyID); ok {
		fmt.Fprintf(&text, "Key: %s · %s\n", key.Name, fingerprintHint(key.Fingerprint))
	}
	fmt.Fprintf(&text, "\n%s\n", detailHeading("Saved status"))
	if task.ResumeID == "" {
		text.WriteString("Saved evidence requires review; no supported continuation.\n")
	} else {
		text.WriteString("Saved work needs follow-up. Continuing rechecks saved state.\n")
		if task.RegistrationID != "" && task.ResumeID == task.RegistrationID {
			text.WriteString("Continue will reconcile remote state before retry.\n")
		}
	}
	text.WriteString("Saved evidence only; not a live SSH check.\n")
	fmt.Fprintf(&text, "\n%s\n%sv opens source IDs, saved checkpoints and recovery guidance.\n", detailHeading("Next actions"), m.detailActions(m.taskEntry(task)))
	return text.String()
}
