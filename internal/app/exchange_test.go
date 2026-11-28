package app_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/roster"
	"ssh-keyflow/internal/state"
)

// testSigner is an unprotected key standing in for a fleet root key, so tests
// can sign any roster bytes.
type testSigner struct {
	private string
	key     roster.PublicKey
}

func newSigner(t *testing.T) testSigner {
	t.Helper()
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not on PATH")
	}
	private := filepath.Join(t.TempDir(), "root")
	if output, err := exec.Command("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-C", "test-root", "-f", private).CombinedOutput(); err != nil {
		t.Fatal(err, string(output))
	}
	key, err := roster.ParsePublicKey(strings.TrimSpace(string(read(t, private+".pub"))), false)
	if err != nil {
		t.Fatal(err)
	}
	return testSigner{private: private, key: key}
}

// rosterDir writes data, its signature by signer and fleet-root.pub (the
// signer's key unless root is given) into a new directory.
func rosterDir(t *testing.T, signer testSigner, data []byte, root ...string) string {
	t.Helper()
	dir := t.TempDir()
	file := filepath.Join(dir, "roster.json")
	write(t, file, string(data))
	if output, err := exec.Command("ssh-keygen", "-Y", "sign", "-f", signer.private, "-n", roster.Namespace, file).CombinedOutput(); err != nil {
		t.Fatal(err, string(output))
	}
	text := signer.key.Text + "\n"
	if len(root) > 0 {
		text = root[0]
	}
	write(t, filepath.Join(dir, "fleet-root.pub"), text)
	return dir
}

const testFleetID = "fleet-0f3c9a7e5b2d4c1f8e6a0b9d7c5e3f21"

func testRoster(t *testing.T, version int64, created string) []byte {
	t.Helper()
	data, err := roster.Roster{Format: roster.Format, Schema: roster.Schema, FleetID: testFleetID, Version: version, CreatedAt: created, LoginAccount: "admin",
		Nodes: []roster.Node{{ID: "node-5a1e0c3b7d9f4e2a8b6c1d0e9f7a3b5c", Name: "laptop", Roles: []string{"client"}, ClientKeys: []string{strings.TrimSuffix(otherPublicKey, " other@example")}}}}.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func now() string { return time.Now().UTC().Format(roster.TimeLayout) }

func acceptOptions(t *testing.T, home, dir string, approve bool) (app.FleetOptions, *bytes.Buffer, *bytes.Buffer) {
	options, out, errs := fleetOptions(home, newKeygenRunner(t, always("")), approve)
	options.Dir, options.MaxAge = dir, 30*24*time.Hour
	return options, out, errs
}

// noPrompt fails the test if a prompt is shown.
func noPrompt(t *testing.T) func(string) (bool, error) {
	return func(label string) (bool, error) {
		t.Errorf("unexpected prompt %q", label)
		return false, errors.New("unexpected prompt")
	}
}

func TestFleetExportAddAndAcceptEndToEnd(t *testing.T) {
	admin, member := tempHome(t), tempHome(t)
	adminNode(t, admin)
	adminRunner := newKeygenRunner(t, always("fleet-pass-1"))
	options, out, _ := fleetOptions(admin, adminRunner, true)
	if err := app.InitFleet(context.Background(), options); err != nil {
		t.Fatal(err, out)
	}

	imported(t, member, "desk", otherPublicKey)
	if out, err := setNode(t, member, app.NodeRequest{Name: "desk", Roles: []string{"client"}, Key: "desk"}, true); err != nil {
		t.Fatal(out, err)
	}
	nodeDir := filepath.Join(t.TempDir(), "node")
	var exported bytes.Buffer
	if err := app.ExportNode(app.ExportOptions{Home: member, Out: nodeDir, Output: &exported}); err != nil {
		t.Fatal(err)
	}
	if file, err := roster.ParseNodeFile(read(t, filepath.Join(nodeDir, "node.json"))); err != nil || file.Node.Name != "desk" {
		t.Fatal("node.json:", file, err)
	}
	if !strings.Contains(exported.String(), "Client key: ssh-ed25519 SHA256:") || !strings.Contains(exported.String(), "fleet add") {
		t.Fatal("node export output:", exported.String())
	}

	before := read(t, statePath(admin))
	options, out, _ = fleetOptions(admin, adminRunner, false)
	options.Dir, options.DryRun = nodeDir, true
	if err := app.AddNode(context.Background(), options); err != nil {
		t.Fatal(err, out)
	}
	if !strings.Contains(out.String(), "Adds node desk:") || !strings.Contains(out.String(), "Compare the name, roles and fingerprints with node show on desk") || !bytes.Equal(before, read(t, statePath(admin))) {
		t.Fatal("fleet add --dry-run:", out)
	}
	options, out, _ = fleetOptions(admin, adminRunner, true)
	options.Dir = nodeDir
	if err := app.AddNode(context.Background(), options); err != nil {
		t.Fatal(err, out)
	}
	fleet := load(t, admin).Fleet
	second, err := roster.Parse([]byte(fleet.Accepted.Roster))
	if err != nil || second.Version != 2 || len(second.Nodes) != 2 {
		t.Fatalf("fleet add did not sign version 2 with two nodes: %+v %v", second, err)
	}
	verifyIndependently(t, *fleet)

	exportDir := filepath.Join(t.TempDir(), "roster")
	if err := app.ExportFleet(app.ExportOptions{Home: admin, Out: exportDir, Output: &bytes.Buffer{}}); err != nil {
		t.Fatal(err)
	}
	if string(read(t, filepath.Join(exportDir, "roster.json"))) != fleet.Accepted.Roster || string(read(t, filepath.Join(exportDir, "roster.json.sig"))) != fleet.Accepted.Signature ||
		string(read(t, filepath.Join(exportDir, "fleet-root.pub"))) != fleet.Roots[0].PublicKey+"\n" {
		t.Fatal("fleet export files differ from the accepted roster")
	}

	before = read(t, statePath(member))
	accept, out, _ := acceptOptions(t, member, exportDir, false)
	accept.DryRun, accept.Confirm = true, noPrompt(t)
	if err := app.AcceptFleet(context.Background(), accept); err != nil {
		t.Fatal(err, out)
	}
	if !strings.Contains(out.String(), "First acceptance: this pins fleet root key "+fleet.Roots[0].Fingerprint) || !strings.Contains(out.String(), "Dry run: no changes made.") ||
		!bytes.Equal(before, read(t, statePath(member))) {
		t.Fatal("fleet accept --dry-run:", out)
	}
	accept, out, _ = acceptOptions(t, member, exportDir, true)
	if err := app.AcceptFleet(context.Background(), accept); err != nil {
		t.Fatal(err, out)
	}
	accepted := load(t, member).Fleet
	if accepted == nil || accepted.ID != fleet.ID || accepted.RootKey != nil || accepted.Roots[0].PublicKey != fleet.Roots[0].PublicKey ||
		accepted.Accepted.Roster != fleet.Accepted.Roster || accepted.Accepted.SHA256 != fleet.Accepted.SHA256 {
		t.Fatalf("member did not pin and accept version 2: %+v", accepted)
	}
	if !strings.Contains(out.String(), "Type yes to pin root "+fleet.Roots[0].Fingerprint) {
		t.Fatal("the first-pin prompt does not name the fingerprint:", out)
	}
	verifyIndependently(t, *accepted)

	before = read(t, statePath(member))
	accept, out, _ = acceptOptions(t, member, exportDir, true)
	accept.Confirm = noPrompt(t)
	// Accepting the same roster again needs no terminal (roster-format 2 rule 5).
	accept.RequireTerminal = func() error { t.Error("a settled accept asked for a terminal"); return errors.New("no terminal") }
	if err := app.AcceptFleet(context.Background(), accept); err != nil || !strings.Contains(out.String(), "already accepted") || !bytes.Equal(before, read(t, statePath(member))) {
		t.Fatal("accepting the same roster again:", err, out)
	}

	// A member's re-export of the accepted roster is identical to the admin's.
	reexport := filepath.Join(t.TempDir(), "reexport")
	if err := app.ExportFleet(app.ExportOptions{Home: member, Out: reexport, Output: &bytes.Buffer{}}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(read(t, filepath.Join(reexport, "roster.json")), read(t, filepath.Join(exportDir, "roster.json"))) {
		t.Fatal("a re-export differs")
	}
}

func TestFleetAcceptRejectsAndChangesNothing(t *testing.T) {
	signer, other := newSigner(t), newSigner(t)
	home := tempHome(t)
	accept, out, _ := acceptOptions(t, home, rosterDir(t, signer, testRoster(t, 5, now())), true)
	if err := app.AcceptFleet(context.Background(), accept); err != nil {
		t.Fatal(err, out)
	}
	accepted := load(t, home).Fleet.Accepted
	otherFleet := bytes.Replace(testRoster(t, 6, now()), []byte(testFleetID), []byte("fleet-"+strings.Repeat("ab", 16)), 1)
	tampered := func(t *testing.T) string {
		dir := rosterDir(t, signer, testRoster(t, 6, now()))
		write(t, filepath.Join(dir, "roster.json"), strings.Replace(string(testRoster(t, 6, now())), "laptop", "laptoq", 1))
		return dir
	}
	for name, test := range map[string]struct {
		dir  func(*testing.T) string
		want string
	}{
		"unknown root": {func(t *testing.T) string { return rosterDir(t, other, testRoster(t, 6, now())) }, "is not a pinned root"},
		"bad signature": {func(t *testing.T) string {
			return rosterDir(t, other, testRoster(t, 6, now()), signer.key.Text+"\n")
		}, "is not a valid signature by root"},
		"modified roster": {tampered, "does not match its signature"},
		"rollback":        {func(t *testing.T) string { return rosterDir(t, signer, testRoster(t, 4, now())) }, "older than the accepted version 5"},
		"same version, other bytes": {func(t *testing.T) string {
			return rosterDir(t, signer, testRoster(t, 5, "2026-01-02T00:00:00Z"))
		}, "differs from the accepted roster of the same version"},
		"other fleet": {func(t *testing.T) string { return rosterDir(t, signer, otherFleet) }, "is for fleet"},
		"not canonical": {func(t *testing.T) string {
			return rosterDir(t, signer, bytes.ReplaceAll(testRoster(t, 6, now()), []byte("\n"), []byte("\r\n")))
		}, "not in canonical form"},
		"extra file": {func(t *testing.T) string {
			dir := rosterDir(t, signer, testRoster(t, 6, now()))
			write(t, filepath.Join(dir, "notes.txt"), "x")
			return dir
		}, "must hold exactly"},
		"root with comment": {func(t *testing.T) string {
			return rosterDir(t, signer, testRoster(t, 6, now()), signer.key.Text+" test-root\n")
		}, "without a comment"},
	} {
		t.Run(name, func(t *testing.T) {
			before := read(t, statePath(home))
			for _, dryRun := range []bool{true, false} {
				options, out, _ := acceptOptions(t, home, test.dir(t), true)
				options.DryRun, options.Confirm = dryRun, noPrompt(t)
				err := app.AcceptFleet(context.Background(), options)
				if err == nil || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("dry run %t: expected %q, got %v %s", dryRun, test.want, err, out)
				}
			}
			if !bytes.Equal(before, read(t, statePath(home))) {
				t.Fatal("a rejected roster changed state")
			}
		})
	}
	if load(t, home).Fleet.Accepted.SHA256 != accepted.SHA256 {
		t.Fatal("accepted roster changed")
	}

	options, out, _ := acceptOptions(t, home, rosterDir(t, signer, testRoster(t, 7, now())), false)
	noTerminal := errors.New("no terminal")
	options.RequireTerminal = func() error { return noTerminal }
	if err := app.AcceptFleet(context.Background(), options); !errors.Is(err, noTerminal) || out.Len() != 0 {
		t.Fatal("an accept that needs a prompt must require a terminal before describing:", err, out)
	}
	options, out, _ = acceptOptions(t, home, rosterDir(t, signer, testRoster(t, 7, now())), false)
	if err := app.AcceptFleet(context.Background(), options); !errors.Is(err, app.ErrCancelled) || !strings.Contains(out.String(), "Changes since accepted version 5:") {
		t.Fatal("declined accept:", err, out)
	}
	if load(t, home).Fleet.Accepted.Version != 5 {
		t.Fatal("a declined accept changed state")
	}
}

func TestFleetAcceptWarnsOnAgeAndPinsOnlyAfterYes(t *testing.T) {
	signer := newSigner(t)
	home := tempHome(t)
	options, _, _ := acceptOptions(t, home, rosterDir(t, signer, testRoster(t, 1, "2026-01-01T00:00:00Z")), false)
	options.DryRun = true
	if err := app.AcceptFleet(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh")); !os.IsNotExist(err) {
		t.Fatal("fleet accept --dry-run initialized storage")
	}
	options, _, _ = acceptOptions(t, home, rosterDir(t, signer, testRoster(t, 1, "2026-01-01T00:00:00Z")), false)
	if err := app.AcceptFleet(context.Background(), options); !errors.Is(err, app.ErrCancelled) {
		t.Fatal("declined first accept:", err)
	}
	if doc, err := state.Load(home); err != nil || doc.Fleet != nil {
		t.Fatal("a declined first accept pinned a root:", err)
	}
	options, out, errs := acceptOptions(t, home, rosterDir(t, signer, testRoster(t, 1, "2026-01-01T00:00:00Z")), true)
	if err := app.AcceptFleet(context.Background(), options); err != nil {
		t.Fatal(err, out)
	}
	if !strings.Contains(errs.String(), "older than --max-age 30 days") || strings.Contains(out.String(), "warning") {
		t.Fatal("age warning must be on stderr only:", errs, out)
	}
	if fleet := load(t, home).Fleet; fleet == nil || fleet.Roots[0].Fingerprint != signer.key.Fingerprint || fleet.Accepted.Version != 1 {
		t.Fatalf("old roster not accepted: %+v", fleet)
	}
	options, _, errs = acceptOptions(t, home, rosterDir(t, signer, testRoster(t, 2, "2099-01-01T00:00:00Z")), true)
	if err := app.AcceptFleet(context.Background(), options); err != nil || !strings.Contains(errs.String(), "in the future") {
		t.Fatal("future roster:", err, errs)
	}
}

func TestFleetAddRefusals(t *testing.T) {
	admin := tempHome(t)
	adminNode(t, admin)
	runner := newKeygenRunner(t, always("fleet-pass-1"))
	options, out, _ := fleetOptions(admin, runner, true)
	if err := app.InitFleet(context.Background(), options); err != nil {
		t.Fatal(err, out)
	}
	doc := load(t, admin)
	nodeDir := func(t *testing.T, node roster.Node) string {
		data, err := roster.NodeFile{Format: roster.NodeFormat, Schema: roster.NodeSchema, Node: node}.Marshal()
		if err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		write(t, filepath.Join(dir, "node.json"), string(data))
		return dir
	}
	desk := roster.Node{ID: "node-" + strings.Repeat("ee", 16), Name: "desk", Roles: []string{"client"}, ClientKeys: []string{strings.TrimSuffix(otherPublicKey, " other@example")}}
	for name, test := range map[string]struct {
		dir  func(*testing.T) string
		want string
	}{
		"own id": {func(t *testing.T) string {
			own := desk
			own.ID = doc.Node.ID
			return nodeDir(t, own)
		}, "describes this admin node"},
		"name in use": {func(t *testing.T) string {
			clash := desk
			clash.Name = "laptop"
			return nodeDir(t, clash)
		}, `node name "laptop" is already used`},
		"key in use": {func(t *testing.T) string {
			clash := desk
			clash.ClientKeys = []string{strings.TrimSuffix(publicKey, " fixture@example")}
			return nodeDir(t, clash)
		}, "more than one node"},
		"not canonical": {func(t *testing.T) string {
			dir := nodeDir(t, desk)
			write(t, filepath.Join(dir, "node.json"), strings.ReplaceAll(string(read(t, filepath.Join(dir, "node.json"))), "\n", "\r\n"))
			return dir
		}, "not in canonical form"},
		"no node.json": {func(t *testing.T) string { return t.TempDir() }, "node.json"},
	} {
		t.Run(name, func(t *testing.T) {
			before := read(t, statePath(admin))
			options, out, _ := fleetOptions(admin, runner, true)
			options.Dir, options.Confirm = test.dir(t), noPrompt(t)
			if err := app.AddNode(context.Background(), options); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v %s", test.want, err, out)
			}
			if !bytes.Equal(before, read(t, statePath(admin))) {
				t.Fatal("a refused fleet add changed state")
			}
		})
	}

	options, out, _ = fleetOptions(admin, runner, true)
	options.Dir = nodeDir(t, desk)
	if err := app.AddNode(context.Background(), options); err != nil {
		t.Fatal(err, out)
	}
	moved := desk
	moved.Name = "desk2"
	var label string
	options, out, _ = fleetOptions(admin, runner, false)
	options.Dir = nodeDir(t, moved)
	options.Confirm = func(text string) (bool, error) { label = text; return false, nil }
	if err := app.AddNode(context.Background(), options); !errors.Is(err, app.ErrCancelled) {
		t.Fatal(err, out)
	}
	if !strings.Contains(out.String(), "Replaces the entry of node desk2:") || !strings.Contains(out.String(), "was: desk ") || !strings.Contains(out.String(), "now: desk2 ") ||
		!strings.Contains(out.String()+label, "replace node desk2") {
		t.Fatal("replacement plan:", out, label)
	}

	member := tempHome(t)
	options, _, _ = fleetOptions(member, runner, true)
	options.Dir = nodeDir(t, desk)
	if err := app.AddNode(context.Background(), options); err == nil || !strings.Contains(err.Error(), "run fleet init") {
		t.Fatal("fleet add without a fleet:", err)
	}
}

func TestExportsRefuseAndNeverInitializeStorage(t *testing.T) {
	home := tempHome(t)
	out := filepath.Join(t.TempDir(), "out")
	if err := app.ExportNode(app.ExportOptions{Home: home, Out: out, Output: &bytes.Buffer{}}); err == nil || !strings.Contains(err.Error(), "run node set first") {
		t.Fatal("node export without a node:", err)
	}
	if err := app.ExportFleet(app.ExportOptions{Home: home, Out: out, Output: &bytes.Buffer{}}); err == nil || !strings.Contains(err.Error(), "no accepted roster") {
		t.Fatal("fleet export without a roster:", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh")); !os.IsNotExist(err) {
		t.Fatal("an export initialized storage")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("a refused export created --out")
	}

	imported(t, home, "desk", otherPublicKey)
	if text, err := setNode(t, home, app.NodeRequest{Name: "desk", Roles: []string{"client"}, Key: "desk"}, true); err != nil {
		t.Fatal(text, err)
	}
	full := t.TempDir()
	write(t, filepath.Join(full, "keep.txt"), "keep")
	if err := app.ExportNode(app.ExportOptions{Home: home, Out: full, Output: &bytes.Buffer{}}); err == nil || !strings.Contains(err.Error(), "absent or empty") {
		t.Fatal("node export into a non-empty directory:", err)
	}
	if entries, _ := os.ReadDir(full); len(entries) != 1 {
		t.Fatal("a refused export wrote files")
	}
	empty := t.TempDir()
	if err := app.ExportNode(app.ExportOptions{Home: home, Out: empty, Output: &bytes.Buffer{}}); err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(empty); len(entries) != 1 || entries[0].Name() != "node.json" {
		t.Fatal("unexpected export files", entries)
	}
}

func TestFleetDirectoryArgumentIsAUsageError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	if err := app.CheckInputDirectory(tempHome(t), missing); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "file")
	write(t, file, "x")
	if err := app.CheckInputDirectory(tempHome(t), file); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatal(err)
	}
}
