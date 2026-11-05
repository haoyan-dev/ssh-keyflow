package ssh

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"strings"
	"unicode/utf16"
)

func Quote(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }

func RemoteCommand(system, script string) string {
	if system != "windows" {
		return "sh -c " + Quote(script)
	}
	script = "$ErrorActionPreference='Stop'\ntry {\n& {\n" + script + "\n}\n} catch {\n[Console]::Error.WriteLine($_.Exception.Message)\nexit 1\n}\n"
	if len(script) > 8000 {
		var compressed bytes.Buffer
		writer := gzip.NewWriter(&compressed)
		_, _ = writer.Write([]byte(script))
		_ = writer.Close()
		script = "$keyflowCompressed='" + base64.StdEncoding.EncodeToString(compressed.Bytes()) + "'\n$compressedStream=New-Object IO.MemoryStream(,[Convert]::FromBase64String($keyflowCompressed))\n$gzipStream=New-Object IO.Compression.GzipStream($compressedStream,[IO.Compression.CompressionMode]::Decompress)\n$scriptReader=New-Object IO.StreamReader($gzipStream,[Text.Encoding]::UTF8)\ntry { & ([ScriptBlock]::Create($scriptReader.ReadToEnd())) } finally { $scriptReader.Dispose(); $gzipStream.Dispose(); $compressedStream.Dispose() }\n"
	}
	units := utf16.Encode([]rune(script))
	data := make([]byte, len(units)*2)
	for index, unit := range units {
		data[index*2], data[index*2+1] = byte(unit), byte(unit>>8)
	}
	return "powershell.exe -NoLogo -NoProfile -EncodedCommand " + base64.StdEncoding.EncodeToString(data)
}

func RegistrationProbe(system, account, user string) (string, error) {
	if system == "linux" || system == "macos" {
		kernel := "Linux"
		if system == "macos" {
			kernel = "Darwin"
		}
		return strings.NewReplacer("EXPECTED_KERNEL", Quote(kernel), "EXPECTED_USER", Quote(user)).Replace(posixProbe), nil
	}
	if system != "windows" || (account != "standard" && account != "administrator") {
		return "", fmt.Errorf("confirm destination OS/account")
	}
	target := "Join-Path $HOME '.ssh/authorized_keys'"
	effective := ".ssh/authorized_keys"
	if account == "administrator" {
		target = "Join-Path $env:ProgramData 'ssh/administrators_authorized_keys'"
		effective = "__PROGRAMDATA__/ssh/administrators_authorized_keys"
	}
	return strings.NewReplacer("EXPECTED_USER", "'"+strings.ReplaceAll(user, "'", "''")+"'", "EXPECTED_ACCOUNT", "'"+account+"'", "EXPECTED_FILE", "'"+effective+"'", "TARGET_PATH", target).Replace(windowsProbe), nil
}

func RegistrationRead(system, account string, public []byte) (string, error) {
	plan, err := StandardRegistration(system, account, public)
	if err != nil {
		return "", err
	}
	if system != "windows" {
		start := strings.Index(plan.Script, "set +e\nawk")
		end := strings.Index(plan.Script, "match_status=$?")
		prelude := plan.Script[:strings.Index(plan.Script, "if [ ! -e \"$directory\"")]
		return prelude + "if [ ! -e \"$directory\" ] && [ ! -L \"$directory\" ]; then printf '%s\n' KEYFLOW_ABSENT; exit 0; fi\n[ -d \"$directory\" ] || exit 1\ncheck_path \"$directory\"\nif [ ! -e \"$authorized\" ] && [ ! -L \"$authorized\" ]; then printf '%s\n' KEYFLOW_ABSENT; exit 0; fi\n[ -f \"$authorized\" ] && [ -r \"$authorized\" ] || exit 1\ncheck_path \"$authorized\"\n" + plan.Script[start:end] + "result=$?\nset -e\ncase $result in 0) printf '%s\n' KEYFLOW_PRESENT;; 1) printf '%s\n' KEYFLOW_ABSENT;; *) exit 1;; esac\n", nil
	}
	patternStart := strings.Index(plan.Script, "    $parts =")
	patternEnd := strings.Index(plan.Script, "    if ($found)")
	prelude := "$RegistrationHome=$HOME\n$RegistrationServer=Join-Path $env:ProgramData 'ssh'\n" + plan.Script[strings.Index(plan.Script, "$ErrorActionPreference"):strings.Index(plan.Script, "Write-Output ('Registration target:")]
	helpers := plan.Script[strings.Index(plan.Script, "function Assert-PlainPath"):strings.Index(plan.Script, "function Set-NewACL")]
	return prelude + helpers + "Assert-PlainPath $directory\nif (Test-Path -LiteralPath $directory) { Assert-ACL $directory }\nAssert-PlainPath $authorized\nif (!(Test-Path -LiteralPath $authorized)) { Write-Output 'KEYFLOW_ABSENT'; exit 0 }\nAssert-ACL $authorized\n$utf8=New-Object Text.UTF8Encoding($false,$true)\n$text=$utf8.GetString([IO.File]::ReadAllBytes($authorized)).TrimStart([char]0xFEFF)\nif ($text.Contains([char]0)) { throw 'Unsupported encoding' }\n" + plan.Script[patternStart:patternEnd] + "if ($found) { Write-Output 'KEYFLOW_PRESENT' } else { Write-Output 'KEYFLOW_ABSENT' }\n", nil
}

const posixProbe = `set -eu
[ "$(uname -s)" = EXPECTED_KERNEL ] || { echo 'OS mismatch' >&2; exit 1; }
[ "$(id -un)" = EXPECTED_USER ] || { echo 'Login user mismatch' >&2; exit 1; }
[ -n "$SSH_CONNECTION" ] || exit 1
set -- $SSH_CONNECTION
[ "$#" -eq 4 ] || exit 1
client_address=$1
server_address=$3
server_port=$4
configuration=$(/usr/sbin/sshd -T -C "user=$(id -un),host=$client_address,addr=$client_address,laddr=$server_address,lport=$server_port") || { echo 'Cannot inspect effective sshd configuration; manual review required' >&2; exit 1; }
setting() { printf '%s\n' "$configuration" | sed -n "s/^$1 //p"; }
[ "$(setting usedns)" = no ] || { echo 'DNS dependent Match requires review' >&2; exit 1; }
[ "$(setting pubkeyauthentication)" = yes ] && [ "$(setting strictmodes)" = yes ] || exit 1
[ "$(setting authorizedkeyscommand)" = none ] || exit 1
case "$(setting authorizedkeysfile)" in '.ssh/authorized_keys'|'.ssh/authorized_keys .ssh/authorized_keys2'|"$HOME/.ssh/authorized_keys") ;; *) echo 'Nonstandard AuthorizedKeysFile; manual review required' >&2; exit 1;; esac
[ -d "$HOME" ] && [ ! -L "$HOME" ] || exit 1
printf '%s\n' "KEYFLOW_TARGET:$HOME/.ssh/authorized_keys"
`

const windowsProbe = `$ErrorActionPreference='Stop'
if ($env:OS -ne 'Windows_NT') { throw 'OS mismatch' }
$identity=[Security.Principal.WindowsIdentity]::GetCurrent()
$expectedUser=EXPECTED_USER
if ($identity.Name -ine $expectedUser -and ($identity.Name -split '\\')[-1] -ine $expectedUser) { throw 'Login user mismatch' }
$isAdministrator=@($identity.Groups | ForEach-Object { $_.Value }) -contains 'S-1-5-32-544'
if ((EXPECTED_ACCOUNT -eq 'administrator') -ne $isAdministrator) { throw 'Account type mismatch' }
$connection=$env:SSH_CONNECTION -split ' '
if ($connection.Count -ne 4) { throw 'Missing SSH connection context' }
$sshd=Join-Path $env:WINDIR 'System32/OpenSSH/sshd.exe'
$configuration=@(& $sshd -T -C ('user='+$identity.Name+',host='+$connection[0]+',addr='+$connection[0]+',laddr='+$connection[2]+',lport='+$connection[3]))
if ($LASTEXITCODE -ne 0) { throw 'Cannot inspect effective sshd configuration' }
function Setting([string]$Name) { return @($configuration | Where-Object { $_ -clike ($Name+' *') } | ForEach-Object { $_.Substring($Name.Length+1) }) }
if ((Setting 'usedns') -ne 'no' -or (Setting 'pubkeyauthentication') -ne 'yes' -or (Setting 'strictmodes') -ne 'yes' -or (Setting 'authorizedkeyscommand') -ne 'none') { throw 'Nonstandard sshd configuration' }
if ((Setting 'authorizedkeysfile') -ine EXPECTED_FILE) { throw 'Nonstandard AuthorizedKeysFile; manual review required' }
$target=TARGET_PATH
Write-Output ('KEYFLOW_TARGET:'+$target)
`
