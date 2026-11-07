package tui_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"ssh-keyflow/internal/app"
)

// Reuse the registration fixture; only the external revocation read is added.
// An already-absent result must never send a remote removal command.
type journeyRunner struct{ *resultRunner }

func (r journeyRunner) Run(ctx context.Context, c app.Command) error {
	if c.Name == "ssh" {
		payload := c.Args[len(c.Args)-1]
		switch {
		case strings.Contains(payload, "KEYFLOW_ALTERNATE"):
			r.calls.Add(1)
			fmt.Fprintln(c.Stdout, "KEYFLOW_TARGET:/home/me/.ssh/authorized_keys")
			return nil
		case strings.Contains(payload, "KEYFLOW_REVOCATION_READ"):
			r.calls.Add(1)
			fmt.Fprintln(c.Stdout, "KEYFLOW_ABSENT")
			return nil
		}
	}
	return r.resultRunner.Run(ctx, c)
}

func TestJourneyUnregisterReconciliationUnlocksDeletionWithoutRevivingOnboarding(t *testing.T) {
	home, external := resultHome(t, false)
	runner := journeyRunner{external}
	before, err := app.Snapshot(app.Options{Home: home})
	if err != nil {
		t.Fatal(err)
	}
	registration := before.State.Registrations[0]
	key := before.State.Keys[0]
	native := &handoffIO{answer: strings.NewReader("yes\nyes\n")}
	session := startHandoff(t, app.Options{Home: home, Input: native, Output: native, Runner: runner}, nil)

	session.send(press(tea.KeyTab), press('d'))
	if screen := session.screen(t); !strings.Contains(screen, "Unregister outstanding recorded registrations first") {
		t.Fatalf("pending registration did not block deletion:\n%s", screen)
	}
	calls := runner.calls.Load()
	session.send(press('u'), press(tea.KeyTab), press(tea.KeyRight), reviewKey())
	review := session.screen(t)
	if !strings.Contains(review, registration.ID) || !strings.Contains(review, "Review unregister") {
		t.Fatalf("unregister lost the blocked key's registration:\n%s", review)
	}
	session.send(press('?'), press(tea.KeyEscape), press(tea.KeyTab), press(tea.KeyEnter), reviewKey())
	if screen := session.screen(t); !strings.Contains(screen, "Review unregister") || runner.calls.Load() != calls {
		t.Fatalf("review/help/edit executed or lost unregister intent:\n%s", screen)
	}
	session.send(press(tea.KeyEnter))
	if screen := session.returned(t); !strings.Contains(screen, "Completed; confirmed applicable work recorded") {
		t.Fatalf("read-only absence reconciliation did not complete unregister:\n%s", screen)
	}
	removed, err := app.Snapshot(app.Options{Home: home})
	if err != nil {
		t.Fatal(err)
	}
	saved := removed.State.Registrations[0]
	if saved.ID != registration.ID || saved.Operation != "unregister" || saved.Status != "removed" || saved.Stage != "complete" || len(removed.State.Onboardings) != len(before.State.Onboardings) {
		t.Fatalf("unregister lost identity or historical onboarding: %+v", removed.State)
	}
	if _, err := os.Stat(key.PublicPath); err != nil {
		t.Fatal("unregister removed the local key", err)
	}
	restart := runScreen(t, home, press(tea.KeyTab), press(tea.KeyTab), press(tea.KeyTab))
	if !strings.Contains(restart, "No recorded follow-up") {
		t.Fatalf("restart revived obsolete onboarding after unregister:\n%s", restart)
	}

	// The result link must identify this registration; deletion is a fresh session.
	session.send(press('v'), press('v'))
	if screen := session.screen(t); !strings.Contains(screen, registration.ID) {
		t.Fatalf("result linked to a different registration:\n%s", screen)
	}
	native = &handoffIO{answer: strings.NewReader("yes\n")}
	session = startHandoff(t, app.Options{Home: home, Input: native, Output: native, Runner: runner}, nil)
	session.send(press(tea.KeyTab), press('d'), reviewKey())
	review = session.screen(t)
	if !strings.Contains(review, "Review delete") || !strings.Contains(review, key.ID) || strings.Contains(review, "Auth:") {
		t.Fatalf("local deletion did not retain the exact key or offered authentication:\n%s", review)
	}
	calls = runner.calls.Load()
	session.send(press(tea.KeyEnter))
	if screen := session.returned(t); !strings.Contains(screen, "Completed; confirmed applicable work recorded") {
		t.Fatalf("guarded local deletion failed after unregister:\n%s", screen)
	}
	deleted, err := app.Snapshot(app.Options{Home: home})
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted.State.Keys) != 0 || len(deleted.State.DeletedKeys) != 1 || deleted.State.DeletedKeys[0].Key.ID != key.ID || deleted.State.Registrations[0].ID != registration.ID || len(deleted.State.History) < len(removed.State.History) {
		t.Fatalf("local deletion lost tombstone/registration/history: %+v", deleted.State)
	}
	if _, err := os.Stat(key.PublicPath); !os.IsNotExist(err) {
		t.Fatalf("deleted fixture key remains: %v", err)
	}
	if runner.calls.Load() != calls {
		t.Fatal("local deletion invoked SSH or dependency execution")
	}
	history := runScreen(t, home, press(tea.KeyTab), press('h'), press(tea.KeyEnd))
	if !strings.Contains(history, "Deleted local key") || !strings.Contains(history, key.ID) {
		t.Fatalf("restart cannot inspect retained key identity:\n%s", history)
	}
}
