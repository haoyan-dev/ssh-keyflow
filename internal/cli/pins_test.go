package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ssh-keyflow/internal/cli"
	"ssh-keyflow/internal/state"
)

const hostKeyLine = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB root@server\n"
const otherHostKeyLine = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIC root@server\n"

func runInput(test *testing.T, home, input string, arguments ...string) (int, string, string) {
	test.Helper()
	var output, diagnostics bytes.Buffer
	code := cli.Run(context.Background(), arguments, cli.Options{
		Home: home, Input: strings.NewReader(input), Output: &output, Error: &diagnostics, Runner: forbiddenSSH{test},
	})
	return code, output.String(), diagnostics.String()
}

func TestHostsPinFromPipeRequiresYesAndNeverReplaces(test *testing.T) {
	home := fixture(test)
	before := tree(test, home)
	code, _, diagnostics := runInput(test, home, hostKeyLine, "hosts", "pin", "--host", "server")
	if code != 1 || !strings.Contains(diagnostics, "--yes") {
		test.Fatalf("exit=%d stderr=%s", code, diagnostics)
	}
	if !equalTrees(before, tree(test, home)) {
		test.Fatal("refused pin changed files")
	}
	code, output, diagnostics := runInput(test, home, hostKeyLine, "hosts", "pin", "--host", "server", "--yes")
	if code != 0 || !strings.Contains(output, "SHA256:") {
		test.Fatalf("exit=%d stdout=%s stderr=%s", code, output, diagnostics)
	}
	pinned := "ssh-keyflow-host-server " + strings.Join(strings.Fields(hostKeyLine)[:2], " ") + "\n"
	if data, _ := os.ReadFile(state.PinPath(home)); string(data) != pinned {
		test.Fatalf("pin file %q", data)
	}
	for _, arguments := range [][]string{{"--yes"}, {"--yes", "--replace"}} {
		code, _, diagnostics = runInput(test, home, otherHostKeyLine, append([]string{"hosts", "pin", "--host", "server"}, arguments...)...)
		if code == 0 || !strings.Contains(diagnostics, "replac") {
			test.Fatalf("%v: exit=%d stderr=%s", arguments, code, diagnostics)
		}
	}
	if data, _ := os.ReadFile(state.PinPath(home)); string(data) != pinned {
		test.Fatal("--yes replaced pinned keys")
	}
	code, output, _ = runInput(test, home, "", "hosts", "list", "--json")
	var result struct {
		Pins map[string][]struct{ Type, Fingerprint string } `json:"host_key_pins"`
	}
	if code != 0 || json.Unmarshal([]byte(output), &result) != nil || len(result.Pins["host-server"]) != 1 || result.Pins["host-server"][0].Type != "ssh-ed25519" {
		test.Fatal(output)
	}
	if code, output, _ = runInput(test, home, "", "hosts", "list"); code != 0 || !strings.Contains(output, "pinned (1)") {
		test.Fatal(output)
	}
	// Unpinning, like removal, needs typed confirmation in a real terminal.
	code, output, diagnostics = runInput(test, home, "yes\n", "hosts", "unpin", "--host", "server")
	if code != 1 || output != "" || !strings.Contains(diagnostics, "real terminal") {
		test.Fatal("unpin from a pipe", code, output, diagnostics)
	}
	if code, _, _ = runInput(test, home, "", "hosts", "unpin", "--host", "server", "--yes"); code != 2 {
		test.Fatal("unpin accepted --yes", code)
	}
	if data, _ := os.ReadFile(state.PinPath(home)); string(data) != pinned {
		test.Fatalf("refused unpin changed %q", data)
	}
}

func TestHostsPinFileAndMalformedPinsReportUnknown(test *testing.T) {
	home := fixture(test)
	source := filepath.Join(test.TempDir(), "keys.txt")
	if err := os.WriteFile(source, []byte("$ cat /etc/ssh/ssh_host_*_key.pub\n"+hostKeyLine), 0600); err != nil {
		test.Fatal(err)
	}
	code, output, diagnostics := runInput(test, home, "", "hosts", "pin", "--host", "server", "--file", source, "--yes")
	if code != 0 || !strings.Contains(output, "Ignored line: $ cat") {
		test.Fatalf("exit=%d stdout=%s stderr=%s", code, output, diagnostics)
	}
	if err := os.WriteFile(state.PinPath(home), []byte("garbage\n"), 0600); err != nil {
		test.Fatal(err)
	}
	code, output, diagnostics = runInput(test, home, "", "hosts", "list", "--json")
	if code != 0 || strings.Contains(output, "host_key_pins") || !strings.Contains(diagnostics, "unknown") {
		test.Fatalf("exit=%d stdout=%s stderr=%s", code, output, diagnostics)
	}
	if code, output, _ = runInput(test, home, "", "hosts", "list"); code != 0 || !strings.Contains(output, "unknown") {
		test.Fatal(output)
	}
	code, _, diagnostics = runInput(test, home, hostKeyLine, "hosts", "pin", "--host", "server", "--yes")
	if code != 1 || !strings.Contains(diagnostics, "review the file manually") {
		test.Fatal(code, diagnostics)
	}
	if data, _ := os.ReadFile(state.PinPath(home)); string(data) != "garbage\n" {
		test.Fatal("malformed pin file rewritten")
	}
	if code, _, _ := runInput(test, home, "", "hosts", "pin", "--yes"); code != 2 {
		test.Fatal("missing --host must be a usage error")
	}
}

func equalTrees(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for path, content := range left {
		if right[path] != content {
			return false
		}
	}
	return true
}
