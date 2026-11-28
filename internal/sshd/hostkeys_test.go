package sshd

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func hostKey(seed byte) string {
	var blob []byte
	for _, part := range [][]byte{[]byte("ssh-ed25519"), bytes.Repeat([]byte{seed}, 32)} {
		blob = binary.BigEndian.AppendUint32(blob, uint32(len(part)))
		blob = append(blob, part...)
	}
	return "ssh-ed25519 " + base64.StdEncoding.EncodeToString(blob)
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}

func texts(keys []HostKey) string {
	var out []string
	for _, key := range keys {
		out = append(out, key.Key.Text)
	}
	return strings.Join(out, "|")
}

func TestHostKeyEntriesIncludesAndMatch(t *testing.T) {
	dir := t.TempDir()
	first, second, inMatch := filepath.Join(dir, "keys", "a_key"), filepath.Join(dir, "keys", "b_key"), filepath.Join(dir, "keys", "c_key")
	write(t, first+".pub", hostKey(1)+" root@host\n")
	write(t, second+".pub", hostKey(2)+"\n")
	write(t, inMatch+".pub", hostKey(3)+"\n")
	write(t, filepath.Join(dir, "sshd_config.d", "10-extra.conf"), "hostkey="+second+"\n# HostKey "+inMatch+"\n")
	config := filepath.Join(dir, "sshd_config")
	write(t, config, "# comment\nHostKey \""+first+"\"\nInclude sshd_config.d/*.conf\nHostKey "+first+"\nMatch User x\n  HostKey "+inMatch+"\n")
	keys, skipped, err := ReadHostKeys(config, dir, true)
	if err != nil || len(skipped) != 0 {
		t.Fatal(err, skipped)
	}
	if got, want := texts(keys), hostKey(1)+"|"+hostKey(2); got != want {
		t.Fatalf("got %s want %s", got, want)
	}
	if keys[0].Path != first+".pub" {
		t.Fatal("wrong source path", keys[0].Path)
	}
}

func TestDefaultsWithoutHostKeyAndMissingConfig(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "ssh_host_ed25519_key.pub"), hostKey(5)+"\n")
	keys, skipped, err := ReadHostKeys(filepath.Join(dir, "sshd_config"), dir, false)
	if err != nil || texts(keys) != hostKey(5) || len(skipped) != 2 {
		t.Fatal(keys, skipped, err)
	}
	if _, _, err := ReadHostKeys(filepath.Join(dir, "sshd_config"), dir, true); err == nil {
		t.Fatal("a missing required config was accepted")
	}
	empty := t.TempDir()
	if _, _, err := ReadHostKeys(filepath.Join(empty, "sshd_config"), empty, false); err == nil || !strings.Contains(err.Error(), "no sshd host public key") {
		t.Fatal("no host key was accepted", err)
	}
}

func TestRejectedEntries(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "bad.pub"), "ssh-dss AAAA\n")
	for name, line := range map[string]string{
		"relative path":    "HostKey keys/a_key",
		"token":            "HostKey /etc/ssh/%h_key",
		"two paths":        "HostKey /a /b",
		"unterminated":     "HostKey \"/a",
		"unsupported type": "HostKey " + filepath.Join(dir, "bad"),
	} {
		config := filepath.Join(dir, "sshd_config")
		write(t, config, line+"\n")
		if _, _, err := ReadHostKeys(config, dir, true); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}
