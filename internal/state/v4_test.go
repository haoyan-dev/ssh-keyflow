package state

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ssh-keyflow/internal/roster"
)

func ed25519Key(seed byte) string {
	var blob []byte
	for _, part := range [][]byte{[]byte("ssh-ed25519"), bytes.Repeat([]byte{seed}, 32)} {
		blob = binary.BigEndian.AppendUint32(blob, uint32(len(part)))
		blob = append(blob, part...)
	}
	return "ssh-ed25519 " + base64.StdEncoding.EncodeToString(blob)
}

func ecdsaKey() string {
	var blob []byte
	for _, part := range [][]byte{[]byte("ecdsa-sha2-nistp256"), []byte("nistp256"), bytes.Repeat([]byte{4}, 65)} {
		blob = binary.BigEndian.AppendUint32(blob, uint32(len(part)))
		blob = append(blob, part...)
	}
	return "ecdsa-sha2-nistp256 " + base64.StdEncoding.EncodeToString(blob)
}

func fingerprintOf(text string) string {
	blob, _ := base64.StdEncoding.DecodeString(strings.Fields(text)[1])
	digest := sha256.Sum256(blob)
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(digest[:])
}

func v3Key(home string) string {
	return strings.Replace(pairKey(home), `"connection_check":"unverified",`, "", 1)
}

func TestV3ConvertsReadOnlyAndFirstSaveArchivesIt(t *testing.T) {
	home := t.TempDir()
	path, data := legacyFile(t, home, `{"version":3,"keys":[`+v3Key(home)+`]}`)
	document, err := Load(home)
	if err != nil {
		t.Fatal(err)
	}
	if document.Version != Version || len(document.Keys) != 1 || document.Migration == nil || document.Migration.FromVersion != 3 {
		t.Fatalf("unexpected conversion: %+v", document)
	}
	notice := document.Migration.Notice()
	if !strings.Contains(notice, "schema v3") || !strings.Contains(notice, "saves schema v4.") || strings.Contains(notice, "only") || strings.Contains(notice, "Not carried over") {
		t.Fatal("unexpected v3 notice:", notice)
	}
	if after, _ := os.ReadFile(path); !bytes.Equal(after, data) {
		t.Fatal("loading changed the v3 file")
	}
	if err := saveLegacy(t, home); err != nil {
		t.Fatal(err)
	}
	matches, _ := filepath.Glob(filepath.Join(Directory(home), "state.v3-*.json"))
	if len(matches) != 1 {
		t.Fatal("v3 archive missing", matches)
	}
	if archived, _ := os.ReadFile(matches[0]); !bytes.Equal(archived, data) {
		t.Fatal("archive is not the exact v3 file")
	}
	if backup, _ := os.ReadFile(path + ".bak"); !bytes.Equal(backup, data) {
		t.Fatal("backup is not the previous v3 file")
	}
	saved, err := Load(home)
	if err != nil || saved.Migration != nil || saved.Version != 4 || len(saved.Keys) != 1 {
		t.Fatal("not saved as v4", saved, err)
	}
}

func TestV3StaysStrict(t *testing.T) {
	for name, body := range map[string]string{
		"v4 field in v3": `{"version":3,"keys":[],"node":null}`,
		"missing keys":   `{"version":3}`,
		"unknown field":  `{"version":3,"keys":[],"hosts":[]}`,
	} {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			legacyFile(t, home, body)
			if _, err := Load(home); err == nil {
				t.Fatal("invalid v3 state accepted")
			}
		})
	}
}

func validV4(t *testing.T, home string) State {
	t.Helper()
	var key Key
	if err := json.Unmarshal([]byte(v3Key(home)), &key); err != nil {
		t.Fatal(err)
	}
	text := testRoster(t, 1)
	digest := sha256.Sum256([]byte(text))
	root := ed25519Key(9)
	return State{
		Version: Version, Keys: []Key{key},
		Node: &Node{ID: "node-" + strings.Repeat("ab", 16), Name: "laptop", Roles: []string{"client", "remote"}, Addresses: []string{"192.0.2.10", "laptop.lan"},
			Port: 2222, ClientKeyIDs: []string{key.ID}, HostKeys: []string{ed25519Key(1), ed25519Key(2)}, SSHDConfig: filepath.Join(home, "sshd_config"), UpdatedAt: stamp},
		Fleet: &Fleet{ID: "fleet-" + strings.Repeat("cd", 16), Roots: []Root{{PublicKey: root, Fingerprint: fingerprintOf(root), PinnedAt: stamp}},
			RootKey:  &RootKey{Fingerprint: fingerprintOf(root), PublicPath: filepath.Join(home, "root.pub"), PrivatePath: filepath.Join(home, "root")},
			Accepted: &Accepted{Version: 1, SHA256: hex.EncodeToString(digest[:]), Roster: text, Signature: testSignature, AcceptedAt: stamp}},
	}
}

const testFleetID = "fleet-cdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcd"

// testSignature has the armor and SSHSIG magic; state does not verify it.
var testSignature = "-----BEGIN SSH SIGNATURE-----\n" + base64.StdEncoding.EncodeToString([]byte("SSHSIG\x00\x00\x00\x01")) + "\n-----END SSH SIGNATURE-----\n"

func testRoster(t *testing.T, version int64) string {
	t.Helper()
	data, err := (roster.Roster{Format: roster.Format, Schema: roster.Schema, FleetID: testFleetID, Version: version, CreatedAt: stamp, LoginAccount: "admin",
		Nodes: []roster.Node{{ID: "node-" + strings.Repeat("ab", 16), Name: "laptop", Roles: []string{"client"}, ClientKeys: []string{ed25519Key(3)}}}}).Marshal()
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func setRoster(s *State, text string) {
	digest := sha256.Sum256([]byte(text))
	s.Fleet.Accepted.Roster, s.Fleet.Accepted.SHA256 = text, hex.EncodeToString(digest[:])
}

func TestV4RoundTripAndValidation(t *testing.T) {
	home := t.TempDir()
	document := validV4(t, home)
	if err := CheckWritable(home); err != nil {
		t.Fatal(err)
	}
	if err := WithLock(home, func(store *LockedStore) error { return store.Save(document) }); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(home)
	if err != nil || loaded.Node == nil || loaded.Fleet == nil || loaded.Node.Port != 2222 || loaded.Fleet.Accepted.Roster != document.Fleet.Accepted.Roster || loaded.Fleet.Accepted.Signature != testSignature {
		t.Fatal("v4 round trip failed", loaded, err)
	}
	if err := (State{Version: Version, Keys: []Key{}}).Validate(); err != nil {
		t.Fatal("empty v4 state rejected", err)
	}
	for name, change := range map[string]func(*State){
		"bad node id":           func(s *State) { s.Node.ID = "node-1" },
		"uppercase name":        func(s *State) { s.Node.Name = "Laptop" },
		"roles out of order":    func(s *State) { s.Node.Roles = []string{"remote", "client"} },
		"client without key":    func(s *State) { s.Node.ClientKeyIDs = nil },
		"unmanaged client key":  func(s *State) { s.Node.ClientKeyIDs = []string{"key-unknown"} },
		"port 22 written":       func(s *State) { s.Node.Port = 22 },
		"unsorted addresses":    func(s *State) { s.Node.Addresses = []string{"laptop.lan", "192.0.2.10"} },
		"duplicate host key":    func(s *State) { s.Node.HostKeys = []string{ed25519Key(1), ed25519Key(1)} },
		"host key comment":      func(s *State) { s.Node.HostKeys = []string{ed25519Key(1) + " c"} },
		"client with addresses": func(s *State) { s.Node.Roles = []string{"client"} },
		"relative sshd config":  func(s *State) { s.Node.SSHDConfig = "sshd_config" },
		"bad account":           func(s *State) { s.Node.LoginAccount = "a b" },
		"no root":               func(s *State) { s.Fleet.Roots = nil },
		"non-ed25519 root": func(s *State) {
			s.Fleet.Roots = []Root{{PublicKey: ecdsaKey(), Fingerprint: fingerprintOf(ecdsaKey()), PinnedAt: stamp}}
			s.Fleet.RootKey = nil
		},
		"root fingerprint":  func(s *State) { s.Fleet.Roots[0].Fingerprint = fingerprintOf(ed25519Key(8)) },
		"unpinned root key": func(s *State) { s.Fleet.RootKey.Fingerprint = fingerprintOf(ed25519Key(8)) },
		"roster digest":     func(s *State) { s.Fleet.Accepted.Roster = strings.TrimSuffix(s.Fleet.Accepted.Roster, "\n") },
		"version zero":      func(s *State) { s.Fleet.Accepted.Version = 0 },
		"version mismatch":  func(s *State) { s.Fleet.Accepted.Version = 2 },
		"fleet id mismatch": func(s *State) { s.Fleet.ID = "fleet-" + strings.Repeat("ef", 16) },
		"no signature":      func(s *State) { s.Fleet.Accepted.Signature = "" },
		"crlf signature":    func(s *State) { s.Fleet.Accepted.Signature = strings.ReplaceAll(testSignature, "\n", "\r\n") },
		"non-canonical roster": func(s *State) {
			setRoster(s, strings.Replace(s.Fleet.Accepted.Roster, "  ", "\t", 1))
		},
		"empty roster": func(s *State) { setRoster(s, "{}\n") },
	} {
		t.Run(name, func(t *testing.T) {
			broken := validV4(t, home)
			change(&broken)
			if err := broken.Validate(); err == nil {
				t.Fatal("invalid v4 state accepted")
			}
		})
	}
}

func TestV4UnknownNodeFieldIsRejected(t *testing.T) {
	home := t.TempDir()
	legacyFile(t, home, `{"version":4,"keys":[],"node":{"id":"node-`+strings.Repeat("ab", 16)+`","name":"x","roles":["client"],"updated_at":"`+stamp+`","extra":1}}`)
	if _, err := Load(home); err == nil {
		t.Fatal("unknown node field accepted")
	}
}
