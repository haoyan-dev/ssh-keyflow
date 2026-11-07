package tui

import (
	"fmt"
	"ssh-keyflow/internal/app"
	"strings"
)

// Re-resolve exact IDs against a read-only fresh Snapshot. Labels may change;
// connection targets, key material and saved operation intent may not drift.
func (m *model) resolveFormTargets() error {
	fresh, err := app.Snapshot(m.options)
	if err != nil {
		return fmt.Errorf("Selected target requires review: reload failed: %w; draft retained", err)
	}
	request := m.formTarget
	for _, r := range m.view.State.Registrations {
		if r.ID == request.Registration || r.ID == request.Resume {
			found := false
			for _, current := range fresh.State.Registrations {
				if current.ID == r.ID {
					found = current == r
				}
			}
			if !found {
				return fmt.Errorf("Registration target %s changed or is missing; draft retained, cancel and reselect", r.ID)
			}
			request.Host, request.Key = r.HostID, r.KeyID
		}
	}
	if request.Resume != "" {
		found := false
		for _, old := range m.view.State.Creations {
			if old.ID == request.Resume {
				for _, current := range fresh.State.Creations {
					if old == current {
						found = true
					}
				}
			}
		}
		for _, old := range m.view.State.Deletions {
			if old.ID == request.Resume {
				for _, current := range fresh.State.Deletions {
					if old == current {
						found = true
						request.Key = old.KeyID
					}
				}
			}
		}
		for _, old := range m.view.State.Onboardings {
			if old.ID == request.Resume {
				for _, current := range fresh.State.Onboardings {
					if old == current {
						found = true
						request.Host, request.Key = old.HostID, old.KeyID
						if old.CreationID != "" {
							found = false
							for _, creation := range m.view.State.Creations {
								if creation.ID != old.CreationID || creation.KeyID != old.KeyID {
									continue
								}
								for _, currentCreation := range fresh.State.Creations {
									if creation == currentCreation {
										found = true
										// Incomplete creation reserves an identity before a managed key exists.
										if old.Stage == "key_pending" && creation.Stage != "complete" {
											request.Key = ""
										}
									}
								}
							}
						}
					}
				}
			}
		}
		for _, old := range m.view.State.Registrations {
			if old.ID == request.Resume {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("Saved operation target %s changed or is missing; draft retained, cancel and reselect", request.Resume)
		}
	}
	if request.Host != "" {
		old, ok := m.host(request.Host)
		found := false
		for _, current := range fresh.State.Hosts {
			if current.ID == request.Host && ok {
				old.Alias = current.Alias
				found = old == current
			}
		}
		if !found {
			return fmt.Errorf("Destination target %s changed or is missing; draft retained, cancel and reselect", request.Host)
		}
	}
	if request.Key != "" {
		old, ok := m.key(request.Key)
		found := false
		for _, current := range fresh.State.Keys {
			if current.ID == request.Key && ok {
				old.Name, old.ConnectionCheck = current.Name, current.ConnectionCheck
				found = old == current
			}
		}
		if !found {
			return fmt.Errorf("Managed key target %s changed or is missing; draft retained, cancel and reselect", request.Key)
		}
	}
	m.view = fresh
	return nil
}

func (m *model) targetError(err error) {
	name := "continue"
	switch m.action {
	case "register", "add", "delete":
		name = "key"
	case "unregister":
		name = "registration"
	case "resume":
		name = "resume"
	}
	m.invalid(name, err.Error())
}

func (m model) keyChoices() []entry {
	var choices []entry
	for _, k := range m.view.State.Keys {
		choices = append(choices, m.keyChoiceEntry(k))
	}
	return choices
}

func (m model) resumeChoices(target entry) []entry {
	var choices []entry
	for _, task := range ProjectTasks(m.view) {
		if task.ResumeID == "" {
			continue
		}
		matches := target.kind == "" ||
			(target.kind == "task" && target.id == task.ID) ||
			(target.kind == "host" && target.id == task.HostID) ||
			(target.kind == "key" && target.id == task.KeyID) ||
			(target.kind == "registration" && target.id == task.RegistrationID)
		// Local recovery is also available from a category list.
		if !matches && m.page == "list" && target.kind != "task" {
			for _, c := range m.view.State.Creations {
				if c.ID == task.ResumeID {
					matches = true
				}
			}
			for _, d := range m.view.State.Deletions {
				if d.ID == task.ResumeID {
					matches = true
				}
			}
		}
		if matches {
			choices = append(choices, entry{kind: "resume", id: task.ResumeID, title: task.Title, description: strings.Join(task.Reasons, " · "), searchText: strings.Join(task.SourceIDs, " ")})
		}
	}
	return choices
}

// Keep target identity formatting shared with navigation; no name resolution is
// used for execution, even when multiple keys have the same display name.
func (m model) keyLabel(id string) string {
	if k, ok := m.key(id); ok {
		return k.Name + " [" + id + "]"
	}
	return id
}
func (m model) hostLabel(id string) string {
	if h, ok := m.host(id); ok {
		return h.Alias + " · " + login(h) + " [" + id + "]"
	}
	return id
}

func (m model) fixedOperationLabel() string {
	id := m.formTarget.Registration
	if id == "" {
		id = m.formTarget.Resume
	}
	for _, r := range m.view.State.Registrations {
		if r.ID == id {
			return m.registrationLabel(r) + " [" + id + "]"
		}
	}
	for _, c := range m.resumeChoices(entry{}) {
		if c.id == id {
			return c.title + " [" + id + "]"
		}
	}
	return id
}
