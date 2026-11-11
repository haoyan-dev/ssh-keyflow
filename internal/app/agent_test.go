package app

import (
	"encoding/binary"
	"encoding/pem"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func openSSHPrivate(cipher string) []byte {
	body := []byte("openssh-key-v1\x00")
	for _, field := range []string{cipher, "kdf", ""} {
		body = binary.BigEndian.AppendUint32(body, uint32(len(field)))
		body = append(body, field...)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "OPENSSH PRIVATE KEY", Bytes: body})
}

func TestPassphraseProtectionReadsOnlyKeyFormat(t *testing.T) {
	for name, test := range map[string]struct {
		data []byte
		want protection
	}{
		"openssh-none":      {openSSHPrivate("none"), unprotected},
		"openssh-aes":       {openSSHPrivate("aes256-ctr"), protected},
		"pem-encrypted":     {pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Headers: map[string]string{"Proc-Type": "4,ENCRYPTED", "DEK-Info": "AES-128-CBC,00"}, Bytes: []byte{1}}), protected},
		"pem-plain":         {pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: []byte{1}}), unprotected},
		"pkcs8-encrypted":   {pem.EncodeToMemory(&pem.Block{Type: "ENCRYPTED PRIVATE KEY", Bytes: []byte{1}}), protected},
		"pkcs8-plain":       {pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte{1}}), unprotected},
		"not-a-key":         {[]byte("private fixture"), undetermined},
		"openssh-truncated": {pem.EncodeToMemory(&pem.Block{Type: "OPENSSH PRIVATE KEY", Bytes: []byte("openssh-key-v1\x00\x00\x00")}), undetermined},
		"openssh-bad-magic": {pem.EncodeToMemory(&pem.Block{Type: "OPENSSH PRIVATE KEY", Bytes: []byte("other")}), undetermined},
		"public-key-pem":    {pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: []byte{1}}), undetermined},
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "key")
			if err := os.WriteFile(path, test.data, 0600); err != nil {
				t.Fatal(err)
			}
			if got, _ := passphraseProtection(path); got != test.want {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
	if got, reason := passphraseProtection(filepath.Join(t.TempDir(), "missing")); got != undetermined || reason == "" {
		t.Fatal("missing key was not undetermined with a reason")
	}
}

func TestPassphraseProtectionMatchesNativeSSHKeygen(t *testing.T) {
	keygen, err := exec.LookPath("ssh-keygen")
	if err != nil {
		t.Skip("native ssh-keygen unavailable")
	}
	for passphrase, want := range map[string]protection{"": unprotected, "fixture passphrase": protected} {
		path := filepath.Join(t.TempDir(), "key")
		if output, err := exec.Command(keygen, "-q", "-t", "ed25519", "-N", passphrase, "-C", "", "-f", path).CombinedOutput(); err != nil {
			t.Fatal(err, string(output))
		}
		if got, reason := passphraseProtection(path); got != want {
			t.Fatalf("passphrase %t: got %v (%s), want %v", passphrase != "", got, reason, want)
		}
	}
}

func TestAgentStepsArePerPlatformAndReadOnlyOnLinux(t *testing.T) {
	windows := agentSetup("windows", "C:/Users/alice/key")
	for _, want := range []string{"Get-Service -Name ssh-agent", "Set-Service -Name ssh-agent -StartupType Automatic", "Start-Service -Name ssh-agent", "System32\\OpenSSH\\ssh.exe", "Administrator"} {
		if !strings.Contains(windows.Script, want) {
			t.Fatalf("Windows agent step missing %q", want)
		}
	}
	macos := agentSetup("macos", "/Users/alice/key")
	if !strings.Contains(macos.Instructions, "ssh-add --apple-use-keychain '/Users/alice/key'") || !strings.Contains(macos.Instructions, "Keychain") {
		t.Fatal("macOS step did not offer the Keychain load", macos.Instructions)
	}
	linux := agentSetup("linux", "/home/alice/key")
	for _, forbidden := range []string{"systemctl", ">>", "mkdir", "environment.d/"} {
		if strings.Contains(linux.Script, forbidden) {
			t.Fatalf("Linux step changes files or services: %q", forbidden)
		}
	}
	for _, want := range []string{"systemd user service", "shell profile", "keyring"} {
		if !strings.Contains(linux.Instructions, want) {
			t.Fatalf("Linux manual guide missing %q", want)
		}
	}
	for _, step := range []GuideStep{macos, linux} {
		if !strings.Contains(step.Script, "ssh-add -l") || strings.Contains(step.Script, "ssh-add '") {
			t.Fatal("POSIX step must only check the agent", step.Script)
		}
	}
}

func TestPOSIXAgentCheckReportsReachableAndMissingAgents(t *testing.T) {
	shell := posixShell(t)
	if _, err := exec.LookPath("ssh-agent"); err != nil {
		t.Skip("native ssh-agent unavailable")
	}
	for _, platform := range []string{"linux", "macos"} {
		script := agentSetup(platform, "/home/alice/key").Script
		missing := exec.Command(shell, "-c", script)
		missing.Env = append(os.Environ(), "SSH_AUTH_SOCK="+filepath.ToSlash(filepath.Join(t.TempDir(), "absent.sock")))
		if output, err := missing.CombinedOutput(); err == nil || !strings.Contains(string(output), "No reachable ssh-agent") {
			t.Fatalf("%s: missing agent not reported: %v %s", platform, err, output)
		}
	}
	// A disposable agent on a temporary socket; the user's agent is never used.
	// A short directory keeps the path within the Unix socket length limit.
	dir, err := os.MkdirTemp("", "kfa")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socket := filepath.ToSlash(filepath.Join(dir, "s"))
	agent := exec.Command(shell, "-c", `exec ssh-agent -D -a "$1"`, "sh", socket)
	if err := agent.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = agent.Process.Kill(); _ = agent.Wait() }()
	for _, platform := range []string{"linux", "macos"} {
		// Retry while the agent creates its socket.
		for attempt := 1; ; attempt++ {
			check := exec.Command(shell, "-c", agentSetup(platform, "/home/alice/key").Script)
			check.Env = append(os.Environ(), "SSH_AUTH_SOCK="+socket)
			output, err := check.CombinedOutput()
			if err == nil && strings.Contains(string(output), "ssh-agent is reachable") {
				break
			}
			if attempt == 50 {
				t.Fatalf("%s: reachable agent not reported: %v %s", platform, err, output)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func TestWindowsAgentScriptRunsWithoutChangingTheService(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows PowerShell unavailable")
	}
	query := `$s = Get-Service -Name ssh-agent -ErrorAction SilentlyContinue; if ($s) { "$($s.StartType)/$($s.Status)/" + ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator) } else { 'absent' }`
	for _, name := range []string{"powershell.exe", "pwsh.exe"} {
		shell, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		before, err := exec.Command(shell, "-NoProfile", "-NonInteractive", "-Command", query).Output()
		if err != nil {
			t.Fatal(err)
		}
		state := strings.TrimSpace(string(before))
		configured := strings.HasPrefix(state, "Automatic/Running/")
		if !configured && !strings.HasSuffix(state, "/False") && state != "absent" {
			t.Skip("elevated test process would change the machine's ssh-agent service")
		}
		output, err := runSnippet(t, shell, agentSetup("windows", "C:/Users/alice/key").Script)
		switch {
		case state == "absent":
			if err == nil || !strings.Contains(string(output), "not installed") {
				t.Fatalf("%s: missing service not reported: %v %s", name, err, output)
			}
		case configured:
			if err != nil || !strings.Contains(string(output), "No change") {
				t.Fatalf("%s: configured service not reported: %v %s", name, err, output)
			}
		default:
			if err == nil || !strings.Contains(string(output), "as Administrator") {
				t.Fatalf("%s: non-elevated change not refused: %v %s", name, err, output)
			}
		}
		after, err := exec.Command(shell, "-NoProfile", "-NonInteractive", "-Command", query).Output()
		if err != nil || strings.TrimSpace(string(after)) != state {
			t.Fatalf("%s: service state changed: %s -> %s", name, state, after)
		}
	}
}
