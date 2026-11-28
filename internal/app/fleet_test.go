package app_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/roster"
	"ssh-keyflow/internal/state"
)

// keygenRunner runs the real ssh-keygen. Interactive calls get their
// passphrase from an SSH_ASKPASS helper, standing in for the terminal; the
// product itself never sets SSH_ASKPASS.
type keygenRunner struct {
	t *testing.T
	// passphrase answers each interactive call.
	passphrase func(app.Command) string
	calls      *[]app.Command
}

func newKeygenRunner(t *testing.T, passphrase func(app.Command) string) keygenRunner {
	t.Helper()
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not on PATH")
	}
	return keygenRunner{t: t, passphrase: passphrase, calls: &[]app.Command{}}
}

func always(passphrase string) func(app.Command) string {
	return func(app.Command) string { return passphrase }
}

func (r keygenRunner) LookPath(name string) (string, error) { return exec.LookPath(name) }

func (r keygenRunner) Run(ctx context.Context, command app.Command) error {
	*r.calls = append(*r.calls, command)
	process := exec.CommandContext(ctx, command.Name, command.Args...)
	process.Stdin, process.Stdout, process.Stderr = command.Stdin, command.Stdout, command.Stderr
	process.Env = os.Environ()
	if command.Interactive {
		passphrase := r.passphrase(command)
		helper := filepath.Join(r.t.TempDir(), "askpass.sh")
		body := "#!/bin/sh\nprintf '%s\\n' '" + passphrase + "'\n"
		if runtime.GOOS == "windows" {
			helper = filepath.Join(filepath.Dir(helper), "askpass.cmd")
			body = "@echo " + passphrase + "\r\n"
			if passphrase == "" {
				body = "@echo.\r\n"
			}
		}
		if err := os.WriteFile(helper, []byte(body), 0700); err != nil {
			r.t.Fatal(err)
		}
		process.Env = append(process.Env, "SSH_ASKPASS="+helper, "SSH_ASKPASS_REQUIRE=force", "DISPLAY=ssh-keyflow-test")
	}
	return process.Run()
}

// fakeRunner reports an OpenSSH version and fails on any other call.
type fakeRunner struct {
	t       *testing.T
	version string
}

func (r fakeRunner) LookPath(string) (string, error) {
	return filepath.Join(r.t.TempDir(), "ssh-keygen"), nil
}

func (r fakeRunner) Run(_ context.Context, command app.Command) error {
	if len(command.Args) != 1 || command.Args[0] != "-V" || filepath.Base(command.Name) != "ssh" {
		r.t.Fatalf("unexpected process %s %v", command.Name, command.Args)
	}
	_, err := fmt.Fprintln(command.Stderr, r.version+", OpenSSL 3.0.0")
	return err
}

// adminNode sets a client and remote node whose login account equals the
// common account "admin", so the roster omits it.
func adminNode(t *testing.T, home string) {
	t.Helper()
	imported(t, home, "laptop", publicKey)
	config, dir := sshdFixture(t, home)
	if out, err := setNode(t, home, app.NodeRequest{Name: "laptop", Roles: []string{"client", "remote"}, Key: "laptop",
		Addresses: []string{"laptop.lan"}, LoginAccount: "admin", SSHDConfig: config, SSHDir: dir}, true); err != nil {
		t.Fatal(out, err)
	}
}

func fleetOptions(home string, runner app.Runner, approve bool) (app.FleetOptions, *bytes.Buffer, *bytes.Buffer) {
	var out, errs bytes.Buffer
	return app.FleetOptions{Home: home, Input: strings.NewReader(""), Output: &out, Error: &errs, Runner: runner, LoginAccount: "admin",
		Confirm: func(string) (bool, error) { return approve, nil }}, &out, &errs
}

func statePath(home string) string { return filepath.Join(state.Directory(home), "state.json") }

// fleetEntries lists fleet/, so leftover temporary directories show up.
func fleetEntries(t *testing.T, home string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(state.Directory(home), "fleet"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

// verifyIndependently checks the accepted signature with ssh-keygen directly.
func verifyIndependently(t *testing.T, fleet state.Fleet) {
	t.Helper()
	dir := t.TempDir()
	allowed, signature := filepath.Join(dir, "allowed"), filepath.Join(dir, "roster.sig")
	write(t, allowed, `fleet-root namespaces="ssh-keyflow-roster" `+fleet.Roots[0].PublicKey+"\n")
	write(t, signature, fleet.Accepted.Signature)
	verify := exec.Command("ssh-keygen", "-Y", "verify", "-f", allowed, "-I", "fleet-root", "-n", "ssh-keyflow-roster", "-s", signature)
	verify.Stdin = strings.NewReader(fleet.Accepted.Roster)
	if output, err := verify.CombinedOutput(); err != nil {
		t.Fatalf("accepted signature does not verify: %v %s", err, output)
	}
}

func TestFleetInitSignAndShow(t *testing.T) {
	home := tempHome(t)
	adminNode(t, home)
	runner := newKeygenRunner(t, always("fleet-pass-1"))
	before := read(t, statePath(home))

	options, out, _ := fleetOptions(home, runner, false)
	options.DryRun = true
	if err := app.InitFleet(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Dry run: no changes made.") || !strings.Contains(out.String(), "ssh-keygen: ") || !strings.Contains(out.String(), "OpenSSH") {
		t.Fatal("dry run output:", out)
	}
	if !bytes.Equal(read(t, statePath(home)), before) || fleetEntries(t, home) != nil {
		t.Fatal("dry run wrote state or fleet files")
	}

	options, out, _ = fleetOptions(home, runner, true)
	if err := app.InitFleet(context.Background(), options); err != nil {
		t.Fatal(err, out)
	}
	doc := load(t, home)
	fleet := doc.Fleet
	if fleet == nil || fleet.RootKey == nil || fleet.Accepted == nil || fleet.Accepted.Version != 1 {
		t.Fatalf("fleet not initialized: %+v", fleet)
	}
	private := filepath.Join(state.Directory(home), "fleet", "root_ed25519")
	if fleet.RootKey.PrivatePath != private || fleet.RootKey.PublicPath != private+".pub" {
		t.Fatal("unexpected root key paths", fleet.RootKey)
	}
	if entries := fleetEntries(t, home); !slices.Equal(entries, []string{"root_ed25519", "root_ed25519.pub"}) {
		t.Fatal("temporary files left in fleet/:", entries)
	}
	secret := string(read(t, private))
	saved := string(read(t, statePath(home)))
	body := strings.Split(strings.TrimSpace(secret), "\n")[1]
	if !strings.Contains(secret, "PRIVATE KEY") || strings.Contains(saved, "PRIVATE KEY") || strings.Contains(saved, body) || strings.Contains(out.String(), body) {
		t.Fatal("the private key reached state or output")
	}
	if strings.Contains(fleet.Roots[0].PublicKey, "ssh-keyflow-fleet-root") || !strings.HasPrefix(fleet.Roots[0].PublicKey, "ssh-ed25519 ") {
		t.Fatal("pinned root is not a bare Ed25519 key:", fleet.Roots[0].PublicKey)
	}
	first, err := roster.Parse([]byte(fleet.Accepted.Roster))
	if err != nil {
		t.Fatal(err)
	}
	if first.FleetID != fleet.ID || first.LoginAccount != "admin" || len(first.Nodes) != 1 || first.Nodes[0].ID != doc.Node.ID ||
		first.Nodes[0].LoginAccount != "" || first.Nodes[0].ClientKeys[0] != strings.TrimSuffix(publicKey, " fixture@example") {
		t.Fatalf("unexpected roster version 1: %+v", first)
	}
	verifyIndependently(t, *fleet)

	options, out, _ = fleetOptions(home, runner, true)
	if err := app.InitFleet(context.Background(), options); err == nil || !strings.Contains(err.Error(), "already accepted") {
		t.Fatal("second init:", err)
	}

	time.Sleep(1100 * time.Millisecond) // created_at has whole seconds
	options, out, _ = fleetOptions(home, runner, true)
	if err := app.SignFleet(context.Background(), options); err != nil {
		t.Fatal(err, out)
	}
	if !strings.Contains(out.String(), "unchanged; signing refreshes created_at") {
		t.Fatal("sign output:", out)
	}
	fleet = load(t, home).Fleet
	second, err := roster.Parse([]byte(fleet.Accepted.Roster))
	if err != nil || second.Version != 2 || fleet.Accepted.Version != 2 || second.CreatedAt <= first.CreatedAt || !slices.EqualFunc(second.Nodes, first.Nodes, func(a, b roster.Node) bool { return a.ID == b.ID }) {
		t.Fatalf("unexpected version 2: %+v %v", second, err)
	}
	verifyIndependently(t, *fleet)

	view, warnings, err := app.ShowFleet(home, 30*24*time.Hour, time.Now())
	if err != nil || view.Fleet == nil || len(warnings) != 0 || string(view.Fleet.Roster) != fleet.Accepted.Roster ||
		len(view.Fleet.Roots) != 1 || view.Fleet.Roots[0].Fingerprint != fleet.RootKey.Fingerprint || !view.Fleet.IsAdmin {
		t.Fatal("fleet show:", view, warnings, err)
	}
	if _, warnings, _ := app.ShowFleet(home, 30*24*time.Hour, time.Now().Add(31*24*time.Hour)); len(warnings) != 1 || !strings.Contains(warnings[0], "fleet sign") {
		t.Fatal("expected an age warning:", warnings)
	}
}

func TestFleetInitRefusesEmptyPassphrase(t *testing.T) {
	home := tempHome(t)
	adminNode(t, home)
	options, out, _ := fleetOptions(home, newKeygenRunner(t, always("")), true)
	err := app.InitFleet(context.Background(), options)
	if err == nil || !strings.Contains(err.Error(), "empty passphrase is refused") {
		t.Fatal(err, out)
	}
	if load(t, home).Fleet != nil || len(fleetEntries(t, home)) != 0 {
		t.Fatal("an unprotected key was kept or pinned", fleetEntries(t, home))
	}
}

func TestFleetInitResumesAfterFailedSignature(t *testing.T) {
	home := tempHome(t)
	adminNode(t, home)
	signing := func(command app.Command) bool { return command.Args[0] == "-Y" }
	runner := newKeygenRunner(t, func(command app.Command) string {
		if signing(command) {
			return "wrong-pass"
		}
		return "fleet-pass-1"
	})
	options, out, _ := fleetOptions(home, runner, true)
	if err := app.InitFleet(context.Background(), options); err == nil || !strings.Contains(err.Error(), "nothing was saved") {
		t.Fatal(err, out)
	}
	fleet := load(t, home).Fleet
	if fleet == nil || fleet.RootKey == nil || fleet.Accepted != nil {
		t.Fatalf("expected a pinned key without a roster: %+v", fleet)
	}
	pinned := fleet.RootKey.Fingerprint
	if view, warnings, err := app.ShowFleet(home, 30*24*time.Hour, time.Now()); err != nil || view.Fleet != nil || len(warnings) != 1 || !strings.Contains(warnings[0], "unfinished") {
		t.Fatal("show after a failed signature:", view, warnings, err)
	}
	options, _, _ = fleetOptions(home, runner, true)
	if err := app.SignFleet(context.Background(), options); err == nil || !strings.Contains(err.Error(), "unfinished") {
		t.Fatal("fleet sign before version 1:", err)
	}
	before := read(t, statePath(home))
	options, _, _ = fleetOptions(home, runner, true)
	if err := app.InitFleet(context.Background(), options); err == nil || !bytes.Equal(read(t, statePath(home)), before) {
		t.Fatal("a wrong passphrase changed state", err)
	}

	options, out, _ = fleetOptions(home, newKeygenRunner(t, always("fleet-pass-1")), true)
	if err := app.InitFleet(context.Background(), options); err != nil {
		t.Fatal(err, out)
	}
	if !strings.Contains(out.String(), "Finish fleet init") {
		t.Fatal("resume output:", out)
	}
	fleet = load(t, home).Fleet
	if fleet.Accepted == nil || fleet.Accepted.Version != 1 || fleet.RootKey.Fingerprint != pinned {
		t.Fatalf("resume did not sign version 1 with the saved key: %+v", fleet)
	}
	if entries := fleetEntries(t, home); len(entries) != 2 {
		t.Fatal("temporary files left in fleet/:", entries)
	}
	verifyIndependently(t, *fleet)
}

func TestFleetRefusalsChangeNothing(t *testing.T) {
	for name, test := range map[string]struct {
		prepare func(*testing.T, string)
		runner  func(*testing.T) app.Runner
		approve bool
		account string
		want    string
	}{
		"no node":         {prepare: func(*testing.T, string) {}, want: "run node set first"},
		"invalid account": {account: "a b", want: "--login-account"},
		"existing root file": {want: "never reused or replaced", prepare: func(t *testing.T, home string) {
			adminNode(t, home)
			write(t, filepath.Join(state.Directory(home), "fleet", "root_ed25519.pub"), publicKey+"\n")
		}},
		"old OpenSSH": {want: "OpenSSH_8.3p1) is older than OpenSSH 8.4", runner: func(t *testing.T) app.Runner { return fakeRunner{t: t, version: "OpenSSH_8.3p1"} }},
		"declined":    {want: app.ErrCancelled.Error(), approve: false},
	} {
		t.Run(name, func(t *testing.T) {
			home := tempHome(t)
			if test.prepare != nil {
				test.prepare(t, home)
			} else {
				adminNode(t, home)
			}
			runner := app.Runner(fakeRunner{t: t, version: "OpenSSH_9.5p2"})
			if test.runner != nil {
				runner = test.runner(t)
			}
			before, _ := os.ReadFile(statePath(home))
			entries := fleetEntries(t, home)
			options, _, _ := fleetOptions(home, runner, test.approve)
			if test.account != "" {
				options.LoginAccount = test.account
			}
			err := app.InitFleet(context.Background(), options)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
			if test.account != "" && !errors.Is(err, app.ErrInvalidInput) {
				t.Fatal("an invalid account is not a usage error")
			}
			after, _ := os.ReadFile(statePath(home))
			if !bytes.Equal(before, after) || !slices.Equal(entries, fleetEntries(t, home)) {
				t.Fatal("a refused init changed state or fleet/")
			}
		})
	}
}

func TestFleetSignNeedsAnAdminFleet(t *testing.T) {
	home := tempHome(t)
	adminNode(t, home)
	options, _, _ := fleetOptions(home, fakeRunner{t: t, version: "OpenSSH_9.5p2"}, true)
	if err := app.SignFleet(context.Background(), options); err == nil || !strings.Contains(err.Error(), "run fleet init") {
		t.Fatal(err)
	}
}

func TestFleetShowNeverInitializesStorage(t *testing.T) {
	home := tempHome(t)
	view, warnings, err := app.ShowFleet(home, 30*24*time.Hour, time.Now())
	if err != nil || view.Fleet != nil || len(warnings) != 0 {
		t.Fatal(view, warnings, err)
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh")); !os.IsNotExist(err) {
		t.Fatal("fleet show initialized storage")
	}
}
