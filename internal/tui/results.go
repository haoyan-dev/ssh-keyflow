package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"ssh-keyflow/internal/app"
)

func (r operationResult) last(kind app.ReportKind) app.Report {
	var found app.Report
	for _, fact := range r.reports {
		if fact.Kind == kind {
			found = fact
		}
	}
	return found
}
func (r operationResult) saved(kind app.ReportKind) bool {
	for _, fact := range r.reports {
		if fact.Kind == kind && fact.Saved && fact.Outcome == app.OutcomeApplied {
			return true
		}
	}
	return false
}

// Classify only typed current-operation evidence. Snapshot is separate recorded
// context, and native diagnostics/error text are never completion evidence.
func (r operationResult) summary() string {
	manual := r.last(app.ReportManual)
	if manual.Outcome == app.OutcomeRequired {
		if r.err != nil {
			return "Operation stopped; outcome unknown; manual action required"
		}
		return "Manual action required"
	}
	uncertain := false
	for _, fact := range r.reports {
		if fact.Outcome == app.OutcomeUnknown {
			uncertain = true
		}
	}
	confirmation := r.last(app.ReportConfirmation)
	if confirmation.Outcome == app.OutcomeDeclined && confirmation.NoChanges && !uncertain {
		return "Cancelled before changes"
	}
	if r.last(app.ReportRegistration).Operation == "register" && r.saved(app.ReportRegistration) && r.last(app.ReportVerification).Outcome == app.OutcomeFailed {
		return "Registered; connection check failed"
	}
	if r.last(app.ReportRegistration).Operation == "register" && r.saved(app.ReportRegistration) && r.last(app.ReportAlias).Outcome == app.OutcomeDeclined {
		return "Stopped after registration; managed alias not generated this time"
	}
	include := r.last(app.ReportInclude)
	if r.last(app.ReportRegistration).Operation == "register" && r.saved(app.ReportRegistration) && r.saved(app.ReportAlias) && include.Outcome != app.OutcomeApplied && include.Outcome != app.OutcomeReconciled && include.Outcome != app.OutcomeSkipped {
		return "Registration and managed alias saved; Include setup remains"
	}
	for _, kind := range []app.ReportKind{app.ReportCreation, app.ReportDeletion, app.ReportImport, app.ReportRegistration, app.ReportAlias, app.ReportInclude, app.ReportVerification, app.ReportManual, app.ReportConfirmation} {
		fact := r.last(kind)
		if fact.Outcome == app.OutcomeUnknown || fact.Outcome == app.OutcomeFailed || fact.Outcome == app.OutcomeRequiresReview || fact.Outcome == app.OutcomeDeclined {
			return "Operation stopped; outcome unknown or requires review"
		}
	}
	if r.err != nil {
		return "Operation stopped; outcome unknown; saved checkpoints remain"
	}
	reg := r.last(app.ReportRegistration)
	complete := false
	switch r.request.Operation {
	case "create":
		complete = r.saved(app.ReportCreation) && r.last(app.ReportCreation).Stage == "complete"
	case "import":
		complete = r.saved(app.ReportImport)
	case "delete":
		complete = r.saved(app.ReportDeletion) && r.last(app.ReportDeletion).Stage == "complete"
	default:
		if r.saved(app.ReportDeletion) && r.last(app.ReportDeletion).Stage == "complete" {
			complete = true
		}
		if r.saved(app.ReportCreation) && r.last(app.ReportCreation).Stage == "complete" && r.target.HostID == "" {
			complete = true
		}
		if reg.Stage == "complete" && (reg.Saved || reg.Outcome == app.OutcomeReconciled) {
			if reg.Operation == "unregister" && reg.Status == "removed" {
				complete = true
			}
			verification := r.last(app.ReportVerification).Outcome
			if reg.Operation == "register" && reg.Status == "registered" && (verification == app.OutcomeVerified || verification == app.OutcomeUnavailable) {
				complete = true
				if r.request.Configure {
					complete = r.saved(app.ReportAlias) && (include.Outcome == app.OutcomeApplied || include.Outcome == app.OutcomeReconciled)
					// Public-only configuration is explicitly inapplicable. A skipped
					// alias alone never establishes requested configuration completion.
					if verification == app.OutcomeUnavailable && r.last(app.ReportAlias).Outcome == app.OutcomeSkipped && include.Outcome == app.OutcomeSkipped {
						complete = true
					}
				}
			}
		}
	}
	if complete {
		return "Completed; confirmed applicable work recorded"
	}
	return "Operation stopped; outcome unknown; completion not recorded"
}

func (m model) resultTasks() []Task {
	target := m.result.target
	ids := []string{target.RegistrationID, target.OnboardingID, target.CreationID, target.DeletionID}
	var matches []Task
	for _, task := range ProjectTasks(m.view) {
		match := target.HostID != "" && target.KeyID != "" && task.HostID == target.HostID && task.KeyID == target.KeyID
		for _, id := range ids {
			for _, source := range task.SourceIDs {
				if id != "" && id == source {
					match = true
				}
			}
		}
		for _, report := range m.result.reports {
			if (report.Kind == app.ReportAlias || report.Kind == app.ReportInclude) && report.Path != "" && task.ID == "config:"+report.Path {
				match = true
			}
			if report.Kind == app.ReportAlias || report.Kind == app.ReportInclude {
				for _, source := range task.SourceIDs {
					if report.Path != "" && report.Path == source {
						match = true
					}
				}
			}
		}
		if match {
			matches = append(matches, task)
		}
	}
	return matches
}
func (m model) resultEntity() entry {
	t := m.result.target
	if t.RegistrationID != "" {
		for _, r := range m.view.State.Registrations {
			if r.ID == t.RegistrationID {
				_, active := m.key(r.KeyID)
				return entry{kind: "registration", id: r.ID, readOnly: !active}
			}
		}
	}
	if _, ok := m.key(t.KeyID); ok {
		return entry{kind: "key", id: t.KeyID}
	}
	if _, ok := m.host(t.HostID); ok {
		return entry{kind: "host", id: t.HostID}
	}
	return entry{}
}
func (m model) resultControls() []string {
	if m.stale {
		return []string{"Refresh", "Exit", "Help"}
	}
	controls := []string{}
	if m.resultCanContinue() {
		controls = append(controls, "Continue saved work")
	}
	if m.result.last(app.ReportManual).Outcome == app.OutcomeRequired && m.result.last(app.ReportManual).Operation == "register" {
		if _, ok := m.host(m.result.target.HostID); ok {
			if _, ok := m.key(m.result.target.KeyID); ok {
				controls = append(controls, "Register again")
			}
		}
	}
	if len(m.resultTasks()) > 0 {
		controls = append(controls, "View task")
	}
	if m.resultEntity().id != "" {
		controls = append(controls, "View entity")
	}
	if m.result.diagnostics != "" {
		controls = append(controls, "Instructions")
	}
	return append(controls, "Refresh", "Exit", "Help")
}

func (m model) resultCanContinue() bool {
	for _, task := range m.resultTasks() {
		if task.ResumeID != "" {
			return true
		}
	}
	return false
}
func (m model) resultFooter() string {
	controls := m.resultControls()
	footer := "enter: " + controls[m.resultControl%len(controls)]
	position := fmt.Sprintf("%d/%d", m.resultControl%len(controls)+1, len(controls))
	instructions := ""
	if m.result.diagnostics != "" && !m.stale {
		instructions = "o instructions · "
	}
	if m.width < 60 {
		return footer + "\n" + instructions + "Tab " + position + "\n↑/↓ " + m.scrollPosition() + " · ? help · esc back"
	}
	return footer + " · " + instructions + "Tab " + position + "\n↑/↓ scroll · ? help · esc back · q exit"
}
func (m *model) resultAction(action string) {
	if m.stale && action != "Refresh" && action != "Help" && action != "Exit" {
		m.status = "Saved state stale; F5 Refresh before changing anything"
		return
	}
	switch action {
	case "Instructions":
		if m.result.diagnostics == "" {
			return
		}
		m.openDocument("instructions", entry{}, detailHeading("Diagnostics / instructions")+"\nSession output; not saved after exit.\n\n"+ansi.Strip(m.result.diagnostics))
	case "Refresh":
		m.refreshSnapshot()
	case "Exit": // Handled by updateResult.
	case "Help":
		m.openDocument("help", entry{}, m.helpText())
	case "View entity":
		if item := m.resultEntity(); item.id != "" {
			m.openDocument("details", item, m.entityDetails(item))
		} else {
			m.status = "Result entity unavailable; F5 Refresh saved state"
		}
	case "View task":
		tasks := m.resultTasks()
		if len(tasks) == 1 {
			m.openDocument("details", m.taskEntry(tasks[0]), m.taskDetails(tasks[0]))
		} else if len(tasks) > 1 {
			// This is a read-only task picker; choosing opens details, never executes.
			var entries []entry
			for _, task := range tasks {
				entries = append(entries, m.taskEntry(task))
			}
			m.openPicker("View result task", entries)
		} else {
			m.status = "No saved recovery task; instructions remain in this result"
		}
	case "Continue saved work":
		if m.stale {
			m.status = "Saved state stale; F5 Refresh before changing anything"
			return
		}
		var choices []entry
		for _, task := range m.resultTasks() {
			if task.ResumeID != "" {
				choices = append(choices, entry{kind: "resume", id: task.ResumeID, title: task.Title, description: strings.Join(task.SourceIDs, " · ")})
			}
		}
		m.formTarget = app.Request{Operation: "resume"}
		if len(choices) == 1 {
			m.formTarget.Resume = choices[0].id
			m.openForm("resume")
			return
		}
		if len(choices) > 1 {
			m.openPicker("Choose result operation to continue", choices)
			return
		}
		m.status = "No supported saved continuation"
	case "Register again":
		if m.stale {
			m.status = "Saved state stale; F5 Refresh before changing anything"
			return
		}
		t := m.result.target
		if _, ok := m.host(t.HostID); ok {
			if _, ok := m.key(t.KeyID); ok {
				m.formTarget = app.Request{Operation: "register", Host: t.HostID, Key: t.KeyID}
				m.openForm("register")
				return
			}
		}
		m.status = "Exact result destination/key unavailable; F5 Refresh"
	}
}
func (m model) updateResult(message tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	controls := m.resultControls()
	switch message.String() {
	case "tab":
		m.resultControl = (m.resultControl + 1) % len(controls)
	case "shift+tab":
		m.resultControl = (m.resultControl + len(controls) - 1) % len(controls)
	case "enter":
		action := controls[m.resultControl%len(controls)]
		if action == "Exit" {
			return m, tea.Quit
		}
		m.resultAction(action)
	case "v":
		m.resultAction("View entity")
	case "t":
		m.resultAction("View task")
	case "s":
		m.resultAction("Continue saved work")
	case "r":
		found := false
		for _, action := range controls {
			if action == "Register again" {
				found = true
			}
		}
		if found {
			m.resultAction("Register again")
		} else if m.stale {
			m.status = "Saved state stale; F5 Refresh before changing anything"
		}
	case "o":
		m.resultAction("Instructions")
	default:
		var cmd tea.Cmd
		m.content, cmd = m.content.Update(message)
		return m, cmd
	}
	return m, nil
}

func (m *model) renderOperationResult() {
	var text strings.Builder
	m.status = ""
	fmt.Fprintf(&text, "%s\n", detailHeading("Result: "+m.result.summary()))
	controls := m.resultControls()
	fmt.Fprintf(&text, "Next: %s\n", controls[0])
	if host, ok := m.host(m.result.target.HostID); ok {
		fmt.Fprintf(&text, "Destination: %s\n", host.Alias)
	}
	if key, ok := m.recordedKey(m.result.target.KeyID); ok {
		fmt.Fprintf(&text, "%s\n", key.Name)
	}
	text.WriteString("\n" + detailHeading("Next steps") + "\n")
	if m.result.reloadErr != nil {
		m.status = "Saved state stale; F5 Refresh / q Exit"
		fmt.Fprintf(&text, "Reload failed: %v\nPrevious saved state is stale; mutations disabled until Refresh succeeds.\n", m.result.reloadErr)
	}
	tasks := m.resultTasks()
	if !m.stale && len(tasks) == 0 {
		text.WriteString("No saved recovery task. ")
		if m.result.last(app.ReportManual).Outcome == app.OutcomeRequired {
			if m.result.last(app.ReportManual).Operation == "unregister" {
				text.WriteString("Inspect the saved registration and Unregister again after exit/restart; local key files are preserved. Session instructions are not saved.\n")
			} else {
				text.WriteString("Register again after exit/restart; session instructions are not saved.\n")
			}
		} else {
			if m.result.request.Configure && m.result.last(app.ReportInclude).Outcome != app.OutcomeApplied && m.result.last(app.ReportInclude).Outcome != app.OutcomeReconciled {
				text.WriteString("Include/manual setup may remain. Register or Add with configuration requested repeats registration/reconciliation; no config-only operation. Session follow-up is not saved.\n")
			} else {
				text.WriteString("No additional follow-up is recorded.\n")
			}
		}
	} else if !m.stale && m.resultCanContinue() {
		text.WriteString("Next: Continue saved work or View task; reconcile before remote retry.\n")
	} else if !m.stale {
		text.WriteString("View task; no supported continuation. Inspect saved evidence and recovery guidance.\n")
	}
	if m.result.diagnostics != "" && !m.stale {
		text.WriteString("o: Open session instructions / diagnostics.\n")
	}
	if m.result.err != nil {
		fmt.Fprintf(&text, "Error: %v\n", m.result.err)
	}
	if len(m.result.reports) == 0 {
		text.WriteString("Outcome unknown: no operation facts recorded.\n")
	}
	text.WriteString("\n" + detailHeading("Current operation facts:") + "\n")
	for _, report := range m.result.reports {
		if report.Kind == app.ReportResolvedTarget {
			continue
		}
		fmt.Fprintf(&text, "%s: %s", report.Kind, report.Outcome)
		if report.Saved {
			text.WriteString(" (saved checkpoint)")
		}
		text.WriteString("\n")
		if report.Kind == app.ReportAlias && report.Outcome == app.OutcomeDeclined {
			text.WriteString("Managed alias not generated this time; prior bindings/files remain separate evidence.\n")
		}
		if report.Kind == app.ReportInclude && report.Outcome == app.OutcomeDeclined {
			text.WriteString("Main config unchanged by this request; Include/manual setup remains.\n")
		}
	}
	text.WriteString("\n" + detailHeading("Checkpoint details:") + "\n")
	fmt.Fprintf(&text, "Request: %s\n", m.result.request.Operation)
	for _, item := range []struct{ label, id string }{{"Destination", m.result.target.HostID}, {"Key", m.result.target.KeyID}, {"Registration", m.result.target.RegistrationID}, {"Onboarding", m.result.target.OnboardingID}, {"Creation", m.result.target.CreationID}, {"Deletion", m.result.target.DeletionID}} {
		if item.id != "" {
			fmt.Fprintf(&text, "%s: %s\n", item.label, item.id)
		}
	}
	for _, report := range m.result.reports {
		if report.Path != "" {
			fmt.Fprintf(&text, "File: %s\n", report.Path)
		}
		if report.Status != "" {
			fmt.Fprintf(&text, "Saved registration: %s / %s / %s; destination connection: %s\n", report.Operation, report.Status, report.Stage, report.ConnectionCheck)
		}
	}
	text.WriteString("\n" + detailHeading("Saved evidence") + "\nMay predate this operation; not a live probe.\n")
	if m.stale {
		text.WriteString("Previous view is stale; Refresh retries Snapshot only, never execution.\n")
	}
	if item := m.resultEntity(); item.id != "" {
		text.WriteString(m.technicalDetails(item))
	} else {
		text.WriteString("Entity not recorded; allocated IDs do not establish a saved entity.\n")
	}
	text.WriteString("\n" + detailHeading("Diagnostics / instructions") + "\n" + ansi.Strip(m.result.diagnostics))
	m.setContent(text.String())
}
