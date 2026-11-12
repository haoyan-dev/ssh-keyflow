package tui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"ssh-keyflow/internal/state"
)

func TestPinKeysTargetOnlyDestinations(t *testing.T) {
	for _, key := range []rune{'t', 'T'} {
		m := fixture(t)
		next, command := m.Update(press(key))
		if m = next.(model); command != nil || m.running || !strings.Contains(m.status, "Select a destination") {
			t.Fatal("pin handed off for a key", string(key))
		}
		m.tab = 1
		m = update(m, press(tea.KeyEnter))
		if !strings.Contains(m.document, "Host keys: not pinned") {
			t.Fatal("details lack pin status", m.document)
		}
		next, command = m.Update(press(key))
		m = next.(model)
		if command == nil || !m.running || m.request.Host != "host-one" {
			t.Fatal("pin did not hand off", string(key))
		}
		if key == 't' && (m.request.Operation != "pin-host" || !m.request.Replace) {
			t.Fatalf("pin request %+v", m.request)
		}
		if key == 'T' && m.request.Operation != "unpin-host" {
			t.Fatalf("unpin request %+v", m.request)
		}
		m.signals.finish()
	}
}

func TestPinHandoffReadsPasteAndConfirmationFromNativeInput(t *testing.T) {
	hostKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB"
	for _, answer := range []string{"yes\n", "no\n"} {
		m := fixture(t)
		exec := &execution{ctx: context.Background(), options: m.options}
		exec.request.Operation, exec.request.Host, exec.request.Replace = "pin-host", "host-one", true
		exec.SetStdin(strings.NewReader(hostKey + " root@example\n\n" + answer))
		err := exec.Run()
		m = update(m, exec.complete(err))
		if m.page != "result" || !strings.Contains(m.document, "Pin host keys") || !strings.Contains(m.document, "SHA256:") {
			t.Fatal("result screen", m.document)
		}
		entries, loadErr := state.LoadPins(m.options.Home)
		if loadErr != nil || (answer == "yes\n") != (len(entries) == 1) {
			t.Fatal(answer, entries, loadErr, err)
		}
		if answer == "yes\n" {
			m.tab = 1
			m = update(m, press(tea.KeyEscape), press(tea.KeyEnter))
			if !strings.Contains(m.document, "Host keys: pinned (1)") {
				t.Fatal("details did not show pinned status", m.document)
			}
		}
	}
}
