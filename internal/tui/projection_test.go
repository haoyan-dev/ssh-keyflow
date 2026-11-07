package tui_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"slices"
	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/state"
	"ssh-keyflow/internal/tui"
	"strings"
	"testing"
)

func projectionView() app.View {
	return app.View{State: state.State{
		Hosts:         []state.Host{{ID: "host-1", Alias: "build", Hostname: "example.test", User: "alice", Port: 22}},
		Keys:          []state.Key{{ID: "key-1", Name: "laptop", PrivatePath: "private", Fingerprint: "SHA256:fixture"}},
		Registrations: []state.Registration{{ID: "registration-1", HostID: "host-1", KeyID: "key-1", Operation: "register", Status: "unknown", Stage: "pending", ConnectionCheck: "unverified"}},
		Onboardings:   []state.Onboarding{{ID: "onboarding-1", HostID: "host-1", KeyID: "key-1", Stage: "planned"}},
	}}
}

func TestProjectionAC09Precedence(t *testing.T) {
	cases := []struct {
		name         string
		prepare      func(*app.View)
		ids, resumes []string
		priorities   []int
		contains     string
	}{
		{"linked-register", func(v *app.View) {}, []string{"onboarding-1"}, []string{"registration-1"}, []int{1}, "Outcome unknown"},
		{"creation-before-current-registration", func(v *app.View) {
			v.State.Creations = []state.Creation{{ID: "creation-1", KeyID: "key-1", Stage: "pending"}}
			v.State.Onboardings[0].CreationID = "creation-1"
			v.State.Onboardings[0].Stage = "key_pending"
		}, []string{"onboarding-1"}, []string{"onboarding-1"}, []int{1}, "then registration"},
		{"completed-creation-prefers-current-registration", func(v *app.View) {
			v.State.Creations = []state.Creation{{ID: "creation-1", KeyID: "key-1", Stage: "complete"}}
			v.State.Onboardings[0].CreationID = "creation-1"
		}, []string{"onboarding-1"}, []string{"registration-1"}, []int{1}, "reconcile remote"},
		{"creation-alone-does-not-finish-onboarding", func(v *app.View) {
			v.State.Registrations = nil
			v.State.Creations = []state.Creation{{ID: "creation-1", KeyID: "key-1", Stage: "complete"}}
			v.State.Onboardings[0].CreationID = "creation-1"
		}, []string{"onboarding-1"}, []string{"onboarding-1"}, []int{1}, "completed local creation alone"},
		{"unregister-wins", func(v *app.View) { v.State.Registrations[0].Operation = "unregister" }, []string{"onboarding-1"}, []string{"registration-1"}, []int{1}, "supersedes onboarding"},
		{"unregister-failed", func(v *app.View) {
			r := &v.State.Registrations[0]
			r.Operation = "unregister"
			r.Status = "failed"
			r.Stage = "failed"
		}, []string{"onboarding-1"}, []string{"registration-1"}, []int{1}, "Continue unregister"},
		{"removed-complete-no-re-registration", func(v *app.View) {
			r := &v.State.Registrations[0]
			r.Operation = "unregister"
			r.Status = "removed"
			r.Stage = "complete"
		}, nil, nil, nil, ""},
		{"failed-private-registration", func(v *app.View) { r := &v.State.Registrations[0]; r.Status = "failed"; r.Stage = "failed" }, []string{"onboarding-1"}, []string{"registration-1"}, []int{1}, "Continue registration"},
		{"manual-stage-is-not-proof-of-manual-outcome", func(v *app.View) { v.State.Onboardings[0].Stage = "manual" }, []string{"onboarding-1"}, []string{"registration-1"}, []int{1}, "Manual versus interrupted outcome is not recorded"},
		{"failed-check-private", func(v *app.View) {
			r := &v.State.Registrations[0]
			r.Status = "registered"
			r.Stage = "complete"
			r.ConnectionCheck = "failed"
		}, []string{"onboarding-1"}, []string{"registration-1"}, []int{3}, "Continue registration includes reconciliation/verification"},
		{"unknown-check-private", func(v *app.View) {
			r := &v.State.Registrations[0]
			r.Status = "registered"
			r.Stage = "complete"
			r.ConnectionCheck = "unknown"
		}, []string{"onboarding-1"}, []string{"registration-1"}, []int{3}, "specified-private-key"},
		{"unverified-check-private", func(v *app.View) { r := &v.State.Registrations[0]; r.Status = "registered"; r.Stage = "complete" }, []string{"onboarding-1"}, []string{"registration-1"}, []int{3}, "no Verify-only"},
		{"public-only-complete", func(v *app.View) {
			r := &v.State.Registrations[0]
			r.Status = "registered"
			r.Stage = "complete"
			v.State.Keys[0].PrivatePath = ""
		}, nil, nil, nil, ""},
		{"public-only-pending-still-follows-up", func(v *app.View) { v.State.Keys[0].PrivatePath = "" }, []string{"onboarding-1"}, []string{"registration-1"}, []int{1}, "Outcome unknown"},
		{"verified-suppresses-stale-onboarding", func(v *app.View) {
			r := &v.State.Registrations[0]
			r.Status = "registered"
			r.Stage = "complete"
			r.ConnectionCheck = "verified"
			v.State.Onboardings[0].Stage = "manual"
		}, nil, nil, nil, ""},
		{"standalone-creation", func(v *app.View) {
			v.State.Onboardings = nil
			v.State.Registrations = nil
			v.State.Creations = []state.Creation{{ID: "creation-1", KeyID: "future-key", Name: "work", Stage: "planned"}}
		}, []string{"creation-1"}, []string{"creation-1"}, []int{1}, "local key only"},
		{"complete-creation-history-only", func(v *app.View) {
			v.State.Onboardings = nil
			v.State.Registrations = nil
			v.State.Creations = []state.Creation{{ID: "creation-1", KeyID: "key-1", Stage: "complete"}}
		}, nil, nil, nil, ""},
		{"missing-host", func(v *app.View) { v.State.Hosts = nil }, []string{"onboarding-1"}, []string{""}, []int{0}, "Missing/contradictory"},
		{"missing-key", func(v *app.View) { v.State.Keys = nil }, []string{"onboarding-1"}, []string{""}, []int{0}, "no continuation inferred"},
		{"missing-creation", func(v *app.View) { v.State.Onboardings[0].CreationID = "missing" }, []string{"onboarding-1"}, []string{""}, []int{0}, "no continuation inferred"},
		{"contradictory-creation-key", func(v *app.View) {
			v.State.Onboardings[0].CreationID = "creation-1"
			v.State.Creations = []state.Creation{{ID: "creation-1", KeyID: "other", Stage: "complete"}}
		}, []string{"onboarding-1"}, []string{""}, []int{0}, "no continuation inferred"},
		{"contradictory-current-registrations", func(v *app.View) {
			v.State.Registrations = append(v.State.Registrations, state.Registration{ID: "registration-2", HostID: "host-1", KeyID: "key-1", Operation: "unregister"})
		}, []string{"onboarding-1"}, []string{""}, []int{0}, "no continuation inferred"},
		{"pending-deletion-independent", func(v *app.View) {
			v.State.Deletions = []state.LocalDeletion{{ID: "deletion-1", KeyID: "key-1", Stage: "pending", PublicRemoved: true}}
		}, []string{"deletion-1", "onboarding-1"}, []string{"deletion-1", "registration-1"}, []int{1, 1}, "original path/digest"},
		{"review-does-not-disable-deletion", func(v *app.View) {
			v.Review = "external digest conflict"
			v.State.Registrations = nil
			v.State.Onboardings = nil
			v.State.Deletions = []state.LocalDeletion{{ID: "deletion-1", KeyID: "key-1", Stage: "pending"}}
		}, []string{"state.json", "deletion-1"}, []string{"", "deletion-1"}, []int{0, 1}, "existing guards"},
		{"config-path-grouping", func(v *app.View) {
			v.State.Onboardings = nil
			v.State.Registrations = nil
			v.State.Configs = []state.Config{{Path: "/config", PendingSHA256: "pending"}}
			v.State.ConfigEdits = []state.ConfigEdit{{Path: "/config", BeforeSHA256: "before", AfterSHA256: "after", BackupPath: "/backup"}}
		}, []string{"config:/config"}, []string{""}, []int{0}, "No standalone config/Include resume"},
		{"config-paths-stay-independent", func(v *app.View) {
			v.State.Onboardings = nil
			v.State.Registrations = nil
			v.State.Configs = []state.Config{{Path: "/z", PendingSHA256: "pending"}, {Path: "/a", PendingSHA256: "pending"}}
		}, []string{"config:/a", "config:/z"}, []string{"", ""}, []int{0, 0}, "not config-only recovery"},
		{"completed-config-is-not-task", func(v *app.View) {
			v.State.Onboardings = nil
			v.State.Registrations = nil
			v.State.Configs = []state.Config{{Path: "/config", SHA256: "saved"}}
			v.State.ConfigEdits = []state.ConfigEdit{{Path: "/config", Complete: true}}
		}, nil, nil, nil, ""},
		{"tombstone-complete-deletion-history-only", func(v *app.View) {
			v.State.Registrations[0].Operation = "unregister"
			v.State.Registrations[0].Status = "removed"
			v.State.Registrations[0].Stage = "complete"
			v.State.DeletedKeys = []state.DeletedKey{{Key: v.State.Keys[0], DeletionID: "deletion-1"}}
			v.State.Keys = nil
			v.State.Deletions = []state.LocalDeletion{{ID: "deletion-1", KeyID: "key-1", Stage: "complete"}}
			v.State.History = []state.RegistrationEvent{{RegistrationID: "registration-1", Status: "registered", ConnectionCheck: "failed"}}
		}, nil, nil, nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			view := projectionView()
			tc.prepare(&view)
			before, _ := json.Marshal(view)
			tasks := tui.ProjectTasks(view)
			if len(tasks) != len(tc.ids) {
				t.Fatalf("expected %v, got %+v", tc.ids, tasks)
			}
			for i, task := range tasks {
				if task.ID != tc.ids[i] || task.ResumeID != tc.resumes[i] || task.Priority != tc.priorities[i] {
					t.Fatalf("wrong precedence/identity: %+v", tasks)
				}
			}
			if tc.contains != "" {
				data, _ := json.Marshal(tasks)
				if !strings.Contains(string(data), tc.contains) {
					t.Fatalf("missing %q in %s", tc.contains, data)
				}
			}
			after, _ := json.Marshal(view)
			if string(before) != string(after) {
				t.Fatal("projection mutated its input")
			}
			slices.Reverse(view.State.Onboardings)
			slices.Reverse(view.State.Registrations)
			slices.Reverse(view.State.Creations)
			slices.Reverse(view.State.Configs)
			slices.Reverse(view.State.ConfigEdits)
			if again := tui.ProjectTasks(view); !reflect.DeepEqual(tasks, again) {
				t.Fatalf("projection depends on input order:\n%+v\n%+v", tasks, again)
			}
		})
	}
}

func TestProjectionMultipleOnboardingReferencesKeepStableSources(t *testing.T) {
	view := projectionView()
	view.State.Onboardings = append(view.State.Onboardings, state.Onboarding{ID: "onboarding-2", HostID: "host-1", KeyID: "key-1", Stage: "planned"})
	tasks := tui.ProjectTasks(view)
	if len(tasks) != 1 || !slices.Equal(tasks[0].SourceIDs, []string{"onboarding-1", "onboarding-2", "registration-1"}) {
		t.Fatalf("lost grouped source identity: %+v", tasks)
	}
	beforeID := tasks[0].ID
	view.State.Registrations[0].Stage = "complete"
	view.State.Registrations[0].Status = "registered"
	view.State.Registrations[0].ConnectionCheck = "failed"
	tasks = tui.ProjectTasks(view)
	if len(tasks) != 1 || tasks[0].ID != beforeID || tasks[0].ResumeID != "registration-1" || tasks[0].Priority != 3 {
		t.Fatalf("identity changed with continuation/reason: %+v", tasks)
	}
}

func TestProjectionIncompleteCreationUsesReferencingOnboarding(t *testing.T) {
	view := projectionView()
	view.State.Creations = []state.Creation{{ID: "creation-1", KeyID: "key-1", Stage: "pending"}}
	view.State.Onboardings = append(view.State.Onboardings, state.Onboarding{ID: "onboarding-z", HostID: "host-1", KeyID: "key-1", CreationID: "creation-1", Stage: "key_pending"})
	tasks := tui.ProjectTasks(view)
	if len(tasks) != 1 || tasks[0].ID != "onboarding-1" || tasks[0].ResumeID != "onboarding-z" {
		t.Fatalf("did not select actual creation owner: %+v", tasks)
	}
	view.State.Onboardings[0].CreationID = "creation-1"
	view.State.Onboardings[0].Stage = "key_pending"
	tasks = tui.ProjectTasks(view)
	if len(tasks) != 1 || tasks[0].Priority != 0 || tasks[0].ResumeID != "" {
		t.Fatalf("ambiguous incomplete onboarding chose continuation: %+v", tasks)
	}
}

func TestProjectionConfigurationUsesOnlyUnambiguousSavedParent(t *testing.T) {
	managed := filepath.FromSlash("/home/.ssh/ssh-keyflow/config")
	main := filepath.FromSlash("/home/.ssh/config")
	for _, scenario := range []string{"pending-registration", "public-complete", "verified-complete", "shared", "missing-binding", "unrelated-path", "unregister"} {
		t.Run(scenario, func(t *testing.T) {
			view := projectionView()
			view.State.Hosts[0].ConfigKeyID = "key-1"
			view.State.Configs = []state.Config{{Path: managed, PendingSHA256: "pending"}}
			view.State.ConfigEdits = []state.ConfigEdit{{Path: main, BeforeSHA256: "before", AfterSHA256: "after", BackupPath: "backup"}}
			resume := "registration-1"
			switch scenario {
			case "public-complete", "verified-complete":
				view.State.Registrations[0].Status = "registered"
				view.State.Registrations[0].Stage = "complete"
				view.State.Registrations[0].ConnectionCheck = "verified"
				if scenario == "public-complete" {
					view.State.Keys[0].PrivatePath = ""
				}
				resume = ""
			case "shared":
				view.State.Hosts = append(view.State.Hosts, state.Host{ID: "host-2", ConfigKeyID: "key-1"})
				view.State.Registrations = append(view.State.Registrations, state.Registration{ID: "registration-2", HostID: "host-2", KeyID: "key-1", Operation: "register", Status: "registered", Stage: "complete", ConnectionCheck: "verified"})
			case "missing-binding":
				view.State.Hosts[0].ConfigKeyID = "missing"
			case "unrelated-path":
				view.State.ConfigEdits[0].Path = filepath.FromSlash("/elsewhere/config")
			case "unregister":
				view.State.Registrations[0].Operation = "unregister"
			}
			before, _ := json.Marshal(view)
			tasks := tui.ProjectTasks(view)
			if scenario == "shared" || scenario == "missing-binding" {
				if len(tasks) != 3 {
					t.Fatalf("ambiguous config must stay by path: %+v", tasks)
				}
				for _, task := range tasks {
					if strings.HasPrefix(task.ID, "config:") && task.ResumeID != "" {
						t.Fatal("invented config resume")
					}
				}
			} else {
				expected := 1
				if scenario == "unrelated-path" {
					expected = 2
				}
				if len(tasks) != expected {
					t.Fatalf("config duplicated parent work: %+v", tasks)
				}
				var parent tui.Task
				for _, task := range tasks {
					if task.ID == "onboarding-1" {
						parent = task
					}
				}
				if parent.Priority != 0 || parent.ResumeID != resume || !slices.Contains(parent.SourceIDs, managed) || !strings.Contains(strings.Join(parent.Reasons, " "), "Pending digest") {
					t.Fatalf("lost parent recovery or config evidence: %+v", parent)
				}
				if scenario != "unrelated-path" && (!slices.Contains(parent.SourceIDs, main) || !strings.Contains(strings.Join(parent.Reasons, " "), "Incomplete Include edit")) {
					t.Fatalf("lost Include substep: %+v", parent)
				}
			}
			after, _ := json.Marshal(view)
			if string(before) != string(after) {
				t.Fatal("projection wrote input")
			}
			slices.Reverse(view.State.Hosts)
			slices.Reverse(view.State.Registrations)
			if !reflect.DeepEqual(tasks, tui.ProjectTasks(view)) {
				t.Fatal("config parent depends on order")
			}
		})
	}
}
