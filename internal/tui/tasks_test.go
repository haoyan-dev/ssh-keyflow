package tui_test

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/state"
	"ssh-keyflow/internal/tui"
	"strings"
	"testing"
)

func TestTasksGroupSavedOnboardingAndContinueExactRegistration(t *testing.T) {
	home, view := handoffHome(t)
	screen := runScreen(t, home, press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab))
	if !strings.Contains(screen, "Outcome unknown") || !strings.Contains(screen, "fixture") {
		t.Fatalf("saved onboarding follow-up is missing:\n%s", screen)
	}
	review := runScreen(t, home, press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyEnter), press('s'), reviewKey())
	for _, want := range []string{"Review resume", view.State.Registrations[0].ID} {
		if !strings.Contains(review, want) {
			t.Fatalf("task continuation missing %q:\n%s", want, review)
		}
	}
	after, err := app.Snapshot(app.Options{Home: home})
	if err != nil || len(after.State.Registrations) != len(view.State.Registrations) {
		t.Fatal("read-only task navigation changed state", err)
	}
}

func TestResultManualHasHonestSessionAndRestartFollowup(t *testing.T) {
	home, view := handoffHome(t)
	document := view.State
	document.Onboardings = nil
	document.Registrations = []state.Registration{}
	saveNavigationState(t, home, document)
	native := &handoffIO{answer: strings.NewReader("yes\n")}
	session := startHandoff(t, app.Options{Home: home, Input: native, Output: native, Runner: publicRunner{}}, nil)
	session.send(press('r'), press(tea.KeyEnter), reviewKey(), press(tea.KeyEnter))
	screen := session.returned(t)
	if !strings.Contains(screen, "Manual action required") {
		t.Fatalf("manual nil-error has no required primary summary:\n%s", screen)
	}
	session.send(press(tea.KeyEnd))
	if !strings.Contains(session.screen(t), "ssh-ed25519") {
		t.Fatal("manual instructions unavailable")
	}
	session.send(press(tea.KeyHome), press('t'))
	if !strings.Contains(session.screen(t), "No saved recovery task") {
		t.Fatal("manual result fabricated durable follow-up")
	}
	restart := runScreen(t, home, press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab))
	if !strings.Contains(restart, "No recorded follow-up") {
		t.Fatalf("restart fabricated manual Task:\n%s", restart)
	}
}

func TestProjectionContradictoryOnboardingLinksNeverGuessResume(t *testing.T) {
	document := state.State{
		Hosts:       []state.Host{{ID: "host", Alias: "build", Hostname: "example.test", User: "me", Port: 22}},
		Keys:        []state.Key{{ID: "key", Name: "laptop", PrivatePath: "private"}},
		Creations:   []state.Creation{{ID: "creation-old", KeyID: "key", Stage: "complete"}, {ID: "creation-new", KeyID: "key", Stage: "pending"}},
		Onboardings: []state.Onboarding{{ID: "onboarding-a", HostID: "host", KeyID: "key", CreationID: "creation-old", Stage: "planned"}, {ID: "onboarding-z", HostID: "host", KeyID: "key", CreationID: "creation-new", Stage: "key_pending"}},
	}
	tasks := tui.ProjectTasks(app.View{State: document})
	if len(tasks) != 1 || tasks[0].Priority != 0 || tasks[0].ResumeID != "" {
		t.Fatalf("contradictory creations guessed a continuation: %+v", tasks)
	}
}

func TestTasksSearchAndCompactRecoveryControls(t *testing.T) {
	home, view := handoffHome(t)
	for _, size := range [][2]int{{80, 24}, {60, 18}, {32, 12}} {
		screen := runSizedScreen(t, home, size[0], size[1], press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab), press('/'), text(view.State.Registrations[0].ID), press(tea.KeyEnter), press(tea.KeyEnter))
		if !strings.Contains(screen, "s Continue") || !strings.Contains(screen, "esc back") {
			t.Fatalf("task recovery controls missing at %v:\n%s", size, screen)
		}
		for _, line := range strings.Split(screen, "\n") {
			if ansi.StringWidth(line) > size[0] {
				t.Fatalf("task width exceeds %v: %q", size, line)
			}
		}
		if lipgloss.Height(screen) > size[1] {
			t.Fatalf("task height exceeds %v:\n%s", size, screen)
		}
	}
}
