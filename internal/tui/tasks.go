package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/state"
)

// Task is a read-only projection. ID and SourceIDs name saved sources; ResumeID
// is only an existing supported operation, never a new workflow identifier.
type Task struct {
	ID, Title, HostID, KeyID, RegistrationID, ResumeID string
	SourceIDs, Reasons                                 []string
	Priority                                           int
}

// ProjectTasks does no I/O. Snapshot load errors belong to the caller; Review is
// an integrity warning, not permission to suppress guarded deletion recovery.
func ProjectTasks(view app.View) []Task {
	document := view.State
	var tasks []Task
	keys := map[string]state.Key{}
	hosts := map[string]state.Host{}
	deleted := map[string]bool{}
	creations := map[string]state.Creation{}
	for _, k := range document.Keys {
		keys[k.ID] = k
	}
	for _, h := range document.Hosts {
		hosts[h.ID] = h
	}
	for _, k := range document.DeletedKeys {
		deleted[k.Key.ID] = true
	}
	for _, c := range document.Creations {
		creations[c.ID] = c
	}
	type group struct {
		host, key string
		jobs      []state.Onboarding
		regs      []state.Registration
	}
	groups := map[string]*group{}
	pair := func(h, k string) *group {
		id := h + "/" + k
		if groups[id] == nil {
			groups[id] = &group{host: h, key: k}
		}
		return groups[id]
	}
	for _, j := range document.Onboardings {
		g := pair(j.HostID, j.KeyID)
		g.jobs = append(g.jobs, j)
	}
	for _, r := range document.Registrations {
		g := pair(r.HostID, r.KeyID)
		g.regs = append(g.regs, r)
	}
	linked := map[string]bool{}
	safeParents := map[string]Task{}
	for _, g := range groups {
		sort.Slice(g.jobs, func(i, j int) bool { return g.jobs[i].ID < g.jobs[j].ID })
		sort.Slice(g.regs, func(i, j int) bool { return g.regs[i].ID < g.regs[j].ID })
		task := Task{HostID: g.host, KeyID: g.key, Priority: 1}
		host, hostOK := hosts[g.host]
		key, keyOK := keys[g.key]
		label := host.Alias + " / " + key.Name + " · " + login(host)
		invalid := !hostOK || len(g.regs) > 1
		var incomplete state.Creation
		incompleteJobID := ""
		incompleteJobs := 0
		creationLinks := map[string]bool{}
		for _, j := range g.jobs {
			task.SourceIDs = append(task.SourceIDs, j.ID)
			if j.CreationID != "" {
				creationLinks[j.CreationID] = true
				task.SourceIDs = append(task.SourceIDs, j.CreationID)
				c, ok := creations[j.CreationID]
				if !ok || c.KeyID != g.key {
					invalid = true
				} else if c.Stage != "complete" {
					linked[c.ID] = true
					incomplete = c
					incompleteJobs++
					if j.Stage != "key_pending" {
						invalid = true
					}
					if incompleteJobID == "" {
						incompleteJobID = j.ID
					}
					task.Reasons = append(task.Reasons, "Local creation "+c.ID+": "+c.Stage)
				}
			}
		}
		if len(creationLinks) > 1 || incompleteJobs > 1 {
			invalid = true
		}
		var reg state.Registration
		if len(g.regs) > 0 {
			reg = g.regs[0]
			task.RegistrationID = reg.ID
			task.SourceIDs = append(task.SourceIDs, reg.ID)
		}
		if !keyOK && incomplete.ID == "" && !deleted[g.key] {
			invalid = true
		}
		if deleted[g.key] && reg.ID != "" && !(reg.Operation == "unregister" && reg.Status == "removed" && reg.Stage == "complete") {
			invalid = true
		}
		task.SourceIDs = uniqueSorted(task.SourceIDs)
		// Prefer onboarding identity throughout its saved lifetime, even when the
		// executable continuation changes from onboarding to registration.
		if len(g.jobs) > 0 {
			task.ID = g.jobs[0].ID
		} else {
			task.ID = reg.ID
		}
		if !invalid {
			safeParents[g.host+"/"+g.key] = task
		}
		if invalid {
			task.Priority = 0
			task.Title = "Requires review: saved links"
			task.SourceIDs = uniqueSorted(append(task.SourceIDs, g.host, g.key))
			task.Reasons = append(task.Reasons, "Missing/contradictory destination, key, creation or registration links; no continuation inferred")
		} else if reg.Operation == "unregister" {
			if reg.Status == "removed" && reg.Stage == "complete" {
				continue
			}
			task.ResumeID = reg.ID
			task.Title = "Outcome unknown: Continue unregister · " + label
			task.Reasons = append(task.Reasons, "Current unregister intent supersedes onboarding; reconcile remote state before retry. Local key files are preserved.")
		} else if deleted[g.key] {
			continue
		} else if incomplete.ID != "" {
			task.ResumeID = incompleteJobID
			task.Title = "Outcome unknown: Continue destination setup · " + host.Alias
			task.Reasons = append(task.Reasons, "Continue onboarding to reconcile key creation, then registration; this is not local-only recovery")
		} else if reg.ID != "" {
			if reg.Operation != "register" {
				task.Priority = 0
				task.Title = "Requires review: registration intent"
			} else if reg.Status != "registered" || reg.Stage != "complete" {
				task.ResumeID = reg.ID
				task.Title = "Outcome unknown: Continue registration · " + label
				task.Reasons = append(task.Reasons, "Saved registration "+reg.Status+" / "+reg.Stage+"; reconcile remote state before retry. Manual versus interrupted outcome is not recorded.")
			} else if key.PrivatePath != "" && reg.ConnectionCheck != "verified" {
				task.Priority = 3
				task.ResumeID = reg.ID
				task.Title = "Connection check " + reg.ConnectionCheck + ": Continue registration · " + label
				task.Reasons = append(task.Reasons, "Registration saved; specified-private-key check needs follow-up. Continue registration includes reconciliation/verification; no Verify-only operation.")
			} else {
				continue
			}
		} else if len(g.jobs) > 0 {
			task.ResumeID = g.jobs[0].ID
			task.Title = "Outcome unknown: Continue destination setup · " + host.Alias
			task.Reasons = append(task.Reasons, "Registration not recorded; completed local creation alone does not finish destination setup. Continuation includes registration.")
		}
		task.Reasons = append(task.Reasons, "Destination ID: "+g.host+"; Key ID: "+g.key, "Configuration request/activation: not recorded by onboarding; inspect separate file evidence")
		tasks = append(tasks, task)
	}
	for _, c := range document.Creations {
		if c.Stage == "complete" || linked[c.ID] {
			continue
		}
		tasks = append(tasks, Task{ID: c.ID, KeyID: c.KeyID, ResumeID: c.ID, Priority: 1, Title: "Outcome unknown: Continue local creation · " + c.Name, SourceIDs: []string{c.ID}, Reasons: []string{"Saved creation: " + c.Stage, "Reconcile/create local key only; no remote registration is promised"}})
	}
	for _, d := range document.Deletions {
		if d.Stage == "complete" {
			continue
		}
		task := Task{ID: d.ID, KeyID: d.KeyID, ResumeID: d.ID, Priority: 1, Title: "Interrupted local deletion · " + keys[d.KeyID].Name, SourceIDs: []string{d.ID}, Reasons: []string{fmt.Sprintf("Public removed: %t; private removed: %t (saved checkpoints)", d.PublicRemoved, d.PrivateRemoved), "Continue local deletion with original path/digest/ownership and registration checks; no authentication. General integrity warnings do not waive these checks."}}
		if _, ok := keys[d.KeyID]; !ok {
			task.Priority = 0
			task.ResumeID = ""
			task.Title = "Requires review: deletion key missing"
		}
		tasks = append(tasks, task)
	}
	configTasks := map[string]*Task{}
	configTask := func(path string) *Task {
		if configTasks[path] == nil {
			configTasks[path] = &Task{ID: "config:" + path, Title: "Requires review: configuration · " + path, SourceIDs: []string{path}, Priority: 0, Reasons: []string{"No standalone config/Include resume operation. Inspect diagnostics and preserve files/backups. Register or Add with configuration requested repeats registration/reconciliation when eligible; it is not config-only recovery. A binding or empty integrity warning does not establish Include activation."}}
		}
		return configTasks[path]
	}
	for _, c := range document.Configs {
		if c.PendingSHA256 != "" {
			task := configTask(c.Path)
			task.Reasons = append(task.Reasons, "Pending digest: "+c.PendingSHA256+"; previous saved digest: "+c.SHA256)
		}
	}
	for _, e := range document.ConfigEdits {
		if !e.Complete {
			task := configTask(e.Path)
			task.Reasons = append(task.Reasons, "Incomplete Include edit; before: "+e.BeforeSHA256+"; after: "+e.AfterSHA256+"; backup: "+e.BackupPath)
		}
	}
	// The dedicated file and its Include are shared by every saved binding.
	// Only one valid binding and one conventional saved path identify a parent.
	var parent Task
	bound := 0
	for _, host := range document.Hosts {
		if host.ConfigKeyID != "" {
			bound++
			parent = safeParents[host.ID+"/"+host.ConfigKeyID]
		}
	}
	managedPath, includePath := "", ""
	if bound == 1 && parent.RegistrationID != "" && len(document.Configs) == 1 {
		path := document.Configs[0].Path
		if filepath.Base(path) == "config" && filepath.Base(filepath.Dir(path)) == "ssh-keyflow" && filepath.Base(filepath.Dir(filepath.Dir(path))) == ".ssh" {
			managedPath = path
			includePath = filepath.Join(filepath.Dir(filepath.Dir(path)), "config")
		}
	}
	paths := make([]string, 0, len(configTasks))
	for path := range configTasks {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		config := configTasks[path]
		if managedPath != "" && (path == managedPath || path == includePath) {
			index := -1
			for i, task := range tasks {
				if task.ID == parent.ID {
					index = i
					break
				}
			}
			if index < 0 {
				parent.Title = "Requires review: configuration · " + hosts[parent.HostID].Alias + " / " + keys[parent.KeyID].Name
				tasks = append(tasks, parent)
				index = len(tasks) - 1
			}
			task := &tasks[index]
			if task.Priority != 0 {
				task.Reasons = append(task.Reasons, task.Title)
				task.Title = "Requires review: configuration · " + hosts[parent.HostID].Alias + " / " + keys[parent.KeyID].Name
			}
			task.Priority = 0
			task.SourceIDs = uniqueSorted(append(task.SourceIDs, config.SourceIDs...))
			task.Reasons = uniqueSorted(append(task.Reasons, config.Reasons...))
		} else {
			sort.Strings(config.Reasons)
			tasks = append(tasks, *config)
		}
	}
	if view.Review != "" {
		tasks = append(tasks, Task{ID: "state.json", Title: "Requires review: saved-state integrity", Priority: 0, SourceIDs: []string{"state.json"}, Reasons: []string{view.Review, "Inspect files and saved evidence. Dedicated pending deletion recovery remains available with its existing guards."}})
	}
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].Priority != tasks[j].Priority {
			return tasks[i].Priority < tasks[j].Priority
		}
		return tasks[i].ID < tasks[j].ID
	})
	return tasks
}

func uniqueSorted(values []string) []string {
	sort.Strings(values)
	result := []string{}
	for _, v := range values {
		if v != "" && (len(result) == 0 || result[len(result)-1] != v) {
			result = append(result, v)
		}
	}
	return result
}

func (m model) taskEntry(t Task) entry {
	return entry{kind: "task", id: t.ID, title: t.Title, description: strings.Join(t.SourceIDs, " · "), searchText: strings.Join(t.Reasons, " ") + " " + m.hostLabel(t.HostID) + " " + m.keyLabel(t.KeyID), readOnly: t.ResumeID == ""}
}
func (m model) taskDetails(t Task) string {
	action := "Read-only review; no supported saved continuation"
	if t.ResumeID != "" {
		action = "s: Continue saved operation [" + t.ResumeID + "]"
	}
	return t.Title + "\n" + action + "\nSource IDs: " + strings.Join(t.SourceIDs, ", ") + "\n\n" + strings.Join(t.Reasons, "\n\n") + "\n\nHistory remains read-only. No Mark done or automatic remote retry."
}
