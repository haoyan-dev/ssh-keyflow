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
	if home == "" {
		return "", fmt.Errorf("Home is required")
	}
	document, err := state.Load(home)
	if err != nil {
		return "", err
	}
	hostIndex, err := destinationIndex(document, hostReference)
	if err != nil {
		return "", err
	}
	host := document.Hosts[hostIndex]
	key, err := usableKey(document, keyReference)
	if err != nil {
		return "", err
	}
	public, err := publicText(key)
	if err != nil {
		return "", err
	}
	var text strings.Builder
	fmt.Fprintf(&text, "Copy-paste setup: %s (%s@%s:%d)\nKey: %s [%s]\nFingerprint: %s\nRemote registration and connection: not checked.\nCopying this guide changes no files and records no success.\n\nPublic key (copy this complete line):\n%s\n\n", host.Alias, host.User, host.Hostname, host.Port, key.Name, key.ID, key.Fingerprint, public)
	switch host.OS {
	case "linux", "macos":
		text.WriteString("On the destination, sign in as the login user. Paste the public key as its own line in ~/.ssh/authorized_keys; preserve all existing lines. Use that user's home directory, with .ssh mode 700 and authorized_keys mode 600, owned by that user.\n")
	case "windows":
		if host.WindowsAccount == "administrator" {
			text.WriteString(`On the destination, use an elevated editor to paste the public key as its own line in %ProgramData%\ssh\administrators_authorized_keys. This file is shared by administrator logins. Preserve other keys; restrict its ACL to SYSTEM and Administrators.` + "\n")
		} else if host.WindowsAccount == "standard" {
			text.WriteString(`On the destination, sign in as the login user and paste the public key as its own line in %USERPROFILE%\.ssh\authorized_keys. Preserve other keys; restrict write access to that user, SYSTEM and Administrators.` + "\n")
		} else {
			return "", fmt.Errorf("confirm the destination Windows account type before preparing guidance")
		}
	default:
		return "", fmt.Errorf("confirm the destination OS before preparing guidance")
	}
	text.WriteString("These are standard OpenSSH locations. Check the destination's effective AuthorizedKeysFile/Match settings if customized. An SSH server and network access must already be available.\n\nManual removal: keep another working access method, then remove only the line matching the public key above from the same destination file. Preserve other lines. SSH KeyFlow does not verify removal.\n\n")
	if key.PrivatePath == "" {
		text.WriteString("Public-only key: use the matching private key on your client. No usable IdentityFile or connection command can be generated here.\n")
	} else {
		path := filepath.ToSlash(key.PrivatePath)
		for _, character := range path {
			if unicode.IsControl(character) {
				return "", fmt.Errorf("private path contains control characters; cannot prepare a snippet")
			}
		}
		if strings.Contains(path, "${") {
			text.WriteString("OpenSSH expands ${...} environment variables in identity paths. This private path cannot be represented literally here; no IdentityFile snippet or connection command was generated. Use the public-key placement guidance above and select a private-key path without ${...} yourself.\n")
			fmt.Fprintf(&text, "\nLocal files (manual cleanup only; never delete a private key still needed for access):\nPublic: %s\nPrivate: %s\n", key.PublicPath, key.PrivatePath)
			return text.String(), nil
		}
		// SSH expands percent tokens in IdentityFile even inside quotes.
		configPath := strings.NewReplacer("%", "%%", "\\", "\\\\", "\"", "\\\"").Replace(path)
		configUser := strings.NewReplacer("\\", "\\\\", "\"", "\\\"").Replace(host.User)
		fmt.Fprintf(&text, "SSH config snippet (paste into your client ~/.ssh/config after checking for an existing Host %s; SSH KeyFlow does not inspect or edit it):\nHost %s\n    HostName %s\n    User \"%s\"\n    Port %d\n    IdentityFile \"%s\"\n    IdentitiesOnly yes\n\n", host.Alias, host.Alias, host.Hostname, configUser, host.Port, configPath)
		fmt.Fprintf(&text, "Connection command (POSIX shell / PowerShell, after pasting the SSH config snippet):\nssh %s\n", host.Alias)
	}
	fmt.Fprintf(&text, "\nLocal files (manual cleanup only; never delete a private key still needed for access):\nPublic: %s\nPrivate: %s\n", key.PublicPath, key.PrivatePath)
	return text.String(), nil
}
