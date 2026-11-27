package cli_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/cli"
	"ssh-keyflow/internal/state"
)

func TestHelpAndStartupDoNotInitializeStorage(test *testing.T) {
	for _, arguments := range [][]string{nil, {"--help"}, {"keys"}, {"keys", "public", "--help"}, {"keys", "import", "--help"}, {"resume", "--help"}} {
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

func TestListJSONIsReadOnlyAndContainsOnlyKeyRecords(test *testing.T) {
	home := test.TempDir()
	code, output, diagnostics := run(test, home, "keys", "list", "--json")
	var result map[string]json.RawMessage
	if code != 0 || json.Unmarshal([]byte(output), &result) != nil || diagnostics != "" {
		test.Fatalf("list: exit=%d stdout=%s stderr=%s", code, output, diagnostics)
	}
	if len(result) != 2 || string(result["keys"]) != "[]" || string(result["creations"]) != "[]" {
		test.Fatalf("unexpected list schema: %s", output)
	}
	assertEmptyHome(test, home)
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
		{[]string{"keys", "import", "--name", "existing", "--public", "source.pub"}, 2, "private"},
		{[]string{"keys", "import", "--name", "existing", "--public", "source.pub", "--private", "source"}, 1, "terminal"},
		{[]string{"keys", "public"}, 2, "key"},
		{[]string{"keys", "remove"}, 2, "key"},
		{[]string{"keys", "remove", "--key", "known"}, 1, "terminal"},
		{[]string{"resume"}, 2, "id"},
		{[]string{"resume", "--id", "creation-123"}, 1, "terminal"},
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

func TestRetiredCommandsAndFlagsAreUsageErrorsWithoutSideEffects(test *testing.T) {
	cases := [][]string{
		{"tui"},
		{"hosts", "list"},
		{"hosts", "add", "--alias", "server", "--hostname", "server.example", "--user", "alice", "--os", "linux", "--key", "known"},
		{"hosts", "guide", "--host", "server", "--key", "known"},
		{"hosts", "pin", "--host", "server"},
		{"keys", "delete", "--key", "known"},
		{"register", "--host", "server", "--key", "known"},
		{"keys", "import", "--name", "n", "--public", "p", "--private", "q", "--public-only"},
	}
	for _, arguments := range cases {
		test.Run(strings.Join(arguments, " "), func(test *testing.T) {
			home := test.TempDir()
			code, output, diagnostics := run(test, home, arguments...)
			if code != 2 || output != "" || !strings.Contains(diagnostics, "unknown") || !strings.Contains(diagnostics, "Usage:") {
				test.Fatalf("retired interface: exit=%d stdout=%s stderr=%s", code, output, diagnostics)
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
	for _, arguments := range [][]string{{"keys", "unknown"}, {"keys", "create"}, {"keys", "list", "extra"}, {"keys", "public", "--key", "known", "extra"}} {
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
	for _, arguments := range [][]string{{"keys", "list", "--json"}, {"keys", "public", "--key", "known"}} {
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
	for _, reference := range []string{"known", "key-known"} {
		code, output, diagnostics := run(test, home, "keys", "public", "--key", reference)
		if code != 0 || output != public+"\n" || strings.Contains(output, "comment") || diagnostics != "" {
			test.Fatalf("public %s: exit=%d stdout=%q stderr=%s", reference, code, output, diagnostics)
		}
	}
	for _, arguments := range [][]string{{"keys", "list"}, {"keys", "list", "--json"}} {
		code, output, diagnostics := run(test, home, arguments...)
		if code != 0 || !strings.Contains(output, "key-known") || diagnostics != "" {
			test.Fatalf("saved list %v: exit=%d stdout=%s stderr=%s", arguments, code, output, diagnostics)
		}
	}
	if after := tree(test, home); !reflect.DeepEqual(before, after) {
		test.Fatalf("read-only commands changed files: before=%v after=%v", before, after)
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
	code, output, diagnostics = run(test, home, "keys", "public", "--key", "known")
	if code != 1 || output != "" || diagnostics == "" {
		test.Fatalf("missing key: exit=%d stdout=%s stderr=%s", code, output, diagnostics)
	}
	if after := tree(test, home); !reflect.DeepEqual(before, after) {
		test.Fatal("failed reads changed files")
	}
}

func TestLegacyStateListShowsMigrationNoticeWithoutWriting(test *testing.T) {
	home := fixture(test)
	path := filepath.Join(state.Directory(home), "state.json")
	legacy := `{"version":2,"keys":[],"hosts":[{"id":"host-1","alias":"server","hostname":"server.example","user":"alice","port":22,"os":"linux"}],"registrations":[]}`
	if err := os.WriteFile(path, []byte(legacy), 0600); err != nil {
		test.Fatal(err)
	}
	before := tree(test, home)
	code, output, diagnostics := run(test, home, "keys", "list", "--json")
	if code != 0 || !strings.Contains(diagnostics, "schema v2") || !strings.Contains(diagnostics, "1 destination") || !strings.Contains(output, `"keys":[]`) {
		test.Fatalf("legacy list: exit=%d stdout=%s stderr=%s", code, output, diagnostics)
	}
	if !reflect.DeepEqual(before, tree(test, home)) {
		test.Fatal("listing a legacy state changed files")
	}
}

func TestCopyableOutputReportsWriterFailure(test *testing.T) {
	home := fixture(test)
	for _, arguments := range [][]string{{"keys", "list", "--json"}, {"keys", "list"}, {"keys", "public", "--key", "known"}} {
		var diagnostics bytes.Buffer
		code := cli.Run(context.Background(), arguments, cli.Options{
			Home: home, Input: strings.NewReader(""), Output: brokenWriter{}, Error: &diagnostics, Runner: forbiddenSSH{test},
		})
		if code != 1 || !strings.Contains(diagnostics.String(), "output unavailable") || strings.Contains(diagnostics.String(), "Usage:") {
			test.Fatalf("output failure %v: exit=%d stderr=%s", arguments, code, diagnostics.String())
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
	const public = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB comment\n"
	const private = "private fixture"
	blob, err := base64.StdEncoding.DecodeString(strings.Fields(public)[1])
	if err != nil {
		test.Fatal(err)
	}
	digest := sha256.Sum256(blob)
	privateDigest := sha256.Sum256([]byte(private))
	publicPath := filepath.Join(home, ".ssh", "known.pub")
	privatePath := filepath.Join(home, ".ssh", "known")
	document := state.State{
		Version: state.Version,
		Keys: []state.Key{{
			ID: "key-known", Name: "known", Origin: "imported", ImportedAt: "2026-11-08T00:00:00Z",
			Fingerprint: "SHA256:" + base64.RawStdEncoding.EncodeToString(digest[:]),
			PublicPath:  publicPath, PrivatePath: privatePath, PrivateSHA256: hex.EncodeToString(privateDigest[:]),
		}},
	}
	data, err := json.Marshal(document)
	if err != nil {
		test.Fatal(err)
	}
	for path, content := range map[string][]byte{
		filepath.Join(directory, "state.json"): data,
		publicPath:                             []byte(public),
		privatePath:                            []byte(private),
		filepath.Join(home, ".ssh", "config"):  []byte("Host unmanaged\n Include preserved.conf\n"),
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
