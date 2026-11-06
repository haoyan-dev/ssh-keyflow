package app_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unicode/utf16"

	"ssh-keyflow/internal/app"
	"ssh-keyflow/internal/state"
)

func TestNativeOpenSSHParsesIsolatedSpacedPaths(testingContext *testing.T) {
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		testingContext.Skip("native OpenSSH unavailable")
	}
	home, external := prepareRegistration(testingContext, true)
	identityPath := filepath.Join(home, "key with spaces")
	spacedHome := filepath.Join(home, "isolated home with spaces")
	if err := os.MkdirAll(spacedHome, 0700); err != nil {
		testingContext.Fatal(err)
	}
	if err := os.Rename(filepath.Join(home, ".ssh"), filepath.Join(spacedHome, ".ssh")); err != nil {
		testingContext.Fatal(err)
	}
	home, external.home = spacedHome, spacedHome
	external.ssh = func(command app.Command) error {
		arguments := append([]string{"-G"}, command.Args[:len(command.Args)-1]...)
		execution := exec.Command(sshPath, arguments...)
		configuration, err := execution.CombinedOutput()
		if err != nil {
			testingContext.Fatalf("native SSH config parsing: %v %s", err, configuration)
		}
		for _, expected := range []string{"controlmaster false", "identityagent none", "certificatefile none", filepath.ToSlash(filepath.Join(home, ".ssh", "known_hosts")), filepath.ToSlash(filepath.Join(home, ".ssh", "known_hosts2"))} {
			if !strings.Contains(string(configuration), expected) {
				testingContext.Fatalf("native parser lost %q: %s", expected, configuration)
			}
		}
		for _, line := range strings.Split(string(configuration), "\n") {
			if strings.HasPrefix(line, "controlpath ") && strings.TrimSpace(line) != "controlpath none" {
				testingContext.Fatalf("native parser retained a control socket: %s", line)
			}
		}
		payload := command.Args[len(command.Args)-1]
		if strings.Contains(payload, "KEYFLOW_PROBE") {
			fmt.Fprintln(command.Stdout, "KEYFLOW_TARGET:/home/me/.ssh/authorized_keys")
		} else if strings.Contains(payload, "KEYFLOW_REGISTER") {
			fmt.Fprintln(command.Stdout, "KEYFLOW_REGISTERED")
		} else {
			if !strings.Contains(string(configuration), "identityfile "+identityPath) {
				testingContext.Fatalf("native parser lost target identity: %s", configuration)
			}
			fmt.Fprintln(command.Stdout, "KEYFLOW_VERIFIED")
		}
		return nil
	}
	output, err := runCLI(testingContext, home, "1\na\n1\nyes\np\nyes\nno\n0\n", external)
	if err != nil || !strings.Contains(output, "connection=verified") {
		testingContext.Fatal(err, output)
	}
}

func TestWindowsRegistrationRunsInIsolatedNTFSFixture(testingContext *testing.T) {
	if runtime.GOOS != "windows" {
		testingContext.Skip("requires native Windows PowerShell and NTFS")
	}
	identityCommand := exec.Command("powershell.exe", "-NoProfile", "-Command", `$identity=[Security.Principal.WindowsIdentity]::GetCurrent(); $identity.Name; @($identity.Groups | ForEach-Object { $_.Value }) -contains 'S-1-5-32-544'`)
	identityOutput, err := identityCommand.CombinedOutput()
	if err != nil {
		testingContext.Fatal(err, string(identityOutput))
	}
	identityLines := strings.Fields(string(identityOutput))
	if len(identityLines) != 2 {
		testingContext.Fatal("unexpected native identity", string(identityOutput))
	}
	if identityLines[1] != "False" {
		testingContext.Skip("native standard-account fixture requires a non-administrator identity; administrator scripts are parsed separately")
	}
	home, external := prepareRegistration(testingContext, false)
	if err := state.Update(home, func(document *state.State) error {
		document.Hosts[0].OS = "windows"
		document.Hosts[0].User = identityLines[0]
		document.Hosts[0].WindowsAccount = "standard"
		return nil
	}); err != nil {
		testingContext.Fatal(err)
	}
	remote := filepath.Join(fixtureTempDir(testingContext), "remote home with spaces")
	if err := os.MkdirAll(remote, 0700); err != nil {
		testingContext.Fatal(err)
	}
	daemon := writeFixture(testingContext, remote, "sshd.ps1", `if ($args.Count -ne 3 -or $args[0] -cne '-T' -or $args[1] -cne '-C') { throw 'invalid daemon query' }
$global:LASTEXITCODE=0
'usedns no'
'pubkeyauthentication yes'
'strictmodes yes'
'authorizedkeyscommand none'
'authorizedkeysfile .ssh/authorized_keys'
`)
	external.ssh = func(command app.Command) error {
		encoded := strings.Fields(command.Args[len(command.Args)-1])
		script := decodePowerShell(testingContext, encoded[len(encoded)-1])
		script = strings.ReplaceAll(script, "$HOME", "$fixtureHome")
		script = strings.ReplaceAll(script, `$sshd=Join-Path $env:WINDIR 'System32/OpenSSH/sshd.exe'`, "$sshd='"+strings.ReplaceAll(daemon, "'", "''")+"'")
		script = "$fixtureHome='" + strings.ReplaceAll(remote, "'", "''") + "'\n" + script
		execution := exec.Command("powershell.exe", "-NoProfile", "-EncodedCommand", encodePowerShell(script))
		execution.Env = append(os.Environ(), "SSH_CONNECTION=192.0.2.1 12345 192.0.2.2 22", "PSModulePath="+filepath.Join(os.Getenv("WINDIR"), "System32/WindowsPowerShell/v1.0/Modules"))
		execution.Stdout, execution.Stderr = command.Stdout, command.Stderr
		return execution.Run()
	}
	output, err := runCLI(testingContext, home, "1\na\n1\nyes\np\nyes\nno\n0\n", external)
	authorized := filepath.Join(remote, ".ssh/authorized_keys")
	written, readErr := os.ReadFile(authorized)
	if err != nil || readErr != nil || strings.Count(string(written), strings.Fields(publicKey)[1]) != 1 {
		testingContext.Fatalf("native Windows registration: %v %v %s", err, readErr, output)
	}
	original := append([]byte{239, 187, 191}, []byte("# preserved\r\n"+otherPublicKey+"\r\n"+string(written))...)
	if err := os.WriteFile(authorized, original, 0600); err != nil {
		testingContext.Fatal(err)
	}
	output, err = runCLI(testingContext, home, "1\na\n1\nyes\np\nyes\nno\n0\n", external)
	retained, _ := os.ReadFile(authorized)
	document, _ := state.Load(home)
	if err != nil || string(retained) != string(original) || document.Registrations[0].Status != "registered" || len(document.History) != 3 {
		testingContext.Fatalf("native Windows reconciliation: %v %+v %s", err, document.Registrations, output)
	}
	testingContext.Log("isolated standard-account NTFS ACL registration and read-only reconciliation passed")
}

func decodePowerShell(testingContext *testing.T, encoded string) string {
	testingContext.Helper()
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		testingContext.Fatal(err)
	}
	units := make([]uint16, len(data)/2)
	for index := range units {
		units[index] = binary.LittleEndian.Uint16(data[index*2:])
	}
	script := string(utf16.Decode(units))
	if strings.HasPrefix(script, "$keyflowCompressed='") {
		compressed, err := base64.StdEncoding.DecodeString(strings.Split(script, "'")[1])
		if err != nil {
			testingContext.Fatal(err)
		}
		reader, err := gzip.NewReader(bytes.NewReader(compressed))
		if err != nil {
			testingContext.Fatal(err)
		}
		defer reader.Close()
		contents, err := io.ReadAll(reader)
		if err != nil {
			testingContext.Fatal(err)
		}
		return string(contents)
	}
	return script
}

func TestWindowsAutomaticPayloadFailsClosed(testingContext *testing.T) {
	if runtime.GOOS != "windows" {
		testingContext.Skip("requires native Windows PowerShell")
	}
	for _, account := range []string{"standard", "administrator"} {
		testingContext.Run(account, func(testingContext *testing.T) {
			home, external := prepareRegistration(testingContext, false)
			if err := state.Update(home, func(document *state.State) error {
				document.Hosts[0].OS = "windows"
				document.Hosts[0].WindowsAccount = account
				return nil
			}); err != nil {
				testingContext.Fatal(err)
			}
			observedRegistration := false
			external.ssh = func(command app.Command) error {
				payload := command.Args[len(command.Args)-1]
				encoded := strings.Fields(payload)
				data, err := base64.StdEncoding.DecodeString(encoded[len(encoded)-1])
				if err != nil {
					testingContext.Fatal(err)
				}
				units := make([]uint16, len(data)/2)
				for index := range units {
					units[index] = binary.LittleEndian.Uint16(data[index*2:])
				}
				script := string(utf16.Decode(units))
				if strings.Contains(script, "KEYFLOW_REGISTER") {
					observedRegistration = true
					probeStart := strings.Index(script, "$observed=@(& {")
					if probeStart < 0 {
						testingContext.Fatal("missing guarded target probe")
					}
					bodyStart := strings.Index(script, "param([string]$RegistrationHome")
					if bodyStart < 0 {
						testingContext.Fatal("shared registration script not reused")
					}
					suffix := ""
					if wrapperEnd := strings.LastIndex(script, "\n}\n} catch {"); wrapperEnd >= 0 {
						suffix = script[wrapperEnd:]
					}
					script = script[:probeStart] + "Write-Error 'fixture write denied'\nWrite-Output 'KEYFLOW_REGISTERED'" + suffix
					execution := exec.Command("powershell.exe", "-NoProfile", "-EncodedCommand", encodePowerShell(script))
					execution.Stdout, execution.Stderr = command.Stdout, command.Stderr
					return execution.Run()
				}
				fmt.Fprintln(command.Stdout, "KEYFLOW_TARGET:C:\\isolated fixture\\authorized_keys")
				return nil
			}
			input := "1\na\n1\nyes\np\nyes\n"
			if account == "administrator" {
				input += "yes\n"
			}
			output, err := runCLI(testingContext, home, input+"0\n", external)
			document, _ := state.Load(home)
			if err != nil || !observedRegistration || document.Registrations[0].Status != "unknown" || document.Registrations[0].Stage != "unknown" {
				testingContext.Fatalf("Windows error reported success: %v %+v %s", err, document.Registrations, output)
			}
		})
	}
}

func encodePowerShell(script string) string {
	units := utf16.Encode([]rune(script))
	data := make([]byte, len(units)*2)
	for index, unit := range units {
		binary.LittleEndian.PutUint16(data[index*2:], unit)
	}
	return base64.StdEncoding.EncodeToString(data)
}

func TestRegistrationExecutesGuardedScriptsInIsolatedPOSIXFixture(testingContext *testing.T) {
	shell := "bash"
	if runtime.GOOS == "windows" {
		shell = "C:/Program Files/Git/bin/bash.exe"
	}
	if _, err := exec.LookPath(shell); err != nil {
		testingContext.Skip("Git Bash/POSIX shell unavailable")
	}
	for _, configuration := range []string{"standard", "custom", "mismatch", "unsafe-read"} {
		testingContext.Run(configuration, func(testingContext *testing.T) {
			home, external := prepareRegistration(testingContext, false)
			remote := fixtureTempDir(testingContext)
			binaryDirectory := filepath.Join(remote, "bin")
			system, kernel, mismatch := "linux", "Linux", "Darwin"
			if runtime.GOOS == "darwin" {
				system, kernel, mismatch = "macos", "Darwin", "Linux"
			}
			if err := state.Update(home, func(document *state.State) error { document.Hosts[0].OS = system; return nil }); err != nil {
				testingContext.Fatal(err)
			}
			writeFixture(testingContext, remote, "bin/uname", "#!/bin/sh\nprintf '%s\\n' "+kernel+"\n")
			writeFixture(testingContext, remote, "bin/id", "#!/bin/sh\nif [ \"$1\" = -un ]; then printf '%s\\n' me; else /usr/bin/id \"$@\"; fi\n")
			effective := "usedns no\npubkeyauthentication yes\nstrictmodes yes\nauthorizedkeyscommand none\nauthorizedkeysfile .ssh/authorized_keys .ssh/authorized_keys2\n"
			if configuration == "custom" {
				effective = strings.ReplaceAll(effective, ".ssh/authorized_keys .ssh/authorized_keys2", "/custom/authorized")
			}
			if configuration == "mismatch" {
				writeFixture(testingContext, remote, "bin/uname", "#!/bin/sh\nprintf '%s\\n' "+mismatch+"\n")
			}
			daemon := writeFixture(testingContext, remote, "bin/sshd", "#!/bin/sh\n[ \"$1\" = -T ] && [ \"$2\" = -C ] || exit 1\ncase \"$3\" in user=me,host=192.0.2.1,addr=192.0.2.1,laddr=192.0.2.2,lport=22) ;; *) exit 1;; esac\nprintf '%s' '"+effective+"'\n")
			initial := "# unrelated\r\n" + otherPublicKey + "\r\n"
			authorized := writeFixture(testingContext, remote, ".ssh/authorized_keys", initial)
			prepare := exec.Command(shell, "-c", `chmod 700 "$HOME" "$HOME/.ssh" "$HOME/bin" && chmod 600 "$HOME/.ssh/authorized_keys" && chmod 700 "$HOME/bin/"*`)
			prepare.Env = append(os.Environ(), "HOME="+filepath.ToSlash(remote))
			if output, err := prepare.CombinedOutput(); err != nil {
				testingContext.Fatalf("fixture preparation: %v %s", err, output)
			}
			external.ssh = func(command app.Command) error {
				payload := strings.ReplaceAll(command.Args[len(command.Args)-1], "/usr/sbin/sshd", filepath.ToSlash(daemon))
				execute := exec.CommandContext(context.Background(), shell, "-c", "PATH=\"$HOME/bin:/usr/bin:/bin\"; export PATH; "+payload)
				execute.Env = append(os.Environ(), "HOME="+filepath.ToSlash(remote), "PATH="+filepath.ToSlash(binaryDirectory)+":/usr/bin:/bin", "SSH_CONNECTION=192.0.2.1 12345 192.0.2.2 22")
				execute.Stdout, execute.Stderr = command.Stdout, command.Stderr
				return execute.Run()
			}
			output, err := runCLI(testingContext, home, "1\na\n1\nyes\np\nyes\nno\n0\n", external)
			after, readErr := os.ReadFile(authorized)
			document, loadErr := state.Load(home)
			if err != nil || readErr != nil || loadErr != nil {
				testingContext.Fatalf("fixture: %v %v %v %s", err, readErr, loadErr, output)
			}
			if configuration == "custom" || configuration == "mismatch" {
				if string(after) != initial || document.Registrations[0].Status != "unknown" {
					testingContext.Fatal("unsafe effective target or OS wrote remote file", output)
				}
				return
			}
			if !strings.HasPrefix(string(after), initial) || strings.Count(string(after), strings.Fields(publicKey)[1]) != 1 || document.Registrations[0].Status != "registered" {
				testingContext.Fatalf("script registration: %s %s", after, output)
			}
			if configuration == "unsafe-read" {
				writeFixture(testingContext, remote, "bin/stat", "#!/bin/sh\ncase \"$3\" in */.ssh) printf '0 777\\n';; *) /usr/bin/stat \"$@\";; esac\n")
				prepare := exec.Command(shell, "-c", `chmod 700 "$HOME/bin/stat"`)
				prepare.Env = append(os.Environ(), "HOME="+filepath.ToSlash(remote))
				if result, err := prepare.CombinedOutput(); err != nil {
					testingContext.Fatal(err, string(result))
				}
				output, err := runCLI(testingContext, home, "1\na\n1\nyes\np\nyes\n0\n", external)
				afterRead, _ := state.Load(home)
				if err != nil || len(afterRead.History) != len(document.History) || !strings.Contains(output, "read-only reconciliation failed") {
					testingContext.Fatalf("unsafe remote directory accepted: %v %s", err, output)
				}
			} else {
				runCLI(testingContext, home, "1\na\n1\nyes\np\nyes\nno\n0\n", external)
			}
			repeated, _ := os.ReadFile(authorized)
			if string(repeated) != string(after) {
				testingContext.Fatal("reconciliation appended a duplicate")
			}
		})
	}
}

func TestRegisterAnotherManagedKeyThroughMenuThree(testingContext *testing.T) {
	home, external := prepareRegistration(testingContext, false)
	second := writeFixture(testingContext, home, "second.pub", otherPublicKey)
	output, err := runCLI(testingContext, home, importInput(second, "second"), runner{})
	if err != nil {
		testingContext.Fatal(err, output)
	}
	output, err = runCLI(testingContext, home, "3\n1\n2\nyes\np\nyes\nno\n2\n0\n", external)
	document, loadErr := state.Load(home)
	if err != nil || loadErr != nil || len(document.Registrations) != 2 || document.Registrations[1].Status != "registered" || document.Registrations[0].Status != "unknown" || !strings.Contains(output, "registered") {
		testingContext.Fatalf("second registration: %v %v %+v %s", err, loadErr, document, output)
	}
}

func TestInteractiveSystemRunnerRejectsScriptInput(testingContext *testing.T) {
	var output strings.Builder
	err := (app.SystemRunner{}).Run(context.Background(), app.Command{Name: "ssh", Stdin: strings.NewReader("secret"), Stdout: &output, Stderr: &output, Interactive: true})
	if err == nil || !strings.Contains(fmt.Sprint(err), "real terminal") || output.Len() != 0 {
		testingContext.Fatal("nonterminal credentials were accepted", err)
	}
}

func nativePOSIXSystem() string {
	if runtime.GOOS == "darwin" {
		return "macos"
	}
	return "linux"
}
