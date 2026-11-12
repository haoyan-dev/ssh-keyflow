package app

import (
	"fmt"
	"os"
	"path/filepath"
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
	return PrepareGuideWithOptions(home, hostReference, keyReference, GuideOptions{})
}

// PrepareGuideWithOptions selects the terminals independently of the app's OS.
func PrepareGuideWithOptions(home, hostReference, keyReference string, options GuideOptions) (SetupGuide, error) {
	if home == "" {
		return SetupGuide{}, fmt.Errorf("Home is required")
	}
	if err := options.Validate(); err != nil {
		return SetupGuide{}, err
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
	if options.RemoteOS != "" && options.RemoteOS != host.OS {
		host.OS, host.WindowsAccount = options.RemoteOS, ""
	}
	if options.WindowsAccount != "" {
		if host.OS != "windows" {
			return SetupGuide{}, fmt.Errorf("%w: Windows account applies only to a Windows destination", ErrInvalidInput)
		}
		host.WindowsAccount = options.WindowsAccount
	}
	localOS := options.LocalOS
	if localOS == "" {
		localOS = LocalPlatform()
	}
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
	fmt.Fprintf(&text, "Local platform: %s\nDestination platform: %s", localOS, host.OS)
	if host.OS == "windows" {
		fmt.Fprintf(&text, " (%s)", host.WindowsAccount)
	}
	text.WriteString("\n\n")
	var placement GuideStep
	switch host.OS {
	case "linux", "macos":
		placement = posixPlacement(host, public)
	case "windows":
		if host.WindowsAccount != "administrator" && host.WindowsAccount != "standard" {
			return SetupGuide{}, fmt.Errorf("%w: confirm the destination Windows account type before preparing guidance", ErrInvalidInput)
		}
		placement = windowsPlacement(host, public)
	default:
		return SetupGuide{}, fmt.Errorf("%w: confirm the destination OS before preparing guidance", ErrInvalidInput)
	}
	placement.Instructions += "\nCheck effective AuthorizedKeysFile/Match settings if customized. An SSH server and network access must already be available. Existing keys are preserved; an already present key is not appended again.\nManual removal: keep another working access method, then remove only the matching public-key line from the destination file. SSH KeyFlow does not verify removal."
	guide.Steps = append(guide.Steps, placement)
	notice := ""
	publicOnly := key.PrivatePath == ""
	path := filepath.ToSlash(key.PrivatePath)
	if publicOnly {
		// OpenSSH selects the agent key matching this public key file.
		path = filepath.ToSlash(key.PublicPath)
	}
	if options.IdentityFile != "" {
		path = options.IdentityFile
		if localOS == "windows" {
			path = strings.ReplaceAll(path, "\\", "/")
		}
	} else if localOS != LocalPlatform() {
		if !publicOnly {
			return SetupGuide{}, fmt.Errorf("%w: a different local OS requires identity-file, the matching private-key path in that terminal", ErrInvalidInput)
		}
		path = ""
		notice = "Public-only key: the saved public key path is for this computer. Supply identity-file, the matching key path in the selected local terminal, to get an SSH config step and connection command.\n"
	}
	for _, character := range path {
		if unicode.IsControl(character) {
			return SetupGuide{}, fmt.Errorf("identity path contains control characters; cannot prepare a snippet")
		}
	}
	switch {
	case path == "":
	case strings.Contains(path, "${"):
		notice = "OpenSSH expands ${...} environment variables in identity paths. This identity path cannot be represented literally here; no IdentityFile snippet or connection command was generated. Select a key path without ${...} yourself.\n"
	default:
		// SSH expands percent tokens in IdentityFile even inside quotes.
		configPath := quoteConfigPath(path)
		configUser := strings.NewReplacer("\\", "\\\\", "\"", "\\\"").Replace(host.User)
		// A path supplied for another terminal is never inspected (ADR 0003).
		keyProtection, reason := undetermined, "the private key path was supplied for this terminal and is not inspected"
		if publicOnly {
			// The agent belongs to the key holder, such as 1Password or a FIDO token.
			keyProtection, reason = unprotected, ""
		} else if options.IdentityFile == "" {
			keyProtection, reason = passphraseProtection(key.PrivatePath)
		}
		agentConfig := ""
		if keyProtection != unprotected {
			agentConfig = "    AddKeysToAgent yes\n"
			if localOS == "macos" {
				agentConfig += "    IgnoreUnknown UseKeychain\n    UseKeychain yes\n"
			}
		}
		pinning, pinned := hostKeyPinning(home, host, localOS)
		config := fmt.Sprintf("Host %s\n    HostName %s\n    User \"%s\"\n    Port %d\n    IdentityFile \"%s\"\n%s    IdentitiesOnly yes\n%s", host.Alias, host.Hostname, configUser, host.Port, configPath, agentConfig, pinning.lines)
		local := clientSetup(host, config, localOS)
		if publicOnly {
			local.Instructions += "\nPublic-only key: IdentityFile names the public key so ssh uses the matching private key from your SSH agent (for example 1Password, or ssh-agent with a FIDO key). That key must be loaded in the agent. On Windows, an \"UNPROTECTED PRIVATE KEY FILE\" or \"bad permissions\" warning means it was not."
		} else if options.IdentityFile != "" {
			local.Instructions += "\nIdentityFile is the path you supplied for this terminal. Ensure it refers to the matching private key; SSH KeyFlow has not checked that path or copied a key."
		}
		if pinning.instructions != "" {
			local.Instructions += "\n" + pinning.instructions
		}
		if keyProtection == undetermined {
			// The config step can be copied alone, so it carries the notice too.
			local.Instructions += "\n" + undeterminedNotice(reason)
		} else {
			reason = ""
		}
		if !pinned && pinning.steps {
			guide.Steps = append(guide.Steps, hostKeyPrintStep(host), pinStep(host, localOS))
		}
		guide.Steps = append(guide.Steps, local)
		if keyProtection != unprotected {
			guide.Steps = append(guide.Steps, agentSetup(localOS, path, reason))
		}
		guide.Steps = append(guide.Steps, GuideStep{
			Title:        fmt.Sprintf("LOCAL terminal — %s — connect (POSIX shell / PowerShell)", localOS),
			Instructions: "Run in the selected local terminal after completing the destination step and local config step. This opens an SSH connection; SSH KeyFlow does not run it or verify success.",
			Script:       fmt.Sprintf("ssh %s\n", host.Alias),
		})
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
