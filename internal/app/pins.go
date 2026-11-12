package app

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"ssh-keyflow/internal/state"
)

const (
	maxHostKeyPaste = 64 << 10
	maxHostKeys     = 16
)

// HostKeyPaste is the reviewed result of parsing pasted host key output.
type HostKeyPaste struct {
	Keys []state.HostKey
	// Ignored holds sanitized, shortened lines that were not accepted as keys.
	Ignored []string
}

// ParseHostKeys accepts only "<type> <base64> [comment]" lines for sshd host key
// algorithms. Other non-blank lines are reported but never block valid keys.
// Comments are dropped; only parsed fields are ever written.
func ParseHostKeys(text string) (HostKeyPaste, error) {
	if len(text) > maxHostKeyPaste {
		return HostKeyPaste{}, fmt.Errorf("%w: pasted host keys exceed %d KiB", ErrInvalidInput, maxHostKeyPaste>>10)
	}
	var paste HostKeyPaste
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "\ufeff"))
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		key := state.HostKey{Type: fields[0]}
		if len(fields) >= 2 {
			key.Blob = fields[1]
		}
		if key.Validate() != nil {
			paste.Ignored = append(paste.Ignored, displayLine(line))
			continue
		}
		if !slices.Contains(paste.Keys, key) {
			paste.Keys = append(paste.Keys, key)
		}
	}
	if len(paste.Keys) == 0 {
		return paste, fmt.Errorf("%w: no host public key found; paste lines such as \"ssh-ed25519 AAAA... root@host\"", ErrInvalidInput)
	}
	if len(paste.Keys) > maxHostKeys {
		return paste, fmt.Errorf("%w: more than %d host keys pasted", ErrInvalidInput, maxHostKeys)
	}
	return paste, nil
}

func displayLine(line string) string {
	line = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return '?'
		}
		return r
	}, line)
	if runes := []rune(line); len(runes) > 60 {
		line = string(runes[:57]) + "..."
	}
	return line
}

// HostKeyFingerprint is the OpenSSH SHA256 fingerprint of a host key.
func HostKeyFingerprint(key state.HostKey) string {
	blob, _ := base64.StdEncoding.DecodeString(key.Blob)
	digest := sha256.Sum256(blob)
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(digest[:])
}

// HostKeyPins returns pinned keys by destination ID. It never repairs the file.
func HostKeyPins(home string) (map[string][]state.HostKey, error) {
	entries, err := state.LoadPins(home)
	if err != nil {
		return nil, err
	}
	pins := map[string][]state.HostKey{}
	for _, entry := range entries {
		pins[entry.HostID] = append(pins[entry.HostID], entry.Key)
	}
	return pins, nil
}

func sameKeys(left, right []state.HostKey) bool {
	if len(left) != len(right) {
		return false
	}
	for _, key := range left {
		if !slices.Contains(right, key) {
			return false
		}
	}
	return true
}

func writeFingerprints(output io.Writer, heading string, keys []state.HostKey) error {
	if _, err := fmt.Fprintln(output, heading); err != nil {
		return err
	}
	for _, key := range keys {
		if _, err := fmt.Fprintf(output, "  %s %s\n", key.Type, HostKeyFingerprint(key)); err != nil {
			return err
		}
	}
	return nil
}

// HostKeyCommand prints the destination's host public keys without changing it.
func HostKeyCommand(platform string) string {
	if platform == "windows" {
		return `Get-Content "$env:ProgramData\ssh\ssh_host_*_key.pub"`
	}
	return "cat /etc/ssh/ssh_host_*_key.pub"
}

// readPaste reads byte by byte, so a later confirmation reads the next line.
// It stops at end of input or at an empty line after content.
func readPaste(input io.Reader) (string, error) {
	var text strings.Builder
	for {
		line, err := readLine(input)
		if line != "" {
			text.WriteString(line + "\n")
		}
		if text.Len() > maxHostKeyPaste {
			return "", fmt.Errorf("%w: pasted host keys exceed %d KiB", ErrInvalidInput, maxHostKeyPaste>>10)
		}
		if errors.Is(err, io.EOF) || (err == nil && line == "" && text.Len() > 0) {
			return text.String(), nil
		}
		if err != nil {
			return "", err
		}
	}
}

func pinHost(options Options) error {
	request := options.Request
	document, err := state.Load(options.Home)
	if err != nil {
		return err
	}
	index, err := destinationIndex(document, request.Host)
	if err != nil {
		return err
	}
	host := document.Hosts[index]
	options.target = ReportTarget{HostID: host.ID}
	text := request.HostKeys
	if text == "" {
		elevated := ""
		if host.OS == "windows" {
			elevated = " in PowerShell opened with Run as Administrator"
		}
		if _, err := fmt.Fprintf(options.Output, "Pin host keys: %s (%s@%s:%d) [%s]\nOn the destination%s, run:\n  %s\nPaste its complete output here, then press Enter on an empty line.\n", host.Alias, host.User, host.Hostname, host.Port, host.ID, elevated, HostKeyCommand(host.OS)); err != nil {
			return err
		}
		if text, err = readPaste(options.Input); err != nil {
			return err
		}
	}
	paste, err := ParseHostKeys(text)
	for _, line := range paste.Ignored {
		fmt.Fprintf(options.Output, "Ignored line: %s\n", line)
	}
	if err != nil {
		return err
	}
	if err := writeFingerprints(options.Output, fmt.Sprintf("Host keys pasted for %s; compare with ssh-keygen -lf on the destination:", host.Alias), paste.Keys); err != nil {
		return err
	}
	if err := state.CheckWritable(options.Home); err != nil {
		return err
	}
	return state.WithLock(options.Home, func(store *state.LockedStore) error {
		document, err := store.Load()
		if err != nil {
			return err
		}
		if _, err := destinationIndex(document, host.ID); err != nil {
			return err
		}
		entries, err := state.LoadPins(options.Home)
		if err != nil {
			return err
		}
		existing := state.HostPins(entries, host.ID)
		if sameKeys(existing, paste.Keys) {
			_, err := fmt.Fprintln(options.Output, "These host keys are already pinned; no changes.")
			return err
		}
		label := "Type yes to pin these host keys:"
		if len(existing) > 0 {
			if err := writeFingerprints(options.Output, "WARNING: different host keys are already pinned for this destination:", existing); err != nil {
				return err
			}
			if !request.Replace {
				return fmt.Errorf("%w: different host keys are already pinned for %s; a changed host key can mean an attack. Confirm the new keys on the destination, then rerun with --replace", ErrInvalidInput, host.Alias)
			}
			if _, err := fmt.Fprintln(options.Output, "Replace them only if you confirmed the change on the destination itself."); err != nil {
				return err
			}
			label = "Type yes to REPLACE the pinned host keys:"
		}
		if _, err := prompt(options, label); err != nil {
			return err
		}
		if err := store.SavePins(host.ID, paste.Keys); err != nil {
			options.report(Report{Kind: ReportHostKeys, Outcome: OutcomeUnknown, Path: state.PinPath(options.Home)})
			return err
		}
		options.report(Report{Kind: ReportHostKeys, Outcome: OutcomeApplied, Saved: true, Path: state.PinPath(options.Home)})
		_, err = fmt.Fprintf(options.Output, "Pinned %d host key(s) for %s in %s.\nGenerate the guide again to get the pinned SSH config lines. If this Host is already in your SSH config, add those host-key lines to it yourself.\n", len(paste.Keys), host.Alias, state.PinPath(options.Home))
		return err
	})
}

func unpinHost(options Options) error {
	document, err := state.Load(options.Home)
	if err != nil {
		return err
	}
	index, err := destinationIndex(document, options.Request.Host)
	if err != nil {
		return err
	}
	host := document.Hosts[index]
	options.target = ReportTarget{HostID: host.ID}
	if err := state.CheckWritable(options.Home); err != nil {
		return err
	}
	return state.WithLock(options.Home, func(store *state.LockedStore) error {
		document, err := store.Load()
		if err != nil {
			return err
		}
		if _, err := destinationIndex(document, host.ID); err != nil {
			return err
		}
		entries, err := state.LoadPins(options.Home)
		if err != nil {
			return err
		}
		existing := state.HostPins(entries, host.ID)
		if len(existing) == 0 {
			_, err := fmt.Fprintf(options.Output, "No host keys are pinned for %s; no changes.\n", host.Alias)
			return err
		}
		if err := writeFingerprints(options.Output, fmt.Sprintf("Pinned host keys for %s:", host.Alias), existing); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(options.Output, "If your SSH config has this destination's HostKeyAlias, UserKnownHostsFile and StrictHostKeyChecking lines, remove them after unpinning or ssh will refuse the connection."); err != nil {
			return err
		}
		if _, err := prompt(options, "Type yes to unpin these host keys:"); err != nil {
			return err
		}
		if err := store.SavePins(host.ID, nil); err != nil {
			options.report(Report{Kind: ReportHostKeys, Outcome: OutcomeUnknown, Path: state.PinPath(options.Home)})
			return err
		}
		options.report(Report{Kind: ReportHostKeys, Outcome: OutcomeApplied, Saved: true, Path: state.PinPath(options.Home)})
		_, err = fmt.Fprintf(options.Output, "Unpinned host keys for %s. SSH access on the destination is unchanged.\n", host.Alias)
		return err
	})
}

type guidePinning struct {
	lines, instructions string
	// steps adds the print and pin steps before the SSH config step.
	steps bool
}

// hostKeyPinning reads the pin file only for this computer's terminal, since the
// file path and the pin command belong to it. It never fails the guide.
func hostKeyPinning(home string, host state.Host, localOS string) (guidePinning, bool) {
	if localOS != LocalPlatform() {
		return guidePinning{}, false
	}
	pins, err := HostKeyPins(home)
	if err != nil {
		return guidePinning{instructions: fmt.Sprintf("Host keys: unknown (%v). No host-key pinning lines were generated.", err)}, false
	}
	keys := pins[host.ID]
	if len(keys) == 0 {
		return guidePinning{steps: true, instructions: "Host keys: not pinned. To connect without a first-connection host key prompt, complete the print and pin steps, then generate this guide again before copying this step."}, false
	}
	lines, notice := pinnedConfig(home, host.ID)
	if notice != "" {
		return guidePinning{instructions: notice}, true
	}
	return guidePinning{lines: lines, instructions: fmt.Sprintf("Host keys: pinned (%d). ssh accepts only those keys for this Host and never asks to trust a new one. If this Host is already in your SSH config, add the HostKeyAlias, UserKnownHostsFile and StrictHostKeyChecking lines to it yourself. If the destination's host keys change, confirm them on the destination and run ssh-keyflow hosts pin --host %s --replace.", len(keys), host.ID)}, true
}

// pinnedConfig returns the Host lines for a pinned destination, or a notice when
// the pin file path cannot be written literally into SSH config.
func pinnedConfig(home, hostID string) (string, string) {
	path := filepath.ToSlash(state.PinPath(home))
	for _, character := range path {
		if unicode.IsControl(character) {
			return "", "The host key pin file path contains control characters; no pinning lines were generated.\n"
		}
	}
	if strings.Contains(path, "${") {
		return "", "OpenSSH expands ${...} in UserKnownHostsFile. The pin file path cannot be written literally; no pinning lines were generated.\n"
	}
	quoted := strings.NewReplacer("%", "%%", "\\", "\\\\", "\"", "\\\"").Replace(path)
	return fmt.Sprintf("    HostKeyAlias %s\n    UserKnownHostsFile \"%s\"\n    StrictHostKeyChecking yes\n", state.PinAlias(hostID), quoted), ""
}

func hostKeyPrintStep(host state.Host) GuideStep {
	step := GuideStep{
		Title:        "DESTINATION — POSIX shell — print host keys",
		Instructions: "Run on the destination itself, in a terminal you already trust (console or existing session). It only reads the SSH server's host public keys. Copy the complete output for the next step.",
		Script:       HostKeyCommand(host.OS) + "\n",
	}
	if host.OS == "windows" {
		step.Title = "DESTINATION — Windows PowerShell (Run as Administrator) — print host keys"
		step.Instructions = "Open PowerShell with Run as Administrator on the destination itself: standard users cannot read these files. It only reads the SSH server's host public keys. Copy the complete output for the next step."
	}
	step.Instructions += "\nOptional check: ssh-keygen -lf <file> on the destination shows the fingerprints that the next step prints."
	return step
}

func pinStep(host state.Host, localOS string) GuideStep {
	return GuideStep{
		Title:        fmt.Sprintf("LOCAL terminal — %s — pin host keys", localOS),
		Instructions: "Run in this computer's terminal (use the full path to ssh-keyflow if it is not on PATH). Paste the destination output, press Enter on an empty line, review the fingerprints and type yes. Then generate this guide again: the SSH config step will include the pinning lines.",
		Script:       fmt.Sprintf("ssh-keyflow hosts pin --host %s\n", host.ID),
	}
}

// PinSummary is one pinned host key as shown to people and scripts.
type PinSummary struct {
	Type        string `json:"type"`
	Fingerprint string `json:"fingerprint"`
}

// PinSummaries lists pinned key fingerprints by destination ID, read-only.
func PinSummaries(home string) (map[string][]PinSummary, error) {
	pins, err := HostKeyPins(home)
	if err != nil {
		return nil, err
	}
	summaries := map[string][]PinSummary{}
	for id, keys := range pins {
		for _, key := range keys {
			summaries[id] = append(summaries[id], PinSummary{Type: key.Type, Fingerprint: HostKeyFingerprint(key)})
		}
	}
	return summaries, nil
}
