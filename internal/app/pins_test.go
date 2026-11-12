package app_test

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/state"
)

// Host key fixtures reuse valid Ed25519 blobs; the parser only checks format.
var (
	hostKeyA = strings.Join(strings.Fields(publicKey)[:2], " ")
	hostKeyB = strings.Join(strings.Fields(otherPublicKey)[:2], " ")
)

func ed25519Blob(public []byte) string {
	blob := append([]byte{0, 0, 0, 11}, "ssh-ed25519"...)
	blob = append(blob, 0, 0, 0, byte(len(public)))
	return base64.StdEncoding.EncodeToString(append(blob, public...))
}

func pinnedDestination(t *testing.T, system, account string) (string, state.Key, state.Host) {
	t.Helper()
	home := tempHome(t)
	key := imported(t, home, "key", publicKey, true)
	if _, err := run(t, home, app.Request{Operation: "add", Key: key.ID, Destination: destination("server", system, account)}); err != nil {
		t.Fatal(err)
	}
	return home, key, load(t, home).Hosts[0]
}

func pin(t *testing.T, home string, request app.Request, confirm bool) (string, error) {
	t.Helper()
	request.Operation = "pin-host"
	return run(t, home, request, func(options *app.Options) {
		options.Confirm = func(string) (bool, error) { return confirm, nil }
	})
}

func TestParseHostKeysAcceptsOnlyHostKeyLines(t *testing.T) {
	cert := "ssh-ed25519-cert-v01@openssh.com " + strings.Fields(publicKey)[1]
	text := "\ufeffuser@server:~$ cat /etc/ssh/ssh_host_*_key.pub\r\n" + hostKeyA + " root@server\r\n\n" +
		hostKeyA + "\n@cert-authority * " + hostKeyB + "\n@revoked " + hostKeyB + "\n" + cert + "\nssh-rsa " + strings.Fields(publicKey)[1] + "\n" +
		"  " + hostKeyB + "   comment with spaces \x1b[31m\n"
	paste, err := app.ParseHostKeys(text)
	if err != nil {
		t.Fatal(err)
	}
	if len(paste.Keys) != 2 || paste.Keys[0].Blob != strings.Fields(publicKey)[1] || paste.Keys[1].Blob != strings.Fields(otherPublicKey)[1] {
		t.Fatalf("keys %+v", paste.Keys)
	}
	if len(paste.Ignored) != 5 {
		t.Fatalf("ignored %q", paste.Ignored)
	}
	for _, line := range paste.Ignored {
		if strings.ContainsAny(line, "\x1b\r") || len([]rune(line)) > 60 {
			t.Fatalf("unsafe ignored line %q", line)
		}
	}
	for _, bad := range []string{"", "\n\n", "prompt$ \n", cert, strings.Repeat(hostKeyA+"\n", 1) + strings.Repeat("x", 64<<10)} {
		if _, err := app.ParseHostKeys(bad); !errors.Is(err, app.ErrInvalidInput) {
			t.Fatalf("accepted %.40q: %v", bad, err)
		}
	}
}

func TestParseHostKeysLimitsKeyCount(t *testing.T) {
	var lines []string
	for i := range 17 {
		blob := make([]byte, 32)
		blob[0] = byte(i)
		lines = append(lines, "ssh-ed25519 "+ed25519Blob(blob))
	}
	if _, err := app.ParseHostKeys(strings.Join(lines[:16], "\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := app.ParseHostKeys(strings.Join(lines, "\n")); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatal("accepted 17 keys", err)
	}
}

func TestPinHostWritesOnlyParsedFieldsAndRepinIsNoChange(t *testing.T) {
	home, _, host := pinnedDestination(t, "linux", "")
	statePath := filepath.Join(state.Directory(home), "state.json")
	before := read(t, statePath)
	out, err := pin(t, home, app.Request{Host: "server", HostKeys: hostKeyA + " root@server\n"}, true)
	if err != nil {
		t.Fatal(err, out)
	}
	fingerprint := app.HostKeyFingerprint(state.HostKey{Type: "ssh-ed25519", Blob: strings.Fields(publicKey)[1]})
	if !strings.Contains(out, fingerprint) || !strings.Contains(out, "Generate the guide again") {
		t.Fatal(out)
	}
	want := "ssh-keyflow-" + host.ID + " " + hostKeyA + "\n"
	if got := string(read(t, state.PinPath(home))); got != want {
		t.Fatalf("pin file %q", got)
	}
	if !bytes.Equal(before, read(t, statePath)) {
		t.Fatal("pinning changed state.json")
	}
	info, _ := os.Stat(state.PinPath(home))
	out, err = pin(t, home, app.Request{Host: host.ID, HostKeys: hostKeyA}, false)
	if err != nil || !strings.Contains(out, "already pinned; no changes") {
		t.Fatal(out, err)
	}
	if after, _ := os.Stat(state.PinPath(home)); !after.ModTime().Equal(info.ModTime()) {
		t.Fatal("identical re-pin rewrote the file")
	}
}

func TestPinHostRefusesDifferentKeysWithoutReplace(t *testing.T) {
	home, _, host := pinnedDestination(t, "linux", "")
	if _, err := pin(t, home, app.Request{Host: host.ID, HostKeys: hostKeyA}, true); err != nil {
		t.Fatal(err)
	}
	original := read(t, state.PinPath(home))
	out, err := pin(t, home, app.Request{Host: host.ID, HostKeys: hostKeyB}, true)
	if !errors.Is(err, app.ErrInvalidInput) || !strings.Contains(err.Error(), "--replace") || !strings.Contains(out, "WARNING") {
		t.Fatal(out, err)
	}
	if _, err := pin(t, home, app.Request{Host: host.ID, HostKeys: hostKeyB, Replace: true}, false); !errors.Is(err, app.ErrCancelled) {
		t.Fatal("declined replace", err)
	}
	if !bytes.Equal(original, read(t, state.PinPath(home))) {
		t.Fatal("refused replace changed pins")
	}
	out, err = pin(t, home, app.Request{Host: host.ID, HostKeys: hostKeyB, Replace: true}, true)
	if err != nil || !strings.Contains(out, "REPLACE") {
		t.Fatal(out, err)
	}
	if got := string(read(t, state.PinPath(home))); got != "ssh-keyflow-"+host.ID+" "+hostKeyB+"\n" {
		t.Fatal(got)
	}
}

func TestPinHostReadsPasteThenConfirmationFromOneInput(t *testing.T) {
	home, _, host := pinnedDestination(t, "windows", "standard")
	input := strings.NewReader(hostKeyA + "\r\n" + hostKeyB + "\r\n\r\nyes\r\n")
	var out bytes.Buffer
	request := app.Request{Operation: "pin-host", Host: host.ID}
	err := app.Run(t.Context(), app.Options{Home: home, Input: input, Output: &out, Request: &request, Confirm: func(string) (bool, error) { return app.ReadConfirmation(input) }})
	if err != nil {
		t.Fatal(err, out.String())
	}
	if !strings.Contains(out.String(), "Run as Administrator") || !strings.Contains(out.String(), `$env:ProgramData\ssh\ssh_host_*_key.pub`) {
		t.Fatal(out.String())
	}
	if entries, err := state.LoadPins(home); err != nil || len(entries) != 2 {
		t.Fatal(entries, err)
	}
}

// endless never ends a line, so an unbounded reader would never return.
type endless struct{ read int }

func (e *endless) Read(buffer []byte) (int, error) {
	e.read += len(buffer)
	for index := range buffer {
		buffer[index] = 'A'
	}
	return len(buffer), nil
}

func TestPinHostPasteWithoutNewlineStopsAtLimit(t *testing.T) {
	home, _, host := pinnedDestination(t, "linux", "")
	input := &endless{}
	request := app.Request{Operation: "pin-host", Host: host.ID}
	err := app.Run(t.Context(), app.Options{Home: home, Input: input, Output: &bytes.Buffer{}, Request: &request, Confirm: func(string) (bool, error) { return true, nil }})
	if !errors.Is(err, app.ErrInvalidInput) || input.read > 64<<10+1 {
		t.Fatal(err, input.read)
	}
	if _, statErr := os.Stat(state.PinPath(home)); !os.IsNotExist(statErr) {
		t.Fatal("oversize paste created the pin file", statErr)
	}
}

func TestPinSummariesListOnlyManagedDestinations(t *testing.T) {
	home, _, host := pinnedDestination(t, "linux", "")
	if _, err := pin(t, home, app.Request{Host: host.ID, HostKeys: hostKeyA}, true); err != nil {
		t.Fatal(err)
	}
	if summaries, err := app.PinSummaries(home, load(t, home).Hosts); err != nil || len(summaries[host.ID]) != 1 {
		t.Fatal(summaries, err)
	}
	if _, err := run(t, home, app.Request{Operation: "remove-host", Host: host.ID}); err != nil {
		t.Fatal(err)
	}
	if summaries, err := app.PinSummaries(home, load(t, home).Hosts); err != nil || len(summaries) != 0 {
		t.Fatal("removed destination listed", summaries, err)
	}
}

func TestPinAndUnpinRefuseMalformedFileAndUnknownHosts(t *testing.T) {
	home, _, host := pinnedDestination(t, "linux", "")
	write(t, state.PinPath(home), "garbage\n")
	if _, err := pin(t, home, app.Request{Host: host.ID, HostKeys: hostKeyA}, true); err == nil || !strings.Contains(err.Error(), "review the file manually") {
		t.Fatal("pinned over malformed file", err)
	}
	if _, err := run(t, home, app.Request{Operation: "unpin-host", Host: host.ID}); err == nil {
		t.Fatal("unpinned over malformed file")
	}
	if string(read(t, state.PinPath(home))) != "garbage\n" {
		t.Fatal("malformed file rewritten")
	}
	if _, err := pin(t, home, app.Request{Host: "missing", HostKeys: hostKeyA}, true); err == nil {
		t.Fatal("pinned unknown destination")
	}
	if _, err := run(t, home, app.Request{Operation: "pin-host"}); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatal("pin without host", err)
	}
}

func TestUnpinHostRemovesOnlyThatDestination(t *testing.T) {
	home, key, host := pinnedDestination(t, "linux", "")
	if _, err := run(t, home, app.Request{Operation: "add", Key: key.ID, Destination: destination("other", "linux", "")}); err != nil {
		t.Fatal(err)
	}
	other := load(t, home).Hosts[1]
	for _, id := range []string{host.ID, other.ID} {
		if _, err := pin(t, home, app.Request{Host: id, HostKeys: hostKeyA}, true); err != nil {
			t.Fatal(err)
		}
	}
	out, err := run(t, home, app.Request{Operation: "unpin-host", Host: "server"}, func(options *app.Options) {
		options.Confirm = func(string) (bool, error) { return false, nil }
	})
	if !errors.Is(err, app.ErrCancelled) || !strings.Contains(out, "remove them") {
		t.Fatal(out, err)
	}
	if _, err := run(t, home, app.Request{Operation: "unpin-host", Host: "server"}); err != nil {
		t.Fatal(err)
	}
	if got := string(read(t, state.PinPath(home))); got != "ssh-keyflow-"+other.ID+" "+hostKeyA+"\n" {
		t.Fatal(got)
	}
	out, err = run(t, home, app.Request{Operation: "unpin-host", Host: "server"})
	if err != nil || !strings.Contains(out, "no changes") {
		t.Fatal(out, err)
	}
}

func TestRemovingPinnedDestinationKeepsPinsAndPointsToUnpin(t *testing.T) {
	home, _, host := pinnedDestination(t, "linux", "")
	out, err := run(t, home, app.Request{Operation: "remove-host", Host: host.ID})
	if err != nil || strings.Contains(out, "hosts unpin") {
		t.Fatal("unpinned removal mentioned unpin", out, err)
	}
	home, _, host = pinnedDestination(t, "linux", "")
	if _, err := pin(t, home, app.Request{Host: host.ID, HostKeys: hostKeyA}, true); err != nil {
		t.Fatal(err)
	}
	before := read(t, state.PinPath(home))
	out, err = run(t, home, app.Request{Operation: "remove-host", Host: host.ID})
	if err != nil || !strings.Contains(out, "hosts unpin --host "+host.ID+" first") {
		t.Fatal(out, err)
	}
	if after := read(t, state.PinPath(home)); string(after) != string(before) {
		t.Fatal("removal changed the pin file", string(after))
	}
}

func configStep(t *testing.T, guide app.SetupGuide) app.GuideStep {
	t.Helper()
	for _, step := range guide.Steps {
		if strings.Contains(step.Script, "Host server\n") {
			return step
		}
	}
	t.Fatal("no config step", guide.Text)
	return app.GuideStep{}
}

func TestGuideAddsPinStepsUntilPinnedThenPinnedLines(t *testing.T) {
	home, key, host := pinnedDestination(t, "linux", "")
	guide, err := app.PrepareGuide(home, "server", key.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(guide.Steps[1].Title, "print host keys") || guide.Steps[1].Script != "cat /etc/ssh/ssh_host_*_key.pub\n" ||
		guide.Steps[2].Script != "ssh-keyflow hosts pin --host "+host.ID+"\n" {
		t.Fatal("pin steps missing", guide.Text)
	}
	unpinned := configStep(t, guide)
	if strings.Contains(unpinned.Script, "HostKeyAlias") || !strings.Contains(unpinned.Instructions, "Host keys: not pinned") {
		t.Fatal(unpinned)
	}
	if _, err := pin(t, home, app.Request{Host: host.ID, HostKeys: hostKeyA}, true); err != nil {
		t.Fatal(err)
	}
	guide, err = app.PrepareGuide(home, "server", key.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(withoutPinSteps(guide.Steps)) != len(guide.Steps) {
		t.Fatal("pinned guide kept pin steps", guide.Text)
	}
	pinned := configStep(t, guide)
	lines := "    HostKeyAlias ssh-keyflow-" + host.ID + "\n    UserKnownHostsFile \"" + filepath.ToSlash(state.PinPath(home)) + "\"\n    StrictHostKeyChecking yes\n"
	// Only the lines are added: everything else in the snippet is unchanged.
	if !strings.Contains(pinned.Script, "IdentitiesOnly yes\n"+lines) || strings.Replace(pinned.Script, lines, "", 1) != unpinned.Script || !strings.Contains(pinned.Instructions, "Host keys: pinned (1)") {
		t.Fatalf("pinned snippet %q", pinned.Script)
	}
}

func TestGuideOmitsPinningForAnotherLocalTerminalAndUnreadablePins(t *testing.T) {
	home, key, host := pinnedDestination(t, "windows", "administrator")
	foreign := "windows"
	if app.LocalPlatform() == foreign {
		foreign = "linux"
	}
	guide, err := app.PrepareGuideWithOptions(home, "server", key.ID, app.GuideOptions{LocalOS: foreign, IdentityFile: "~/key"})
	if err != nil || len(withoutPinSteps(guide.Steps)) != len(guide.Steps) || strings.Contains(guide.Text, "Host keys:") {
		t.Fatal("pinning offered for another computer", err, guide.Text)
	}
	guide, err = app.PrepareGuide(home, "server", key.ID)
	if err != nil || !strings.Contains(guide.Steps[1].Title, "Run as Administrator") || !strings.Contains(guide.Steps[1].Script, `$env:ProgramData\ssh\`) {
		t.Fatal("Windows host key step", err, guide.Text)
	}
	write(t, state.PinPath(home), "ssh-keyflow-"+host.ID+" broken\n")
	guide, err = app.PrepareGuide(home, "server", key.ID)
	if err != nil || len(withoutPinSteps(guide.Steps)) != len(guide.Steps) || strings.Contains(guide.Text, "HostKeyAlias") || !strings.Contains(configStep(t, guide).Instructions, "Host keys: unknown") {
		t.Fatal("malformed pins were guessed at", err, guide.Text)
	}
}

func TestNativeOpenSSHParsesPinnedSnippet(t *testing.T) {
	ssh, err := exec.LookPath("ssh")
	if err != nil {
		t.Skip("native OpenSSH unavailable")
	}
	home, key, host := pinnedDestination(t, "linux", "")
	if _, err := pin(t, home, app.Request{Host: host.ID, HostKeys: hostKeyA}, true); err != nil {
		t.Fatal(err)
	}
	guide, err := app.PrepareGuide(home, "server", key.ID)
	if err != nil {
		t.Fatal(err)
	}
	script := configStep(t, guide).Script
	start := strings.Index(script, "Host server\n")
	end := strings.Index(script, "    StrictHostKeyChecking yes\n") + len("    StrictHostKeyChecking yes\n")
	config := filepath.Join(t.TempDir(), "config")
	write(t, config, script[start:end])
	output, err := exec.Command(ssh, "-G", "-F", config, "server").CombinedOutput()
	text := strings.ToLower(string(output))
	if err != nil || !strings.Contains(text, "hostkeyalias ssh-keyflow-"+host.ID) || !strings.Contains(text, "stricthostkeychecking true") || !strings.Contains(text, "userknownhostsfile") {
		t.Fatalf("native parser rejected pinned config: %v %s", err, output)
	}
}

// repeated returns one byte forever.
type repeated struct {
	value byte
	read  int
}

func (r *repeated) Read(buffer []byte) (int, error) {
	r.read += len(buffer)
	for index := range buffer {
		buffer[index] = r.value
	}
	return len(buffer), nil
}

func pinFromInput(t *testing.T, home, hostID string, input io.Reader) error {
	t.Helper()
	request := app.Request{Operation: "pin-host", Host: hostID}
	return app.Run(t.Context(), app.Options{Home: home, Input: input, Output: &bytes.Buffer{}, Request: &request, Confirm: func(string) (bool, error) { return true, nil }})
}

func TestPinHostPasteLimitCountsAllInput(t *testing.T) {
	home, _, host := pinnedDestination(t, "linux", "")
	blank := &repeated{value: '\n'}
	if err := pinFromInput(t, home, host.ID, blank); !errors.Is(err, app.ErrInvalidInput) || blank.read > 64<<10+1 {
		t.Fatal("endless blank lines", err, blank.read)
	}
	exact := hostKeyA + "\n" + strings.Repeat(" ", 64<<10-len(hostKeyA)-1)
	if err := pinFromInput(t, home, host.ID, strings.NewReader(exact)); err != nil {
		t.Fatal("refused exactly 64 KiB", err)
	}
	if err := pinFromInput(t, home, host.ID, strings.NewReader(exact+" ")); !errors.Is(err, app.ErrInvalidInput) {
		t.Fatal("accepted 64 KiB + 1", err)
	}
}

func TestParseHostKeysDropsBOMAfterLeadingWhitespace(t *testing.T) {
	paste, err := app.ParseHostKeys(" \ufeff" + hostKeyA)
	if err != nil || len(paste.Keys) != 1 || len(paste.Ignored) != 0 {
		t.Fatal(paste, err)
	}
}

func TestGuideOmitsPinStepsWhenPinFilePathIsNotLiteral(t *testing.T) {
	home := filepath.Join(tempHome(t), "a${b}")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	key := imported(t, home, "key", publicKey, true)
	if _, err := run(t, home, app.Request{Operation: "add", Key: key.ID, Destination: destination("server", "linux", "")}); err != nil {
		t.Fatal(err)
	}
	guide, err := app.PrepareGuideWithOptions(home, "server", key.ID, app.GuideOptions{LocalOS: app.LocalPlatform(), IdentityFile: "~/key"})
	if err != nil || len(withoutPinSteps(guide.Steps)) != len(guide.Steps) || !strings.Contains(guide.Text, "cannot be written literally") {
		t.Fatal("pin steps offered for an unusable pin file path", err, guide.Text)
	}
}
