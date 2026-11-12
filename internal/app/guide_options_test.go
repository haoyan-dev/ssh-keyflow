package app_test

import (
	"bytes"
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/state"
)

func TestGuidePlatformMatrixPreservesSavedRecords(t *testing.T) {
	home := tempHome(t)
	key := imported(t, home, "pair", publicKey, true)
	if _, err := run(t, home, app.Request{Operation: "add", Key: key.ID, Destination: destination("server", "windows", "administrator")}); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(state.Directory(home), "state.json")
	before := read(t, statePath)
	for _, local := range []string{"linux", "macos", "windows"} {
		for _, remote := range []string{"linux", "macos", "windows"} {
			t.Run(local+"/"+remote, func(t *testing.T) {
				path := "/home/alice/key%20's with spaces"
				if local == "windows" {
					path = `C:\Users\alice\key%20's with spaces`
				}
				options := app.GuideOptions{LocalOS: local, RemoteOS: remote, IdentityFile: path}
				if remote == "windows" {
					options.WindowsAccount = "standard"
				}
				guide, err := app.PrepareGuideWithOptions(home, "server", key.ID, options)
				guide.Steps = withoutPinSteps(guide.Steps)
				// The supplied terminal path is not inspected, so the ssh-agent step is included.
				if err != nil || len(guide.Steps) != 4 {
					t.Fatal(err, guide)
				}
				for _, want := range []string{"Local platform: " + local, "Destination platform: " + remote, "key%%20's with spaces", "not checked"} {
					if !strings.Contains(guide.Text, want) {
						t.Fatalf("missing %s: %s", want, guide.Text)
					}
				}
				if !strings.Contains(guide.Steps[3].Title, local) || guide.Steps[3].Script != "ssh server\n" ||
					!strings.Contains(guide.Steps[2].Title, "ssh-agent") || strings.HasPrefix(guide.Steps[2].Script, "& {") != (local == "windows") {
					t.Fatal("connection did not use selected local platform")
				}
				if strings.HasPrefix(guide.Steps[1].Script, "& {") != (local == "windows") ||
					strings.HasPrefix(guide.Steps[0].Script, "& {") != (remote == "windows") {
					t.Fatal("script language does not match selected platforms")
				}
				if remote != "windows" && strings.Contains(guide.Steps[0].Script, "administrators_authorized_keys") {
					t.Fatal("Windows account leaked into a POSIX destination")
				}
				if !strings.Contains(guide.Steps[1].Instructions, "has not checked that path") {
					t.Fatal("foreign private-key path implied verification")
				}
				out, err := run(t, home, app.Request{Operation: "guide", Host: "server", Key: key.ID, GuideOptions: options})
				if err != nil || out != guide.Text+"\n" {
					t.Fatal("app.Run dropped guide options", err)
				}
			})
		}
	}
	if !bytes.Equal(before, read(t, statePath)) {
		t.Fatal("guide options changed saved records")
	}
	guide, err := app.PrepareGuide(home, "server", key.ID)
	guide.Steps = withoutPinSteps(guide.Steps)
	if err != nil || !strings.Contains(guide.Text, "Local platform: "+app.LocalPlatform()) ||
		!strings.Contains(guide.Steps[0].Script, "administrators_authorized_keys") ||
		!strings.Contains(guide.Steps[1].Script, filepath.ToSlash(key.PrivatePath)) {
		t.Fatal("default platform, account or identity changed", err, guide)
	}
	expected := runtime.GOOS
	if expected == "darwin" {
		expected = "macos"
	}
	if app.LocalPlatform() != expected {
		t.Fatal("local platform detection changed")
	}
}

func TestInvalidGuideOptionsNeverProduceScripts(t *testing.T) {
	home := tempHome(t)
	key := imported(t, home, "pair", publicKey, true)
	if _, err := run(t, home, app.Request{Operation: "add", Key: key.ID, Destination: destination("server", "linux", "")}); err != nil {
		t.Fatal(err)
	}
	foreign := "windows"
	if app.LocalPlatform() == foreign {
		foreign = "linux"
	}
	for _, options := range []app.GuideOptions{
		{LocalOS: "win"}, {RemoteOS: "darwin"}, {WindowsAccount: "root"},
		{WindowsAccount: "standard"}, {RemoteOS: "linux", WindowsAccount: "standard"},
		{RemoteOS: "windows"}, {LocalOS: foreign},
		{LocalOS: "linux", IdentityFile: `C:\Users\alice\key`},
		{LocalOS: "windows", IdentityFile: "/home/alice/key"},
		{IdentityFile: "relative/key"}, {IdentityFile: "~/key\nHost injected"},
	} {
		guide, err := app.PrepareGuideWithOptions(home, "server", key.ID, options)
		if !errors.Is(err, app.ErrInvalidInput) || len(guide.Steps) != 0 || guide.Text != "" {
			t.Fatalf("invalid options produced guidance: %+v %v %+v", options, err, guide)
		}
	}
	for _, local := range []string{"linux", "macos", "windows"} {
		guide, err := app.PrepareGuideWithOptions(home, "server", key.ID, app.GuideOptions{LocalOS: local, IdentityFile: "~/key"})
		if err != nil || !strings.Contains(withoutPinSteps(guide.Steps)[1].Script, `IdentityFile "~/key"`) {
			t.Fatal("home-relative path rejected", local, err)
		}
	}
	guide, err := app.PrepareGuideWithOptions(home, "server", key.ID, app.GuideOptions{LocalOS: "linux", IdentityFile: "/home/${USER}/key"})
	if err != nil || len(guide.Steps) != 1 || !strings.Contains(guide.Text, "cannot be represented literally") {
		t.Fatal("environment expansion guard lost", err, guide)
	}
}

func TestPublicOnlyOverrideDoesNotInventPrivateIdentity(t *testing.T) {
	home := tempHome(t)
	key := imported(t, home, "public", publicKey, false)
	if _, err := run(t, home, app.Request{Operation: "add", Key: key.ID, Destination: destination("server", "linux", "")}); err != nil {
		t.Fatal(err)
	}
	guide, err := app.PrepareGuideWithOptions(home, "server", key.ID, app.GuideOptions{LocalOS: "macos", RemoteOS: "windows", WindowsAccount: "administrator", IdentityFile: "~/key.pub"})
	steps := withoutPinSteps(guide.Steps)
	if err != nil || len(steps) != 3 || !strings.Contains(steps[1].Script, `IdentityFile "~/key.pub"`) || strings.Contains(guide.Text, "AddKeysToAgent") || strings.Contains(guide.Text, "UseKeychain") || !strings.Contains(steps[1].Instructions, "must be loaded in the agent") {
		t.Fatal("public-only key became a key pair", err, guide)
	}
	foreign := "windows"
	if app.LocalPlatform() == foreign {
		foreign = "linux"
	}
	// The saved public key path belongs to this computer, not another terminal.
	guide, err = app.PrepareGuideWithOptions(home, "server", key.ID, app.GuideOptions{LocalOS: foreign})
	if err != nil || len(guide.Steps) != 1 || strings.Contains(guide.Text, "IdentityFile \"") || !strings.Contains(guide.Steps[0].Instructions, "Supply identity-file") {
		t.Fatal("public-only key used this computer's path for another terminal", err, guide)
	}
}
