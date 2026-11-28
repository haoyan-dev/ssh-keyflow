package cli_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"ssh-keyflow/internal/roster"
	"ssh-keyflow/internal/state"
)

func TestFleetUsageErrorsExitTwoWithoutWriting(test *testing.T) {
	for _, arguments := range [][]string{
		{"fleet", "init"},
		{"fleet", "init", "--login-account", "a b"},
		{"fleet", "init", "--login-account", "admin", "extra"},
		{"fleet", "sign", "extra"},
		{"fleet", "show", "--max-age", "0"},
		{"fleet", "show", "--max-age", "x"},
		{"fleet", "show", "--max-age", "36501"},
		{"fleet", "show", "--max-age", "200000"},
	} {
		home := test.TempDir()
		code, output, diagnostics := run(test, home, arguments...)
		if code != 2 || output != "" || !strings.Contains(diagnostics, "Usage:") {
			test.Fatalf("%v: exit=%d stdout=%s stderr=%s", arguments, code, output, diagnostics)
		}
		assertEmptyHome(test, home)
	}
}

func TestFleetChangesNeedATerminal(test *testing.T) {
	for _, arguments := range [][]string{{"fleet", "init", "--login-account", "admin"}, {"fleet", "sign"}} {
		home := test.TempDir()
		code, _, diagnostics := run(test, home, arguments...)
		if code != 1 || !strings.Contains(diagnostics, "real terminal") || strings.Contains(diagnostics, "Usage:") {
			test.Fatalf("%v: exit=%d stderr=%s", arguments, code, diagnostics)
		}
		assertEmptyHome(test, home)
	}
}

func TestFleetDryRunWithoutNodeStopsBeforeProbing(test *testing.T) {
	home := test.TempDir()
	code, _, diagnostics := run(test, home, "fleet", "init", "--login-account", "admin", "--dry-run")
	if code != 1 || !strings.Contains(diagnostics, "run node set first") {
		test.Fatalf("exit=%d stderr=%s", code, diagnostics)
	}
	assertEmptyHome(test, home)
}

func TestFleetShowBeforeAnyRoster(test *testing.T) {
	home := test.TempDir()
	code, output, diagnostics := run(test, home, "fleet", "show", "--json")
	if code != 0 || output != "{\"fleet\":null}\n" || diagnostics != "" {
		test.Fatalf("exit=%d stdout=%q stderr=%s", code, output, diagnostics)
	}
	code, output, _ = run(test, home, "fleet", "show")
	if code != 0 || !strings.Contains(output, "No accepted roster yet") {
		test.Fatalf("exit=%d stdout=%s", code, output)
	}
	assertEmptyHome(test, home)
}

// acceptedFleet writes a state whose accepted roster was signed long ago.
// State checks the signature's form, not its validity.
func acceptedFleet(test *testing.T) (home, text string) {
	test.Helper()
	home = test.TempDir()
	key := func(seed byte) string {
		var blob []byte
		for _, part := range [][]byte{[]byte("ssh-ed25519"), bytes.Repeat([]byte{seed}, 32)} {
			blob = append(blob, 0, 0, 0, byte(len(part)))
			blob = append(blob, part...)
		}
		return "ssh-ed25519 " + base64.StdEncoding.EncodeToString(blob)
	}
	root := key(9)
	parsed, err := roster.ParsePublicKey(root, false)
	if err != nil {
		test.Fatal(err)
	}
	fleetID := "fleet-" + strings.Repeat("cd", 16)
	data, err := (roster.Roster{Format: roster.Format, Schema: roster.Schema, FleetID: fleetID, Version: 7, CreatedAt: "2026-01-01T00:00:00Z", LoginAccount: "admin",
		Nodes: []roster.Node{{ID: "node-" + strings.Repeat("ab", 16), Name: "laptop", Roles: []string{"client"}, ClientKeys: []string{key(1)}}}}).Marshal()
	if err != nil {
		test.Fatal(err)
	}
	digest := sha256.Sum256(data)
	signature := "-----BEGIN SSH SIGNATURE-----\n" + base64.StdEncoding.EncodeToString([]byte("SSHSIG\x00\x00\x00\x01")) + "\n-----END SSH SIGNATURE-----\n"
	document := state.State{Version: state.Version, Keys: []state.Key{}, Fleet: &state.Fleet{ID: fleetID,
		Roots:    []state.Root{{PublicKey: root, Fingerprint: parsed.Fingerprint, PinnedAt: "2026-01-01T00:00:00Z"}},
		Accepted: &state.Accepted{Version: 7, SHA256: hex.EncodeToString(digest[:]), Roster: string(data), Signature: signature, AcceptedAt: "2026-01-01T00:00:00Z"}}}
	if err := document.Validate(); err != nil {
		test.Fatal(err)
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		test.Fatal(err)
	}
	if err := os.MkdirAll(state.Directory(home), 0700); err != nil {
		test.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(state.Directory(home), "state.json"), encoded, 0600); err != nil {
		test.Fatal(err)
	}
	return home, string(data)
}

func TestFleetShowJSONAndAgeWarning(test *testing.T) {
	home, text := acceptedFleet(test)
	before := tree(test, home)
	code, output, diagnostics := run(test, home, "fleet", "show", "--json", "--max-age", "1")
	if code != 0 || !strings.HasSuffix(output, "}\n") || strings.Count(output, "\n") != 1 {
		test.Fatalf("exit=%d stdout=%s stderr=%s", code, output, diagnostics)
	}
	if !strings.Contains(diagnostics, "roster version 7 was signed 2026-01-01T00:00:00Z") || !strings.Contains(diagnostics, "--max-age 1 days") || strings.Contains(output, "warning") {
		test.Fatalf("age warning must be on stderr only: stderr=%s", diagnostics)
	}
	var shown struct {
		Fleet struct {
			Roots  []map[string]string `json:"roots"`
			SHA256 string              `json:"sha256"`
			Roster json.RawMessage     `json:"roster"`
		} `json:"fleet"`
	}
	decoder := json.NewDecoder(strings.NewReader(output))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&shown); err != nil {
		test.Fatal(err)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, []byte(text)); err != nil {
		test.Fatal(err)
	}
	digest := sha256.Sum256([]byte(text))
	if len(shown.Fleet.Roots) != 1 || !strings.HasPrefix(shown.Fleet.Roots[0]["fingerprint"], "SHA256:") || len(shown.Fleet.Roots[0]) != 1 ||
		shown.Fleet.SHA256 != hex.EncodeToString(digest[:]) || string(shown.Fleet.Roster) != compact.String() {
		test.Fatalf("unexpected fleet show --json: %s", output)
	}
	code, output, _ = run(test, home, "fleet", "show")
	if code != 0 || !strings.Contains(output, "Roster version: 7, created 2026-01-01T00:00:00Z (") || !strings.Contains(output, " days old)") || !strings.Contains(output, "Admin node: false") || !strings.Contains(output, "laptop node-") {
		test.Fatalf("text view: exit=%d stdout=%s", code, output)
	}
	if !reflect.DeepEqual(before, tree(test, home)) {
		test.Fatal("fleet show changed files")
	}
}
