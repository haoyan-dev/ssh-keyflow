package roster

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"strings"
	"testing"
)

func wire(parts ...[]byte) string {
	var blob []byte
	for _, part := range parts {
		blob = binary.BigEndian.AppendUint32(blob, uint32(len(part)))
		blob = append(blob, part...)
	}
	return base64.StdEncoding.EncodeToString(blob)
}

func TestNamesAccountsAndAddresses(t *testing.T) {
	for value, want := range map[string]bool{"laptop": true, "vps-1": true, "a": true, "Laptop": false, "-x": false, "x-": false,
		"a.b": false, "": false, strings.Repeat("a", 64): false} {
		if ValidName(value) != want {
			t.Errorf("ValidName(%q) != %t", value, want)
		}
	}
	for value, want := range map[string]bool{"user": true, "Admin_1": true, "a.b-c": true, ".x": false, "DOMAIN\\user": false,
		"user@domain": false, "has space": false} {
		if ValidAccount(value) != want {
			t.Errorf("ValidAccount(%q) != %t", value, want)
		}
	}
	for value, want := range map[string]bool{"192.0.2.10": true, "2001:db8::1": true, "laptop.lan": true, "host": true,
		"192.0.2.010": false, "2001:DB8::1": false, "[2001:db8::1]": false, "fe80::1%eth0": false, "::ffff:192.0.2.1": true, "::FFFF:192.0.2.1": false,
		"host:22": false, "Host.lan": false, "host.": false, "1.2.3": false, "": false} {
		if ValidAddress(value) != want {
			t.Errorf("ValidAddress(%q) != %t", value, want)
		}
	}
}

func TestRolesAreOrderedAndStrict(t *testing.T) {
	if roles, err := Roles([]string{"remote", "client"}); err != nil || strings.Join(roles, ",") != "client,remote" {
		t.Fatal(roles, err)
	}
	for _, bad := range [][]string{nil, {"host"}, {"client", "client"}, {""}} {
		if _, err := Roles(bad); err == nil {
			t.Errorf("Roles(%q) accepted", bad)
		}
	}
}

func TestParsePublicKey(t *testing.T) {
	ed := wire([]byte("ssh-ed25519"), bytes.Repeat([]byte{1}, 32))
	key, err := ParsePublicKey("ssh-ed25519 "+ed+" comment@example\n", false)
	if err != nil || key.Text != "ssh-ed25519 "+ed || !strings.HasPrefix(key.Fingerprint, "SHA256:") {
		t.Fatal(key, err)
	}
	ecdsa := wire([]byte("ecdsa-sha2-nistp256"), []byte("nistp256"), bytes.Repeat([]byte{4}, 65))
	if _, err := ParsePublicKey("ecdsa-sha2-nistp256 "+ecdsa, false); err != nil {
		t.Fatal(err)
	}
	modulus := func(bits int) []byte { return append([]byte{0}, bytes.Repeat([]byte{0xff}, bits/8)...) }
	rsa := wire([]byte("ssh-rsa"), []byte{1, 0, 1}, modulus(2048))
	if _, err := ParsePublicKey("ssh-rsa "+rsa, true); err != nil {
		t.Fatal("2048-bit RSA host key rejected", err)
	}
	for name, test := range map[string]struct {
		line string
		host bool
	}{
		"rsa client key":     {"ssh-rsa " + rsa, false},
		"short rsa host key": {"ssh-rsa " + wire([]byte("ssh-rsa"), []byte{1, 0, 1}, modulus(1024)), true},
		"dsa":                {"ssh-dss " + wire([]byte("ssh-dss"), []byte{1}), true},
		"certificate":        {"ssh-ed25519-cert-v01@openssh.com " + ed, true},
		"type mismatch":      {"ecdsa-sha2-nistp256 " + ed, true},
		"curve mismatch":     {"ecdsa-sha2-nistp384 " + wire([]byte("ecdsa-sha2-nistp384"), []byte("nistp256"), []byte{4}), true},
		"short ed25519":      {"ssh-ed25519 " + wire([]byte("ssh-ed25519"), []byte{1}), false},
		"truncated":          {"ssh-ed25519 " + ed[:20], false},
		"two lines":          {"ssh-ed25519 " + ed + "\nssh-ed25519 " + ed, false},
		"type only":          {"ssh-ed25519", false},
	} {
		if _, err := ParsePublicKey(test.line, test.host); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}
