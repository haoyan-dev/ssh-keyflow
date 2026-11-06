package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"ssh-keyflow/internal/app"
)

type onboardingRunner struct {
	testingContext *testing.T
	home           string
	generate       func(app.Command, string) error
}

func (external onboardingRunner) LookPath(name string) (string, error) { return name, nil }
func (external onboardingRunner) Run(ctx context.Context, command app.Command) error {
	if command.Name != "ssh-keygen" {
		external.testingContext.Fatalf("unexpected process: %s", command.Name)
	}
	if len(command.Args) > 0 && command.Args[0] == "-y" {
		fmt.Fprintln(command.Stdout, publicKey)
		return nil
	}
	if len(command.Args) < 4 || command.Args[0] != "-t" || command.Args[1] != "ed25519" || !command.Interactive {
		external.testingContext.Fatalf("not native interactive Ed25519: %+v", command)
	}
	path := command.Args[len(command.Args)-1]
	data, err := os.ReadFile(filepath.Join(external.home, ".ssh/ssh-keyflow/state.json"))
	if err != nil || !strings.Contains(string(data), "\"stage\": \"pending\"") || !strings.Contains(string(data), "\"creations\"") {
		external.testingContext.Fatalf("creation intent absent before effect: %v %s", err, data)
	}
	if _, err := os.Stat(filepath.Join(external.home, ".ssh/ssh-keyflow/state.lock")); err != nil {
		external.testingContext.Fatal("generation is not locked", err)
	}
	if external.generate != nil {
		return external.generate(command, path)
	}
	if err := os.WriteFile(path, []byte("private fixture"), 0600); err != nil {
		return err
	}
	return os.WriteFile(path+".pub", []byte(publicKey), 0600)
}

func TestCreateEd25519PersistsIntentAndListsCreatedProvenance(testingContext *testing.T) {
	home := fixtureTempDir(testingContext)
	output, err := runCLI(testingContext, home, "5\nc\n日本語 My laptop/../key\nyes\n2\n0\n", onboardingRunner{testingContext: testingContext, home: home})
	if err != nil || !strings.Contains(output, "Created") || !strings.Contains(output, "origin=created") || !strings.Contains(output, "日本語 My laptop/../key") {
		testingContext.Fatalf("creation: %v\n%s", err, output)
	}
	data, err := os.ReadFile(filepath.Join(home, ".ssh/ssh-keyflow/state.json"))
	var document struct {
		Keys []struct {
			PrivatePath string `json:"private_path"`
			ImportedAt  string `json:"imported_at"`
			CreatedAt   string `json:"created_at"`
		}
	}
	if err != nil || json.Unmarshal(data, &document) != nil || len(document.Keys) != 1 {
		testingContext.Fatalf("state: %v %s", err, data)
	}
	key := document.Keys[0]
	if filepath.Dir(key.PrivatePath) != filepath.Join(home, ".ssh/ssh-keyflow/keys") || !strings.HasPrefix(filepath.Base(key.PrivatePath), "id_ed25519_") || key.CreatedAt == "" || key.ImportedAt != "" {
		testingContext.Fatalf("unsafe path or invalid provenance: %+v", key)
	}
}

func TestInterruptedCreationRecoversSamePathWithoutOverwriting(testingContext *testing.T) {
	for _, partial := range []string{"neither", "private", "pair", "public"} {
		testingContext.Run(partial, func(testingContext *testing.T) {
			home := fixtureTempDir(testingContext)
			originalPath := ""
			interrupted := onboardingRunner{testingContext: testingContext, home: home, generate: func(command app.Command, path string) error {
				originalPath = path
				if partial == "private" || partial == "pair" {
					writeFixture(testingContext, home, strings.TrimPrefix(path, home), "private fixture")
				}
				if partial == "public" || partial == "pair" {
					if err := os.WriteFile(path+".pub", []byte(publicKey), 0600); err != nil {
						testingContext.Fatal(err)
					}
				}
				return fmt.Errorf("simulated interruption")
			}}
			output, err := runCLI(testingContext, home, "5\nc\nresume me\nyes\n0\n", interrupted)
			if err != nil || !strings.Contains(output, "interrupted") {
				testingContext.Fatalf("interruption: %v %s", err, output)
			}
			recovery := onboardingRunner{testingContext: testingContext, home: home, generate: func(command app.Command, path string) error {
				if partial != "neither" || path != originalPath {
					testingContext.Fatal("attempted overwrite or allocated another pair", path)
				}
				if err := os.WriteFile(path, []byte("private fixture"), 0600); err != nil {
					return err
				}
				return os.WriteFile(path+".pub", []byte(publicKey), 0600)
			}}
			output, err = runCLI(testingContext, home, "5\nr\n1\nyes\n2\n0\n", recovery)
			if partial == "public" {
				if err != nil || !strings.Contains(output, "requires review") || strings.Contains(output, "origin=created") {
					testingContext.Fatalf("public-only recovery: %v %s", err, output)
				}
				if _, err := os.Stat(originalPath); !os.IsNotExist(err) {
					testingContext.Fatal("public-only recovery created a private key")
				}
			} else if err != nil || !strings.Contains(output, "origin=created") || !strings.Contains(output, originalPath) {
				testingContext.Fatalf("recovery: %v %s", err, output)
			}
		})
	}
}

func TestDestinationReusesImportedPublicKeyAndPersistsManualHistory(testingContext *testing.T) {
	home := fixtureTempDir(testingContext)
	publicPath := writeFixture(testingContext, home, ".ssh/existing.pub", publicKey)
	if output, err := runCLI(testingContext, home, importInput(publicPath, "共有鍵"), runner{}); err != nil || !strings.Contains(output, "Imported") {
		testingContext.Fatalf("import: %v %s", err, output)
	}
	before, _ := os.ReadFile(publicPath)
	output, err := runCLI(testingContext, home, "1\nn\n自分のVPS\nserver.tailnet.ts.net\n利用者\n2222\nunknown\nlinux\n1\nyes\n0\n2\n0\n", onboardingRunner{testingContext: testingContext, home: home, generate: func(command app.Command, path string) error {
		testingContext.Fatal("reuse generated another key")
		return nil
	}})
	for _, expected := range []string{"自分のVPS", "利用者@server.tailnet.ts.net:2222", "origin=imported", "status=unknown", "connection=unverified", "authorized_keys", "openssh-server", "ssh-ed25519 AAAA", "recorded history"} {
		if err != nil || !strings.Contains(output, expected) {
			testingContext.Fatalf("missing %q: %v\n%s", expected, err, output)
		}
	}
	after, _ := os.ReadFile(publicPath)
	if string(before) != string(after) {
		testingContext.Fatal("reuse changed imported file")
	}
	output, err = runCLI(testingContext, home, "1\nr\n1\nyes\n0\n", runner{})
	if err != nil || !strings.Contains(output, "authorized_keys") {
		testingContext.Fatalf("manual recovery: %v %s", err, output)
	}
}

func TestManualGuidanceIsCopyableForEveryDestinationOS(testingContext *testing.T) {
	for _, target := range []struct {
		system, account string
		expected        []string
	}{
		{"linux", "", []string{"sudo apt install openssh-server", "systemctl", "authorized_keys", "awk", "umask 077", "stat -c", "ssh-keyflow-registration.lock"}},
		{"macos", "", []string{"Remote Login", "allowed users", "systemsetup -setremotelogin on", "authorized_keys", "stat -f"}},
		{"windows", "standard", []string{"Add-WindowsCapability", "Start-Service sshd", "Set-Service", "$HOME", "authorized_keys", "UTF8Encoding", "Get-Acl", "S-1-5-32-544", "GetString"}},
		{"windows", "administrator", []string{"administrators_authorized_keys", "$env:ProgramData", "shared", "Read-Host", "S-1-5-18", "Get-Acl"}},
	} {
		testingContext.Run(target.system+target.account, func(testingContext *testing.T) {
			home := fixtureTempDir(testingContext)
			publicPath := writeFixture(testingContext, home, ".ssh/original.pub", publicKey+" $(danger); 'comment'")
			if output, err := runCLI(testingContext, home, importInput(publicPath, "existing"), runner{}); err != nil || !strings.Contains(output, "Imported") {
				testingContext.Fatalf("import: %v %s", err, output)
			}
			accountInput := ""
			if target.account != "" {
				accountInput = target.account + "\n"
			}
			output, err := runCLI(testingContext, home, "1\nn\nhost\n100.64.0.1\nme\n\n"+target.system+"\n"+accountInput+"1\nyes\n0\n", runner{})
			for _, expected := range target.expected {
				if err != nil || !strings.Contains(output, expected) {
					testingContext.Fatalf("missing %q: %v %s", expected, err, output)
				}
			}
			if strings.Contains(output, "$(danger)") {
				testingContext.Fatal("public comment leaked into executable guidance")
			}
		})
	}
}

func TestAbandonedOnboardingCannotClaimStandaloneCreation(testingContext *testing.T) {
	home := fixtureTempDir(testingContext)
	output, err := runCLI(testingContext, home, "1\nn\nabandoned\nexample.com\nme\n\nlinux\nn\nyes\nno\n5\nc\nindependent\nyes\n2\n0\n", onboardingRunner{testingContext: testingContext, home: home})
	if err != nil || !strings.Contains(output, "Created independent") || strings.Contains(output, "managed host abandoned") || strings.Contains(output, "registration registration-") {
		testingContext.Fatalf("unrelated onboarding claimed standalone key: %v %s", err, output)
	}
}

func TestDestinationAliasCannotShadowRecursiveUnmanagedHost(testingContext *testing.T) {
	home := fixtureTempDir(testingContext)
	config := "Include conf/*.conf\nMatch exec never-execute-this\n"
	writeFixture(testingContext, home, ".ssh/config", config)
	writeFixture(testingContext, home, ".ssh/conf/host.conf", "Host Existing backup-*\n HostName original\n")
	for _, alias := range []string{"existing", "EXISTING"} {
		output, err := runCLI(testingContext, home, "1\nn\n"+alias+"\nexample.test\nme\n\nlinux\n0\n", runner{})
		if err != nil || !strings.Contains(output, "existing Host") {
			testingContext.Fatalf("alias collision accepted: %v %s", err, output)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh/ssh-keyflow/state.json")); !os.IsNotExist(err) {
		testingContext.Fatal("alias conflict changed state")
	}
	data, _ := os.ReadFile(filepath.Join(home, ".ssh/config"))
	if string(data) != config {
		testingContext.Fatal("unmanaged configuration rewritten")
	}
}

func TestWildcardHostDefaultsPreserveManualOnboarding(testingContext *testing.T) {
	for _, pattern := range []string{"*", "backup-*"} {
		testingContext.Run(pattern, func(testingContext *testing.T) {
			home := fixtureTempDir(testingContext)
			config := "Host " + pattern + "\n  ForwardAgent no\n"
			configPath := writeFixture(testingContext, home, ".ssh/config", config)
			publicPath := writeFixture(testingContext, home, ".ssh/source.pub", publicKey)
			runCLI(testingContext, home, importInput(publicPath, "reuse"), runner{})
			output, err := runCLI(testingContext, home, "1\nn\nbackup-new\n100.64.0.1\nme\n\nlinux\n1\nyes\n0\n", runner{})
			data, _ := os.ReadFile(configPath)
			if err != nil || string(data) != config || !strings.Contains(output, "Wildcard Host") || !strings.Contains(output, "review") {
				testingContext.Fatalf("wildcard handling: %v %s", err, output)
			}
			data, _ = os.ReadFile(filepath.Join(home, ".ssh/ssh-keyflow/state.json"))
			var document struct {
				Hosts         []json.RawMessage
				Registrations []json.RawMessage
			}
			if json.Unmarshal(data, &document) != nil || len(document.Hosts) != 1 || len(document.Registrations) != 1 {
				testingContext.Fatalf("wildcard blocked manual onboarding: %s", data)
			}
		})
	}
}

func TestNewDestinationResumesSameKeyAfterInterruption(testingContext *testing.T) {
	home := fixtureTempDir(testingContext)
	keyPath := ""
	external := onboardingRunner{testingContext: testingContext, home: home, generate: func(command app.Command, path string) error {
		keyPath = path
		data, err := os.ReadFile(filepath.Join(home, ".ssh/ssh-keyflow/state.json"))
		if err != nil || !strings.Contains(string(data), "\"stage\": \"key_pending\"") || !strings.Contains(string(data), "my-vps") {
			testingContext.Fatalf("destination intent not saved: %v %s", err, data)
		}
		if err := os.WriteFile(path, []byte("private fixture"), 0600); err != nil {
			return err
		}
		return fmt.Errorf("interrupted")
	}}
	output, err := runCLI(testingContext, home, "1\nn\nmy-vps\n2001:db8::1\nme\n\nmacos\nn\nyes\nyes\n0\n", external)
	if err != nil || !strings.Contains(output, "interrupted") {
		testingContext.Fatalf("interruption: %v %s", err, output)
	}
	output, err = runCLI(testingContext, home, "1\nr\n1\nyes\n0\n2\n0\n", onboardingRunner{testingContext: testingContext, home: home, generate: func(command app.Command, path string) error {
		testingContext.Fatal("recovery generated another pair")
		return nil
	}})
	if err != nil || !strings.Contains(output, keyPath) || !strings.Contains(output, "Remote Login") || !strings.Contains(output, "status=unknown") {
		testingContext.Fatalf("resume: %v %s", err, output)
	}
	data, _ := os.ReadFile(filepath.Join(home, ".ssh/ssh-keyflow/state.json"))
	var document struct {
		Hosts         []any
		Keys          []any
		Registrations []any
		Creations     []any
		Onboardings   []any
	}
	if json.Unmarshal(data, &document) != nil || len(document.Hosts) != 1 || len(document.Keys) != 1 || len(document.Registrations) != 1 || len(document.Creations) != 1 || len(document.Onboardings) != 1 {
		testingContext.Fatalf("duplicated recovery state: %s", data)
	}
}

type changingInput struct {
	reader    io.Reader
	remaining int
	change    func()
}

func TestCreationDoesNotClaimFilesAppearingDuringConfirmation(testingContext *testing.T) {
	home := fixtureTempDir(testingContext)
	input := "5\nc\ncollision\nyes\n0\n"
	var output strings.Builder
	privatePath := ""
	err := app.Run(context.Background(), app.Options{Home: home, Input: &changingInput{reader: strings.NewReader(input), remaining: len(input) - 2, change: func() {
		_, suffix, _ := strings.Cut(output.String(), "\nPrivate: ")
		privatePath, _, _ = strings.Cut(suffix, "\n")
		if !strings.HasPrefix(privatePath, filepath.Join(home, ".ssh/ssh-keyflow/keys")) {
			testingContext.Fatal("unsafe fixture path", privatePath)
		}
		if err := os.WriteFile(privatePath, []byte("unrelated private bytes"), 0600); err != nil {
			testingContext.Fatal(err)
		}
	}}, Output: &output, Runner: onboardingRunner{testingContext: testingContext, home: home}})
	data, _ := os.ReadFile(privatePath)
	if err != nil || !strings.Contains(output.String(), "requires review") || strings.Contains(output.String(), "Created collision") || string(data) != "unrelated private bytes" {
		testingContext.Fatalf("creation claimed unrelated files: %v %s", err, output.String())
	}
	if _, err := os.Stat(privatePath + ".pub"); !os.IsNotExist(err) {
		testingContext.Fatal("creation modified unrelated pair")
	}
}

func TestOnboardingRejectsInvalidDestinationFields(testingContext *testing.T) {
	for _, fields := range []string{"-option\nexample.test\nme\n\nlinux", "host\n$(command)\nme\n\nlinux", "host\nexample.test\nme;command\n\nlinux", "host\nexample.test\nme\n65536\nlinux", "host\nexample.test\nme\n0\nlinux", "host\nexample.test\nme\n22\nunsupported"} {
		home := fixtureTempDir(testingContext)
		output, err := runCLI(testingContext, home, "1\nn\n"+fields+"\n0\n", runner{})
		if err != nil || !strings.Contains(output, "invalid destination") {
			testingContext.Fatalf("invalid destination accepted: %v %s", err, output)
		}
		if _, err := os.Stat(filepath.Join(home, ".ssh/ssh-keyflow/state.json")); !os.IsNotExist(err) {
			testingContext.Fatal("invalid destination persisted")
		}
	}
}

func TestCreationAndOnboardingHistoryRejectsInconsistentState(testingContext *testing.T) {
	for _, broken := range []string{"provenance", "creation-result", "creation-stage", "onboarding-reference", "manual-history"} {
		testingContext.Run(broken, func(testingContext *testing.T) {
			home := fixtureTempDir(testingContext)
			output, err := runCLI(testingContext, home, "1\nn\nhost\nexample.test\nme\n\nlinux\nn\nyes\nyes\n0\n", onboardingRunner{testingContext: testingContext, home: home})
			if err != nil || !strings.Contains(output, "Intent saved") {
				testingContext.Fatalf("history fixture: %v %s", err, output)
			}
			statePath := filepath.Join(home, ".ssh/ssh-keyflow/state.json")
			data, _ := os.ReadFile(statePath)
			var document map[string]any
			if err := json.Unmarshal(data, &document); err != nil {
				testingContext.Fatal(err)
			}
			switch broken {
			case "provenance":
				document["keys"].([]any)[0].(map[string]any)["imported_at"] = "2026-11-04T00:00:00Z"
			case "creation-result":
				document["creations"].([]any)[0].(map[string]any)["key_id"] = "missing"
			case "creation-stage":
				document["creations"].([]any)[0].(map[string]any)["stage"] = "invented"
			case "onboarding-reference":
				document["onboardings"].([]any)[0].(map[string]any)["creation_id"] = "missing"
			case "manual-history":
				document["registrations"] = []any{}
			}
			data, err = json.Marshal(document)
			if err != nil {
				testingContext.Fatal(err)
			}
			if err := os.WriteFile(statePath, data, 0600); err != nil {
				testingContext.Fatal(err)
			}
			if output, err := runCLI(testingContext, home, "0\n", runner{}); err == nil {
				testingContext.Fatal("inconsistent history accepted", output)
			}
			after, _ := os.ReadFile(statePath)
			if !bytes.Equal(data, after) {
				testingContext.Fatal("inconsistent history rewritten")
			}
		})
	}
}

func (input *changingInput) Read(buffer []byte) (int, error) {
	count, err := input.reader.Read(buffer)
	input.remaining -= count
	if input.remaining <= 0 && input.change != nil {
		input.change()
		input.change = nil
	}
	return count, err
}

func TestDestinationRechecksImportedKeyAfterConfirmation(testingContext *testing.T) {
	home := fixtureTempDir(testingContext)
	publicPath := writeFixture(testingContext, home, ".ssh/original.pub", publicKey)
	runCLI(testingContext, home, importInput(publicPath, "original"), runner{})
	statePath := filepath.Join(home, ".ssh/ssh-keyflow/state.json")
	before, _ := os.ReadFile(statePath)
	input := "1\nn\nhost\nexample.com\nme\n\nlinux\n1\nyes\n0\n"
	var output strings.Builder
	err := app.Run(context.Background(), app.Options{Home: home, Input: &changingInput{reader: strings.NewReader(input), remaining: len(input) - 2, change: func() { writeFixture(testingContext, home, ".ssh/original.pub", otherPublicKey) }}, Output: &output, Runner: runner{}})
	after, _ := os.ReadFile(statePath)
	if err != nil || !strings.Contains(output.String(), "requires review") || string(before) != string(after) {
		testingContext.Fatalf("changed import committed a destination: %v %s", err, output.String())
	}
}

func TestCopyablePOSIXRegistrationPreservesBytesAndMatchesKeyOptions(testingContext *testing.T) {
	shell := "sh"
	if runtime.GOOS == "windows" {
		shell = "C:/Program Files/Git/bin/bash.exe"
	}
	if _, err := exec.LookPath(shell); err != nil {
		testingContext.Skip("POSIX shell unavailable")
	}
	for _, initial := range []string{"# retain me\r\n" + otherPublicKey + "\r\n", "# no trailing newline", "command=\"printf 'hello world'\",restrict " + publicKey + "\n", "# " + publicKey + "\n"} {
		testingContext.Run(initial, func(testingContext *testing.T) {
			home := fixtureTempDir(testingContext)
			publicPath := writeFixture(testingContext, home, ".ssh/source.pub", publicKey)
			runCLI(testingContext, home, importInput(publicPath, "existing"), runner{})
			output, err := runCLI(testingContext, home, "1\nn\nhost\nexample.test\nme\n\n"+nativePOSIXSystem()+"\n1\nyes\n0\n", runner{})
			if err != nil {
				testingContext.Fatal(err)
			}
			_, script, found := strings.Cut(output, "BEGIN REGISTRATION SCRIPT\n")
			script, _, _ = strings.Cut(script, "END REGISTRATION SCRIPT")
			if !found {
				testingContext.Fatal("no copyable script", output)
			}
			fixture := fixtureTempDir(testingContext)
			authorized := writeFixture(testingContext, fixture, ".ssh/authorized_keys", initial)
			scriptPath := writeFixture(testingContext, fixture, "register.sh", script)
			for attempt := 0; attempt < 2; attempt++ {
				command := exec.Command(shell, "-c", `chmod 700 "$HOME" "$HOME/.ssh" && chmod 600 "$HOME/.ssh/authorized_keys" && sh "$1"`, "fixture", filepath.ToSlash(scriptPath))
				command.Env = append(os.Environ(), "HOME="+filepath.ToSlash(fixture))
				if result, err := command.CombinedOutput(); err != nil {
					testingContext.Fatalf("generated script: %v %s", err, result)
				}
			}
			after, err := os.ReadFile(authorized)
			if err != nil || !strings.HasPrefix(string(after), initial) {
				testingContext.Fatalf("existing bytes altered: %v %q", err, after)
			}
			if strings.HasPrefix(initial, "command=") {
				if string(after) != initial {
					testingContext.Fatal("option-bearing identical key duplicated")
				}
			} else if strings.Count(string(after), strings.Fields(publicKey)[1]) != strings.Count(initial, strings.Fields(publicKey)[1])+1 {
				testingContext.Fatalf("registration not idempotent: %q", after)
			}
		})
	}
}

func TestCopyableWindowsRegistrationScripts(testingContext *testing.T) {
	if runtime.GOOS != "windows" {
		testingContext.Skip("Windows PowerShell fixture")
	}
	for _, account := range []string{"standard", "administrator"} {
		testingContext.Run(account, func(testingContext *testing.T) {
			home := fixtureTempDir(testingContext)
			publicPath := writeFixture(testingContext, home, ".ssh/source.pub", publicKey)
			runCLI(testingContext, home, importInput(publicPath, "existing"), runner{})
			output, err := runCLI(testingContext, home, "1\nn\nhost\nexample.test\nme\n\nwindows\n"+account+"\n1\nyes\n0\n", runner{})
			if err != nil {
				testingContext.Fatal(err)
			}
			_, script, found := strings.Cut(output, "BEGIN REGISTRATION SCRIPT\n")
			script, _, _ = strings.Cut(script, "END REGISTRATION SCRIPT")
			if !found {
				testingContext.Fatal("no copyable script", output)
			}
			fixture := fixtureTempDir(testingContext)
			scriptPath := writeFixture(testingContext, fixture, "register.ps1", script)
			helper := writeFixture(testingContext, fixture, "fixture.ps1", windowsRegistrationFixture)
			command := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", helper, scriptPath, fixture, account)
			command.Env = append(os.Environ(), "PSModulePath="+filepath.Join(os.Getenv("WINDIR"), "System32/WindowsPowerShell/v1.0/Modules"))
			command.Stdin = strings.NewReader("yes\nyes\nyes\n")
			result, err := command.CombinedOutput()
			if err != nil {
				testingContext.Fatalf("generated Windows script: %v %s", err, result)
			}
			testingContext.Log(string(result))
		})
	}
}

const windowsRegistrationFixture = `param($ScriptPath, $Fixture, $Account)
$ErrorActionPreference = 'Stop'
$parseTokens = $null
$parseErrors = $null
$null = [Management.Automation.Language.Parser]::ParseFile($ScriptPath, [ref]$parseTokens, [ref]$parseErrors)
if ($parseErrors.Count) { throw ($parseErrors | Out-String) }
Write-Output 'PowerShell 5.1 syntax passed'
$identity = [Security.Principal.WindowsIdentity]::GetCurrent()
$adminGroup = @($identity.Groups | ForEach-Object { $_.Value }) -contains 'S-1-5-32-544'
$elevated = (New-Object Security.Principal.WindowsPrincipal($identity)).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (($Account -eq 'administrator') -ne $adminGroup -or ($adminGroup -and !$elevated)) { Write-Output 'Native ACL execution requires matching account/elevation; syntax checked'; exit 0 }
$fixtureHome = Join-Path $Fixture 'home'
$fixtureServer = Join-Path $Fixture 'server'
$null = New-Item -ItemType Directory -Path $fixtureHome
$null = New-Item -ItemType Directory -Path $fixtureServer
& $ScriptPath -RegistrationHome $fixtureHome -RegistrationServer $fixtureServer
if ($Account -eq 'administrator') { $authorized = Join-Path $fixtureServer 'administrators_authorized_keys' } else { $authorized = Join-Path $fixtureHome '.ssh/authorized_keys' }
$original = [IO.File]::ReadAllText($authorized)
$expected = [byte[]](239,187,191) + [Text.Encoding]::UTF8.GetBytes(('# preserved' + [char]13 + [char]10 + $original))
[IO.File]::WriteAllBytes($authorized, $expected)
& $ScriptPath -RegistrationHome $fixtureHome -RegistrationServer $fixtureServer
& $ScriptPath -RegistrationHome $fixtureHome -RegistrationServer $fixtureServer
if ([Convert]::ToBase64String([IO.File]::ReadAllBytes($authorized)) -cne [Convert]::ToBase64String($expected)) { throw 'Existing BOM, CRLF or key bytes changed' }
Write-Output 'Isolated NTFS registration and idempotence passed'
`
