package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"

	"ssh-keyflow/internal/state"
)

// PublicKey returns only the public algorithm and blob, never a file comment.
func PublicKey(home, reference string) (string, error) {
	if home == "" {
		return "", fmt.Errorf("Home is required")
	}
	document, err := state.Load(home)
	if err != nil {
		return "", err
	}
	key, err := usableKey(document, reference)
	if err != nil {
		return "", err
	}
	return publicText(key)
}

func publicText(key state.Key) (string, error) {
	data, err := os.ReadFile(key.PublicPath)
	if err != nil {
		return "", err
	}
	identity, err := fingerprint(data)
	if err != nil || identity != key.Fingerprint {
		return "", fmt.Errorf("requires review: public key fingerprint mismatch")
	}
	fields := strings.Fields(string(data))
	return fields[0] + " " + fields[1], nil
}

// Guide is text for the user to review and copy. It never runs a remote process,
// edits SSH configuration, or records registration/connection success.
func Guide(home, hostReference, keyReference string) (string, error) {
	guide, err := PrepareGuide(home, hostReference, keyReference)
	return guide.Text, err
}

// GuideStep separates terminal instructions from the script copied by the TUI.
type GuideStep struct {
	Title, Instructions, Script string
}

type SetupGuide struct {
	Text  string
	Steps []GuideStep
}

// PrepareGuide validates identities before producing any copyable scripts.
func PrepareGuide(home, hostReference, keyReference string) (SetupGuide, error) {
	if home == "" {
		return SetupGuide{}, fmt.Errorf("Home is required")
	}
	document, err := state.Load(home)
	if err != nil {
		return SetupGuide{}, err
	}
	hostIndex, err := destinationIndex(document, hostReference)
	if err != nil {
		return SetupGuide{}, err
	}
	host := document.Hosts[hostIndex]
	key, err := usableKey(document, keyReference)
	if err != nil {
		return SetupGuide{}, err
	}
	public, err := publicText(key)
	if err != nil {
		return SetupGuide{}, err
	}
	guide := SetupGuide{}
	var text strings.Builder
	fmt.Fprintf(&text, "Copy-paste setup: %s (%s@%s:%d)\nKey: %s [%s]\nFingerprint: %s\nRemote registration and connection: not checked.\nCopying this guide changes no files and records no success.\n\nPublic key (copy this complete line):\n%s\n\n", host.Alias, host.User, host.Hostname, host.Port, key.Name, key.ID, key.Fingerprint, public)
	var placement GuideStep
	switch host.OS {
	case "linux", "macos":
		placement = posixPlacement(host, public)
	case "windows":
		if host.WindowsAccount != "administrator" && host.WindowsAccount != "standard" {
			return SetupGuide{}, fmt.Errorf("confirm the destination Windows account type before preparing guidance")
		}
		placement = windowsPlacement(host, public)
	default:
		return SetupGuide{}, fmt.Errorf("confirm the destination OS before preparing guidance")
	}
	placement.Instructions += "\nCheck effective AuthorizedKeysFile/Match settings if customized. An SSH server and network access must already be available. Existing keys are preserved; an already present key is not appended again.\nManual removal: keep another working access method, then remove only the matching public-key line from the destination file. SSH KeyFlow does not verify removal."
	guide.Steps = append(guide.Steps, placement)
	notice := ""
	if key.PrivatePath == "" {
		notice = "Public-only key: use the matching private key on your client. No usable IdentityFile or connection command can be generated here.\n"
	} else {
		path := filepath.ToSlash(key.PrivatePath)
		for _, character := range path {
			if unicode.IsControl(character) {
				return SetupGuide{}, fmt.Errorf("private path contains control characters; cannot prepare a snippet")
			}
		}
		if strings.Contains(path, "${") {
			notice = "OpenSSH expands ${...} environment variables in identity paths. This private path cannot be represented literally here; no IdentityFile snippet or connection command was generated. Select a private-key path without ${...} yourself.\n"
		} else {
			// SSH expands percent tokens in IdentityFile even inside quotes.
			configPath := strings.NewReplacer("%", "%%", "\\", "\\\\", "\"", "\\\"").Replace(path)
			configUser := strings.NewReplacer("\\", "\\\\", "\"", "\\\"").Replace(host.User)
			config := fmt.Sprintf("Host %s\n    HostName %s\n    User \"%s\"\n    Port %d\n    IdentityFile \"%s\"\n    IdentitiesOnly yes\n", host.Alias, host.Hostname, configUser, host.Port, configPath)
			guide.Steps = append(guide.Steps, clientSetup(host, config, runtime.GOOS), GuideStep{
				Title:        "LOCAL terminal — connect (POSIX shell / PowerShell)",
				Instructions: "Run on this computer after completing the destination step and local config step. This opens an SSH connection; SSH KeyFlow does not run it or verify success.",
				Script:       fmt.Sprintf("ssh %s\n", host.Alias),
			})
		}
	}
	for i := range guide.Steps {
		guide.Steps[i].Instructions += fmt.Sprintf("\nKey: %s [%s]\nFingerprint: %s", key.Name, key.ID, key.Fingerprint)
		step := guide.Steps[i]
		fmt.Fprintf(&text, "Step %d/%d: %s\n%s\n\n%s\n", i+1, len(guide.Steps), step.Title, step.Instructions, step.Script)
	}
	text.WriteString(notice)
	fmt.Fprintf(&text, "\nLocal files (manual cleanup only; never delete a private key still needed for access):\nPublic: %s\nPrivate: %s\n", key.PublicPath, key.PrivatePath)
	// Show public-only/unsupported-path guidance on the single copyable step too.
	if notice != "" {
		guide.Steps[0].Instructions += "\n" + notice
	}
	guide.Text = text.String()
	return guide, nil
}
