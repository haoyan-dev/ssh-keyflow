package roster

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func key(seed byte) string {
	return "ssh-ed25519 " + wire([]byte("ssh-ed25519"), bytes.Repeat([]byte{seed}, 32))
}

func sample() Roster {
	return Roster{Format: Format, Schema: Schema, FleetID: "fleet-" + strings.Repeat("cd", 16), Version: 3, CreatedAt: "2026-11-28T00:00:00Z", LoginAccount: "admin",
		Nodes: []Node{
			{ID: "node-" + strings.Repeat("aa", 16), Name: "server", Roles: []string{RoleClient, RoleRemote}, LoginAccount: "deploy",
				Addresses: []string{"192.0.2.10", "server.lan"}, Port: 2222, ClientKeys: []string{key(1)}, HostKeys: []string{key(2)}},
			{ID: "node-" + strings.Repeat("bb", 16), Name: "laptop", Roles: []string{RoleClient}, ClientKeys: []string{key(3)}},
		}}
}

func TestMarshalParseRoundTrip(t *testing.T) {
	data, err := sample().Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasSuffix(data, []byte("}\n")) || bytes.Contains(data, []byte("\r")) || !bytes.HasPrefix(data, []byte("{\n  \"format\": ")) {
		t.Fatalf("unexpected canonical bytes:\n%s", data)
	}
	parsed, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := parsed.Marshal()
	if !bytes.Equal(again, data) {
		t.Fatal("round trip changed the bytes")
	}
}

func TestParseNeverNormalizes(t *testing.T) {
	data, err := sample().Marshal()
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	var compact bytes.Buffer
	if err := json.Compact(&compact, data); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"crlf":            strings.ReplaceAll(text, "\n", "\r\n"),
		"compact":         compact.String() + "\n",
		"no final lf":     strings.TrimSuffix(text, "\n"),
		"tab indent":      strings.Replace(text, "  ", "\t", 1),
		"duplicate field": strings.Replace(text, `"schema": 1,`, `"schema": 1, "schema": 1,`, 1),
		"uppercase field": strings.Replace(text, `"version"`, `"VERSION"`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]byte(body)); !errors.Is(err, ErrNotCanonical) {
				t.Fatal("expected ErrNotCanonical, got", err)
			}
		})
	}
	for name, body := range map[string]string{
		"unknown field":  strings.Replace(text, `"schema": 1,`, `"schema": 1,`+"\n  \"extra\": 1,", 1),
		"trailing data":  text + "{}\n",
		"string schema":  strings.Replace(text, `"schema": 1`, `"schema": "1"`, 1),
		"other format":   strings.Replace(text, `"ssh-keyflow-roster"`, `"other"`, 1),
		"not an object":  "[]\n",
		"oversized":      text + strings.Repeat(" ", MaxBytes),
		"invalid roster": strings.Replace(text, `"version": 3`, `"version": 0`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]byte(body)); err == nil {
				t.Fatal("invalid roster accepted")
			}
		})
	}
	if _, err := Parse([]byte(strings.Replace(text, `"schema": 1`, `"schema": 2`, 1))); !errors.Is(err, ErrNewerSchema) {
		t.Fatal("schema 2 should ask for an upgrade, got", err)
	}
}

func TestValidateRules(t *testing.T) {
	for name, change := range map[string]func(*Roster){
		"no nodes":              func(r *Roster) { r.Nodes = nil },
		"version zero":          func(r *Roster) { r.Version = 0 },
		"version too large":     func(r *Roster) { r.Version = MaxVersion + 1 },
		"fractional created_at": func(r *Roster) { r.CreatedAt = "2026-11-28T00:00:00.5Z" },
		"offset created_at":     func(r *Roster) { r.CreatedAt = "2026-11-28T09:00:00+09:00" },
		"bad fleet id":          func(r *Roster) { r.FleetID = "fleet-1" },
		"bad common account":    func(r *Roster) { r.LoginAccount = "a b" },
		"unsorted nodes":        func(r *Roster) { r.Nodes[0], r.Nodes[1] = r.Nodes[1], r.Nodes[0] },
		"duplicate name":        func(r *Roster) { r.Nodes[1].Name = "server" },
		"key on two nodes":      func(r *Roster) { r.Nodes[1].ClientKeys = []string{key(1)} },
		"client and host key":   func(r *Roster) { r.Nodes[0].HostKeys = []string{key(1)} },
		"port 22":               func(r *Roster) { r.Nodes[0].Port = 22 },
		"port too large":        func(r *Roster) { r.Nodes[0].Port = 65536 },
		"login equals common":   func(r *Roster) { r.Nodes[0].LoginAccount = "admin" },
		"roles out of order":    func(r *Roster) { r.Nodes[0].Roles = []string{RoleRemote, RoleClient} },
		"client keys on remote": func(r *Roster) { r.Nodes[0].Roles = []string{RoleRemote} },
		"remote fields on client": func(r *Roster) {
			r.Nodes[1].Addresses = []string{"laptop.lan"}
		},
		"client without keys":  func(r *Roster) { r.Nodes[1].ClientKeys = nil },
		"remote without hosts": func(r *Roster) { r.Nodes[0].HostKeys = nil },
		"unsorted addresses":   func(r *Roster) { r.Nodes[0].Addresses = []string{"server.lan", "192.0.2.10"} },
		"key with comment":     func(r *Roster) { r.Nodes[1].ClientKeys = []string{key(3) + " c"} },
	} {
		t.Run(name, func(t *testing.T) {
			broken := sample()
			change(&broken)
			if _, err := broken.Marshal(); err == nil {
				t.Fatal("invalid roster marshalled")
			}
		})
	}
}

func TestValidSignature(t *testing.T) {
	body := base64.StdEncoding.EncodeToString([]byte("SSHSIG\x00\x00\x00\x01"))
	good := "-----BEGIN SSH SIGNATURE-----\n" + body + "\n-----END SSH SIGNATURE-----\n"
	if !ValidSignature(good) {
		t.Fatal("valid signature rejected")
	}
	for name, text := range map[string]string{
		"crlf":          strings.ReplaceAll(good, "\n", "\r\n"),
		"no final lf":   strings.TrimSuffix(good, "\n"),
		"empty body":    "-----BEGIN SSH SIGNATURE-----\n-----END SSH SIGNATURE-----\n",
		"wrong magic":   strings.Replace(good, body, base64.StdEncoding.EncodeToString([]byte("NOTSIG")), 1),
		"not base64":    strings.Replace(good, body, "!!!!", 1),
		"other armor":   strings.ReplaceAll(good, "SSH SIGNATURE", "PGP SIGNATURE"),
		"trailing text": good + "x\n",
	} {
		if ValidSignature(text) {
			t.Error(name, "accepted")
		}
	}
}

func TestAgeWarning(t *testing.T) {
	roster := sample()
	created, _ := time.Parse(TimeLayout, roster.CreatedAt)
	maxAge := 30 * 24 * time.Hour
	if warning := AgeWarning(roster, maxAge, created.Add(maxAge)); warning != "" {
		t.Fatal("warned at exactly --max-age:", warning)
	}
	if warning := AgeWarning(roster, maxAge, created.Add(31*24*time.Hour)); !strings.Contains(warning, "31 days ago") || !strings.Contains(warning, "--max-age 30 days") || !strings.Contains(warning, "fleet sign") {
		t.Fatal("unexpected stale warning:", warning)
	}
	if warning := AgeWarning(roster, maxAge, created.Add(-FutureSkew)); warning != "" {
		t.Fatal("warned within the clock skew:", warning)
	}
	if warning := AgeWarning(roster, maxAge, created.Add(-FutureSkew-time.Second)); !strings.Contains(warning, "future") {
		t.Fatal("unexpected future warning:", warning)
	}
}
