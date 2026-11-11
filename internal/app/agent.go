package app

import (
	"bytes"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
)

type protection int

const (
	undetermined protection = iota
	unprotected
	protected
)

// passphraseProtection reads only the private-key format header. It never
// decrypts, prompts, starts a process or returns key material.
func passphraseProtection(path string) (protection, string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return undetermined, "the private key file could not be read"
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return undetermined, "the private key format is not recognized"
	}
	switch block.Type {
	case "OPENSSH PRIVATE KEY":
		const magic = "openssh-key-v1\x00"
		body := block.Bytes
		if !bytes.HasPrefix(body, []byte(magic)) || len(body) < len(magic)+4 {
			return undetermined, "the OpenSSH private key header is invalid"
		}
		body = body[len(magic):]
		length := binary.BigEndian.Uint32(body)
		if length == 0 || uint64(length) > uint64(len(body)-4) {
			return undetermined, "the OpenSSH private key header is invalid"
		}
		if string(body[4:4+length]) == "none" {
			return unprotected, ""
		}
		return protected, ""
	case "ENCRYPTED PRIVATE KEY":
		return protected, ""
	case "PRIVATE KEY", "RSA PRIVATE KEY", "DSA PRIVATE KEY", "EC PRIVATE KEY":
		if strings.Contains(block.Headers["Proc-Type"], "ENCRYPTED") {
			return protected, ""
		}
		return unprotected, ""
	}
	return undetermined, "the private key format is not recognized"
}

func posixQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

// agentSetup prepares a copy-paste step; SSH KeyFlow never runs it or starts an agent.
func agentSetup(platform, identityPath string) GuideStep {
	const loaded = "AddKeysToAgent in the config step loads the key into the agent at the first connection that asks for its passphrase. It stays loaded until the agent stops."
	switch platform {
	case "windows":
		return GuideStep{
			Title:        "LOCAL computer — windows — ssh-agent (PowerShell)",
			Instructions: "Open PowerShell as Administrator in the selected local terminal. The script sets the OpenSSH Authentication Agent service (ssh-agent) to start automatically and starts it; it changes nothing when that is already done. Windows OpenSSH keeps added keys in the service across restarts.\n" + loaded + "\nOnly Windows OpenSSH (System32\\OpenSSH\\ssh.exe) uses this service; another ssh on PATH, such as Git for Windows, does not.",
			Script: `& {
    $ErrorActionPreference = 'Stop'
    $service = Get-Service -Name ssh-agent -ErrorAction SilentlyContinue
    if (-not $service) {
        throw 'The ssh-agent service is not installed. Add the Windows OpenSSH Client optional feature first.'
    }
    $expected = Join-Path $env:SystemRoot 'System32\OpenSSH\ssh.exe'
    $ssh = Get-Command -Name ssh -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($ssh -and $ssh.Source -ine $expected) {
        Write-Warning "ssh on PATH is $($ssh.Source), not $expected; it does not use the Windows ssh-agent service."
    }
    if ($service.StartType -eq 'Automatic' -and $service.Status -eq 'Running') {
        Write-Output 'ssh-agent service already starts automatically and is running. No change.'
        return
    }
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    if (-not ([Security.Principal.WindowsPrincipal]::new($identity)).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        throw 'Open PowerShell as Administrator to configure the ssh-agent service.'
    }
    if ($service.StartType -ne 'Automatic') { Set-Service -Name ssh-agent -StartupType Automatic }
    if ($service.Status -ne 'Running') { Start-Service -Name ssh-agent }
    Write-Output 'ssh-agent service now starts automatically and is running.'
}
`,
		}
	case "macos":
		return GuideStep{
			Title:        "LOCAL computer — macos — ssh-agent (POSIX shell)",
			Instructions: "Run in the selected local terminal. macOS launchd starts ssh-agent for each login session; this script only checks that it is reachable and changes nothing.\n" + loaded + " UseKeychain stores the passphrase in the login Keychain.\nTo load the key now instead, run: ssh-add --apple-use-keychain " + posixQuote(identityPath),
			Script:       posixAgentCheck("Sign out and back in, then check that SSH_AUTH_SOCK is not overridden in your shell profile."),
		}
	}
	return GuideStep{
		Title: "LOCAL computer — linux — ssh-agent (POSIX shell, manual guide)",
		Instructions: "Run in the selected local terminal. The script only checks whether SSH_AUTH_SOCK points to a reachable ssh-agent; it installs nothing and edits no files.\n" + loaded + "\nIf no agent is reachable, choose one option yourself:\n" +
			"- Desktop keyring: GNOME Keyring / gcr-ssh-agent or KDE Wallet often provide an agent and set SSH_AUTH_SOCK after login. Enable it in your desktop settings.\n" +
			"- systemd user service: if your distribution ships an ssh-agent user unit, run systemctl --user enable --now on it (or write one that runs ssh-agent -D -a \"$XDG_RUNTIME_DIR/ssh-agent.socket\"), and set SSH_AUTH_SOCK to that socket in ~/.config/environment.d/.\n" +
			"- shell profile: start one agent per login from ~/.profile, for example eval \"$(ssh-agent -s)\", and stop it at logout. A new agent per shell does not share loaded keys.\n" +
			"Run this check again after signing in.",
		Script: posixAgentCheck("Choose one of the options in this step's instructions, sign in again, and rerun this check."),
	}
}

func posixAgentCheck(advice string) string {
	return fmt.Sprintf(`(
set -u
command -v ssh-add >/dev/null 2>&1 || { printf 'ssh-add not found. Install the OpenSSH client first.\n' >&2; exit 1; }
status=0
ssh-add -l >/dev/null 2>&1 || status=$?
case "$status" in
0|1) printf 'ssh-agent is reachable (SSH_AUTH_SOCK=%%s). No change.\n' "${SSH_AUTH_SOCK:-}" ;;
*) printf 'No reachable ssh-agent in this terminal. %%s\n' %s >&2; exit 1 ;;
esac
)
`, posixQuote(advice))
}
