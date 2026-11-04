package app_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/state"
)

type alternateKeyRunner struct {
	registrationRunner
	derived string
}

func (external alternateKeyRunner) Run(ctx context.Context, command app.Command) error {
	if command.Name == "ssh-keygen" {
		if !command.Interactive || len(command.Args) != 3 || command.Args[0] != "-y" || command.Args[1] != "-f" {
			external.testingContext.Fatal("alternate identity was not derived interactively", command.Args)
		}
		fmt.Fprintln(command.Stdout, external.derived)
		return nil
	}
	return external.registrationRunner.Run(ctx, command)
}

func TestNativeSSHParsesOnlyDistinctAlternateIdentityForRevocation(testingContext *testing.T) {
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		testingContext.Skip("native OpenSSH unavailable")
	}
	home, external := prepareRegistration(testingContext, true)
	alternate := writeFixture(testingContext, home, "distinct alternate key", "alternate private fixture")
	document, _ := state.Load(home)
	target := document.Keys[0].PrivatePath
	calls := 0
	external.ssh = func(command app.Command) error {
		calls++
		arguments := append([]string{"-G"}, command.Args[:len(command.Args)-1]...)
		configuration, err := exec.Command(sshPath, arguments...).CombinedOutput()
		if err != nil {
			testingContext.Fatal(err, string(configuration))
		}
		for _, expected := range []string{"identitiesonly yes", "identityagent none", "certificatefile none", "controlmaster false", "preferredauthentications publickey", "passwordauthentication no", "kbdinteractiveauthentication no", "gssapiauthentication no", "hostbasedauthentication no", "identityfile " + alternate} {
			if !strings.Contains(string(configuration), expected) {
				testingContext.Fatal("native authentication isolation missing", expected, string(configuration))
			}
		}
		if strings.Contains(string(configuration), target) {
			testingContext.Fatal("target identity leaked into alternate authentication")
		}
		for _, line := range strings.Split(string(configuration), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "identityfile ") && line != "identityfile none" && strings.TrimPrefix(line, "identityfile ") != alternate {
				testingContext.Fatal("default identity fallback", line)
			}
			if strings.HasPrefix(line, "controlpath ") && strings.TrimSpace(line) != "controlpath none" {
				testingContext.Fatal("multiplexing fallback", line)
			}
		}
		payload := command.Args[len(command.Args)-1]
		if strings.Contains(payload, "KEYFLOW_ALTERNATE") {
			fmt.Fprintln(command.Stdout, "KEYFLOW_TARGET:/home/me/.ssh/authorized_keys")
		} else if strings.Contains(payload, "KEYFLOW_REVOCATION_READ") {
			fmt.Fprintln(command.Stdout, "KEYFLOW_PRESENT")
		} else {
			fmt.Fprintln(command.Stdout, "KEYFLOW_REMOVED")
		}
		return nil
	}
	output, err := runCLI(testingContext, home, "4\n1\nk\n"+alternate+"\nyes\nyes\n0\n", alternateKeyRunner{registrationRunner: external, derived: otherPublicKey})
	after, _ := state.Load(home)
	if err != nil || calls != 3 || after.Registrations[0].Status != "removed" {
		testingContext.Fatal(err, calls, output)
	}
}

func TestRevocationExecutesGuardedPayloadInIsolatedPOSIXFixture(testingContext *testing.T) {
	shell := "bash"
	if runtime.GOOS == "windows" {
		shell = "C:/Program Files/Git/bin/bash.exe"
	}
	if _, err := exec.LookPath(shell); err != nil {
		testingContext.Skip("POSIX shell unavailable")
	}
	for _, system := range []string{"linux", "macos"} {
		testingContext.Run(system, func(testingContext *testing.T) {
			home, external := prepareRegistration(testingContext, false)
			if err := state.Update(home, func(document *state.State) error {
				document.Hosts[0].OS = system
				return nil
			}); err != nil {
				testingContext.Fatal(err)
			}
			remote := testingContext.TempDir()
			kernel := "Linux"
			if system == "macos" {
				kernel = "Darwin"
			}
			writeFixture(testingContext, remote, "bin/uname", "#!/bin/sh\nprintf '%s\\n' "+kernel+"\n")
			writeFixture(testingContext, remote, "bin/id", "#!/bin/sh\nif [ \"$1\" = -un ]; then printf '%s\\n' me; else /usr/bin/id \"$@\"; fi\n")
			effective := "usedns no\npubkeyauthentication yes\nstrictmodes yes\nauthorizedkeyscommand none\nauthorizedkeysfile .ssh/authorized_keys\n"
			daemon := writeFixture(testingContext, remote, "bin/sshd", "#!/bin/sh\n[ \"$1\" = -T ] && [ \"$2\" = -C ] || exit 1\nprintf '%s' '"+effective+"'\n")
			initial := "\xef\xbb\xbf# keep 日本語\r\ncommand=\"echo " + strings.TrimSpace(publicKey) + "\" " + strings.TrimSpace(otherPublicKey) + "\r\n"
			authorized := writeFixture(testingContext, remote, ".ssh/authorized_keys", initial+"restrict "+strings.TrimSpace(publicKey)+"\n")
			writeFixture(testingContext, remote, "sshd_config", "AuthorizedKeysFile .ssh/authorized_keys\n")
			prepare := exec.Command(shell, "-c", `chmod 700 "$HOME" "$HOME/.ssh" "$HOME/bin" && chmod 600 "$HOME/.ssh/authorized_keys" "$HOME/sshd_config" && chmod 700 "$HOME/bin/"*`)
			prepare.Env = append(os.Environ(), "HOME="+filepath.ToSlash(remote))
			if output, err := prepare.CombinedOutput(); err != nil {
				testingContext.Fatal(err, string(output))
			}
			external.ssh = func(command app.Command) error {
				payload := strings.ReplaceAll(command.Args[len(command.Args)-1], "/usr/sbin/sshd", filepath.ToSlash(daemon))
				configuration := filepath.ToSlash(filepath.Join(remote, "sshd_config"))
				if runtime.GOOS == "windows" {
					configuration = "/" + strings.ToLower(configuration[:1]) + configuration[2:]
				}
				payload = strings.ReplaceAll(payload, "/etc/ssh/sshd_config", configuration)
				execute := exec.Command(shell, "-c", "PATH=\"$HOME/bin:/usr/bin:/bin\"; export PATH; "+payload)
				execute.Env = append(os.Environ(), "HOME="+filepath.ToSlash(remote), "SSH_CONNECTION=192.0.2.1 12345 192.0.2.2 22")
				execute.Stdout, execute.Stderr = command.Stdout, command.Stderr
				return execute.Run()
			}
			output, err := runCLI(testingContext, home, "4\n1\np\nyes\nyes\n4\n1\np\nyes\n0\n", external)
			after, readErr := os.ReadFile(authorized)
			document, loadErr := state.Load(home)
			if err != nil || readErr != nil || loadErr != nil || string(after) != initial || document.Registrations[0].Status != "removed" || !strings.Contains(output, "Already absent") {
				testingContext.Fatal("guarded script failed", err, readErr, loadErr, document.Registrations, output)
			}
		})
	}
}

func TestWindowsRevocationExecutesGuardedPayloadInNTFSFixture(testingContext *testing.T) {
	if runtime.GOOS != "windows" {
		testingContext.Skip("requires native Windows PowerShell and NTFS")
	}
	identityOutput, err := exec.Command("powershell.exe", "-NoProfile", "-Command", `$identity=[Security.Principal.WindowsIdentity]::GetCurrent(); $identity.Name; @($identity.Groups | ForEach-Object { $_.Value }) -contains 'S-1-5-32-544'`).CombinedOutput()
	identityLines := strings.Fields(string(identityOutput))
	if err != nil || len(identityLines) != 2 {
		testingContext.Fatal("native identity unavailable", err, string(identityOutput))
	}
	if identityLines[1] != "False" {
		testingContext.Skip("standard-account fixture requires non-administrator identity")
	}
	home, external := prepareRegistration(testingContext, true)
	if err := state.Update(home, func(document *state.State) error {
		document.Hosts[0].OS, document.Hosts[0].WindowsAccount, document.Hosts[0].User = "windows", "standard", identityLines[0]
		return nil
	}); err != nil {
		testingContext.Fatal(err)
	}
	remote := testingContext.TempDir()
	fixtureHome := filepath.Join(remote, "home with spaces")
	retained := "\xef\xbb\xbf# keep 日本語\r\ncommand=\"echo " + publicKey + "\",restrict " + otherPublicKey + "\n# final"
	initial := strings.Replace(retained, "# final", "command=\"echo \\\"hi\\\"\",restrict "+publicKey+"\r\n# final", 1)
	authorized := writeFixture(testingContext, fixtureHome, ".ssh/authorized_keys", initial)
	server := filepath.Join(remote, "ssh")
	configuration := writeFixture(testingContext, server, "sshd_config", "AuthorizedKeysFile .ssh/authorized_keys\r\n")
	daemon := writeFixture(testingContext, remote, "sshd.ps1", "$global:LASTEXITCODE=0\n'usedns no'\n'pubkeyauthentication yes'\n'strictmodes yes'\n'authorizedkeyscommand none'\n'authorizedkeysfile .ssh/authorized_keys'\n")
	setup := "$paths=@('" + strings.Join([]string{fixtureHome, filepath.Dir(authorized), authorized, server, configuration}, "','") + "')\n" + `$ErrorActionPreference='Stop'
$identity=[Security.Principal.WindowsIdentity]::GetCurrent()
foreach ($path in $paths) {
    if ((Get-Item -LiteralPath $path).PSIsContainer) { $acl=New-Object Security.AccessControl.DirectorySecurity } else { $acl=New-Object Security.AccessControl.FileSecurity }
    $acl.SetAccessRuleProtection($true,$false)
    $acl.SetOwner($identity.User)
    $acl.AddAccessRule((New-Object Security.AccessControl.FileSystemAccessRule($identity.User,'FullControl','Allow')))
    Set-Acl -LiteralPath $path -AclObject $acl
}

(Get-Acl -LiteralPath $paths[2]).Sddl
`
	fixtureEnvironment := append(os.Environ(), "PSModulePath="+filepath.Join(os.Getenv("WINDIR"), "System32/WindowsPowerShell/v1.0/Modules"))
	setupCommand := exec.Command("powershell.exe", "-NoProfile", "-EncodedCommand", encodePowerShell(setup))
	setupCommand.Env = fixtureEnvironment
	beforeACL, err := setupCommand.Output()
	if err != nil {
		testingContext.Fatal("fixture ACL setup", err, string(beforeACL))
	}
	external.ssh = func(command app.Command) error {
		encoded := strings.Fields(command.Args[len(command.Args)-1])
		script := decodePowerShell(testingContext, encoded[len(encoded)-1])
		script = strings.ReplaceAll(script, "$HOME", "$fixtureHome")
		script = strings.ReplaceAll(script, "$env:ProgramData", "$fixtureProgramData")
		script = strings.ReplaceAll(script, `$sshd=Join-Path $env:WINDIR 'System32/OpenSSH/sshd.exe'`, "$sshd='"+daemon+"'")
		script = "$fixtureHome='" + fixtureHome + "'\n$fixtureProgramData='" + remote + "'\n" + script
		execution := exec.Command("powershell.exe", "-NoProfile", "-EncodedCommand", encodePowerShell(script))
		execution.Env = append(os.Environ(), "SSH_CONNECTION=192.0.2.1 12345 192.0.2.2 22", "PSModulePath="+filepath.Join(os.Getenv("WINDIR"), "System32/WindowsPowerShell/v1.0/Modules"))
		execution.Stdout, execution.Stderr = command.Stdout, command.Stderr
		return execution.Run()
	}
	output, err := runCLI(testingContext, home, "4\n1\np\nyes\nyes\n0\n", external)
	if err != nil || !strings.Contains(output, "Public key removed; local files preserved") {
		testingContext.Fatal("native Windows guarded revocation", err, output)
	}
	output, err = runCLI(testingContext, home, "4\n1\np\nyes\n0\n", external)
	after, readErr := os.ReadFile(authorized)
	aclCommand := exec.Command("powershell.exe", "-NoProfile", "-Command", "(Get-Acl -LiteralPath '"+authorized+"').Sddl")
	aclCommand.Env = fixtureEnvironment
	afterACL, aclErr := aclCommand.Output()
	document, loadErr := state.Load(home)
	if err != nil || readErr != nil || aclErr != nil || loadErr != nil || string(after) != retained || string(afterACL) != string(beforeACL) || document.Registrations[0].Status != "removed" || !strings.Contains(output, "Already absent") {
		testingContext.Fatal("native Windows byte/ACL preservation or read-only reconciliation", err, readErr, aclErr, loadErr, output)
	}
}
