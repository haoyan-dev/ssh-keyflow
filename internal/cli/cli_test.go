package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/cli"
)

func TestHelpAndNonterminalStartupDoNotInitializeStorage(test *testing.T) {
	for _, arguments := range [][]string{{"--help"}, {}} {
		home := test.TempDir()
		var output, diagnostics bytes.Buffer
		code := cli.Run(context.Background(), arguments, cli.Options{
			Home: home, Input: strings.NewReader(""), Output: &output, Error: &diagnostics,
		})
		if code != 0 || !strings.Contains(output.String(), "Usage:") || diagnostics.Len() != 0 {
			test.Fatalf("help: exit=%d stdout=%s stderr=%s", code, output.String(), diagnostics.String())
		}
		if _, err := os.Stat(filepath.Join(home, ".ssh")); !os.IsNotExist(err) {
			test.Fatal("help initialized SSH storage")
		}
	}
}

func TestListJSONIsReadOnlyAndContainsOnlyRequestedRecords(test *testing.T) {
	for _, category := range []string{"hosts", "keys", "registrations"} {
		home := test.TempDir()
		var output, diagnostics bytes.Buffer
		code := cli.Run(context.Background(), []string{category, "list", "--json"}, cli.Options{
			Home: home, Input: strings.NewReader(""), Output: &output, Error: &diagnostics,
		})
		var result map[string]json.RawMessage
		if code != 0 || json.Unmarshal(output.Bytes(), &result) != nil || string(result[category]) != "[]" || diagnostics.Len() != 0 {
			test.Fatalf("list %s: exit=%d stdout=%s stderr=%s", category, code, output.String(), diagnostics.String())
		}
		if _, err := os.Stat(filepath.Join(home, ".ssh")); !os.IsNotExist(err) {
			test.Fatal("listing initialized SSH storage")
		}
	}
}

func TestDirectCommandsRejectMissingArgumentsAndNonterminalChanges(test *testing.T) {
	cases := []struct {
		arguments []string
		code      int
		message   string
	}{
		{[]string{"keys", "create"}, 2, "name"},
		{[]string{"keys", "create", "--name", "direct"}, 1, "terminal"},
		{[]string{"keys", "import", "--name", "existing", "--public", "source.pub"}, 1, "terminal"},
		{[]string{"keys", "delete", "--key", "key-123"}, 1, "terminal"},
		{[]string{"register", "--host", "server", "--key", "key-123"}, 1, "terminal"},
		{[]string{"unregister", "--registration", "registration-123"}, 1, "terminal"},
		{[]string{"resume", "--id", "creation-123"}, 1, "terminal"},
		{[]string{"tui"}, 2, "terminal"},
		{[]string{"hosts", "add", "--alias", "server", "--hostname", "server.example", "--user", "alice", "--os", "windows", "--key", "known"}, 2, "account"},
		{[]string{"hosts", "add", "--alias", "server", "--hostname", "server.example", "--user", "alice", "--os", "linux", "--key", "known"}, 1, "terminal"},
		{[]string{"register", "--host", "server", "--key", "known", "--auth", "key"}, 2, "identity"},
		{[]string{"register", "--host", "server", "--key", "known", "--auth", "guess"}, 2, "authentication"},
	}
	for _, example := range cases {
		test.Run(strings.Join(example.arguments, " "), func(test *testing.T) {
			home := test.TempDir()
			var output, diagnostics bytes.Buffer
			code := cli.Run(context.Background(), example.arguments, cli.Options{
				Home: home, Input: strings.NewReader("yes\n"), Output: &output, Error: &diagnostics,
			})
			if code != example.code || !strings.Contains(diagnostics.String(), example.message) {
				test.Fatalf("exit=%d stdout=%s stderr=%s", code, output.String(), diagnostics.String())
			}
			if _, err := os.Stat(filepath.Join(home, ".ssh")); !os.IsNotExist(err) {
				test.Fatal("rejected operation initialized SSH storage")
			}
		})
	}
}

func TestVersionUsageAndRuntimeFailuresHaveDistinctOutput(test *testing.T) {
	home := test.TempDir()
	var output, diagnostics bytes.Buffer
	options := cli.Options{Home: home, Version: "v0.2.0-test", Input: strings.NewReader(""), Output: &output, Error: &diagnostics}
	code := cli.Run(context.Background(), []string{"--version"}, options)
	if code != 0 || strings.TrimSpace(output.String()) != "SSH KeyFlow v0.2.0-test" || diagnostics.Len() != 0 {
		test.Fatalf("version: exit=%d stdout=%s stderr=%s", code, output.String(), diagnostics.String())
	}
	for _, arguments := range [][]string{{"keys", "unknown"}, {"keys", "create"}, {"hosts", "list", "extra"}} {
		output.Reset()
		diagnostics.Reset()
		code = cli.Run(context.Background(), arguments, options)
		if code != 2 || output.Len() != 0 || !strings.Contains(diagnostics.String(), "Usage:") {
			test.Fatalf("usage: args=%v exit=%d stdout=%s stderr=%s", arguments, code, output.String(), diagnostics.String())
		}
	}
	directory := filepath.Join(home, ".ssh", "ssh-keyflow")
	if err := os.MkdirAll(directory, 0700); err != nil {
		test.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "state.json"), []byte("{"), 0600); err != nil {
		test.Fatal(err)
	}
	output.Reset()
	diagnostics.Reset()
	code = cli.Run(context.Background(), []string{"keys", "list", "--json"}, options)
	if code != 1 || output.Len() != 0 || !strings.Contains(diagnostics.String(), "invalid state.json") || strings.Contains(diagnostics.String(), "Usage:") {
		test.Fatalf("runtime: exit=%d stdout=%s stderr=%s", code, output.String(), diagnostics.String())
	}
}

func TestListsShowUnmanagedDiscoveryWithoutWritingOrInvokingSSH(test *testing.T) {
	home := test.TempDir()
	files := map[string]string{
		"id_external.pub": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB outside\n",
		"config":          "Host outside\n HostName outside.example\nMatch exec never-run\n IdentityFile ~/.ssh/id_external\n",
	}
	if err := os.Mkdir(filepath.Join(home, ".ssh"), 0700); err != nil {
		test.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(home, ".ssh", name), []byte(content), 0600); err != nil {
			test.Fatal(err)
		}
	}
	for _, category := range []string{"keys", "hosts"} {
		for _, asJSON := range []bool{false, true} {
			arguments := []string{category, "list"}
			if asJSON {
				arguments = append(arguments, "--json")
			}
			var output, diagnostics bytes.Buffer
			code := cli.Run(context.Background(), arguments, cli.Options{
				Home: home, Input: strings.NewReader(""), Output: &output, Error: &diagnostics, Runner: forbiddenSSH{test},
			})
			if code != 0 || diagnostics.Len() != 0 {
				test.Fatalf("read-only list failed: %d %s", code, diagnostics.String())
			}
			discovery := output.String()
			if asJSON {
				var result struct{ Discovery string }
				if err := json.Unmarshal(output.Bytes(), &result); err != nil {
					test.Fatalf("JSON mixed diagnostics or text: %v\n%s", err, output.String())
				}
				discovery = result.Discovery
			} else if !strings.Contains(discovery, "Discovery (read-only; unmanaged files are not imported)") {
				test.Fatalf("text list does not distinguish unmanaged candidates:\n%s", discovery)
			}
			for _, expected := range []string{"unmanaged key candidate", "id_external.pub", "unmanaged Host outside", "Match exec is never executed"} {
				if !strings.Contains(discovery, expected) {
					test.Fatalf("list %v omitted %q:\n%s", arguments, expected, discovery)
				}
			}
		}
	}
	for name, content := range files {
		current, err := os.ReadFile(filepath.Join(home, ".ssh", name))
		if err != nil || string(current) != content {
			test.Fatalf("discovery changed unmanaged %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh", "ssh-keyflow")); !os.IsNotExist(err) {
		test.Fatal("discovery initialized management storage")
	}
}

type forbiddenSSH struct{ test *testing.T }

func (external forbiddenSSH) LookPath(string) (string, error) {
	external.test.Fatal("read-only listing probed SSH dependencies")
	return "", nil
}

func (external forbiddenSSH) Run(context.Context, app.Command) error {
	external.test.Fatal("read-only listing executed an SSH command")
	return nil
}
