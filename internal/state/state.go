package state

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"ssh-keyflow/internal/roster"
)

// Version is the only schema this binary writes. v4 adds this machine's
// node description and fleet data (ADR 0008); it is unreleased until fleet
// sync phase 1 ships, so its layout may still change before then.
const Version = 4

// Key is a managed local key pair. Private key contents are never stored;
// PrivateSHA256 only detects replacement of the recorded file.
type Key struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Fingerprint   string `json:"fingerprint"`
	PublicPath    string `json:"public_path"`
	PrivatePath   string `json:"private_path"`
	PrivateSHA256 string `json:"private_sha256"`
	Origin        string `json:"origin"`
	CreatedAt     string `json:"created_at,omitempty"`
	ImportedAt    string `json:"imported_at,omitempty"`
}

// Creation is an unfinished local key creation that resume can finish.
// Completed creations are removed in the same save that adds their key.
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

// Node is this machine's node description. ClientKeyIDs refer to managed
// keys; HostKeys hold the sshd host public keys as "<type> <base64>", read
// from SSHDConfig when the description was last set.
type Node struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Roles        []string `json:"roles"`
	LoginAccount string   `json:"login_account,omitempty"`
	Addresses    []string `json:"addresses,omitempty"`
	Port         int      `json:"port,omitempty"`
	ClientKeyIDs []string `json:"client_key_ids,omitempty"`
	HostKeys     []string `json:"host_keys,omitempty"`
	SSHDConfig   string   `json:"sshd_config,omitempty"`
	UpdatedAt    string   `json:"updated_at"`
}

// Fleet is the fleet this node belongs to. Roots are the pinned fleet root
// public keys; RootKey is set only on an admin node, which holds the private
// key at RootKey.PrivatePath (never its contents). Accepted is the highest
// accepted roster, kept as its exact signed text.
type Fleet struct {
	ID       string    `json:"id"`
	Roots    []Root    `json:"roots"`
	RootKey  *RootKey  `json:"root_key,omitempty"`
	Accepted *Accepted `json:"accepted,omitempty"`
}

type Root struct {
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint"`
	PinnedAt    string `json:"pinned_at"`
}

type RootKey struct {
	Fingerprint string `json:"fingerprint"`
	PublicPath  string `json:"public_path"`
	PrivatePath string `json:"private_path"`
}

type Accepted struct {
	Version    int64  `json:"version"`
	SHA256     string `json:"sha256"`
	Roster     string `json:"roster"`
	AcceptedAt string `json:"accepted_at"`
}

type State struct {
	Version   int        `json:"version"`
	Keys      []Key      `json:"keys"`
	Creations []Creation `json:"creations,omitempty"`
	Node      *Node      `json:"node,omitempty"`
	Fleet     *Fleet     `json:"fleet,omitempty"`
	// Migration is set when the file on disk uses an older schema. It is never saved.
	Migration *Migration `json:"-"`
}

// Migration describes what the first save will do to an older state file.
type Migration struct {
	FromVersion int
	// Dropped lists records that are not carried into the current schema.
	Dropped []string
}

func Directory(home string) string { return filepath.Join(home, ".ssh", "ssh-keyflow") }

func empty() State { return State{Version: Version, Keys: []Key{}} }

func Load(home string) (State, error) {
	data, err := os.ReadFile(filepath.Join(Directory(home), "state.json"))
	if os.IsNotExist(err) {
		return empty(), nil
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
	var header struct {
		Version *int `json:"version"`
	}
	if err := json.Unmarshal(data, &header); err != nil || header.Version == nil {
		return State{}, fmt.Errorf("invalid state.json; restore a verified backup: missing or invalid version")
	}
	switch *header.Version {
	case Version:
	case 3:
		return decodeV3(data)
	case 1, 2:
		return decodeLegacy(data)
	default:
		return State{}, fmt.Errorf("unsupported state.json schema version %d; updates stopped", *header.Version)
	}
	var document State
	if err := strictDecode(data, &document); err != nil {
		return State{}, err
	}
	if err := document.Validate(); err != nil {
		return State{}, fmt.Errorf("invalid state.json: %w", err)
	}
	return document, nil
}

func strictDecode(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid state.json; restore a verified backup: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("invalid state.json: trailing content")
	}
	return nil
}

// Notice explains, before any change, what the next save does to an older
// state file. Every changing command and keys list show it.
func (migration *Migration) Notice() string {
	note := fmt.Sprintf("state.json uses schema v%d. The next change copies it to %s next to state.json and saves schema v%d", migration.FromVersion, ArchivePattern, Version)
	if migration.FromVersion < 3 {
		note += " with local key pairs and unfinished creations only"
	}
	note += ". SSH KeyFlow v0.11.0 and earlier cannot read schema v" + fmt.Sprint(Version) + "."
	if len(migration.Dropped) > 0 {
		note += " Not carried over: " + strings.Join(migration.Dropped, "; ") + "."
	}
	return note
}

// decodeV3 converts schema v3 in memory, with v3's strict rules. Every v3
// record carries over; the original file is archived on the first save.
func decodeV3(data []byte) (State, error) {
	var v3 struct {
		Version   int        `json:"version"`
		Keys      []Key      `json:"keys"`
		Creations []Creation `json:"creations,omitempty"`
	}
	if err := strictDecode(data, &v3); err != nil {
		return State{}, err
	}
	document := State{Version: Version, Keys: v3.Keys, Creations: v3.Creations}
	if err := document.Validate(); err != nil {
		return State{}, fmt.Errorf("invalid state.json: %w", err)
	}
	document.Migration = &Migration{FromVersion: 3}
	return document, nil
}

// legacyKey is a v1/v2 key; connection_check is accepted and dropped.
type legacyKey struct {
	Key
	ConnectionCheck string `json:"connection_check"`
}

// decodeLegacy converts schema v1/v2 in memory. Unknown fields and wrongly
// typed arrays are rejected as before. Only local key pairs and unfinished
// creations are carried over; every other non-empty record type is listed in
// Migration.Dropped, and the original file is archived on the first save.
func decodeLegacy(data []byte) (State, error) {
	var legacy struct {
		Version       int               `json:"version"`
		Keys          []legacyKey       `json:"keys"`
		Hosts         []json.RawMessage `json:"hosts"`
		Registrations []json.RawMessage `json:"registrations"`
		Creations     []Creation        `json:"creations"`
		Associations  []json.RawMessage `json:"associations"`
		RemovedKeys   []json.RawMessage `json:"removed_keys"`
		RemovedHosts  []json.RawMessage `json:"removed_hosts"`
		DeletedKeys   []json.RawMessage `json:"deleted_keys"`
		Deletions     []json.RawMessage `json:"deletions"`
		Onboardings   []json.RawMessage `json:"onboardings"`
		History       []json.RawMessage `json:"history"`
		Configs       []json.RawMessage `json:"configs"`
		ConfigEdits   []json.RawMessage `json:"config_edits"`
	}
	if err := strictDecode(data, &legacy); err != nil {
		return State{}, err
	}
	if legacy.Keys == nil || legacy.Hosts == nil || legacy.Registrations == nil {
		return State{}, fmt.Errorf("unsupported or incomplete state.json schema/version; updates stopped")
	}
	pendingDeletion := map[string]bool{}
	for _, raw := range legacy.Deletions {
		var deletion struct {
			KeyID string `json:"key_id"`
			Stage string `json:"stage"`
		}
		if err := json.Unmarshal(raw, &deletion); err != nil {
			return State{}, fmt.Errorf("invalid state.json; restore a verified backup: deletions: %w", err)
		}
		if deletion.Stage != "complete" {
			pendingDeletion[deletion.KeyID] = true
		}
	}
	document := empty()
	migration := &Migration{FromVersion: legacy.Version}
	drop := func(format string, arguments ...any) {
		migration.Dropped = append(migration.Dropped, fmt.Sprintf(format, arguments...))
	}
	for _, raw := range legacy.ConfigEdits {
		var edit struct {
			Path     string `json:"path"`
			Complete bool   `json:"complete"`
		}
		if err := json.Unmarshal(raw, &edit); err != nil {
			return State{}, fmt.Errorf("invalid state.json; restore a verified backup: config_edits: %w", err)
		}
		if !edit.Complete {
			drop("an unfinished SSH config edit of %q (check that file manually)", edit.Path)
		}
	}
	for _, legacyKey := range legacy.Keys {
		key := legacyKey.Key
		switch {
		case key.PrivatePath == "":
			drop("public-only key %q [%s]", key.Name, key.ID)
		case pendingDeletion[key.ID]:
			drop("key %q [%s] with an unfinished legacy deletion", key.Name, key.ID)
		default:
			document.Keys = append(document.Keys, key)
		}
	}
	for _, creation := range legacy.Creations {
		if creation.Stage != "complete" {
			document.Creations = append(document.Creations, creation)
		}
	}
	for _, records := range []struct {
		items []json.RawMessage
		label string
	}{
		{legacy.Hosts, "destination(s)"},
		{legacy.Associations, "association(s)"},
		{legacy.RemovedKeys, "archived removed key record(s)"},
		{legacy.RemovedHosts, "archived removed destination record(s)"},
		{legacy.DeletedKeys, "deleted key record(s)"},
		{legacy.Deletions, "local key deletion record(s)"},
		{legacy.Registrations, "registration record(s)"},
		{legacy.History, "registration history event(s)"},
		{legacy.Onboardings, "onboarding record(s)"},
		{legacy.Configs, "SSH config record(s)"},
		{legacy.ConfigEdits, "SSH config edit record(s)"},
	} {
		if len(records.items) > 0 {
			drop("%d %s", len(records.items), records.label)
		}
	}
	if err := document.Validate(); err != nil {
		return State{}, fmt.Errorf("invalid legacy state.json: %w", err)
	}
	document.Migration = migration
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
	cleaned := filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return strings.ToLower(cleaned)
	}
	return cleaned
}

func validTime(value string) bool {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	return err == nil && !parsed.IsZero()
}

func validFingerprint(value string) bool {
	digest, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(value, "SHA256:"))
	return strings.HasPrefix(value, "SHA256:") && err == nil && len(digest) == 32
}

func validDigest(value string) bool {
	digest, err := hex.DecodeString(value)
	return err == nil && len(digest) == 32
}

func (document State) Validate() error {
	if document.Version != Version || document.Keys == nil {
		return fmt.Errorf("unsupported or incomplete schema/version")
	}
	identities, fingerprints, paths := map[string]bool{}, map[string]bool{}, map[string]bool{}
	claim := func(set map[string]bool, value string) bool {
		if set[value] {
			return false
		}
		set[value] = true
		return true
	}
	for _, key := range document.Keys {
		provenance := (key.Origin == "created" && validTime(key.CreatedAt) && key.ImportedAt == "") ||
			(key.Origin == "imported" && validTime(key.ImportedAt) && key.CreatedAt == "")
		if !provenance {
			return fmt.Errorf("invalid key provenance %q", key.ID)
		}
		if !clean(key.ID) || !clean(key.Name) || !validFingerprint(key.Fingerprint) || !validDigest(key.PrivateSHA256) ||
			!filepath.IsAbs(key.PublicPath) || !filepath.IsAbs(key.PrivatePath) || pathIdentity(key.PublicPath) == pathIdentity(key.PrivatePath) {
			return fmt.Errorf("invalid key metadata %q", key.ID)
		}
		if !claim(identities, key.ID) || !claim(fingerprints, key.Fingerprint) || !claim(paths, pathIdentity(key.PublicPath)) || !claim(paths, pathIdentity(key.PrivatePath)) {
			return fmt.Errorf("duplicate key identity, fingerprint or path %q", key.ID)
		}
	}
	for _, creation := range document.Creations {
		if !clean(creation.ID) || !clean(creation.KeyID) || !clean(creation.Name) || !filepath.IsAbs(creation.PrivatePath) ||
			pathIdentity(creation.PublicPath) != pathIdentity(creation.PrivatePath+".pub") || !validTime(creation.CreatedAt) ||
			!oneOf(creation.Stage, "planned", "pending") ||
			(creation.PrivateSHA256 != "" && !validDigest(creation.PrivateSHA256)) ||
			(creation.Fingerprint != "" && !validFingerprint(creation.Fingerprint)) {
			return fmt.Errorf("invalid creation intent %q", creation.ID)
		}
		if !claim(identities, creation.ID) || !claim(identities, creation.KeyID) || !claim(paths, pathIdentity(creation.PrivatePath)) || !claim(paths, pathIdentity(creation.PublicPath)) {
			return fmt.Errorf("creation identity/path conflict %q", creation.ID)
		}
	}
	if document.Node != nil {
		if err := document.validateNode(); err != nil {
			return fmt.Errorf("invalid node: %w", err)
		}
	}
	if document.Fleet != nil {
		if err := document.Fleet.validate(); err != nil {
			return fmt.Errorf("invalid fleet: %w", err)
		}
	}
	return nil
}

func validID(prefix, value string) bool {
	digest, err := hex.DecodeString(strings.TrimPrefix(value, prefix+"-"))
	return strings.HasPrefix(value, prefix+"-") && err == nil && len(digest) == 16 && strings.ToLower(value) == value
}

// uniqueSorted reports whether values are strictly increasing in byte order,
// which also rules out duplicates.
func uniqueSorted(values []string) bool {
	return slices.IsSorted(values) && len(slices.Compact(slices.Clone(values))) == len(values)
}

func (document State) validateNode() error {
	node := document.Node
	if !validID("node", node.ID) || !roster.ValidName(node.Name) || !validTime(node.UpdatedAt) {
		return fmt.Errorf("invalid id, name or time")
	}
	roles, err := roster.Roles(node.Roles)
	if err != nil || !slices.Equal(roles, node.Roles) {
		return fmt.Errorf("roles must be client, remote or both, in that order")
	}
	client, remote := slices.Contains(roles, roster.RoleClient), slices.Contains(roles, roster.RoleRemote)
	if client != (len(node.ClientKeyIDs) > 0) || len(node.ClientKeyIDs) > roster.MaxKeys || !uniqueSorted(node.ClientKeyIDs) {
		return fmt.Errorf("a client needs 1 to %d managed keys, and only a client has them", roster.MaxKeys)
	}
	for _, id := range node.ClientKeyIDs {
		if !slices.ContainsFunc(document.Keys, func(key Key) bool { return key.ID == id }) {
			return fmt.Errorf("client key %q is not managed", id)
		}
	}
	if !remote {
		if node.LoginAccount != "" || node.Addresses != nil || node.Port != 0 || node.HostKeys != nil || node.SSHDConfig != "" {
			return fmt.Errorf("only a remote has login_account, addresses, port, host_keys and sshd_config")
		}
		return nil
	}
	if node.LoginAccount != "" && !roster.ValidAccount(node.LoginAccount) {
		return fmt.Errorf("invalid login_account")
	}
	if len(node.Addresses) == 0 || len(node.Addresses) > roster.MaxAddresses || !uniqueSorted(node.Addresses) ||
		slices.ContainsFunc(node.Addresses, func(address string) bool { return !roster.ValidAddress(address) }) {
		return fmt.Errorf("a remote needs 1 to %d valid, sorted, unique addresses", roster.MaxAddresses)
	}
	if node.Port == roster.DefaultPort || node.Port < 0 || node.Port > 65535 {
		return fmt.Errorf("port must be 1-65535 and is omitted when 22")
	}
	if len(node.HostKeys) == 0 || len(node.HostKeys) > roster.MaxKeys || !uniqueSorted(node.HostKeys) {
		return fmt.Errorf("a remote needs 1 to %d sorted, unique host keys", roster.MaxKeys)
	}
	for _, text := range node.HostKeys {
		if key, err := roster.ParsePublicKey(text, true); err != nil || key.Text != text {
			return fmt.Errorf("invalid host key")
		}
	}
	if !filepath.IsAbs(node.SSHDConfig) {
		return fmt.Errorf("sshd_config must be an absolute path")
	}
	return nil
}

func (fleet Fleet) validate() error {
	if !validID("fleet", fleet.ID) || len(fleet.Roots) == 0 {
		return fmt.Errorf("invalid id or no pinned root")
	}
	pinned := map[string]bool{}
	for _, root := range fleet.Roots {
		key, err := roster.ParsePublicKey(root.PublicKey, false)
		if err != nil || key.Type != "ssh-ed25519" || key.Text != root.PublicKey || key.Fingerprint != root.Fingerprint || !validTime(root.PinnedAt) || pinned[key.Text] {
			return fmt.Errorf("invalid, non-Ed25519 or duplicate pinned root")
		}
		pinned[key.Text] = true
	}
	if root := fleet.RootKey; root != nil {
		if !validFingerprint(root.Fingerprint) || !filepath.IsAbs(root.PublicPath) || !filepath.IsAbs(root.PrivatePath) ||
			pathIdentity(root.PublicPath) == pathIdentity(root.PrivatePath) ||
			!slices.ContainsFunc(fleet.Roots, func(pin Root) bool { return pin.Fingerprint == root.Fingerprint }) {
			return fmt.Errorf("invalid fleet root key, or it is not pinned")
		}
	}
	if accepted := fleet.Accepted; accepted != nil {
		digest := sha256.Sum256([]byte(accepted.Roster))
		if accepted.Version < 1 || accepted.Version > 1<<53-1 || accepted.SHA256 != hex.EncodeToString(digest[:]) || !validTime(accepted.AcceptedAt) {
			return fmt.Errorf("invalid accepted roster record")
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

// Save validates and atomically replaces state.json. The previous valid bytes
// go to state.json.bak. An older-schema file is first copied to a new, never
// overwritten state.v<N>-<UTC time>.json archive.
func (store *LockedStore) Save(document State) error {
	if !store.active {
		return fmt.Errorf("operation lock is no longer held")
	}
	document.Migration = nil
	if err := document.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(Directory(store.home), "state.json")
	previous, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		current, err := decode(previous)
		if err != nil {
			return err
		}
		if current.Migration != nil {
			if _, err := archive(path, current.Migration.FromVersion, previous); err != nil {
				return fmt.Errorf("archive of schema v%d state failed; state unchanged: %w", current.Migration.FromVersion, err)
			}
		}
		if err := atomicWrite(path+".bak", previous); err != nil {
			return fmt.Errorf("backup failed; state unchanged: %w", err)
		}
	}
	return atomicWrite(path, append(data, '\n'))
}

// ArchivePattern matches files written by archive, for documentation and tests.
const ArchivePattern = "state.v*-*.json"

// now is replaced in tests to make archive names predictable.
var now = time.Now

// archive writes data to a new state.v<N>-<UTC>.json. If an earlier attempt
// already archived the same bytes (its save then failed), that file is reused
// so retries do not pile up identical archives.
func archive(statePath string, version int, data []byte) (string, error) {
	directory := filepath.Dir(statePath)
	existing, err := filepath.Glob(filepath.Join(directory, fmt.Sprintf("state.v%d-*.json", version)))
	if err != nil {
		return "", err
	}
	for _, candidate := range existing {
		if previous, err := os.ReadFile(candidate); err == nil && bytes.Equal(previous, data) {
			return candidate, nil
		}
	}
	base := filepath.Join(directory, fmt.Sprintf("state.v%d-%s", version, now().UTC().Format("20060102T150405Z")))
	for suffix := 0; suffix < 100; suffix++ {
		candidate := base + ".json"
		if suffix > 0 {
			candidate = fmt.Sprintf("%s_%d.json", base, suffix)
		}
		file, err := os.OpenFile(candidate, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		_, writeErr := file.Write(data)
		syncErr := file.Sync()
		closeErr := file.Close()
		for _, err := range []error{writeErr, syncErr, closeErr} {
			if err != nil {
				return "", err
			}
		}
		return candidate, nil
	}
	return "", fmt.Errorf("no free archive name for %s", base)
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
