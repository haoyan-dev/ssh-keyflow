package tui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"ssh-keyflow/internal/state"
)

// These are production model views reached through public events, not copies of
// the browser concept. Exercise both ends of every document and resize origins.
func TestGroupedPanelPagesFitSupportedFrames(t *testing.T) {
	home, saved := navigationHome(t)
	journeys := map[string][]tea.Msg{
		"destinations": nil, "keys": {press(tea.KeyTab)},
		"registrations": {press(tea.KeyTab), press(tea.KeyTab)},
		"tasks":         {press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab)},
		"destination":   {press(tea.KeyEnter)},
		"key":           {press(tea.KeyTab), press(tea.KeyEnter)},
		"registration":  {press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyEnter)},
		"task":          {press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyEnter)},
		"key-picker":    {press('r')},
		"host-picker":   {press(tea.KeyTab), press('r')},
		"create":        {press('c')},
		"import":        {press('i')},
		"add":           {press('a')},
		"add-key":       append(destinationDraft("linux"), press(tea.KeyTab), press(tea.KeyEnter)),
		"add-access":    append(destinationDraft("linux"), press(tea.KeyTab), press(tea.KeyEnter), press(tea.KeyTab), press(tea.KeyEnter), press(tea.KeyEnter), press(tea.KeyEnter)),
		"register":      {press('r'), press(tea.KeyEnter)},
		"unregister":    {press('u')},
		"validation":    {press('c'), reviewKey()},
		"review":        {press('r'), press(tea.KeyEnter), reviewKey()},
		"discard":       {press('c'), text("Work laptop"), interruptKey()},
		"technical":     {press(tea.KeyEnter), press('v')},
		"key-technical": {press(tea.KeyTab), press(tea.KeyEnter), press('v')},
		"help":          {press('?')},
		"history":       {press('h')},
		"discovery":     {press('g')},
		"no-matches":    {press('/'), text("no-matching-record"), press(tea.KeyEnter)},
		"picker-empty":  {press('r'), press('/'), text("no-matching-key"), press(tea.KeyEnter)},
		"long-field":    {press('i'), text("Work laptop"), press(tea.KeyTab), tea.PasteMsg{Content: "C:/keys/" + strings.Repeat("長い-directory/", 25) + "key-tail.pub"}},
	}
	for name, journey := range journeys {
		t.Run(name, func(t *testing.T) {
			verifyGroupedPage(t, home, name, journey)
		})
	}
	empty := t.TempDir()
	for tabs := 0; tabs < 4; tabs++ {
		var journey []tea.Msg
		for range tabs {
			journey = append(journey, press(tea.KeyTab))
		}
		verifyGroupedPage(t, empty, []string{"empty-destinations", "empty-keys", "empty-registrations", "empty-tasks"}[tabs], journey)
	}
	saved.Registrations[0].Stage = "pending"
	saveNavigationState(t, home, saved)
	verifyGroupedPage(t, home, "resume", []tea.Msg{press('s')})
	saved.Registrations = append(saved.Registrations, state.Registration{
		ID: "registration-extra", HostID: saved.Hosts[0].ID, KeyID: saved.Keys[1].ID,
		Operation: "register", Stage: "complete", Status: "registered", ConnectionCheck: "verified",
	})
	saveNavigationState(t, home, saved)
	verifyGroupedPage(t, home, "mixed-pairs", []tea.Msg{press(tea.KeyEnter)})
	verifyGroupedPage(t, home, "registration-picker", []tea.Msg{press('u')})
	saved.Registrations[len(saved.Registrations)-1].Stage = "pending"
	saveNavigationState(t, home, saved)
	verifyGroupedPage(t, home, "resume-picker", []tea.Msg{press('s')})
	for i := range saved.Registrations {
		saved.Registrations[i].Operation, saved.Registrations[i].Status, saved.Registrations[i].Stage = "unregister", "removed", "complete"
	}
	saveNavigationState(t, home, saved)
	verifyGroupedPage(t, home, "delete", []tea.Msg{press(tea.KeyTab), press('d')})
	verifyGroupedPage(t, home, "delete-review", []tea.Msg{press(tea.KeyTab), press('d'), reviewKey()})
}

func verifyGroupedPage(t *testing.T, home, name string, journey []tea.Msg) {
	t.Helper()
	for _, size := range [][2]int{{80, 24}, {60, 18}, {32, 12}} {
		for _, end := range []bool{false, true} {
			events := append([]tea.Msg{}, journey...)
			if end {
				// Forms reserve Home/End for editing; PgDown scrolls their full copy.
				for range 30 {
					events = append(events, press(tea.KeyPgDown))
				}
			}
			screen := runSizedScreen(t, home, size[0], size[1], events...)
			if lipgloss.Width(screen) > size[0] || lipgloss.Height(screen) > size[1] {
				t.Fatalf("%s at %v (end=%v) overflow:\n%s", name, size, end, screen)
			}
			// A form may initially scroll its active field into view.
			if !end && !strings.ContainsAny(screen, "┌│") {
				t.Fatalf("%s did not use the selected panels:\n%s", name, screen)
			}
			captureGroupedScreen(t, name, size, end, screen)
		}
	}
}

func captureGroupedScreen(t *testing.T, name string, size [2]int, end bool, screen string) {
	t.Helper()
	root := os.Getenv("KEYFLOW_CAPTURE")
	if root == "" {
		return
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	suffix := "-top"
	if end {
		suffix = "-end"
	}
	path := filepath.Join(root, name+"-"+sizeName(size)+suffix)
	if err := os.WriteFile(path+".txt", []byte(ansi.Strip(screen)), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".ansi", []byte(screen), 0600); err != nil {
		t.Fatal(err)
	}
}
