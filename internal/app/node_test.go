package app_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/state"
)

// hostKeyFixture is a valid ssh-ed25519 public key distinct from publicKey.
const hostKeyFixture = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMD"

func sshdFixture(t *testing.T, home string) (config, dir string) {
	t.Helper()
	dir = filepath.Join(home, "etc-ssh")
	key := filepath.Join(dir, "ssh_host_ed25519_key")
	write(t, key+".pub", hostKeyFixture+" root@host\n")
	config = filepath.Join(dir, "sshd_config")
	write(t, config, "HostKey "+key+"\n")
	return config, dir
}

func setNode(t *testing.T, home string, request app.NodeRequest, approve bool) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := app.SetNode(app.NodeOptions{Home: home, Output: &out, Request: request,
		Confirm: func(string) (bool, error) { return approve, nil }})
	return out.String(), err
}

func TestNodeSetDryRunNeverInitializesStorage(t *testing.T) {
	home := tempHome(t)
	config, dir := sshdFixture(t, home)
	out, err := setNode(t, home, app.NodeRequest{Name: "vps", Roles: []string{"remote"}, Addresses: []string{"vps.example"},
		Port: 22, SSHDConfig: config, SSHDir: dir, DryRun: true}, false)
	if err != nil || !strings.Contains(out, "Dry run: no changes made.") || !strings.Contains(out, "Port: 22") || !strings.Contains(out, "Host key: ssh-ed25519 SHA256:") {
		t.Fatal(out, err)
	}
	if _, err := os.Stat(state.Directory(home)); !os.IsNotExist(err) {
		t.Fatal("dry run created storage", err)
	}
}

func TestNodeSetSavesAfterYesAndKeepsIDOnReplace(t *testing.T) {
	home := tempHome(t)
	key := imported(t, home, "laptop-key", publicKey)
	config, dir := sshdFixture(t, home)
	request := app.NodeRequest{Name: "laptop", Roles: []string{"remote", "client"}, Key: key.Name, Addresses: []string{"laptop.lan", "192.0.2.10"},
		Port: 2222, SSHDConfig: config, SSHDir: dir}
	before := read(t, filepath.Join(state.Directory(home), "state.json"))
	if _, err := setNode(t, home, request, false); !errors.Is(err, app.ErrCancelled) {
		t.Fatal("refusal did not cancel", err)
	}
	if !bytes.Equal(before, read(t, filepath.Join(state.Directory(home), "state.json"))) {
		t.Fatal("cancelled node set changed state")
	}
	if out, err := setNode(t, home, request, true); err != nil {
		t.Fatal(out, err)
	}
	node := load(t, home).Node
	if node == nil || strings.Join(node.Roles, ",") != "client,remote" || strings.Join(node.Addresses, ",") != "192.0.2.10,laptop.lan" ||
		node.ClientKeyIDs[0] != key.ID || node.HostKeys[0] != hostKeyFixture || node.Port != 2222 {
		t.Fatalf("unexpected node %+v", node)
	}
	view, _, err := app.ShowNode(home)
	if err != nil || view.Node.ID != node.ID || view.Node.ClientKeys[0] != strings.Join(strings.Fields(publicKey)[:2], " ") || view.IsAdmin {
		t.Fatal("unexpected view", view, err)
	}
	out, err := setNode(t, home, app.NodeRequest{Name: "laptop2", Roles: []string{"client"}, Key: key.ID}, true)
	if err != nil || !strings.Contains(out, "Replace this machine's node description "+node.ID) {
		t.Fatal(out, err)
	}
	if replaced := load(t, home).Node; replaced.ID != node.ID || replaced.Addresses != nil || replaced.Name != "laptop2" {
		t.Fatalf("replacement lost the id or kept remote fields %+v", replaced)
	}
}

func TestNodeSetRejectsBadInputAsUsage(t *testing.T) {
	home := tempHome(t)
	for name, request := range map[string]app.NodeRequest{
		"bad name":            {Name: "Laptop", Roles: []string{"client"}, Key: "k"},
		"unknown role":        {Name: "a", Roles: []string{"host"}},
		"client without key":  {Name: "a", Roles: []string{"client"}},
		"key without client":  {Name: "a", Roles: []string{"remote"}, Key: "k", Addresses: []string{"a.lan"}},
		"remote no addresses": {Name: "a", Roles: []string{"remote"}},
		"address with port":   {Name: "a", Roles: []string{"remote"}, Addresses: []string{"a.lan:22"}},
		"duplicate address":   {Name: "a", Roles: []string{"remote"}, Addresses: []string{"a.lan", "a.lan"}},
		"client with port":    {Name: "a", Roles: []string{"client"}, Key: "k", Port: 2222},
		"port too large":      {Name: "a", Roles: []string{"remote"}, Addresses: []string{"a.lan"}, Port: 70000},
		"domain account":      {Name: "a", Roles: []string{"remote"}, Addresses: []string{"a.lan"}, LoginAccount: `DOMAIN\user`},
		"missing key":         {Name: "a", Roles: []string{"client"}, Key: "absent"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := setNode(t, home, request, true); !errors.Is(err, app.ErrInvalidInput) {
				t.Fatal("not a usage error:", err)
			}
		})
	}
	if _, err := os.Stat(state.Directory(home)); !os.IsNotExist(err) {
		t.Fatal("rejected input created storage", err)
	}
}

func TestRemoveRefusesTheNodeClientKey(t *testing.T) {
	home := tempHome(t)
	key := imported(t, home, "laptop-key", publicKey)
	if _, err := setNode(t, home, app.NodeRequest{Name: "laptop", Roles: []string{"client"}, Key: key.ID}, true); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, home, app.Request{Operation: "remove", Key: key.ID}); err == nil || !strings.Contains(err.Error(), "node set") {
		t.Fatal("removing the node's client key was allowed", err)
	}
	if len(load(t, home).Keys) != 1 {
		t.Fatal("key removed")
	}
}

func TestShowNodeBeforeSetIsReadOnly(t *testing.T) {
	home := tempHome(t)
	view, review, err := app.ShowNode(home)
	if err != nil || view.Node != nil || view.IsAdmin || review != "" {
		t.Fatal(view, review, err)
	}
	if _, err := os.Stat(state.Directory(home)); !os.IsNotExist(err) {
		t.Fatal("show created storage", err)
	}
}
