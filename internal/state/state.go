package state

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"
)

type Key struct {
	Origin          string `json:"origin"`
	ImportedAt      string `json:"imported_at"`
	CreatedAt       string `json:"created_at,omitempty"`
	ID              string `json:"id"`
	Name            string `json:"name"`
	Fingerprint     string `json:"fingerprint"`
	PublicPath      string `json:"public_path"`
	PrivatePath     string `json:"private_path,omitempty"`
	PrivateSHA256   string `json:"private_sha256,omitempty"`
	ConnectionCheck string `json:"connection_check"`
}

type Host struct {
	ConfigKeyID    string `json:"config_key_id,omitempty"`
	ID             string `json:"id"`
	Alias          string `json:"alias"`
	Hostname       string `json:"hostname"`
	User           string `json:"user"`
	Port           int    `json:"port"`
	OS             string `json:"os"`
	WindowsAccount string `json:"windows_account,omitempty"`
}

func (host Host) Validate() error {
	validText := func(value, punctuation string) bool {
		if !clean(value) || strings.HasPrefix(value, "-") || strings.TrimSpace(value) != value {
			return false
		}
		for _, character := range value {
			if !unicode.IsLetter(character) && !unicode.IsNumber(character) && !unicode.IsMark(character) && !strings.ContainsRune(punctuation, character) {
				return false
			}
		}
		return true
	}
	if !validText(host.Alias, "._-") || (!validText(host.Hostname, ".-_") && net.ParseIP(host.Hostname) == nil) || !validText(host.User, "._-\\$ ") || host.Port < 1 || host.Port > 65535 || !oneOf(host.OS, "unknown", "windows", "linux", "macos") || (host.WindowsAccount != "" && (host.OS != "windows" || !oneOf(host.WindowsAccount, "standard", "administrator"))) {
		return fmt.Errorf("invalid destination fields")
	}
	return nil
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

type Association struct {
	HostID string `json:"host_id"`
	KeyID  string `json:"key_id"`
}

type State struct {
	RemovedKeys   []Key               `json:"removed_keys,omitempty"`
	RemovedHosts  []Host              `json:"removed_hosts,omitempty"`
	Associations  []Association       `json:"associations,omitempty"`
	DeletedKeys   []DeletedKey        `json:"deleted_keys,omitempty"`
	Deletions     []LocalDeletion     `json:"deletions,omitempty"`
	Version       int                 `json:"version"`
	Keys          []Key               `json:"keys"`
	Hosts         []Host              `json:"hosts"`
	Registrations []Registration      `json:"registrations"`
	Creations     []Creation          `json:"creations,omitempty"`
	Onboardings   []Onboarding        `json:"onboardings,omitempty"`
	History       []RegistrationEvent `json:"history,omitempty"`
	Configs       []Config            `json:"configs,omitempty"`
	ConfigEdits   []ConfigEdit        `json:"config_edits,omitempty"`
}

type DeletedKey struct {
	Key        Key    `json:"key"`
	DeletedAt  string `json:"deleted_at"`
	DeletionID string `json:"deletion_id"`
}

type LocalDeletion struct {
	ID             string `json:"id"`
	KeyID          string `json:"key_id"`
	CreatedAt      string `json:"created_at"`
	Stage          string `json:"stage"`
	PublicSHA256   string `json:"public_sha256"`
	PrivateSHA256  string `json:"private_sha256,omitempty"`
	PublicRemoved  bool   `json:"public_removed"`
	PrivateRemoved bool   `json:"private_removed"`
}

type ConfigEdit struct {
	Path         string `json:"path"`
	BeforeSHA256 string `json:"before_sha256"`
	AfterSHA256  string `json:"after_sha256"`
	BackupPath   string `json:"backup_path,omitempty"`
	Complete     bool   `json:"complete"`
}

type RegistrationEvent struct {
	Operation       string `json:"operation,omitempty"`
	RegistrationID  string `json:"registration_id"`
	At              string `json:"at"`
	Status          string `json:"status"`
	ConnectionCheck string `json:"connection_check"`
	Stage           string `json:"stage"`
}

type Config struct {
	Path          string `json:"path"`
	SHA256        string `json:"sha256,omitempty"`
	PendingSHA256 string `json:"pending_sha256,omitempty"`
}

type Onboarding struct {
	ID         string `json:"id"`
	HostID     string `json:"host_id"`
	KeyID      string `json:"key_id"`
	CreationID string `json:"creation_id,omitempty"`
	Stage      string `json:"stage"`
	CreatedAt  string `json:"created_at"`
}

type Creation struct {
	ID            string `json:"id"`
	KeyID         string `json:"key_id"`
	Name          string `json:"name"`
	PrivatePath   string `json:"private_path"`
	PublicPath    string `json:"public_path"`
	CreatedAt     string `json:"created_at"`
	Stage         string `json:"stage"`
	PrivateSHA256 string `json:"private_sha256,omitempty"`
	Fingerprint   string `json:"fingerprint,omitempty"`
}

func Directory(home string) string { return filepath.Join(home, ".ssh", "ssh-keyflow") }

func Load(home string) (State, error) {
	data, err := os.ReadFile(filepath.Join(Directory(home), "state.json"))
	if os.IsNotExist(err) {
		return State{Version: 2, Keys: []Key{}, Hosts: []Host{}, Registrations: []Registration{}}, nil
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
	if (document.Version != 1 && document.Version != 2) || document.Keys == nil || document.Hosts == nil || document.Registrations == nil {
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

func pathIdentity(path string) string {
	if path == "" {
		return ""
	}
	cleaned := filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return strings.ToLower(cleaned)
	}
	return cleaned
}

func (document State) Validate() error {
	if (document.Version != 1 && document.Version != 2) || document.Keys == nil || document.Hosts == nil || document.Registrations == nil {
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
	allKeys := append([]Key{}, document.Keys...)
	removedKeys := map[string]bool{}
	for _, key := range document.RemovedKeys {
		removedKeys[key.ID] = true
		allKeys = append(allKeys, key)
	}
	retired := map[string]DeletedKey{}
	activeKeys := map[string]bool{}
	for _, key := range document.Keys {
		activeKeys[key.ID] = true
	}
	for _, deleted := range document.DeletedKeys {
		deletedAt, err := time.Parse(time.RFC3339Nano, deleted.DeletedAt)
		if err != nil || deletedAt.IsZero() || !clean(deleted.DeletionID) || retired[deleted.Key.ID].Key.ID != "" || activeKeys[deleted.Key.ID] {
			return fmt.Errorf("invalid deleted key tombstone")
		}
		retired[deleted.Key.ID] = deleted
		allKeys = append(allKeys, deleted.Key)
	}
	for _, key := range allKeys {
		provenanceTime := key.ImportedAt
		if key.Origin == "created" {
			provenanceTime = key.CreatedAt
		}
		parsedTime, timeErr := time.Parse(time.RFC3339Nano, provenanceTime)
		if !oneOf(key.Origin, "imported", "created") || timeErr != nil || parsedTime.IsZero() || (key.Origin == "imported" && key.CreatedAt != "") || (key.Origin == "created" && (key.ImportedAt != "" || key.PrivatePath == "")) {
			return fmt.Errorf("invalid key provenance %q", key.ID)
		}
		digest, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(key.Fingerprint, "SHA256:"))
		if !claim(key.ID) || !clean(key.Name) || !strings.HasPrefix(key.Fingerprint, "SHA256:") || err != nil || len(digest) != 32 || (!removedKeys[key.ID] && (fingerprints[key.Fingerprint] || paths[pathIdentity(key.PublicPath)])) || !filepath.IsAbs(key.PublicPath) || !oneOf(key.ConnectionCheck, "unverified", "verified", "failed", "unknown") {
			return fmt.Errorf("invalid or duplicate key %q", key.ID)
		}
		if (key.PrivatePath == "") != (key.PrivateSHA256 == "") || (key.PrivatePath != "" && (!filepath.IsAbs(key.PrivatePath) || pathIdentity(key.PrivatePath) == pathIdentity(key.PublicPath) || (!removedKeys[key.ID] && paths[pathIdentity(key.PrivatePath)]) || len(key.PrivateSHA256) != 64)) {
			return fmt.Errorf("invalid private key metadata %q", key.ID)
		}
		if key.PrivateSHA256 != "" {
			if _, err := hex.DecodeString(key.PrivateSHA256); err != nil {
				return fmt.Errorf("invalid private key digest %q", key.ID)
			}
		}
		keys[key.ID] = true
		if !removedKeys[key.ID] {
			fingerprints[key.Fingerprint], paths[pathIdentity(key.PublicPath)] = true, true
			if key.PrivatePath != "" {
				paths[pathIdentity(key.PrivatePath)] = true
			}
		}
	}
	activeHosts := map[string]bool{}
	for _, host := range document.Hosts {
		activeHosts[host.ID] = true
	}
	allHosts := append(append([]Host{}, document.Hosts...), document.RemovedHosts...)
	for _, host := range allHosts {
		if !claim(host.ID) || host.Validate() != nil || (activeHosts[host.ID] && aliases[strings.ToLower(host.Alias)]) {
			return fmt.Errorf("invalid or duplicate host %q", host.ID)
		}
		hosts[host.ID] = true
		if activeHosts[host.ID] {
			aliases[strings.ToLower(host.Alias)] = true
		}
	}
	if document.Version == 1 && (len(document.Associations) > 0 || len(document.RemovedKeys) > 0 || len(document.RemovedHosts) > 0) {
		return fmt.Errorf("associations and removed records require state version 2")
	}
	associations := map[string]bool{}
	for _, association := range document.Associations {
		pair := association.KeyID + "/" + association.HostID
		if !activeKeys[association.KeyID] || !activeHosts[association.HostID] || associations[pair] {
			return fmt.Errorf("invalid or duplicate key association")
		}
		associations[pair] = true
	}
	for _, registration := range document.Registrations {
		pair := registration.KeyID + "/" + registration.HostID
		if !claim(registration.ID) || !keys[registration.KeyID] || !hosts[registration.HostID] || pairs[pair] || !oneOf(registration.Status, "pending", "registered", "removed", "unknown", "failed") || !oneOf(registration.ConnectionCheck, "unverified", "verified", "failed", "unknown") || !oneOf(registration.Operation, "register", "unregister") || !oneOf(registration.Stage, "planned", "pending", "complete", "unknown", "failed") {
			return fmt.Errorf("invalid registration or relationship %q", registration.ID)
		}
		pairs[pair] = true
	}
	for _, host := range allHosts {
		if host.ConfigKeyID != "" && (!(activeKeys[host.ConfigKeyID] || removedKeys[host.ConfigKeyID]) || !pairs[host.ConfigKeyID+"/"+host.ID]) {
			return fmt.Errorf("invalid configuration key for %q", host.ID)
		}
	}
	for _, event := range document.History {
		matched := false
		for _, registration := range document.Registrations {
			if registration.ID == event.RegistrationID {
				matched = true
			}
		}
		observedAt, err := time.Parse(time.RFC3339Nano, event.At)
		if !matched || err != nil || observedAt.IsZero() || !oneOf(event.Operation, "", "register", "unregister") || !oneOf(event.Status, "pending", "registered", "removed", "unknown", "failed") || !oneOf(event.ConnectionCheck, "unverified", "verified", "failed", "unknown") || !oneOf(event.Stage, "planned", "pending", "complete", "unknown", "failed") {
			return fmt.Errorf("invalid registration history")
		}
	}
	validDigest := func(value string) bool {
		decoded, err := hex.DecodeString(value)
		return err == nil && len(decoded) == 32
	}
	deletionKeys := map[string]bool{}
	for _, deletion := range document.Deletions {
		createdAt, err := time.Parse(time.RFC3339Nano, deletion.CreatedAt)
		if !claim(deletion.ID) || !keys[deletion.KeyID] || deletionKeys[deletion.KeyID] || err != nil || createdAt.IsZero() || !oneOf(deletion.Stage, "pending", "complete") || !validDigest(deletion.PublicSHA256) {
			return fmt.Errorf("invalid local deletion intent")
		}
		deletionKeys[deletion.KeyID] = true
		for _, key := range allKeys {
			if key.ID == deletion.KeyID && (deletion.PrivateSHA256 != key.PrivateSHA256 || (key.PrivatePath == "" && !deletion.PrivateRemoved)) {
				return fmt.Errorf("inconsistent deletion fingerprint/digest")
			}
		}
		if deletion.Stage == "complete" && (!deletion.PublicRemoved || !deletion.PrivateRemoved || retired[deletion.KeyID].DeletionID != deletion.ID) {
			return fmt.Errorf("incomplete deletion tombstone")
		}
		if deletion.Stage == "pending" && !activeKeys[deletion.KeyID] {
			return fmt.Errorf("pending deletion without managed key")
		}
		for _, registration := range document.Registrations {
			if registration.KeyID == deletion.KeyID && (registration.Status != "removed" || registration.Stage != "complete" || registration.Operation != "unregister") {
				return fmt.Errorf("deletion with outstanding recorded registration")
			}
		}
	}
	for keyID := range retired {
		if !deletionKeys[keyID] {
			return fmt.Errorf("tombstone missing deletion history")
		}
	}
	configPaths := map[string]bool{}
	for _, config := range document.Configs {
		if !filepath.IsAbs(config.Path) || configPaths[pathIdentity(config.Path)] || (config.SHA256 == "" && config.PendingSHA256 == "") || (config.SHA256 != "" && !validDigest(config.SHA256)) || (config.PendingSHA256 != "" && !validDigest(config.PendingSHA256)) {
			return fmt.Errorf("invalid configuration ownership")
		}
		configPaths[pathIdentity(config.Path)] = true
	}
	for _, edit := range document.ConfigEdits {
		if !filepath.IsAbs(edit.Path) || !validDigest(edit.BeforeSHA256) || !validDigest(edit.AfterSHA256) || (edit.BackupPath != "" && (!filepath.IsAbs(edit.BackupPath) || pathIdentity(filepath.Dir(edit.BackupPath)) != pathIdentity(filepath.Dir(edit.Path)) || pathIdentity(edit.BackupPath) == pathIdentity(edit.Path))) {
			return fmt.Errorf("invalid configuration edit intent")
		}
	}
	creationKeys := map[string]bool{}
	for _, creation := range document.Creations {
		createdAt, timeErr := time.Parse(time.RFC3339Nano, creation.CreatedAt)
		if !claim(creation.ID) || !clean(creation.KeyID) || creationKeys[creation.KeyID] || !clean(creation.Name) || !filepath.IsAbs(creation.PrivatePath) || pathIdentity(creation.PublicPath) != pathIdentity(creation.PrivatePath+".pub") || timeErr != nil || createdAt.IsZero() || !oneOf(creation.Stage, "planned", "pending", "complete") {
			return fmt.Errorf("invalid creation intent %q", creation.ID)
		}
		creationKeys[creation.KeyID] = true
		if creation.Stage == "complete" {
			matched := false
			for _, key := range allKeys {
				if key.ID == creation.KeyID && key.Origin == "created" && pathIdentity(key.PrivatePath) == pathIdentity(creation.PrivatePath) && pathIdentity(key.PublicPath) == pathIdentity(creation.PublicPath) && key.CreatedAt == creation.CreatedAt && key.PrivateSHA256 == creation.PrivateSHA256 && key.Fingerprint == creation.Fingerprint {
					matched = true
				}
			}
			if !matched {
				return fmt.Errorf("creation result missing or inconsistent %q", creation.ID)
			}
		} else {
			if !claim(creation.KeyID) || paths[pathIdentity(creation.PrivatePath)] || paths[pathIdentity(creation.PublicPath)] {
				return fmt.Errorf("creation identity/path conflict %q", creation.ID)
			}
			paths[pathIdentity(creation.PrivatePath)], paths[pathIdentity(creation.PublicPath)] = true, true
		}
		if creation.PrivateSHA256 != "" {
			digest, err := hex.DecodeString(creation.PrivateSHA256)
			if err != nil || len(digest) != 32 {
				return fmt.Errorf("invalid creation digest %q", creation.ID)
			}
		}
		if creation.Fingerprint != "" {
			digest, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(creation.Fingerprint, "SHA256:"))
			if !strings.HasPrefix(creation.Fingerprint, "SHA256:") || err != nil || len(digest) != 32 {
				return fmt.Errorf("invalid creation fingerprint %q", creation.ID)
			}
		}
	}
	onboardingHosts := map[string]bool{}
	for _, onboarding := range document.Onboardings {
		createdAt, timeErr := time.Parse(time.RFC3339Nano, onboarding.CreatedAt)
		if !claim(onboarding.ID) || !hosts[onboarding.HostID] || onboardingHosts[onboarding.HostID] || !oneOf(onboarding.Stage, "key_pending", "planned", "manual") || timeErr != nil || createdAt.IsZero() {
			return fmt.Errorf("invalid onboarding %q", onboarding.ID)
		}
		onboardingHosts[onboarding.HostID] = true
		if onboarding.CreationID != "" {
			found := false
			for _, creation := range document.Creations {
				if creation.ID == onboarding.CreationID && creation.KeyID == onboarding.KeyID {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("invalid onboarding creation %q", onboarding.ID)
			}
		}
		if !keys[onboarding.KeyID] && !(onboarding.Stage == "key_pending" && creationKeys[onboarding.KeyID] && onboarding.CreationID != "") {
			return fmt.Errorf("invalid onboarding key %q", onboarding.ID)
		}
		if onboarding.Stage == "manual" && !pairs[onboarding.KeyID+"/"+onboarding.HostID] {
			return fmt.Errorf("missing onboarding registration %q", onboarding.ID)
		}
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
	if document.Version != 1 && document.Version != 2 {
		return fmt.Errorf("unsupported schema/version")
	}
	document.Version = 2
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
