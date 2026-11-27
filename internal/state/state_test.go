package state

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const stamp = "2026-11-28T00:00:00Z"

func legacyFile(t *testing.T, home, body string) (string, []byte) {
	t.Helper()
	directory := Directory(home)
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "state.json")
	data := []byte(body)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path, data
}

func pairKey(home string) string {
	digest := sha256.Sum256([]byte("private"))
	private := filepath.Join(home, "id")
	return fmt.Sprintf(`{"id":"key-1","name":"one","fingerprint":"SHA256:%s","public_path":%q,"private_path":%q,"private_sha256":%q,"connection_check":"unverified","origin":"imported","imported_at":%q}`,
		strings.Repeat("A", 43), private+".pub", private, hex.EncodeToString(digest[:]), stamp)
}

func fixedClock(t *testing.T) {
	t.Helper()
	previous := now
	now = func() time.Time { return time.Date(2026, 11, 28, 1, 2, 3, 0, time.UTC) }
	t.Cleanup(func() { now = previous })
}

func TestLegacyV1ConvertsReadOnlyAndListsEveryDroppedRecordType(t *testing.T) {
	home := t.TempDir()
	body := `{"version":1,"keys":[` + pairKey(home) + `],"hosts":[{}],"registrations":[{}],"history":[{}],"onboardings":[{}],"configs":[{}],` +
		`"config_edits":[{"path":"/home/a/.ssh/config","before_sha256":"","after_sha256":"","complete":false}],"removed_keys":[{}],"deleted_keys":[{}]}`
	path, data := legacyFile(t, home, body)
	document, err := Load(home)
	if err != nil {
		t.Fatal(err)
	}
	if document.Version != Version || len(document.Keys) != 1 || document.Migration == nil || document.Migration.FromVersion != 1 {
		t.Fatalf("unexpected conversion: %+v", document)
	}
	notice := document.Migration.Notice()
	for _, want := range []string{"schema v1", "1 destination(s)", "1 registration record(s)", "1 registration history event(s)", "1 onboarding record(s)",
		"1 SSH config record(s)", "1 SSH config edit record(s)", `unfinished SSH config edit of "/home/a/.ssh/config"`, "1 archived removed key record(s)", "1 deleted key record(s)"} {
		if !strings.Contains(notice, want) {
			t.Errorf("notice lacks %q: %s", want, notice)
		}
	}
	if after, _ := os.ReadFile(path); !bytes.Equal(after, data) {
		t.Fatal("loading changed the legacy file")
	}
}

func TestLegacyDecodingStaysStrict(t *testing.T) {
	for name, body := range map[string]string{
		"unknown top-level field": `{"version":2,"keys":[],"hosts":[],"registrations":[],"bogus":1}`,
		"wrongly typed array":     `{"version":2,"keys":[],"hosts":[],"registrations":[],"removed_keys":"garbage"}`,
		"unknown key field":       `{"version":2,"keys":[{"id":"k","extra":1}],"hosts":[],"registrations":[]}`,
		"missing hosts":           `{"version":2,"keys":[],"registrations":[]}`,
		"trailing content":        `{"version":2,"keys":[],"hosts":[],"registrations":[]} {}`,
	} {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			legacyFile(t, home, body)
			if _, err := Load(home); err == nil {
				t.Fatal("corrupt legacy state was accepted")
			}
		})
	}
}

func saveLegacy(t *testing.T, home string) error {
	t.Helper()
	return WithLock(home, func(store *LockedStore) error {
		document, err := store.Load()
		if err != nil {
			return err
		}
		return store.Save(document)
	})
}

func TestArchiveCollisionGetsSuffixAndRetryReusesIdenticalArchive(t *testing.T) {
	fixedClock(t)
	home := t.TempDir()
	path, data := legacyFile(t, home, `{"version":2,"keys":[],"hosts":[],"registrations":[]}`)
	taken := filepath.Join(Directory(home), "state.v2-20261128T010203Z.json")
	if err := os.WriteFile(taken, []byte("someone else's file"), 0600); err != nil {
		t.Fatal(err)
	}
	first, err := archive(path, 2, data)
	if err != nil || filepath.Base(first) != "state.v2-20261128T010203Z_1.json" {
		t.Fatal("collision not resolved with a suffix", first, err)
	}
	if kept, _ := os.ReadFile(taken); string(kept) != "someone else's file" {
		t.Fatal("existing archive name was overwritten")
	}
	again, err := archive(path, 2, data)
	if err != nil || again != first {
		t.Fatal("retry did not reuse the identical archive", again, err)
	}
	if matches, _ := filepath.Glob(filepath.Join(Directory(home), ArchivePattern)); len(matches) != 2 {
		t.Fatal("unexpected archives", matches)
	}
}

func TestArchiveFailureLeavesStateUnchanged(t *testing.T) {
	fixedClock(t)
	home := t.TempDir()
	path, data := legacyFile(t, home, `{"version":2,"keys":[],"hosts":[],"registrations":[]}`)
	base := filepath.Join(Directory(home), "state.v2-20261128T010203Z")
	for suffix := range 100 {
		name := base + ".json"
		if suffix > 0 {
			name = fmt.Sprintf("%s_%d.json", base, suffix)
		}
		if err := os.WriteFile(name, []byte("occupied"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	err := saveLegacy(t, home)
	if err == nil || !strings.Contains(err.Error(), "state unchanged") {
		t.Fatal("archive failure was not reported", err)
	}
	if after, _ := os.ReadFile(path); !bytes.Equal(after, data) {
		t.Fatal("state.json changed after archive failure")
	}
	if _, err := os.Stat(path + ".bak"); !os.IsNotExist(err) {
		t.Fatal("backup written after archive failure", err)
	}
}

func TestFirstSaveArchivesV1ThenWritesCurrentSchema(t *testing.T) {
	home := t.TempDir()
	path, data := legacyFile(t, home, `{"version":1,"keys":[],"hosts":[],"registrations":[]}`)
	if err := saveLegacy(t, home); err != nil {
		t.Fatal(err)
	}
	matches, _ := filepath.Glob(filepath.Join(Directory(home), "state.v1-*.json"))
	if len(matches) != 1 {
		t.Fatal("v1 archive missing", matches)
	}
	if archived, _ := os.ReadFile(matches[0]); !bytes.Equal(archived, data) {
		t.Fatal("archive is not the exact v1 file")
	}
	document, err := Load(home)
	if err != nil || document.Migration != nil || document.Version != Version {
		t.Fatal("state not saved as current schema", document, err)
	}
	if saved, _ := os.ReadFile(path); bytes.Contains(saved, []byte("hosts")) {
		t.Fatal("legacy fields written")
	}
}
