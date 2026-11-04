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
	for _, scenario := range []string{"inspect", "Include", "hardlink", "quote", "encoding", "ACL", "administrator"} {
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
