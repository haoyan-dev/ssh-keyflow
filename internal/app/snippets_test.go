package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"ssh-keyflow/internal/state"
)

const snippetPublic = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB"

func posixShell(t *testing.T) string {
	t.Helper()
	if path, err := exec.LookPath("sh"); err == nil {
		return path
	}
	if runtime.GOOS == "windows" {
		path := `C:\Program Files\Git\bin\sh.exe`
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	t.Skip("POSIX shell unavailable")
	return ""
}

func runSnippet(t *testing.T, shell, script string) ([]byte, error) {
	t.Helper()
	if strings.Contains(strings.ToLower(filepath.Base(shell)), "powershell") || strings.Contains(strings.ToLower(filepath.Base(shell)), "pwsh") {
		// -Command exercises terminal paste behavior, without changing execution policy.
		command := exec.Command(shell, "-NoProfile", "-NonInteractive", "-Command", script)
		// Let each PowerShell edition discover its own standard modules.
		for _, variable := range os.Environ() {
			if !strings.HasPrefix(strings.ToLower(variable), "psmodulepath=") {
				command.Env = append(command.Env, variable)
			}
		}
		return command.CombinedOutput()
	}
	path := filepath.Join(t.TempDir(), "snippet.sh")
	if err := os.WriteFile(path, []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	return exec.Command(shell, path).CombinedOutput()
}

func readSnippetFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestPOSIXPlacementPreservesKeysAndRepeatPaste(t *testing.T) {
	shell := posixShell(t)
	current, err := exec.Command(shell, "-c", "id -un").Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, system := range []string{"linux", "macos"} {
		t.Run(system, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, ".ssh")
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "authorized_keys")
			original := "# existing without trailing newline\nssh-ed25519 other-key keep-me"
			if err := os.WriteFile(path, []byte(original), 0600); err != nil {
				t.Fatal(err)
			}
			step := posixPlacement(state.Host{OS: system, User: strings.TrimSpace(string(current))}, snippetPublic)
			script := strings.ReplaceAll(step.Script, "$HOME", filepath.ToSlash(root))
			for i := 0; i < 2; i++ {
				if output, err := runSnippet(t, shell, script); err != nil {
					t.Fatalf("paste: %v %s", err, output)
				}
			}
			got := readSnippetFile(t, path)
			if !strings.HasPrefix(got, original+"\n") || strings.Count(got, snippetPublic) != 1 {
				t.Fatal("keys changed or duplicate appended", got)
			}
			if runtime.GOOS != "windows" {
				for path, want := range map[string]os.FileMode{dir: 0700, path: 0600} {
					info, err := os.Stat(path)
					if err != nil || info.Mode().Perm() != want {
						t.Fatal("incorrect permissions", path, info, err)
					}
				}
			}
			wrong := strings.ReplaceAll(posixPlacement(state.Host{User: "wrong-keyflow-user"}, snippetPublic).Script, "$HOME", filepath.ToSlash(root))
			if _, err := runSnippet(t, shell, wrong); err == nil || readSnippetFile(t, path) != got {
				t.Fatal("wrong destination account modified keys")
			}
		})
	}
}

func TestClientScriptsPreserveConfigAndLiteralPaths(t *testing.T) {
	host := state.Host{Alias: "server.test", Hostname: "server.example"}
	// These characters must stay data in both shells, not commands or expansion.
	config := "Host server.test\n    HostName server.example\n    IdentityFile \"/tmp/key%%20's $value $(echo injected) `literal` ü\"\n    IdentitiesOnly yes\n"
	for _, target := range []struct{ platform, shell string }{{"linux", "sh"}, {"darwin", "sh"}, {"windows", "pwsh"}, {"windows", "powershell"}} {
		t.Run(target.platform+"/"+target.shell, func(t *testing.T) {
			platform := target.platform
			shell := ""
			if platform == "windows" {
				var err error
				shell, err = exec.LookPath(target.shell)
				if err != nil {
					t.Skip("PowerShell unavailable")
				}
			} else {
				shell = posixShell(t)
			}
			root := t.TempDir()
			dir := filepath.Join(root, ".ssh")
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "config")
			original := "# preserved\nHost other\n    HostName other.example"
			if err := os.WriteFile(path, []byte(original), 0600); err != nil {
				t.Fatal(err)
			}
			step := clientSetup(host, config, platform)
			replacement := filepath.ToSlash(root)
			if platform == "windows" {
				replacement = "'" + strings.ReplaceAll(root, "'", "''") + "'"
			}
			script := strings.ReplaceAll(step.Script, "$HOME", replacement)
			if output, err := runSnippet(t, shell, script); err != nil {
				t.Fatalf("paste: %v %s", err, output)
			}
			got := readSnippetFile(t, path)
			if !strings.HasPrefix(got, original) || !strings.Contains(got, strings.TrimSuffix(config, "\n")) {
				t.Fatal("config changed or literal path expanded", got)
			}
			if _, err := runSnippet(t, shell, script); err == nil || readSnippetFile(t, path) != got {
				t.Fatal("repeat paste duplicated alias or altered config")
			}
			for _, existing := range []string{
				"  hOsT other SERVER.TEST\n    HostName existing.example\n",
				"Host other \"server.test\"\n",
				"Host=server.test\n",
				"Host = server.test\n",
			} {
				if err := os.WriteFile(path, []byte(existing), 0600); err != nil {
					t.Fatal(err)
				}
				if _, err := runSnippet(t, shell, script); err == nil || readSnippetFile(t, path) != existing {
					t.Fatal("existing alias was duplicated", existing)
				}
			}
			if platform == "windows" {
				utf16 := []byte{0xff, 0xfe, 'H', 0}
				if err := os.WriteFile(path, utf16, 0600); err != nil {
					t.Fatal(err)
				}
				if _, err := runSnippet(t, shell, script); err == nil || readSnippetFile(t, path) != string(utf16) {
					t.Fatal("non-UTF8 config was modified")
				}
			}
		})
	}
}

func TestPOSIXScriptsRefuseNonRegularFilesAndSymlinks(t *testing.T) {
	shell := posixShell(t)
	current, err := exec.Command(shell, "-c", "id -un").Output()
	if err != nil {
		t.Fatal(err)
	}
	host := state.Host{Alias: "server", User: strings.TrimSpace(string(current))}
	for _, target := range []struct{ name, script string }{
		{"authorized_keys", posixPlacement(host, snippetPublic).Script},
		{"config", clientSetup(host, "Host server\n", "linux").Script},
	} {
		t.Run(target.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, ".ssh", target.name)
			if err := os.MkdirAll(path, 0755); err != nil {
				t.Fatal(err)
			}
			script := strings.ReplaceAll(target.script, "$HOME", filepath.ToSlash(root))
			if output, err := runSnippet(t, shell, script); err == nil || !strings.Contains(string(output), "regular file") {
				t.Fatal("non-regular target accepted", err, string(output))
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			other := filepath.Join(root, "unrelated")
			if err := os.WriteFile(other, []byte("preserve"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(other, path); err != nil {
				t.Log("symlink check unavailable:", err)
				return
			}
			if _, err := runSnippet(t, shell, script); err == nil || readSnippetFile(t, other) != "preserve" {
				t.Fatal("symlink target was changed")
			}
		})
	}
}

func TestWindowsDestinationScriptPreservesKeysAndRestrictsACL(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows ACL execution requires Windows")
	}
	for _, edition := range []string{"powershell", "pwsh"} {
		t.Run(edition, func(t *testing.T) {
			shell, err := exec.LookPath(edition)
			if err != nil {
				t.Skip("PowerShell edition unavailable")
			}
			checkWindowsDestinationSnippet(t, shell)
		})
	}
}

func checkWindowsDestinationSnippet(t *testing.T, shell string) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, ".ssh", "authorized_keys")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	original := "existing-key without-newline"
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	step := windowsPlacement(state.Host{WindowsAccount: "standard", User: os.Getenv("USERNAME")}, snippetPublic)
	script := strings.ReplaceAll(step.Script, "$env:USERPROFILE", "'"+strings.ReplaceAll(root, "'", "''")+"'")
	for i := 0; i < 2; i++ {
		if output, err := runSnippet(t, shell, script); err != nil {
			t.Fatalf("paste: %v %s", err, output)
		}
	}
	got := readSnippetFile(t, path)
	if !strings.HasPrefix(got, original) || strings.Count(got, snippetPublic) != 1 {
		t.Fatal("keys changed or duplicated", got)
	}
	wrong := windowsPlacement(state.Host{WindowsAccount: "standard", User: "wrong-keyflow-user"}, snippetPublic)
	wrongScript := strings.ReplaceAll(wrong.Script, "$env:USERPROFILE", "'"+strings.ReplaceAll(root, "'", "''")+"'")
	if _, err := runSnippet(t, shell, wrongScript); err == nil || readSnippetFile(t, path) != got {
		t.Fatal("wrong Windows account modified keys")
	}
	check := "$acl = Get-Acl -LiteralPath '" + strings.ReplaceAll(path, "'", "''") + `'
if (-not $acl.AreAccessRulesProtected) { throw 'Inherited ACL remains' }
$expected = @([Security.Principal.WindowsIdentity]::GetCurrent().User.Value, 'S-1-5-18', 'S-1-5-32-544')
$actual = @($acl.Access | ForEach-Object { $_.IdentityReference.Translate([Security.Principal.SecurityIdentifier]).Value })
if ($actual.Count -ne 3 -or @($actual | Where-Object { $_ -notin $expected }).Count) { throw 'Unexpected ACL principals' }
`
	if output, err := runSnippet(t, shell, check); err != nil {
		t.Fatalf("ACL: %v %s", err, output)
	}
	utf16 := []byte{0xff, 0xfe, 'k', 0}
	if err := os.WriteFile(path, utf16, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runSnippet(t, shell, script); err == nil || readSnippetFile(t, path) != string(utf16) {
		t.Fatal("non-UTF8 authorized_keys was modified")
	}
	admin := windowsPlacement(state.Host{WindowsAccount: "administrator"}, snippetPublic)
	if strings.Contains(admin.Script, "$identity.User.Value") || strings.Contains(admin.Script, "DirectorySecurity") || !strings.Contains(admin.Script, "IsInRole") || !strings.Contains(admin.Script, "administrators_authorized_keys") {
		t.Fatal("administrator script did not restrict only its shared key file")
	}
	// Parse the elevated script without executing it or touching ProgramData.
	parser := "$errors = $null; [Management.Automation.Language.Parser]::ParseInput(@'\n" + admin.Script + "\n'@, [ref]$null, [ref]$errors) | Out-Null; if ($errors.Count) { throw $errors[0] }"
	if output, err := runSnippet(t, shell, parser); err != nil {
		t.Fatalf("administrator script syntax: %v %s", err, output)
	}
	// Exercise the elevation guard with ProgramData redirected to a disposable path.
	adminScript := strings.ReplaceAll(admin.Script, "$env:ProgramData", "'"+strings.ReplaceAll(root, "'", "''")+"'")
	if output, err := runSnippet(t, shell, adminScript); err != nil {
		if !strings.Contains(string(output), "Open PowerShell as Administrator first.") {
			t.Fatalf("unexpected administrator failure: %v %s", err, output)
		}
		if _, err := os.Stat(filepath.Join(root, "ssh")); !os.IsNotExist(err) {
			t.Fatal("elevation guard created destination files")
		}
		t.Log("administrator elevation guard passed; elevated execution not exercised")
	} else {
		t.Log("elevated administrator script executed against temporary ProgramData")
	}
}
