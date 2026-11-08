package tui

import (
	"maps"
	"strings"

	tea "charm.land/bubbletea/v2"
	"ssh-keyflow/internal/app"
)

// Draft entry is the baseline; target IDs selected before entry are not edits.
func (m *model) rememberDraft() {
	m.clearHidden()
	m.initialDraft = maps.Clone(m.draft)
	m.initialTarget = m.formTarget
}

func (m *model) cancelDraft() {
	m.saveFields()
	dirty := m.formTarget != m.initialTarget
	for name, value := range m.draft {
		dirty = dirty || value != m.initialDraft[name]
	}
	if !dirty {
		m.discardDraft()
		return
	}
	m.discardSelected = false
	m.openDocument("discard", entry{}, "")
	m.renderDiscard()
}

func (m *model) renderDiscard() {
	keep, discard := "> Keep editing", "  Discard draft"
	if m.discardSelected {
		keep, discard = "  Keep editing", "> Discard draft"
	}
	m.setContent(detailHeading("Discard changes?") + "\n" + actionTitle(m.action) + "\nUnsaved inputs will be lost.\n" + keep + "\n" + discard)
	m.content.GotoTop()
}

func (m *model) discardDraft() {
	// Drop overlays and the form together; the caller's category/query stay intact.
	for m.page != "form" && m.page != "review" && len(m.origins) > 0 {
		m.back()
	}
	m.back()
	m.draft, m.fields, m.initialDraft = nil, nil, nil
	m.request, m.formTarget = app.Request{}, app.Request{}
	m.status = "Changes cancelled"
}

func (m *model) editReview() {
	m.inputGeneration++
	m.page, m.content = "form", m.editContent
	width, height := m.content.Width(), m.content.Height()
	m.resizeBody()
	m.setContent(m.formBody())
	if width != m.content.Width() || height != m.content.Height() {
		m.revealFormFocus()
	}
}

func (m model) updateDiscard(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc", "ctrl+c":
		m.back()
	case "tab", "shift+tab", "up", "down", "left", "right":
		m.discardSelected = !m.discardSelected
		m.renderDiscard()
	case "enter":
		if m.discardSelected {
			m.discardDraft()
		} else {
			m.back()
		}
	case "?":
		m.openDocument("help", entry{}, m.helpText())
	}
	return m, nil
}

// A value copy preserves both the form's scroll and its textinput caret models.
func (m *model) rememberEdit() {
	m.setContent(m.formBody())
	m.editContent = m.content
}

func (m model) reviewControls() string {
	controls := []string{"Continue to terminal", "Edit", "Cancel", "Help"}
	if m.width < 60 {
		controls[0] = "Continue"
		controls[m.reviewControl] = "[" + controls[m.reviewControl] + "]"
		return strings.Join(controls, " ") + "\nTab choose · Enter select\n↑/↓ scroll · e/Esc edit"
	}
	controls[m.reviewControl] = "[" + controls[m.reviewControl] + "]"
	return strings.Join(controls, " · ") + "\nTab choose · Enter select · e/Esc edit · ? Help\n↑/↓ scroll · PgUp/PgDn"
}

func (m model) expectedEffects() string {
	effects := ""
	switch m.action {
	case "create":
		effects = "Expected: create managed Ed25519 files and save the key. Paths are subject to terminal review."
	case "import":
		effects = "Expected: record the selected existing key; source files remain in place."
	case "add":
		effects = "Expected: save destination/setup intent, then register the selected key or prepare manual instructions."
	case "register":
		effects = "Expected: register this public key, or prepare manual instructions; check the connection when a private key is available."
	case "unregister":
		effects = "Expected: remove only the selected public-key fingerprint, or prepare manual instructions. Local key files remain. Fresh alternate authentication is required."
	case "delete":
		effects = "Expected: delete only this managed key's local files after exact approval and all registration, path, digest, ownership and configuration checks. No force/delete-all."
	case "resume":
		effects = "Expected: reconcile the selected saved operation before any guarded retry. Saved checkpoints are preserved; no rollback is promised."
		for _, deletion := range m.view.State.Deletions {
			if deletion.ID == m.request.Resume {
				effects += " Continue only this key's local deletion after registration, path, digest, ownership and configuration checks."
			}
		}
		for _, job := range m.view.State.Onboardings {
			if job.ID == m.request.Resume {
				effects += " Continue saved key work, then registration or manual instructions at this destination."
			}
		}
	}
	if m.request.Configure {
		effects += " Alias generation and main-config Include each remain subject to eligibility and native approval."
	}
	return effects
}

func (m model) draftActive() bool {
	if m.page == "form" || m.page == "review" || m.page == "discard" {
		return true
	}
	for _, origin := range m.origins {
		if origin.page == "form" || origin.page == "review" {
			return true
		}
	}
	return false
}
