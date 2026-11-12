package state

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PinPrefix starts every host-key alias SSH KeyFlow writes, keeping its entries
// apart from names a user writes elsewhere (ADR 0006).
const PinPrefix = "ssh-keyflow-"

// HostKey is one pinned host public key: an algorithm and its base64 blob.
type HostKey struct {
	Type string
	Blob string
}

// PinEntry is one line of the app-owned known_hosts file.
type PinEntry struct {
	HostID string
	Key    HostKey
}

// PinAlias is the HostKeyAlias used for a destination's pinned keys.
func PinAlias(hostID string) string { return PinPrefix + hostID }

// PinPath is the app-owned known_hosts file.
func PinPath(home string) string { return filepath.Join(Directory(home), "known_hosts") }

// LoadPins reads the pin file strictly. A missing file has no pins. Any line not
// in the exact form this package writes is an error: the file is never guessed
// at or repaired.
func LoadPins(home string) ([]PinEntry, error) {
	path := PinPath(home)
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read host key pins: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("host key pins %s is not a regular file; refusing it", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read host key pins: %w", err)
	}
	return decodePins(path, data)
}

func decodePins(path string, data []byte) ([]PinEntry, error) {
	text := string(data)
	if text == "" {
		return nil, nil
	}
	if !strings.HasSuffix(text, "\n") {
		return nil, fmt.Errorf("invalid host key pins %s: missing final newline; review the file manually", path)
	}
	var entries []PinEntry
	seen := map[PinEntry]bool{}
	for number, line := range strings.Split(strings.TrimSuffix(text, "\n"), "\n") {
		fields := strings.Split(line, " ")
		if len(fields) != 3 || !strings.HasPrefix(fields[0], PinPrefix) || len(fields[0]) == len(PinPrefix) || !clean(fields[0]) || strings.ContainsAny(fields[0], "\t\v\f") {
			return nil, fmt.Errorf("invalid host key pins %s line %d; review the file manually", path, number+1)
		}
		entry := PinEntry{HostID: strings.TrimPrefix(fields[0], PinPrefix), Key: HostKey{Type: fields[1], Blob: fields[2]}}
		if err := entry.Key.Validate(); err != nil || seen[entry] {
			return nil, fmt.Errorf("invalid host key pins %s line %d; review the file manually", path, number+1)
		}
		seen[entry] = true
		entries = append(entries, entry)
	}
	return entries, nil
}

// HostKeyTypes are the sshd host key algorithms accepted for pinning.
var HostKeyTypes = []string{"ssh-ed25519", "ecdsa-sha2-nistp256", "ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521", "ssh-rsa"}

// Validate checks the algorithm and that the blob names the same algorithm.
func (key HostKey) Validate() error {
	if !oneOf(key.Type, HostKeyTypes...) {
		return fmt.Errorf("unsupported host key type %q", key.Type)
	}
	blob, err := base64.StdEncoding.Strict().DecodeString(key.Blob)
	// The decoder skips \r and \n, so also require the canonical encoding.
	if err != nil || len(blob) < 4 || base64.StdEncoding.EncodeToString(blob) != key.Blob {
		return fmt.Errorf("invalid host key encoding")
	}
	length := int(uint32(blob[0])<<24 | uint32(blob[1])<<16 | uint32(blob[2])<<8 | uint32(blob[3]))
	if length < 1 || length > len(blob)-4 || string(blob[4:4+length]) != key.Type {
		return fmt.Errorf("host key algorithm does not match its data")
	}
	return nil
}

// SavePins replaces one destination's keys (none removes them) in the position
// of its first entry, and keeps other destinations' entries and order. Strict
// decoding makes the rewritten lines identical to the stored ones. It writes
// atomically under the held state lock.
func (store *LockedStore) SavePins(hostID string, keys []HostKey) error {
	if !store.active {
		return fmt.Errorf("operation lock is no longer held")
	}
	if !clean(hostID) || strings.ContainsAny(hostID, " 	") {
		return fmt.Errorf("invalid destination ID %q for host key pins", hostID)
	}
	for _, key := range keys {
		if err := key.Validate(); err != nil {
			return err
		}
	}
	entries, err := LoadPins(store.home)
	if err != nil {
		return err
	}
	var lines strings.Builder
	replaced := false
	write := func(id string, keys ...HostKey) {
		for _, key := range keys {
			fmt.Fprintf(&lines, "%s %s %s\n", PinAlias(id), key.Type, key.Blob)
		}
	}
	for _, entry := range entries {
		switch {
		case entry.HostID != hostID:
			write(entry.HostID, entry.Key)
		case !replaced:
			write(hostID, keys...)
			replaced = true
		}
	}
	if !replaced {
		write(hostID, keys...)
	}
	if _, err := decodePins(PinPath(store.home), []byte(lines.String())); err != nil {
		return err
	}
	return atomicWrite(PinPath(store.home), []byte(lines.String()))
}
