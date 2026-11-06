package ssh_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"ssh-keyflow/internal/ssh"
)

const revokedPublic = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKcOMg3TlE6eZ28nRHSdnnwEJYEiHoCGkcLIsbHa1IH2"
const retainedPublic = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBcOMg3TlE6eZ28nRHSdnnwEJYEiHoCGkcLIsbHa1IH2"

func TestWindowsLargeRemoteCommandPreservesExecutionAndErrors(testingContext *testing.T) {
	if runtime.GOOS != "windows" {
		testingContext.Skip("requires native Windows PowerShell")
	}
	for _, failure := range []bool{false, true} {
		script := strings.Repeat("$null='日本語'\n", 1500) + "[Console]::WriteLine('payload executed')"
		if failure {
			script += "\nthrow 'payload failure'"
		}
		payload := ssh.RemoteCommand("windows", script)
		if len(payload) >= 30000 {
			testingContext.Fatal("payload exceeds Windows command-line budget", len(payload))
		}
		arguments := strings.Fields(payload)
		output, err := exec.Command(arguments[0], arguments[1:]...).CombinedOutput()
		if (err != nil) != failure || !strings.Contains(string(output), "payload executed") || (failure && !strings.Contains(string(output), "payload failure")) {
			testingContext.Fatal("remote payload behavior changed", failure, err, string(output))
		}
	}
}

func TestPOSIXRevocationRemovesExactIdentityAndPreservesOtherBytes(testingContext *testing.T) {
	plan, err := ssh.StandardRevocation("linux", "", []byte(revokedPublic+" ignored comment"))
	if err != nil {
		testingContext.Fatal(err)
	}
	fixture := testingContext.TempDir()
	retained := "\xef\xbb\xbf# keep 日本語\r\n" + retainedPublic + " keep\ncommand=\"echo " + revokedPublic + "\",restrict " + retainedPublic + "\r\n# " + revokedPublic + "\n# final without newline"
	initial := strings.Replace(retained, "# final", "restrict,command=\"echo \\\"hello world\\\"\" "+revokedPublic+" first\r\n"+revokedPublic+" second\n# final", 1)
	authorized := writeRevocationFixture(testingContext, fixture, ".ssh/authorized_keys", initial)
	writeRevocationFixture(testingContext, fixture, "sshd_config", "AuthorizedKeysFile .ssh/authorized_keys\n")
	for attempt := 0; attempt < 2; attempt++ {
		output, runErr := runPOSIXRevocation(testingContext, fixture, plan.Script)
		if runErr != nil {
			testingContext.Fatalf("generated revocation: %v %s", runErr, output)
		}
		after, readErr := os.ReadFile(authorized)
		if readErr != nil || string(after) != retained {
			testingContext.Fatalf("unrelated bytes changed: %v %q", readErr, after)
		}
		if attempt == 1 && !strings.Contains(output, "Already absent") {
			testingContext.Fatalf("absence not reported: %s", output)
		}
	}
}

func TestPOSIXRevocationRejectsUncertainConfiguration(testingContext *testing.T) {
	for _, configuration := range []string{"Include other.conf\n", "Match User someone\nAuthorizedKeysFile other\n", "AuthorizedKeysFile .ssh/authorized_keys other\n", "AuthorizedKeysCommand /bin/custom\n", "AuthorizedKeysFile .ssh/AUTHORIZED_KEYS\n", "\xef\xbb\xbfAuthorizedKeysFile custom\n"} {
		testingContext.Run(configuration, func(testingContext *testing.T) {
			fixture := testingContext.TempDir()
			authorized := writeRevocationFixture(testingContext, fixture, ".ssh/authorized_keys", revokedPublic+"\n")
			writeRevocationFixture(testingContext, fixture, "sshd_config", configuration)
			plan, err := ssh.StandardRevocation("linux", "", []byte(revokedPublic))
			if err != nil {
				testingContext.Fatal(err)
			}
			output, runErr := runPOSIXRevocation(testingContext, fixture, plan.Script)
			after, readErr := os.ReadFile(authorized)
			if runErr == nil || readErr != nil || string(after) != revokedPublic+"\n" {
				testingContext.Fatalf("unsafe configuration accepted or bytes changed: %v %v %s %q", runErr, readErr, output, after)
			}
		})
	}
}

func TestPOSIXRevocationDoesNotTrustConfigurationOverride(testingContext *testing.T) {
	fixture := testingContext.TempDir()
	initial := revokedPublic + "\n"
	authorized := writeRevocationFixture(testingContext, fixture, ".ssh/authorized_keys", initial)
	writeRevocationFixture(testingContext, fixture, "sshd_config", "AuthorizedKeysFile /custom/authorized\n")
	writeRevocationFixture(testingContext, fixture, "fake_config", "AuthorizedKeysFile .ssh/authorized_keys\n")
	plan, err := ssh.StandardRevocation("linux", "", []byte(revokedPublic))
	if err != nil {
		testingContext.Fatal(err)
	}
	output, runErr := runPOSIXRevocation(testingContext, fixture, "export SSH_KEYFLOW_SSHD_CONFIG=\"$HOME/fake_config\"\n"+plan.Script)
	after, readErr := os.ReadFile(authorized)
	if runErr == nil || readErr != nil || string(after) != initial {
		testingContext.Fatal("environment hid unsafe daemon configuration", runErr, readErr, output)
	}
}

func TestPOSIXRevocationRefusesSecondaryAuthorizationSource(testingContext *testing.T) {
	fixture := testingContext.TempDir()
	initial := revokedPublic + "\n"
	authorized := writeRevocationFixture(testingContext, fixture, ".ssh/authorized_keys", initial)
	secondary := writeRevocationFixture(testingContext, fixture, ".ssh/authorized_keys2", initial)
	writeRevocationFixture(testingContext, fixture, "sshd_config", "")
	plan, _ := ssh.StandardRevocation("linux", "", []byte(revokedPublic))
	output, err := runPOSIXRevocation(testingContext, fixture, plan.Script)
	primaryBytes, _ := os.ReadFile(authorized)
	secondaryBytes, _ := os.ReadFile(secondary)
	if err == nil || string(primaryBytes) != initial || string(secondaryBytes) != initial {
		testingContext.Fatal("secondary authorization source was ignored", err, output)
	}
}

func TestPOSIXRevocationWriteFailurePreservesUnrelatedAuthorization(testingContext *testing.T) {
	fixture := testingContext.TempDir()
	initial := revokedPublic + "\n" + strings.Repeat("# retain unrelated bytes\r\n", 256) + retainedPublic + "\n"
	authorized := writeRevocationFixture(testingContext, fixture, ".ssh/authorized_keys", initial)
	writeRevocationFixture(testingContext, fixture, "sshd_config", "AuthorizedKeysFile .ssh/authorized_keys\n")
	plan, err := ssh.StandardRevocation("linux", "", []byte(revokedPublic))
	if err != nil {
		testingContext.Fatal(err)
	}
	stream := "$stream"
	if strings.Contains(plan.Script, "print {$staged} $retained") {
		stream = "$staged"
	}
	write := "print {" + stream + "} $retained"
	fault := "print {" + stream + "} substr($retained, 0, 1); " + stream + "->flush(); die \"fixture partial write\\n\";\n" + write
	output, runErr := runPOSIXRevocation(testingContext, fixture, strings.Replace(plan.Script, write, fault, 1))
	after, readErr := os.ReadFile(authorized)
	if runErr == nil || readErr != nil || string(after) != initial {
		testingContext.Fatal("failed remote write damaged authorization", runErr, readErr, output)
	}
}

func TestPOSIXRevocationKeepsAnExistingOperationLock(testingContext *testing.T) {
	fixture := testingContext.TempDir()
	authorized := writeRevocationFixture(testingContext, fixture, ".ssh/authorized_keys", revokedPublic+"\n")
	writeRevocationFixture(testingContext, fixture, "sshd_config", "")
	lock := filepath.Join(fixture, ".ssh/ssh-keyflow-registration.lock")
	if err := os.Mkdir(lock, 0700); err != nil {
		testingContext.Fatal(err)
	}
	plan, err := ssh.StandardRevocation("linux", "", []byte(revokedPublic))
	if err != nil {
		testingContext.Fatal(err)
	}
	output, runErr := runPOSIXRevocation(testingContext, fixture, plan.Script)
	after, readErr := os.ReadFile(authorized)
	_, lockErr := os.Stat(lock)
	if runErr == nil || readErr != nil || lockErr != nil || string(after) != revokedPublic+"\n" {
		testingContext.Fatalf("existing operation lock or file changed: %v %v %v %s", runErr, readErr, lockErr, output)
	}
}

func TestPOSIXRevocationInterruptionAndTargetRaceKeepRecoverableBytes(testingContext *testing.T) {
	for _, scenario := range []string{"before replacement", "after replacement", "target changed"} {
		testingContext.Run(scenario, func(testingContext *testing.T) {
			fixture := testingContext.TempDir()
			retained := "\xef\xbb\xbf# unrelated 日本語\r\n" + retainedPublic + "\r\n# final"
			initial := revokedPublic + "\n" + retained
			authorized := writeRevocationFixture(testingContext, fixture, ".ssh/authorized_keys", initial)
			writeRevocationFixture(testingContext, fixture, "sshd_config", "")
			plan, err := ssh.StandardRevocation("linux", "", []byte(revokedPublic))
			if err != nil {
				testingContext.Fatal(err)
			}
			anchor := "my @current = lstat($authorized);"
			injection := "kill 9, $$;\n"
			expected := initial
			if scenario == "after replacement" {
				anchor = "$directory_stream->sync() or die \"Cannot sync replacement directory; backup retained; result unknown\\n\";"
				expected = retained
			}
			if scenario == "target changed" {
				injection = "open(my $changed, '>>', $authorized) or die $!; print {$changed} \"# external change\\n\"; close($changed);\n"
				expected += "# external change\n"
			}
			if strings.Count(plan.Script, anchor) != 1 {
				testingContext.Fatal("fault injection boundary missing", anchor)
			}
			output, runErr := runPOSIXRevocation(testingContext, fixture, strings.Replace(plan.Script, anchor, injection+anchor, 1))
			after, readErr := os.ReadFile(authorized)
			if runErr == nil || readErr != nil || string(after) != expected {
				testingContext.Fatal("interruption or race damaged authorization", runErr, readErr, output, string(after))
			}
			backups, err := filepath.Glob(filepath.Join(fixture, ".ssh/authorized_keys.keyflow-backup-*"))
			if err != nil || len(backups) != 1 {
				testingContext.Fatal("durable backup missing", err, backups)
			}
			backup, err := os.ReadFile(backups[0])
			if err != nil || string(backup) != initial {
				testingContext.Fatal("original authorization is not recoverable", err)
			}
		})
	}
}

func TestPOSIXRevocationInspectionIsReadOnly(testingContext *testing.T) {
	fixture := testingContext.TempDir()
	initial := "\xef\xbb\xbfcommand=\"echo hi\" " + revokedPublic + "\r\n"
	authorized := writeRevocationFixture(testingContext, fixture, ".ssh/authorized_keys", initial)
	writeRevocationFixture(testingContext, fixture, "sshd_config", "")
	plan, err := ssh.StandardRevocation("macos", "", []byte(revokedPublic))
	if err != nil {
		testingContext.Fatal(err)
	}
	output, runErr := runPOSIXRevocation(testingContext, fixture, plan.InspectScript)
	after, readErr := os.ReadFile(authorized)
	if runErr != nil || readErr != nil || string(after) != initial || !strings.Contains(output, "Public key present") {
		testingContext.Fatalf("inspection changed data or misreported presence: %v %v %s %q", runErr, readErr, output, after)
	}
	entries, err := os.ReadDir(filepath.Dir(authorized))
	if err != nil || len(entries) != 1 {
		testingContext.Fatalf("inspection created files: %v %v", err, entries)
	}
}

func TestPOSIXRevocationRefusesUnsafeTargets(testingContext *testing.T) {
	for _, scenario := range []string{"permissions", "hardlink", "quoted options", "encoding", "symlink"} {
		testingContext.Run(scenario, func(testingContext *testing.T) {
			if scenario == "permissions" && runtime.GOOS == "windows" {
				testingContext.Skip("Git Bash chmod does not implement native POSIX permission bits")
			}
			fixture := testingContext.TempDir()
			initial := revokedPublic + "\n"
			if scenario == "quoted options" {
				initial = "command=\"unterminated " + revokedPublic + "\n"
			}
			if scenario == "encoding" {
				initial = "\xff\xfe" + initial + "\x00"
			}
			authorized := writeRevocationFixture(testingContext, fixture, ".ssh/authorized_keys", initial)
			writeRevocationFixture(testingContext, fixture, "sshd_config", "")
			plan, err := ssh.StandardRevocation("linux", "", []byte(revokedPublic))
			if err != nil {
				testingContext.Fatal(err)
			}
			if scenario == "hardlink" || scenario == "symlink" {
				victim := writeRevocationFixture(testingContext, fixture, "other", initial)
				if err := os.Remove(authorized); err != nil {
					testingContext.Fatal(err)
				}
				link := os.Link
				if scenario == "symlink" {
					link = os.Symlink
				}
				if err := link(victim, authorized); err != nil {
					testingContext.Skipf("fixture link unavailable: %v", err)
				}
			}
			if scenario == "permissions" {
				plan.Script = "chmod 666 \"$HOME/.ssh/authorized_keys\"\n" + plan.Script
			}
			output, runErr := runPOSIXRevocation(testingContext, fixture, plan.Script)
			after, readErr := os.ReadFile(authorized)
			if runErr == nil || readErr != nil || string(after) != initial {
				testingContext.Fatalf("unsafe target changed or accepted: %v %v %s %q", runErr, readErr, output, after)
			}
		})
	}
}

func TestWindowsRevocationPreservesBytesAndExistingACL(testingContext *testing.T) {
	if runtime.GOOS != "windows" {
		testingContext.Skip("Windows PowerShell fixture")
	}
	fixture := testingContext.TempDir()
	plan, err := ssh.StandardRevocation("windows", "standard", []byte(revokedPublic))
	if err != nil {
		testingContext.Fatal(err)
	}
	retained := "\xef\xbb\xbf# keep 日本語\r\ncommand=\"echo " + revokedPublic + "\",restrict " + retainedPublic + "\n# final"
	initial := strings.Replace(retained, "# final", "command=\"echo \\\"hi\\\"\",restrict "+revokedPublic+" comment\r\n"+revokedPublic+"\n# final", 1)
	authorized := writeRevocationFixture(testingContext, fixture, "home/.ssh/authorized_keys", initial)
	writeRevocationFixture(testingContext, fixture, "server/sshd_config", "AuthorizedKeysFile .ssh/authorized_keys\r\n")
	scriptPath := writeRevocationFixture(testingContext, fixture, "revoke.ps1", plan.Script)
	helperPath := writeRevocationFixture(testingContext, fixture, "fixture.ps1", windowsRevocationFixture)
	command := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", helperPath, scriptPath, fixture)
	command.Env = append(os.Environ(), "PSModulePath="+filepath.Join(os.Getenv("WINDIR"), "System32/WindowsPowerShell/v1.0/Modules"))
	output, runErr := command.CombinedOutput()
	if runErr != nil {
		testingContext.Fatalf("Windows fixture: %v %s", runErr, output)
	}
	after, readErr := os.ReadFile(authorized)
	if readErr != nil || string(after) != retained {
		testingContext.Fatalf("unrelated bytes changed: %v %q %s", readErr, after, output)
	}
}

func TestWindowsRevocationSafetyAndReadOnlyFixtures(testingContext *testing.T) {
	if runtime.GOOS != "windows" {
		testingContext.Skip("requires native Windows PowerShell and NTFS")
	}
	for _, scenario := range []string{"inspect", "Include", "hardlink", "quote", "encoding", "ACL", "administrator", "lock"} {
		testingContext.Run(scenario, func(testingContext *testing.T) {
			fixture := testingContext.TempDir()
			initial := revokedPublic + "\r\n"
			if scenario == "quote" {
				initial = "command=\"unterminated " + initial
			}
			if scenario == "encoding" {
				initial = "\xff\xfe" + initial + "\x00"
			}
			authorized := writeRevocationFixture(testingContext, fixture, "home/.ssh/authorized_keys", initial)
			configuration := "AuthorizedKeysFile .ssh/authorized_keys\n"
			if scenario == "Include" {
				configuration += "Include ignored.conf\n"
			}
			writeRevocationFixture(testingContext, fixture, "server/sshd_config", configuration)
			if scenario == "hardlink" {
				if err := os.Link(authorized, filepath.Join(fixture, "unrelated")); err != nil {
					testingContext.Fatal(err)
				}
			}
			if scenario == "lock" {
				writeRevocationFixture(testingContext, fixture, "home/.ssh/ssh-keyflow-revocation.lock", "existing operation")
			}
			account := "standard"
			if scenario == "administrator" {
				account = "administrator"
			}
			plan, err := ssh.StandardRevocation("windows", account, []byte(revokedPublic))
			if err != nil {
				testingContext.Fatal(err)
			}
			script := plan.Script
			if scenario == "inspect" {
				script = plan.InspectScript
			}
			helper := windowsRevocationFixture
			before := "Write-Output 'FIXTURE_READY'\n"
			if scenario == "ACL" {
				helper = strings.Replace(helper, "$acl.AddAccessRule($rule)", "$acl.AddAccessRule($rule)\n    if ($path -eq $authorized) { $acl.AddAccessRule((New-Object Security.AccessControl.FileSystemAccessRule((New-Object Security.Principal.SecurityIdentifier('S-1-1-0')),'Read','Allow'))) }", 1)
			}
			helper = strings.Replace(helper, "$beforeACL =", before+"$beforeACL =", 1)
			scriptPath := writeRevocationFixture(testingContext, fixture, "revoke.ps1", script)
			helperPath := writeRevocationFixture(testingContext, fixture, "fixture.ps1", helper)
			command := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", helperPath, scriptPath, fixture)
			command.Env = append(os.Environ(), "PSModulePath="+filepath.Join(os.Getenv("WINDIR"), "System32/WindowsPowerShell/v1.0/Modules"))
			output, runErr := command.CombinedOutput()
			after, readErr := os.ReadFile(authorized)
			if readErr != nil || string(after) != initial || !strings.Contains(string(output), "FIXTURE_READY") {
				testingContext.Fatal("fixture setup failed or protected bytes changed", readErr, runErr, string(output))
			}
			if scenario == "inspect" {
				if runErr != nil || strings.Count(string(output), "Public key present") != 2 {
					testingContext.Fatal("read-only inspection failed", runErr, string(output))
				}
			} else if runErr == nil {
				testingContext.Fatal("unsafe revocation accepted", string(output))
			}
		})
	}
}

func TestWindowsRevocationFaultsKeepBackupBytesAndACL(testingContext *testing.T) {
	if runtime.GOOS != "windows" {
		testingContext.Skip("requires native Windows PowerShell and NTFS")
	}
	for _, scenario := range []string{"partial write", "interrupted", "target changed"} {
		testingContext.Run(scenario, func(testingContext *testing.T) {
			fixture := testingContext.TempDir()
			initial := "\xef\xbb\xbf# retain 日本語\r\n" + revokedPublic + "\n" + retainedPublic + "\r\n# final"
			authorized := writeRevocationFixture(testingContext, fixture, "home/.ssh/authorized_keys", initial)
			writeRevocationFixture(testingContext, fixture, "server/sshd_config", "AuthorizedKeysFile .ssh/authorized_keys\n")
			plan, err := ssh.StandardRevocation("windows", "standard", []byte(revokedPublic))
			if err != nil {
				testingContext.Fatal(err)
			}
			anchor := "$staged.Write($replacement, 0, $replacement.Length)"
			injection := "$staged.Write($replacement, 0, 1); $staged.Flush($true); throw 'fixture partial write'\n"
			expected := initial
			if scenario == "interrupted" {
				anchor = "$staged.Flush($true)"
				injection = "$staged.Flush($true); [Diagnostics.Process]::GetCurrentProcess().Kill()\n"
			}
			if scenario == "target changed" {
				anchor = "$current = [IO.File]::Open($authorized, [IO.FileMode]::Open, [IO.FileAccess]::Read, $sharing)"
				injection = "[IO.File]::Move($authorized, ($authorized + '.raced')); [IO.File]::WriteAllText($authorized, 'external authorization'); Set-Acl -LiteralPath $authorized -AclObject $originalACL\n"
				expected = "external authorization"
			}
			if strings.Count(plan.Script, anchor) != 1 {
				testingContext.Fatal("fault injection boundary missing", anchor)
			}
			script := strings.Replace(plan.Script, anchor, injection+anchor, 1)
			scriptPath := writeRevocationFixture(testingContext, fixture, "revoke.ps1", script)
			helper := strings.Replace(windowsRevocationFixture, "$beforeACL = (Get-Acl -LiteralPath $authorized).Sddl", "$beforeACL = (Get-Acl -LiteralPath $authorized).Sddl\n[IO.File]::WriteAllText((Join-Path $Fixture 'before-acl'), $beforeACL)", 1)
			helperPath := writeRevocationFixture(testingContext, fixture, "fixture.ps1", helper)
			command := exec.Command("powershell.exe", "-NoProfile", "-File", helperPath, scriptPath, fixture)
			command.Env = append(os.Environ(), "PSModulePath="+filepath.Join(os.Getenv("WINDIR"), "System32/WindowsPowerShell/v1.0/Modules"))
			output, runErr := command.CombinedOutput()
			after, readErr := os.ReadFile(authorized)
			if runErr == nil || readErr != nil || string(after) != expected {
				testingContext.Fatal("failed revocation damaged authorization", runErr, readErr, string(output))
			}
			backups, err := filepath.Glob(filepath.Join(fixture, "home/.ssh/authorized_keys.keyflow-backup-*"))
			if err != nil || len(backups) != 1 {
				testingContext.Fatal("durable backup missing", err, backups)
			}
			backup, err := os.ReadFile(backups[0])
			if err != nil || string(backup) != initial {
				testingContext.Fatal("original authorization is not recoverable", err)
			}
			beforeACL, err := os.ReadFile(filepath.Join(fixture, "before-acl"))
			if err != nil {
				testingContext.Fatal(err)
			}
			for _, path := range []string{authorized, backups[0]} {
				aclScript := "(Get-Acl -LiteralPath '" + strings.ReplaceAll(path, "'", "''") + "').Sddl"
				aclCommand := exec.Command("powershell.exe", "-NoProfile", "-Command", aclScript)
				aclCommand.Env = command.Env
				acl, err := aclCommand.CombinedOutput()
				if err != nil || strings.TrimSpace(string(acl)) != string(beforeACL) {
					testingContext.Fatal("authorization ACL changed", err, string(acl))
				}
			}
		})
	}
}

const windowsRevocationFixture = `param($ScriptPath, $Fixture)
$ErrorActionPreference = 'Stop'
$identity = [Security.Principal.WindowsIdentity]::GetCurrent()
$fixtureHome = Join-Path $Fixture 'home'
$fixtureServer = Join-Path $Fixture 'server'
$authorized = Join-Path $fixtureHome '.ssh/authorized_keys'
foreach ($path in @($fixtureHome, (Join-Path $fixtureHome '.ssh'), $authorized, $fixtureServer, (Join-Path $fixtureServer 'sshd_config'))) {
    if ((Get-Item -LiteralPath $path).PSIsContainer) { $acl = New-Object Security.AccessControl.DirectorySecurity } else { $acl = New-Object Security.AccessControl.FileSecurity }
    $acl.SetAccessRuleProtection($true, $false)
    $acl.SetOwner($identity.User)
    $rule = New-Object Security.AccessControl.FileSystemAccessRule($identity.User, 'FullControl', 'Allow')
    $acl.AddAccessRule($rule)
    Set-Acl -LiteralPath $path -AclObject $acl
}
$source = [IO.File]::ReadAllText($ScriptPath)
$groupQuery = '$adminGroup = @($identity.Groups | ForEach-Object { $_.Value }) -contains $administratorSID'
if (-not $source.Contains($groupQuery)) { throw 'Fixture account seam missing' }
$source = $source.Replace($groupQuery, '$adminGroup = $false')
$ScriptPath = Join-Path $Fixture 'isolated-standard.ps1'
[IO.File]::WriteAllText($ScriptPath, $source)
$beforeACL = (Get-Acl -LiteralPath $authorized).Sddl
& $ScriptPath -RevocationHome $fixtureHome -RevocationServer $fixtureServer
& $ScriptPath -RevocationHome $fixtureHome -RevocationServer $fixtureServer
if ((Get-Acl -LiteralPath $authorized).Sddl -cne $beforeACL) { throw 'Existing ACL changed' }
`

func writeRevocationFixture(testingContext *testing.T, fixture, relative, contents string) string {
	testingContext.Helper()
	path := filepath.Join(fixture, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		testingContext.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		testingContext.Fatal(err)
	}
	return path
}

func runPOSIXRevocation(testingContext *testing.T, fixture, script string) (string, error) {
	testingContext.Helper()
	shell := "sh"
	if runtime.GOOS == "windows" {
		shell = "C:/Program Files/Git/bin/bash.exe"
	}
	if _, err := exec.LookPath(shell); err != nil {
		testingContext.Skip("POSIX shell unavailable")
	}
	resolved, err := filepath.EvalSymlinks(fixture)
	if err != nil {
		testingContext.Fatal(err)
	}
	posixFixture := filepath.ToSlash(resolved)
	if runtime.GOOS == "windows" {
		posixFixture = "/" + strings.ToLower(posixFixture[:1]) + posixFixture[2:]
	}
	script = strings.ReplaceAll(script, "/etc/ssh/sshd_config", posixFixture+"/sshd_config")
	scriptPath := writeRevocationFixture(testingContext, resolved, "revoke.sh", script)
	command := exec.Command(shell, "-c", `chmod 700 "$HOME" "$HOME/.ssh"; chmod 600 "$HOME/.ssh/authorized_keys" "$HOME/sshd_config"; sh "$1"`, "fixture", filepath.ToSlash(scriptPath))
	command.Env = append(os.Environ(), "HOME="+filepath.ToSlash(resolved))
	output, err := command.CombinedOutput()
	return string(output), err
}

func TestWindowsRevocationChecksNativeAccountType(testingContext *testing.T) {
	if runtime.GOOS != "windows" {
		testingContext.Skip("requires native Windows identity")
	}
	output, err := exec.Command("powershell.exe", "-NoProfile", "-Command", `$identity=[Security.Principal.WindowsIdentity]::GetCurrent(); @($identity.Groups | ForEach-Object { $_.Value }) -contains 'S-1-5-32-544'; (New-Object Security.Principal.WindowsPrincipal($identity)).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)`).CombinedOutput()
	fields := strings.Fields(string(output))
	if err != nil || len(fields) != 2 {
		testingContext.Fatal("native identity", err, string(output))
	}
	administrator, elevated := fields[0] == "True", fields[1] == "True"
	for _, account := range []string{"standard", "administrator"} {
		testingContext.Run(account, func(testingContext *testing.T) {
			plan, err := ssh.StandardRevocation("windows", account, []byte(revokedPublic))
			if err != nil {
				testingContext.Fatal(err)
			}
			prelude, _, found := strings.Cut(plan.Script, "$allowed =")
			if !found {
				testingContext.Fatal("account guard boundary missing")
			}
			script := writeRevocationFixture(testingContext, testingContext.TempDir(), "guard.ps1", prelude+"Write-Output 'NATIVE_ACCOUNT_ACCEPTED'\n")
			result, runErr := exec.Command("powershell.exe", "-NoProfile", "-File", script).CombinedOutput()
			if (account == "administrator") != administrator {
				if runErr == nil || !strings.Contains(string(result), "Account type mismatch") {
					testingContext.Fatal("native account mismatch accepted", runErr, string(result))
				}
			} else if administrator && !elevated {
				if runErr == nil || !strings.Contains(string(result), "already elevated") {
					testingContext.Fatal("elevation guard bypassed", runErr, string(result))
				}
			} else if runErr != nil || !strings.Contains(string(result), "NATIVE_ACCOUNT_ACCEPTED") {
				testingContext.Fatal("native matching account rejected", runErr, string(result))
			}
		})
	}
}
