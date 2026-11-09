package cli_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/x/term"

	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/cli"
	"ssh-keyflow/internal/state"
)

func TestHelpAndNonterminalStartupDoNotInitializeStorage(test *testing.T) {
	for _, arguments := range [][]string{nil, {"--help"}, {"keys"}, {"hosts"}, {"keys", "public", "--help"}, {"hosts", "add", "--help"}, {"hosts", "guide", "--help"}, {"resume", "--help"}} {
		test.Run(strings.Join(arguments, " "), func(test *testing.T) {
			home := test.TempDir()
			code, output, diagnostics := run(test, home, arguments...)
			if code != 0 || !strings.Contains(output, "Usage:") || diagnostics != "" {
				test.Fatalf("help: exit=%d stdout=%s stderr=%s", code, output, diagnostics)
			}
			assertEmptyHome(test, home)
		})
	}
}

func TestListJSONIsReadOnlyAndContainsOnlyRequestedRecords(test *testing.T) {
	for _, category := range []string{"hosts", "keys"} {
		home := test.TempDir()
		code, output, diagnostics := run(test, home, category, "list", "--json")
		var result map[string]json.RawMessage
		if code != 0 || json.Unmarshal([]byte(output), &result) != nil || string(result[category]) != "[]" || diagnostics != "" {
			test.Fatalf("list %s: exit=%d stdout=%s stderr=%s", category, code, output, diagnostics)
		}
		other := "creations"
		if category == "hosts" {
			other = "associations"
		}
		if len(result) != 2 || string(result[other]) != "[]" {
			test.Fatalf("unexpected list schema: %s", output)
		}
		assertEmptyHome(test, home)
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
		{[]string{"keys", "import", "--name", "existing"}, 2, "public"},
		{[]string{"keys", "import", "--name", "existing", "--public", "source.pub"}, 1, "terminal"},
		{[]string{"keys", "public"}, 2, "key"},
		{[]string{"keys", "remove"}, 2, "key"},
		{[]string{"keys", "remove", "--key", "known"}, 1, "terminal"},
		{[]string{"hosts", "remove"}, 2, "host"},
		{[]string{"hosts", "remove", "--host", "server"}, 1, "terminal"},
		{[]string{"hosts", "dissociate", "--host", "server"}, 2, "key"},
		{[]string{"hosts", "dissociate", "--key", "known"}, 2, "host"},
		{[]string{"hosts", "dissociate", "--host", "server", "--key", "known"}, 1, "terminal"},
		{[]string{"hosts", "guide", "--host", "server"}, 2, "key"},
		{[]string{"hosts", "associate", "--key", "known"}, 2, "host"},
		{[]string{"hosts", "associate", "--host", "server", "--key", "known"}, 1, "terminal"},
		{[]string{"hosts", "add", "--alias", "server", "--hostname", "server.example", "--user", "alice", "--os", "linux"}, 2, "key"},
		{[]string{"hosts", "add", "--alias", "server", "--hostname", "server.example", "--user", "alice", "--os", "windows", "--key", "known"}, 2, "account"},
		{[]string{"hosts", "add", "--alias", "server", "--hostname", "server.example", "--user", "alice", "--os", "linux", "--key", "known"}, 1, "terminal"},
		{[]string{"resume"}, 2, "id"},
		{[]string{"resume", "--id", "creation-123"}, 1, "terminal"},
		{[]string{"tui"}, 2, "terminal"},
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

func TestRemovedCommandsAndFlagsAreUsageErrorsWithoutSideEffects(test *testing.T) {
	cases := [][]string{
		{"keys", "delete", "--key", "known"},
		{"registrations", "list"},
		{"register", "--host", "server", "--key", "known"},
		{"unregister", "--registration", "registration-123"},
	}
	for _, flag := range []string{"--auth=manual", "--identity=key", "--configure", "--new-key=new"} {
		cases = append(cases, []string{"hosts", "add", "--alias", "server", "--hostname", "server.example", "--user", "alice", "--os", "linux", "--key", "known", flag})
		cases = append(cases, []string{"resume", "--id", "creation-123", flag})
	}
	for _, arguments := range cases {
		test.Run(strings.Join(arguments, " "), func(test *testing.T) {
			home := test.TempDir()
			code, output, diagnostics := run(test, home, arguments...)
			if code != 2 || output != "" || !strings.Contains(diagnostics, "unknown") || !strings.Contains(diagnostics, "Usage:") {
				test.Fatalf("removed interface: exit=%d stdout=%s stderr=%s", code, output, diagnostics)
			}
			assertEmptyHome(test, home)
		})
	}
}

func TestVersionUsageAndRuntimeFailuresHaveDistinctOutput(test *testing.T) {
	home := test.TempDir()
	var output, diagnostics bytes.Buffer
	options := cli.Options{Home: home, Version: "v0.3.0-test", Input: strings.NewReader(""), Output: &output, Error: &diagnostics}
	code := cli.Run(context.Background(), []string{"--version"}, options)
	if code != 0 || strings.TrimSpace(output.String()) != "SSH KeyFlow v0.3.0-test" || diagnostics.Len() != 0 {
		test.Fatalf("version: exit=%d stdout=%s stderr=%s", code, output.String(), diagnostics.String())
	}
	assertEmptyHome(test, home)
	for _, arguments := range [][]string{{"keys", "unknown"}, {"keys", "create"}, {"hosts", "list", "extra"}, {"keys", "public", "--key", "known", "extra"}, {"hosts", "guide", "--host", "server", "--key", "known", "extra"}} {
		code, stdout, stderr := run(test, home, arguments...)
		if code != 2 || stdout != "" || !strings.Contains(stderr, "Usage:") {
			test.Fatalf("usage: args=%v exit=%d stdout=%s stderr=%s", arguments, code, stdout, stderr)
		}
	}
	directory := state.Directory(home)
	if err := os.MkdirAll(directory, 0700); err != nil {
		test.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "state.json"), []byte("{"), 0600); err != nil {
		test.Fatal(err)
	}
	for _, arguments := range [][]string{{"keys", "list", "--json"}, {"keys", "public", "--key", "known"}, {"hosts", "guide", "--host", "server", "--key", "known"}} {
		code, stdout, stderr := run(test, home, arguments...)
		if code != 1 || stdout != "" || !strings.Contains(stderr, "invalid state.json") || strings.Contains(stderr, "Usage:") {
			test.Fatalf("runtime: args=%v exit=%d stdout=%s stderr=%s", arguments, code, stdout, stderr)
		}
	}
}

func TestCopyableOutputWorksRedirectedAndPreservesFiles(test *testing.T) {
	home := fixture(test)
	before := tree(test, home)
	public, err := app.PublicKey(home, "known")
	if err != nil {
		test.Fatal(err)
	}
	guide, err := app.Guide(home, "server", "known")
	if err != nil {
		test.Fatal(err)
	}
	for _, example := range []struct {
		arguments []string
		want      string
	}{
		{[]string{"keys", "public", "--key", "known"}, public},
		{[]string{"keys", "public", "--key", "key-known"}, public},
		{[]string{"hosts", "guide", "--host", "server", "--key", "known"}, guide},
		{[]string{"hosts", "guide", "--host", "host-server", "--key", "key-known"}, guide},
	} {
		code, output, diagnostics := run(test, home, example.arguments...)
		if code != 0 || output != example.want || output == "" || diagnostics != "" {
			test.Fatalf("copyable %v: exit=%d stdout=%q want=%q stderr=%s", example.arguments, code, output, example.want, diagnostics)
		}
	}
	for _, arguments := range [][]string{{"keys", "list"}, {"hosts", "list"}, {"keys", "list", "--json"}, {"hosts", "list", "--json"}} {
		code, output, diagnostics := run(test, home, arguments...)
		if code != 0 || output == "" || diagnostics != "" || strings.Contains(output, "Discovery") {
			test.Fatalf("saved list %v: exit=%d stdout=%s stderr=%s", arguments, code, output, diagnostics)
		}
		if arguments[0] == "hosts" && !strings.Contains(output, "key-known") {
			test.Fatalf("host list omitted saved association: %s", output)
		}
	}
	if after := tree(test, home); !reflect.DeepEqual(before, after) {
		test.Fatalf("read-only commands changed files: before=%v after=%v", before, after)
	}
}

func TestGuidePlatformFlagsAndUsageErrors(test *testing.T) {
	home := fixture(test)
	path := filepath.Join(state.Directory(home), "state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		test.Fatal(err)
	}
	var document state.State
	if err := json.Unmarshal(data, &document); err != nil {
		test.Fatal(err)
	}
	privatePath := filepath.Join(home, "private")
	private := []byte("private fixture")
	if err := os.WriteFile(privatePath, private, 0600); err != nil {
		test.Fatal(err)
	}
	digest := sha256.Sum256(private)
	document.Keys[0].PrivatePath = privatePath
	document.Keys[0].PrivateSHA256 = fmt.Sprintf("%x", digest)
	data, err = json.Marshal(document)
	if err != nil {
		test.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		test.Fatal(err)
	}
	before := tree(test, home)
	code, output, diagnostics := run(test, home, "hosts", "guide", "--host", "server", "--key", "known",
		"--local-os", "linux", "--identity-file", "/home/alice/.ssh/key", "--remote-os", "windows", "--windows-account", "administrator")
	for _, want := range []string{"Local platform: linux", "Destination platform: windows (administrator)", "administrators_authorized_keys", `IdentityFile "/home/alice/.ssh/key"`, "set -eu"} {
		if code != 0 || diagnostics != "" || !strings.Contains(output, want) {
			test.Fatalf("override lost %s: exit=%d stderr=%s stdout=%s", want, code, diagnostics, output)
		}
	}
	for _, flags := range [][]string{
		{"--local-os", "win"}, {"--remote-os", "darwin"},
		{"--windows-account", "root"}, {"--windows-account", "standard"},
		{"--remote-os", "windows"}, {"--identity-file", "relative/key"},
		{"--local-os", "linux", "--identity-file", `C:\Users\alice\key`},
	} {
		args := append([]string{"hosts", "guide", "--host", "server", "--key", "known"}, flags...)
		code, output, diagnostics := run(test, home, args...)
		if code != 2 || output != "" || !strings.Contains(diagnostics, "Usage:") {
			test.Fatalf("invalid guide flags %v: exit=%d stdout=%s stderr=%s", flags, code, output, diagnostics)
		}
	}
	if !reflect.DeepEqual(before, tree(test, home)) {
		test.Fatal("guide flags changed saved files")
	}
}

func TestReviewDiagnosticDoesNotCorruptJSON(test *testing.T) {
	home := fixture(test)
	if err := os.Remove(filepath.Join(home, ".ssh", "known.pub")); err != nil {
		test.Fatal(err)
	}
	before := tree(test, home)
	code, output, diagnostics := run(test, home, "keys", "list", "--json")
	var result map[string]json.RawMessage
	if code != 0 || json.Unmarshal([]byte(output), &result) != nil || diagnostics == "" {
		test.Fatalf("review: exit=%d stdout=%s stderr=%s", code, output, diagnostics)
	}
	for _, arguments := range [][]string{{"keys", "public", "--key", "known"}, {"hosts", "guide", "--host", "server", "--key", "known"}} {
		code, output, diagnostics := run(test, home, arguments...)
		if code != 1 || output != "" || diagnostics == "" {
			test.Fatalf("missing key: exit=%d stdout=%s stderr=%s", code, output, diagnostics)
		}
	}
	if after := tree(test, home); !reflect.DeepEqual(before, after) {
		test.Fatal("failed reads changed files")
	}
}

func TestCopyableOutputReportsWriterFailure(test *testing.T) {
	home := fixture(test)
	for _, arguments := range [][]string{{"keys", "list", "--json"}, {"hosts", "list"}, {"keys", "public", "--key", "known"}, {"hosts", "guide", "--host", "server", "--key", "known"}} {
		var diagnostics bytes.Buffer
		code := cli.Run(context.Background(), arguments, cli.Options{
			Home: home, Input: strings.NewReader(""), Output: brokenWriter{}, Error: &diagnostics, Runner: forbiddenSSH{test},
		})
		if code != 1 || !strings.Contains(diagnostics.String(), "output unavailable") || strings.Contains(diagnostics.String(), "Usage:") {
			test.Fatalf("output failure %v: exit=%d stderr=%s", arguments, code, diagnostics.String())
		}
	}
}

func TestTerminalInputWithRedirectedOutputShowsHelp(test *testing.T) {
	if !term.IsTerminal(os.Stdin.Fd()) {
		test.Skip("requires terminal stdin; run with a native PTY")
	}
	for _, arguments := range [][]string{nil, {"tui"}} {
		var output, diagnostics bytes.Buffer
		code := cli.Run(context.Background(), arguments, cli.Options{Home: test.TempDir(), Input: os.Stdin, Output: &output, Error: &diagnostics})
		if len(arguments) == 0 {
			if code != 0 || !strings.Contains(output.String(), "Usage:") || diagnostics.Len() != 0 {
				test.Fatalf("automatic startup: exit=%d stdout=%s stderr=%s", code, output.String(), diagnostics.String())
			}
		} else if code != 2 || !strings.Contains(diagnostics.String(), "terminal") {
			test.Fatalf("explicit tui: exit=%d stderr=%s", code, diagnostics.String())
		}
	}
}

func run(test *testing.T, home string, arguments ...string) (int, string, string) {
	test.Helper()
	var output, diagnostics bytes.Buffer
	code := cli.Run(context.Background(), arguments, cli.Options{
		Home: home, Input: strings.NewReader("yes\n"), Output: &output, Error: &diagnostics, Runner: forbiddenSSH{test},
	})
	return code, output.String(), diagnostics.String()
}

func assertEmptyHome(test *testing.T, home string) {
	test.Helper()
	entries, err := os.ReadDir(home)
	if err != nil || len(entries) != 0 {
		test.Fatalf("command initialized storage: %v %v", entries, err)
	}
}

func fixture(test *testing.T) string {
	test.Helper()
	home := test.TempDir()
	directory := state.Directory(home)
	if err := os.MkdirAll(directory, 0700); err != nil {
		test.Fatal(err)
	}
	const public = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB known\n"
	blob, err := base64.StdEncoding.DecodeString(strings.Fields(public)[1])
	if err != nil {
		test.Fatal(err)
	}
	digest := sha256.Sum256(blob)
	publicPath := filepath.Join(home, ".ssh", "known.pub")
	document := state.State{
		Version:       2,
		Keys:          []state.Key{{ID: "key-known", Name: "known", Origin: "imported", ImportedAt: "2026-11-08T00:00:00Z", Fingerprint: "SHA256:" + base64.RawStdEncoding.EncodeToString(digest[:]), PublicPath: publicPath, ConnectionCheck: "unverified"}},
		Hosts:         []state.Host{{ID: "host-server", Alias: "server", Hostname: "server.example", User: "alice", Port: 2222, OS: "linux"}},
		Registrations: []state.Registration{},
		Associations:  []state.Association{{HostID: "host-server", KeyID: "key-known"}},
	}
	data, err := json.Marshal(document)
	if err != nil {
		test.Fatal(err)
	}
	for path, content := range map[string][]byte{
		filepath.Join(directory, "state.json"):        data,
		publicPath:                                    []byte(public),
		filepath.Join(home, ".ssh", "config"):         []byte("Host unmanaged\n Match exec never-run\n Include preserved.conf\n"),
		filepath.Join(home, ".ssh", "preserved.conf"): []byte("# do not edit\n"),
	} {
		if err := os.WriteFile(path, content, 0600); err != nil {
			test.Fatal(err)
		}
	}
	return home
}

func tree(test *testing.T, home string) map[string]string {
	test.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(home, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			result[path] = "directory"
			return nil
		}
		data, err := os.ReadFile(path)
		result[path] = string(data)
		return err
	})
	if err != nil {
		test.Fatal(err)
	}
	return result
}

type forbiddenSSH struct{ test *testing.T }

func (external forbiddenSSH) LookPath(string) (string, error) {
	external.test.Fatal("read-only or rejected command probed dependencies")
	return "", nil
}

func (external forbiddenSSH) Run(context.Context, app.Command) error {
	external.test.Fatal("read-only or rejected command executed an external command")
	return nil
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("output unavailable") }
