package app_test

import (
	"bytes"
	"encoding/binary"
	"encoding/pem"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/state"
)

// privateWithCipher writes only the OpenSSH key header fields that detection reads.
func privateWithCipher(cipher string) string {
	body := []byte("openssh-key-v1\x00")
	for _, field := range []string{cipher, "kdf", ""} {
		body = binary.BigEndian.AppendUint32(body, uint32(len(field)))
		body = append(body, field...)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "OPENSSH PRIVATE KEY", Bytes: body}))
}

func importedPair(t *testing.T, home, name, private string) state.Key {
	t.Helper()
	path := filepath.Join(home, name+".pub")
	write(t, path, publicKey)
	write(t, strings.TrimSuffix(path, ".pub"), private)
	if out, err := run(t, home, app.Request{Operation: "import", Name: name, PublicPath: path, PrivatePath: strings.TrimSuffix(path, ".pub")}); err != nil {
		t.Fatalf("import: %v %s", err, out)
	}
	doc := load(t, home)
	return doc.Keys[len(doc.Keys)-1]
}

func agentGuide(t *testing.T, private string, options app.GuideOptions) app.SetupGuide {
	t.Helper()
	home := tempHome(t)
	key := importedPair(t, home, "pair", private)
	if _, err := run(t, home, app.Request{Operation: "add", Key: key.ID, Destination: destination("server", "linux", "")}); err != nil {
		t.Fatal(err)
	}
	before := read(t, filepath.Join(state.Directory(home), "state.json"))
	guide, err := app.PrepareGuideWithOptions(home, "server", key.ID, options)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, read(t, filepath.Join(state.Directory(home), "state.json"))) {
		t.Fatal("guide changed state")
	}
	guide.Steps = withoutPinSteps(guide.Steps)
	return guide
}

// withoutPinSteps drops the unpinned print and pin steps (covered in
// pins_test.go) so the other guide tests keep their step positions.
func withoutPinSteps(steps []app.GuideStep) []app.GuideStep {
	var kept []app.GuideStep
	for _, step := range steps {
		if !strings.HasSuffix(step.Title, "print host keys") && !strings.HasSuffix(step.Title, "pin host keys") {
			kept = append(kept, step)
		}
	}
	return kept
}

func TestGuideWithoutPassphraseKeepsCurrentSteps(t *testing.T) {
	guide := agentGuide(t, privateWithCipher("none"), app.GuideOptions{})
	if len(guide.Steps) != 3 || strings.Contains(guide.Text, "AddKeysToAgent") || strings.Contains(guide.Text, "ssh-agent") {
		t.Fatal("unprotected key received agent setup", guide.Text)
	}
}

// Detection reads the managed key, so only the running platform can be protected
// rather than undetermined; other platforms are covered by explicit overrides.
func TestGuideWithPassphraseAddsAgentStep(t *testing.T) {
	guide := agentGuide(t, privateWithCipher("aes256-ctr"), app.GuideOptions{})
	assertAgentGuide(t, guide, app.LocalPlatform(), false)
}

func TestGuideAddsAgentStepWhenProtectionCannotBeChecked(t *testing.T) {
	guide := agentGuide(t, "private fixture", app.GuideOptions{})
	assertAgentGuide(t, guide, app.LocalPlatform(), true)
	for _, local := range []string{"linux", "macos", "windows"} {
		// The managed key has no passphrase, but the supplied terminal path is not inspected.
		guide := agentGuide(t, privateWithCipher("none"), app.GuideOptions{LocalOS: local, IdentityFile: terminalPath(local)})
		assertAgentGuide(t, guide, local, true)
	}
}

// terminalPath is a private-key path as seen by a terminal on another platform.
func terminalPath(local string) string {
	if local == "windows" {
		return `C:\Users\alice\id_ed25519`
	}
	return "/home/alice/id_ed25519"
}

func assertAgentGuide(t *testing.T, guide app.SetupGuide, local string, undetermined bool) {
	t.Helper()
	if len(guide.Steps) != 4 {
		t.Fatalf("%s: want destination, config, agent and connect steps: %s", local, guide.Text)
	}
	config, agent := guide.Steps[1], guide.Steps[2]
	if !strings.Contains(config.Script, "    AddKeysToAgent yes\n") {
		t.Fatal("config lacks AddKeysToAgent", config.Script)
	}
	keychain := strings.Contains(config.Script, "    IgnoreUnknown UseKeychain\n    UseKeychain yes\n")
	if keychain != (local == "macos") || (local != "macos" && strings.Contains(config.Script, "UseKeychain")) {
		t.Fatal("UseKeychain must be macOS only", config.Script)
	}
	if !strings.Contains(agent.Title, "ssh-agent") || !strings.Contains(agent.Title, local) {
		t.Fatal("agent step title", agent.Title)
	}
	if strings.HasPrefix(agent.Script, "& {") != (local == "windows") {
		t.Fatal("agent script language does not match local platform")
	}
	if guide.Steps[3].Script != "ssh server\n" {
		t.Fatal("connection step moved")
	}
	notice := "could not be checked"
	if strings.Contains(agent.Instructions, notice) != undetermined || strings.Contains(config.Instructions, notice) != undetermined || strings.Contains(guide.Text, notice) != undetermined {
		t.Fatal("undetermined notice mismatch", agent.Instructions)
	}
	if strings.Contains(guide.Text, "openssh-key-v1") || strings.Contains(guide.Text, "PRIVATE KEY") {
		t.Fatal("guide disclosed private key material")
	}
}

func TestGuidePublicOnlyKeyHasNoAgentStep(t *testing.T) {
	home := tempHome(t)
	key := imported(t, home, "public", publicKey, false)
	if _, err := run(t, home, app.Request{Operation: "add", Key: key.ID, Destination: destination("server", "linux", "")}); err != nil {
		t.Fatal(err)
	}
	guide, err := app.PrepareGuide(home, "server", key.ID)
	steps := withoutPinSteps(guide.Steps)
	// Destination, config and connect: the agent belongs to the key holder.
	if err != nil || len(steps) != 3 || strings.Contains(guide.Text, "ssh-agent -") || strings.Contains(guide.Text, "AddKeysToAgent") || strings.Contains(guide.Text, "Start-Service") {
		t.Fatal(err, guide.Text)
	}
}

func TestNativeOpenSSHParsesAgentConfigOnEveryPlatform(t *testing.T) {
	ssh, err := exec.LookPath("ssh")
	if err != nil {
		t.Skip("native OpenSSH unavailable")
	}
	for _, local := range []string{"linux", "macos", "windows"} {
		guide := agentGuide(t, privateWithCipher("aes256-ctr"), app.GuideOptions{LocalOS: local, IdentityFile: terminalPath(local)})
		text := guide.Steps[1].Script
		start := strings.Index(text, "Host server\n")
		end := strings.Index(text, "    IdentitiesOnly yes\n")
		if start < 0 || end < start {
			t.Fatalf("%s: Host block not found: %s", local, text)
		}
		config := filepath.Join(t.TempDir(), "config")
		write(t, config, text[start:end+len("    IdentitiesOnly yes\n")])
		output, err := exec.Command(ssh, "-G", "-F", config, "server").CombinedOutput()
		if err != nil || !strings.Contains(string(output), "addkeystoagent true") {
			t.Fatalf("%s: native parser rejected agent config: %v %s", local, err, output)
		}
	}
}
