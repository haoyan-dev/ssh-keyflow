package state

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Key struct {
	Origin          string `json:"origin"`
	ImportedAt      string `json:"imported_at"`
	ID              string `json:"id"`
	Name            string `json:"name"`
	Fingerprint     string `json:"fingerprint"`
	PublicPath      string `json:"public_path"`
	PrivatePath     string `json:"private_path,omitempty"`
	PrivateSHA256   string `json:"private_sha256,omitempty"`
	ConnectionCheck string `json:"connection_check"`
}

type Host struct {
	ID       string `json:"id"`
	Alias    string `json:"alias"`
	Hostname string `json:"hostname"`
	User     string `json:"user"`
	Port     int    `json:"port"`
	OS       string `json:"os"`
}

type Registration struct {
	ID              string `json:"id"`
	KeyID           string `json:"key_id"`
	HostID          string `json:"host_id"`
	Status          string `json:"status"`
	ConnectionCheck string `json:"connection_check"`
	Operation       string `json:"operation"`
	Stage           string `json:"stage"`
}

type State struct {
	Version       int            `json:"version"`
	Keys          []Key          `json:"keys"`
	Hosts         []Host         `json:"hosts"`
	Registrations []Registration `json:"registrations"`
}

func Directory(home string) string { return filepath.Join(home, ".ssh", "ssh-keyflow") }

func Load(home string) (State, error) {
	data, err := os.ReadFile(filepath.Join(Directory(home), "state.json"))
	if os.IsNotExist(err) {
		return State{Version: 1, Keys: []Key{}, Hosts: []Host{}, Registrations: []Registration{}}, nil
	}
	if err != nil {
		return State{}, fmt.Errorf("read state.json: %w", err)
	}
	return decode(data)
}

func decode(data []byte) (State, error) {
	if err := uniqueJSON(json.NewDecoder(bytes.NewReader(data))); err != nil {
		return State{}, fmt.Errorf("invalid state.json: %w", err)
	}
	var document State
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return State{}, fmt.Errorf("invalid state.json; restore a verified backup: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return State{}, fmt.Errorf("invalid state.json: trailing content")
	}
	if document.Version != 1 || document.Keys == nil || document.Hosts == nil || document.Registrations == nil {
		return State{}, fmt.Errorf("unsupported or incomplete state.json schema/version; updates stopped")
	}
	if err := document.Validate(); err != nil {
		return State{}, fmt.Errorf("invalid state.json: %w", err)
	}
	return document, nil
}

func uniqueJSON(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, compound := token.(json.Delim)
	if !compound {
		return nil
	}
	seen := map[string]bool{}
	for decoder.More() {
		if delimiter == '{' {
			field, err := decoder.Token()
			if err != nil {
				return err
			}
			name, valid := field.(string)
			if !valid || name != strings.ToLower(name) || seen[name] {
				return fmt.Errorf("duplicate or invalid field %v", field)
			}
			seen[name] = true
		}
		if err := uniqueJSON(decoder); err != nil {
			return err
		}
	}
	_, err = decoder.Token()
	return err
}

func (document State) Validate() error {
	if document.Version != 1 || document.Keys == nil || document.Hosts == nil || document.Registrations == nil {
		return fmt.Errorf("unsupported or incomplete schema/version")
	}
	identities := map[string]bool{}
	claim := func(identity string) bool {
		if !clean(identity) || identities[identity] {
			return false
		}
		identities[identity] = true
		return true
	}
	keys, hosts, aliases, fingerprints, pairs, paths := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, key := range document.Keys {
		importedAt, timeErr := time.Parse(time.RFC3339Nano, key.ImportedAt)
		if key.Origin != "imported" || timeErr != nil || importedAt.IsZero() {
			return fmt.Errorf("invalid key provenance %q", key.ID)
		}
		digest, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(key.Fingerprint, "SHA256:"))
		if !claim(key.ID) || !clean(key.Name) || !strings.HasPrefix(key.Fingerprint, "SHA256:") || err != nil || len(digest) != 32 || fingerprints[key.Fingerprint] || !filepath.IsAbs(key.PublicPath) || paths[filepath.Clean(key.PublicPath)] || !oneOf(key.ConnectionCheck, "unverified", "verified", "failed", "unknown") {
			return fmt.Errorf("invalid or duplicate key %q", key.ID)
		}
		if (key.PrivatePath == "") != (key.PrivateSHA256 == "") || (key.PrivatePath != "" && (!filepath.IsAbs(key.PrivatePath) || key.PrivatePath == key.PublicPath || paths[filepath.Clean(key.PrivatePath)] || len(key.PrivateSHA256) != 64)) {
			return fmt.Errorf("invalid private key metadata %q", key.ID)
		}
		if key.PrivateSHA256 != "" {
			if _, err := hex.DecodeString(key.PrivateSHA256); err != nil {
				return fmt.Errorf("invalid private key digest %q", key.ID)
			}
		}
		keys[key.ID], fingerprints[key.Fingerprint], paths[filepath.Clean(key.PublicPath)] = true, true, true
		if key.PrivatePath != "" {
			paths[filepath.Clean(key.PrivatePath)] = true
		}
	}
	for _, host := range document.Hosts {
		if !claim(host.ID) || !clean(host.Alias) || strings.ContainsAny(host.Alias, " *?!\\") || aliases[host.Alias] || !clean(host.Hostname) || !clean(host.User) || host.Port < 1 || host.Port > 65535 || !oneOf(host.OS, "unknown", "windows", "linux", "macos") {
			return fmt.Errorf("invalid or duplicate host %q", host.ID)
		}
		hosts[host.ID], aliases[host.Alias] = true, true
	}
	for _, registration := range document.Registrations {
		pair := registration.KeyID + "/" + registration.HostID
		if !claim(registration.ID) || !keys[registration.KeyID] || !hosts[registration.HostID] || pairs[pair] || !oneOf(registration.Status, "pending", "registered", "removed", "unknown", "failed") || !oneOf(registration.ConnectionCheck, "unverified", "verified", "failed", "unknown") || !oneOf(registration.Operation, "register", "unregister") || !oneOf(registration.Stage, "planned", "pending", "complete", "unknown", "failed") {
			return fmt.Errorf("invalid registration or relationship %q", registration.ID)
		}
		pairs[pair] = true
	}
	return nil
}

func clean(value string) bool {
	return strings.TrimSpace(value) != "" && !strings.ContainsAny(value, "\x00\r\n")
}
func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func CheckWritable(home string) error {
	directory := Directory(home)
	if err := os.MkdirAll(filepath.Join(directory, "keys"), 0700); err != nil {
		return fmt.Errorf("storage directory: %w", err)
	}
	probe, err := os.CreateTemp(directory, ".write-check-*")
	if err != nil {
		return fmt.Errorf("storage is not writable: %w", err)
	}
	path := probe.Name()
	if err := probe.Close(); err != nil {
		return err
	}
	return os.Remove(path)
}

type LockedStore struct {
	home   string
	active bool
}

func WithLock(home string, operation func(*LockedStore) error) error {
	directory := Directory(home)
	lockPath := filepath.Join(directory, "state.lock")
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("state is locked or unavailable; retry later; do not remove an unverified lock %s: %w", lockPath, err)
	}
	defer os.Remove(lockPath)
	if _, err := fmt.Fprintf(lock, "pid=%d\n", os.Getpid()); err != nil {
		lock.Close()
		return err
	}
	if err := lock.Close(); err != nil {
		return err
	}
	store := &LockedStore{home: home, active: true}
	defer func() { store.active = false }()
	return operation(store)
}

func (store *LockedStore) Load() (State, error) {
	if !store.active {
		return State{}, fmt.Errorf("operation lock is no longer held")
	}
	return Load(store.home)
}

func Update(home string, change func(*State) error) error {
	return WithLock(home, func(store *LockedStore) error {
		document, err := store.Load()
		if err != nil {
			return err
		}
		if err := change(&document); err != nil {
			return err
		}
		return store.Save(document)
	})
}

func (store *LockedStore) Save(document State) error {
	if !store.active {
		return fmt.Errorf("operation lock is no longer held")
	}
	if err := document.Validate(); err != nil {
		return err
	}
	directory := Directory(store.home)
	_, err := store.Load()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(directory, "state.json")
	previous, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		if _, err := decode(previous); err != nil {
			return err
		}
		if err := atomicWrite(path+".bak", previous); err != nil {
			return fmt.Errorf("backup failed; state unchanged: %w", err)
		}
	}
	return atomicWrite(path, append(data, '\n'))
}

func atomicWrite(path string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".state-*")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}
