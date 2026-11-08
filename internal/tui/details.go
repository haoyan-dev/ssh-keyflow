package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"ssh-keyflow/internal/state"
)

func detailHeading(text string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Cyan).Render(text)
}

func detailEvidence(text, status string) string {
	style := lipgloss.NewStyle()
	switch status {
	case "registered", "verified previously":
		style = style.Foreground(lipgloss.Green)
	case "attempt failed", "failed at last check", "blocked":
		style = style.Foreground(lipgloss.Red)
	case "awaiting registration", "outcome unknown", "not checked", "not recorded", "unfinished":
		style = style.Foreground(lipgloss.Yellow)
	}
	return style.Render(text)
}

func (m model) detailSection(title, body string) string {
	return panel(title, strings.TrimSpace(body), max(10, m.width-2))
}

func detailActionHints(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "Local deletion blocked:"):
			lines[i] = detailEvidence(line, "blocked")
		case strings.HasPrefix(line, "Read-only entry."), strings.HasPrefix(line, "F5 Refresh"):
			lines[i] = detailEvidence(line, "unfinished")
		default:
			lines[i] = lipgloss.NewStyle().Foreground(lipgloss.Cyan).Render(line)
		}
	}
	return strings.Join(lines, "\n")
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
	var status, configuration, identityTitle string
	switch item.kind {
	case "host":
		if host, ok := m.host(item.id); ok {
			identityTitle = "Destination: " + host.Alias
			fmt.Fprintf(&text, "%s\n%s\n", detailHeading("Destination: "+host.Alias), login(host))
			if host.WindowsAccount != "" {
				fmt.Fprintf(&text, "Windows account: %s\n", host.WindowsAccount)
			}
			configuration = "OS: " + host.OS + "\n" + m.configurationSummary(host)
		}
	case "key":
		if key, ok := m.key(item.id); ok {
			identityTitle = "Key: " + key.Name
			fmt.Fprintf(&text, "%s\n%s · %s\n", detailHeading("Key: "+key.Name), key.Origin, keyAvailability(key))
			if key.PrivatePath == "" {
				configuration = "Public key only; private-key verification unavailable.\n"
			} else {
				configuration = "Private-key path recorded; availability is rechecked in the terminal.\n"
			}
		}
	case "registration":
		for _, reg := range m.view.State.Registrations {
			if reg.ID == item.id {
				host, hostOK := m.host(reg.HostID)
				key, keyOK := m.recordedKey(reg.KeyID)
				if !hostOK || !keyOK {
					return m.detailSection("Requires review", "Missing destination/key reference.\nv: Technical details.")
				}
				identityTitle = "Registration: " + host.Alias + " / " + key.Name
				fmt.Fprintf(&text, "%s\n%s\n", detailHeading("Registration: "+host.Alias+" / "+key.Name), login(host))
				status = m.registrationSummary(reg)
				configuration = m.configurationSummary(host)
			}
		}
	}
	if text.Len() == 0 {
		return m.detailSection("Requires review", "Target no longer available.\nRefresh before continuing.")
	}
	if item.kind == "host" || item.kind == "key" {
		var relationships strings.Builder
		found := false
		for _, reg := range m.view.State.Registrations {
			if !relatedRegistration(item, reg) {
				continue
			}
			found = true
			if item.kind == "host" {
				if key, ok := m.recordedKey(reg.KeyID); ok {
					fmt.Fprintf(&relationships, "Key: %s\n", m.keyChoiceEntry(key).title)
				} else {
					relationships.WriteString(detailEvidence("Requires review: key reference missing", "blocked") + "\n")
				}
			} else if host, ok := m.host(reg.HostID); ok {
				fmt.Fprintf(&relationships, "Destination: %s · %s\n", host.Alias, login(host))
			} else {
				relationships.WriteString(detailEvidence("Requires review: destination reference missing", "blocked") + "\n")
			}
			relationships.WriteString(m.registrationSummary(reg) + "\n")
		}
		if !found {
			relationships.WriteString("Registration: none recorded.\nConnection: not recorded.\n")
		}
		status = relationships.String()
	}
	identity := text.String()
	text.Reset()
	actions := m.detailActions(item)
	next := primaryDetailAction(actions)
	width := max(10, m.width-2)
	// The compact first panel is a useful summary, not an arbitrary excerpt
	// of the complete relationship list. Mixed pairs never get one status.
	if m.width < 70 {
		quick := m.quickEvidence(item)
		text.WriteString(panel(identityTitle, "Saved status\n"+quick+"\n"+detailHeading("> "+next), width))
		text.WriteString("\n\n" + panel("Identity", identity, width))
		text.WriteString("\n\n" + m.detailSection("Saved status", status))
		text.WriteString("\n\n" + m.detailSection("Next actions", detailActionHints(actions)))
	} else {
		text.WriteString(panel("Identity", identity, width) + "\n\n")
		leftWidth := (width - 2) * 54 / 100
		// Scope disclosure is part of the evidence group, never a success claim.
		left := panel("Saved status", status, leftWidth)
		right := panel("Next step", detailHeading("> "+next)+"\n\nReview the target before terminal approval.", width-leftWidth-2)
		text.WriteString(panelPair(left, right))
		text.WriteString("\n\n" + m.detailSection("Next actions", detailActionHints(actions)))
	}
	text.WriteString("\n\nSaved evidence only; not a live SSH check.\n")
	if configuration != "" {
		text.WriteString("\n" + m.detailSection("Configuration", configuration) + "\n")
	}
	if item.kind == "key" {
		for _, reg := range m.view.State.Registrations {
			if relatedRegistration(item, reg) {
				if host, ok := m.host(reg.HostID); ok {
					text.WriteString("\n" + m.detailSection("Configuration · "+host.Alias, m.configurationSummary(host)) + "\n")
				}
			}
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
	text.WriteString("\n" + m.detailSection("Further reading", fmt.Sprintf("History: %d recorded event(s). h opens read-only history.\nv opens full IDs, fingerprints, paths and saved checkpoints.", history)))
	return text.String()
}

func primaryDetailAction(actions string) string {
	lines := strings.Split(strings.TrimSpace(actions), "\n")
	next := lines[0]
	for _, line := range lines {
		if strings.HasPrefix(line, "s Continue") {
			next = line
			break
		}
	}
	switch {
	case strings.HasPrefix(next, "s "):
		return "s Continue saved work"
	case strings.HasPrefix(next, "r "):
		return "r Register public key"
	case strings.HasPrefix(next, "c "):
		return "c Create / i Import key"
	case strings.HasPrefix(next, "a "):
		return "a Add destination"
	case strings.HasPrefix(next, "F5 "):
		return "F5 Refresh saved state"
	default:
		return "v Inspect technical details"
	}
}

func (m model) quickEvidence(item entry) string {
	var registrations []state.Registration
	for _, reg := range m.view.State.Registrations {
		if relatedRegistration(item, reg) {
			registrations = append(registrations, reg)
		}
	}
	if len(registrations) > 1 {
		return fmt.Sprintf("%d saved destination/key pairs\nStatus is shown per pair below.", len(registrations))
	}
	if len(registrations) == 0 {
		return "Registration: none recorded\nConnection: not recorded"
	}
	reg := registrations[0]
	registered, checked := registrationStatus(reg.Status), connectionStatus(reg.ConnectionCheck)
	shortConnection := map[string]string{
		"verified": "passed previously", "failed": "failed", "unverified": "not checked", "unknown": "unknown",
	}[reg.ConnectionCheck]
	if shortConnection == "" {
		shortConnection = "not recorded"
	}
	shortRegistration := map[string]string{"pending": "pending", "failed": "failed", "unknown": "unknown"}[reg.Status]
	if shortRegistration == "" {
		shortRegistration = registered
		if registered == "outcome unknown" {
			shortRegistration = "unknown"
		}
	}
	body := detailEvidence("Registration: "+shortRegistration, registered) + "\n" + detailEvidence("Connection: "+shortConnection, checked)
	if item.kind == "host" {
		if key, ok := m.recordedKey(reg.KeyID); ok {
			body = "Key: " + entryTitle(m.keyChoiceEntry(key), max(1, m.width-11)) + "\n" + body
		}
	}
	return body
}

func fingerprintHint(fingerprint string) string {
	if len(fingerprint) > 23 {
		return fingerprint[:23] + "…"
	}
	return fingerprint
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
	registered, checked := registrationStatus(reg.Status), connectionStatus(reg.ConnectionCheck)
	// "Saved status" scopes the entire panel; repeating "(recorded)" beside
	// every value makes the evidence harder to scan and wastes narrow columns.
	text := detailEvidence("Registration: "+registered, registered) + "\n" +
		detailEvidence("Connection: "+checked, checked) + "\n"
	if key, ok := m.recordedKey(reg.KeyID); ok && key.PrivatePath == "" {
		text += "Public key only; private-key verification unavailable.\n"
	}
	if reg.Stage != "complete" {
		text += detailEvidence("Saved work unfinished; inspect Tasks or continue saved work.", "unfinished") + "\n"
	}
	return text
}

func (m model) configurationSummary(host state.Host) string {
	if host.ConfigKeyID == "" {
		return detailEvidence("Configuration: no SSH alias binding recorded.", "not recorded")
	}
	if key, ok := m.key(host.ConfigKeyID); ok {
		return "Configuration: alias bound to " + key.Name + ".\nInclude activation is not established by this saved binding."
	}
	return detailEvidence("Configuration: requires review; bound key is missing.", "blocked")
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
	title := strings.Split(task.Title, " · ")[0]
	if strings.HasPrefix(task.ID, "config:") {
		title = "Requires review: configuration file"
	}
	// The reason precedes metadata and generic recovery explanations, including
	// on a twelve-row terminal. Full original reasons remain in Technical details.
	// A compact reason/action group stays above metadata even at 32×12.
	reason := taskReason(task)
	// File diagnostics remain complete in Technical details; the overview
	// exposes their reason before a long absolute path can fill the screen.
	if before, _, ok := strings.Cut(reason, ": open "); ok {
		reason = before
	}
	next := "v Read full reason"
	if task.ResumeID != "" && !m.stale {
		next = "s Continue saved work"
	}
	if m.stale {
		next = "F5 Refresh saved state"
	}
	summary := title + "\n" + reason + "\n" + detailHeading("> "+next)
	if m.width < 60 {
		summary = reason + "\n" + detailHeading("> "+next)
	}
	text.WriteString(m.detailSection("Task", summary) + "\n\n")
	var identity strings.Builder
	identity.WriteString(title + "\n")
	if task.ResumeID == "" {
		identity.WriteString("Read-only; no supported continuation.\n")
	} else {
		identity.WriteString("s Continue saved work; review first.\n")
	}
	if host, ok := m.host(task.HostID); ok {
		fmt.Fprintf(&identity, "Destination: %s · %s\n", host.Alias, login(host))
	}
	if key, ok := m.recordedKey(task.KeyID); ok {
		fmt.Fprintf(&identity, "Key: %s · %s\n", key.Name, fingerprintHint(key.Fingerprint))
	}
	text.WriteString(m.detailSection("Identity", identity.String()) + "\n\n")
	var status string
	if task.ResumeID == "" {
		status = "Saved evidence requires review; no supported continuation.\n"
	} else {
		status = "Saved work needs follow-up. Continuing rechecks saved state.\n"
		if task.RegistrationID != "" && task.ResumeID == task.RegistrationID {
			status += "Continue will reconcile remote state before retry.\n"
		}
	}
	status = detailEvidence(strings.TrimSpace(status), "unfinished") + "\nSaved evidence only; not a live SSH check.\n"
	fmt.Fprintf(&text, "\n%s\n%sv opens source IDs, saved checkpoints and recovery guidance.\n",
		m.detailSection("Saved status", status), m.detailSection("Next actions", detailActionHints(m.detailActions(m.taskEntry(task)))))
	return text.String()
}
