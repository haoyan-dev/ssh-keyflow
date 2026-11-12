package state

import (
	"encoding/base64"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func hostKeyFixture(kind string, fill byte) HostKey {
	blob := binary.BigEndian.AppendUint32(nil, uint32(len(kind)))
	blob = append(blob, kind...)
	blob = binary.BigEndian.AppendUint32(blob, 32)
	for range 32 {
		blob = append(blob, fill)
	}
	return HostKey{Type: kind, Blob: base64.StdEncoding.EncodeToString(blob)}
}

func pinHome(t *testing.T) string {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(Directory(home), 0700); err != nil {
		t.Fatal(err)
	}
	return home
}

func savePins(t *testing.T, home, hostID string, keys ...HostKey) error {
	t.Helper()
	return WithLock(home, func(store *LockedStore) error { return store.SavePins(hostID, keys) })
}

func TestHostKeyValidateRequiresMatchingAlgorithm(t *testing.T) {
	for _, kind := range HostKeyTypes {
		if err := hostKeyFixture(kind, 1).Validate(); err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
	}
	mismatch := hostKeyFixture("ssh-ed25519", 1)
	mismatch.Type = "ssh-rsa"
	for _, key := range []HostKey{
		mismatch,
		hostKeyFixture("ssh-ed25519-cert-v01@openssh.com", 1),
		{Type: "ssh-ed25519", Blob: "not base64!"},
		{Type: "ssh-ed25519", Blob: "AAAA"},
		{Type: "ssh-ed25519", Blob: strings.TrimRight(hostKeyFixture("ssh-ed25519", 1).Blob, "=") + "A"},
	} {
		if key.Validate() == nil {
			t.Fatalf("accepted %+v", key)
		}
	}
}

func TestSavePinsKeepsOtherDestinationsAndRemovesOnlyOne(t *testing.T) {
	home := pinHome(t)
	if entries, err := LoadPins(home); err != nil || entries != nil {
		t.Fatal("missing file must have no pins", entries, err)
	}
	a, b, c := hostKeyFixture("ssh-ed25519", 1), hostKeyFixture("ssh-ed25519", 2), hostKeyFixture("ssh-rsa", 3)
	if err := savePins(t, home, "h1", a, c); err != nil {
		t.Fatal(err)
	}
	if err := savePins(t, home, "h2", b); err != nil {
		t.Fatal(err)
	}
	if err := savePins(t, home, "h1", b); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(PinPath(home))
	if err != nil {
		t.Fatal(err)
	}
	want := "ssh-keyflow-h1 ssh-ed25519 " + b.Blob + "\nssh-keyflow-h2 ssh-ed25519 " + b.Blob + "\n"
	if string(data) != want {
		t.Fatalf("got %q want %q", data, want)
	}
	if err := savePins(t, home, "h1"); err != nil {
		t.Fatal(err)
	}
	entries, err := LoadPins(home)
	if err != nil || len(entries) != 1 || entries[0].HostID != "h2" {
		t.Fatal(entries, err)
	}
}

func TestMalformedPinFileIsRefusedAndNeverRewritten(t *testing.T) {
	key := hostKeyFixture("ssh-ed25519", 1)
	valid := "ssh-keyflow-h1 ssh-ed25519 " + key.Blob + "\n"
	for name, text := range map[string]string{
		"no final newline": strings.TrimSuffix(valid, "\n"),
		"foreign name":     "server ssh-ed25519 " + key.Blob + "\n",
		"empty id":         "ssh-keyflow- ssh-ed25519 " + key.Blob + "\n",
		"comment":          strings.TrimSuffix(valid, "\n") + " root@host\n",
		"double space":     "ssh-keyflow-h1  ssh-ed25519 " + key.Blob + "\n",
		"tab":              "ssh-keyflow-h1\tx ssh-ed25519 " + key.Blob + "\n",
		"duplicate":        valid + valid,
		"marker":           "@revoked ssh-keyflow-h1 ssh-ed25519 " + key.Blob + "\n",
		"blank line":       valid + "\n",
		"crlf":             strings.TrimSuffix(valid, "\n") + "\r\n",
	} {
		t.Run(name, func(t *testing.T) {
			home := pinHome(t)
			if err := os.WriteFile(PinPath(home), []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadPins(home); err == nil || !strings.Contains(err.Error(), "review the file manually") {
				t.Fatal("accepted malformed file", err)
			}
			if err := savePins(t, home, "h2", key); err == nil {
				t.Fatal("rewrote malformed file")
			}
			if data, _ := os.ReadFile(PinPath(home)); string(data) != text {
				t.Fatal("malformed file changed")
			}
		})
	}
}

func TestSavePinsRefusesUnsafeIDsAndSymlinks(t *testing.T) {
	home := pinHome(t)
	key := hostKeyFixture("ssh-ed25519", 1)
	for _, id := range []string{"", "a b", "a\tb", "a\nb"} {
		if err := savePins(t, home, id, key); err == nil {
			t.Fatalf("accepted id %q", id)
		}
	}
	if err := savePins(t, home, "h1", HostKey{Type: "ssh-ed25519", Blob: "AAAA"}); err == nil {
		t.Fatal("saved invalid key")
	}
	if _, err := os.Lstat(PinPath(home)); !os.IsNotExist(err) {
		t.Fatal("refused writes created the pin file")
	}
	target := filepath.Join(home, "elsewhere")
	if err := os.WriteFile(target, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, PinPath(home)); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	if _, err := LoadPins(home); err == nil {
		t.Fatal("followed symlinked pin file")
	}
}

func TestSavePinsReplacesAtFirstEntryWhenSplitInTheMiddle(t *testing.T) {
	home := pinHome(t)
	a, b, c := hostKeyFixture("ssh-ed25519", 1), hostKeyFixture("ssh-ed25519", 2), hostKeyFixture("ssh-rsa", 3)
	line := func(id string, key HostKey) string {
		return "ssh-keyflow-" + id + " " + key.Type + " " + key.Blob + "\n"
	}
	if err := os.WriteFile(PinPath(home), []byte(line("h0", a)+line("h1", a)+line("h2", b)+line("h1", c)+line("h3", a)), 0600); err != nil {
		t.Fatal(err)
	}
	if err := savePins(t, home, "h1", b, c); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(PinPath(home))
	if err != nil {
		t.Fatal(err)
	}
	if want := line("h0", a) + line("h1", b) + line("h1", c) + line("h2", b) + line("h3", a); string(data) != want {
		t.Fatalf("got %q want %q", data, want)
	}
}
