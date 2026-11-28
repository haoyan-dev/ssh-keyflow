package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"ssh-keyflow/internal/state"
)

func TestNodeCommandsUsageAndTerminalFailures(test *testing.T) {
	cases := []struct {
		arguments []string
		code      int
		message   string
	}{
		{[]string{"node", "set", "--roles", "client"}, 2, "name"},
		{[]string{"node", "set", "--name", "laptop"}, 2, "roles"},
		{[]string{"node", "set", "--name", "Laptop", "--roles", "client", "--key", "known"}, 2, "name"},
		{[]string{"node", "set", "--name", "laptop", "--roles", "host"}, 2, "role"},
		{[]string{"node", "set", "--name", "laptop", "--roles", "client", "--roles", "remote"}, 2, "once"},
		{[]string{"node", "set", "--name", "vps", "--roles", "remote", "--addresses", "a.lan,,b.lan"}, 2, "addresses"},
		{[]string{"node", "set", "--name", "vps", "--roles", "remote", "--addresses", "vps.lan", "--port", "0"}, 2, "--port"},
		{[]string{"node", "set", "--name", "laptop", "--roles", "client", "--key", "known"}, 1, "terminal"},
		{[]string{"node", "show", "extra"}, 2, "unknown"},
	}
	for _, example := range cases {
		test.Run(strings.Join(example.arguments, " "), func(test *testing.T) {
			home := test.TempDir()
			code, output, diagnostics := run(test, home, example.arguments...)
			if code != example.code || output != "" || !strings.Contains(diagnostics, example.message) {
				test.Fatalf("exit=%d stdout=%s stderr=%s", code, output, diagnostics)
			}
			assertEmptyHome(test, home)
		})
	}
}

func TestNodeShowJSONBeforeSetIsNullAndReadOnly(test *testing.T) {
	home := test.TempDir()
	code, output, diagnostics := run(test, home, "node", "show", "--json")
	if code != 0 || output != "{\"node\":null,\"is_admin\":false}\n" || diagnostics != "" {
		test.Fatalf("exit=%d stdout=%q stderr=%s", code, output, diagnostics)
	}
	assertEmptyHome(test, home)
}

func TestNodeSetDryRunNeedsNoTerminalAndWritesNothing(test *testing.T) {
	home := fixture(test)
	path := filepath.Join(state.Directory(home), "state.json")
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Contains(data, []byte(`"version":4,`)) {
		test.Fatal("unexpected fixture", string(data), err)
	}
	if err := os.WriteFile(path, bytes.Replace(data, []byte(`"version":4,`), []byte(`"version":3,`), 1), 0600); err != nil {
		test.Fatal(err)
	}
	before := tree(test, home)
	code, output, diagnostics := run(test, home, "node", "set", "--name", "laptop", "--roles", "client", "--key", "known", "--dry-run")
	if code != 0 || !strings.Contains(output, "Dry run: no changes made.") || !strings.Contains(output, `Client key: "known" [key-known] SHA256:`) {
		test.Fatalf("exit=%d stdout=%s stderr=%s", code, output, diagnostics)
	}
	if !strings.Contains(diagnostics, "schema v3") || strings.Contains(output, "schema v3") {
		test.Fatalf("migration notice not on stderr only: stdout=%s stderr=%s", output, diagnostics)
	}
	if !reflect.DeepEqual(before, tree(test, home)) {
		test.Fatal("dry run changed files")
	}
}
