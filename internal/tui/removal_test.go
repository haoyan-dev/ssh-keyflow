package tui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestRemoveTargetsVisibleKeyOrDestination(t *testing.T) {
	for _, tab := range []int{0, 1} {
		for _, details := range []bool{false, true} {
			m := fixture(t)
			m.tab = tab
			if details {
				m = update(m, press(tea.KeyEnter))
			}
			next, command := m.Update(press('d'))
			m = next.(model)
			if command == nil || !m.running {
				t.Fatal("remove did not hand off to confirmation")
			}
			if tab == 0 && (m.request.Operation != "remove-key" || m.request.Key != "key-one" || m.request.Host != "") {
				t.Fatalf("wrong key target: %+v", m.request)
			}
			if tab == 1 && (m.request.Operation != "remove-host" || m.request.Host != "host-one" || m.request.Key != "") {
				t.Fatalf("wrong destination target: %+v", m.request)
			}
			m.signals.finish()
		}
	}
}

func TestDisconnectPickerContainsOnlyExistingAssociations(t *testing.T) {
	for _, tab := range []int{0, 1} {
		m := fixture(t)
		m.tab = tab
		extra := m.view.State.Keys[0]
		extra.ID = "unassociated-key"
		m.view.State.Keys = append(m.view.State.Keys, extra)
		host := m.view.State.Hosts[0]
		host.ID = "unassociated-host"
		m.view.State.Hosts = append(m.view.State.Hosts, host)
		m = update(m, press('u'))
		if m.page != "picker" || len(m.choices) != 1 {
			t.Fatalf("unlink offered unrelated targets: %+v", m.choices)
		}
		next, command := m.Update(press(tea.KeyEnter))
		m = next.(model)
		if command == nil || !m.running || m.request.Operation != "dissociate" || m.request.Host != "host-one" || m.request.Key != "key-one" {
			t.Fatalf("wrong association: %+v", m.request)
		}
		m.signals.finish()
	}
}

func TestRemovalNativeConfirmationAndReload(t *testing.T) {
	for _, operation := range []string{"remove-key", "remove-host", "dissociate"} {
		for _, answer := range []string{"yes\n", "no\n"} {
			t.Run(operation+"/"+strings.TrimSpace(answer), func(t *testing.T) {
				m := fixture(t)
				exec := &execution{ctx: context.Background(), options: m.options}
				exec.request.Operation, exec.request.Key, exec.request.Host = operation, "key-one", "host-one"
				exec.SetStdin(strings.NewReader(answer))
				err := exec.Run()
				m = update(m, exec.complete(err))
				if m.page != "result" || m.running || m.stale {
					t.Fatal("operation did not reload into result screen")
				}
				approved := answer == "yes\n"
				if approved != (err == nil) {
					t.Fatal("confirmation result mismatch", err)
				}
				wantKeys, wantHosts, wantLinks := 1, 1, 1
				if approved {
					wantLinks = 0
					if operation == "remove-key" {
						wantKeys = 0
					}
					if operation == "remove-host" {
						wantHosts = 0
					}
				}
				if len(m.view.State.Keys) != wantKeys || len(m.view.State.Hosts) != wantHosts || len(m.view.State.Associations) != wantLinks {
					t.Fatal("result screen did not reflect saved state")
				}
				m = update(m, press(tea.KeyEscape))
				if m.page != "list" {
					t.Fatal("result did not return to list")
				}
				m = update(m, tea.WindowSizeMsg{Width: 32, Height: 12})
				assertFits(t, m)
			})
		}
	}
}

func TestRemovalRespectsEmptySelectionStaleStateAndPickerCancellation(t *testing.T) {
	m := emptyModel(t)
	for _, key := range []rune{'d', 'u'} {
		next, command := m.Update(press(key))
		m = next.(model)
		if command != nil || m.running || m.page != "list" {
			t.Fatal("empty selection started a mutation")
		}
	}
	m = fixture(t)
	m = update(m, press('u'), press(tea.KeyEscape))
	if m.running || m.page != "list" {
		t.Fatal("picker cancellation started a mutation")
	}
	m.view.State.Associations = nil
	m = update(m, press('u'))
	if m.page != "list" || m.status != "No local associations to disconnect" {
		t.Fatal("empty association picker opened")
	}
	m.stale = true
	for _, key := range []rune{'d', 'u'} {
		next, command := m.Update(press(key))
		m = next.(model)
		if command != nil || m.running {
			t.Fatal("stale state started a mutation")
		}
	}
}
