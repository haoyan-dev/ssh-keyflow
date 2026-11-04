package ssh

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"regexp"
	"strings"
)

type RegistrationPlan struct {
	Enable string
	Target string
	Script string
	Shared bool
}

func StandardRegistration(system, windowsAccount string, public []byte) (RegistrationPlan, error) {
	fields := strings.Fields(string(public))
	if len(fields) < 2 || strings.ContainsAny(strings.TrimSpace(string(public)), "\r\n\x00") || !regexp.MustCompile(`^[A-Za-z0-9@._+-]+$`).MatchString(fields[0]) {
		return RegistrationPlan{}, fmt.Errorf("invalid public key")
	}
	blob, err := base64.StdEncoding.DecodeString(fields[1])
	if err != nil || len(blob) < 4 {
		return RegistrationPlan{}, fmt.Errorf("invalid public key blob")
	}
	length := int(binary.BigEndian.Uint32(blob[:4]))
	if length < 1 || length > len(blob)-4 || string(blob[4:4+length]) != fields[0] {
		return RegistrationPlan{}, fmt.Errorf("invalid public key algorithm")
	}
	key := fields[0] + " " + fields[1]
	plan := RegistrationPlan{Target: "~/.ssh/authorized_keys"}
	switch system {
	case "linux", "macos":
		statFormat := `stat -c '%u %a'`
		plan.Enable = "Linux: choose the commands for the installed distribution.\nDebian/Ubuntu:\nsudo apt install openssh-server\nsudo systemctl enable --now ssh\nFedora:\nsudo dnf install openssh-server\nsudo systemctl enable --now sshd\nConfirm firewall access and the login user; inspect sshd effective AuthorizedKeysFile before registration."
		if system == "macos" {
			statFormat = `stat -f '%u %Lp'`
			plan.Enable = "macOS: System Settings > General > Sharing > Remote Login. Review allowed users and permit the destination login user. With the required administrator/Full Disk Access permission:\nsudo systemsetup -setremotelogin on\nsudo systemsetup -getremotelogin\nConfirm the standard AuthorizedKeysFile configuration before registration."
		}
		plan.Script = strings.NewReplacer("KEY_TYPE", fields[0], "KEY_BLOB", fields[1], "STAT_FORMAT", statFormat).Replace(posixRegistration)
	case "windows":
		if windowsAccount != "standard" && windowsAccount != "administrator" {
			return RegistrationPlan{}, fmt.Errorf("confirm Windows standard or administrator account")
		}
		plan.Enable = "Windows: in elevated PowerShell, install and start OpenSSH Server:\nAdd-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0\nStart-Service sshd\nSet-Service -Name sshd -StartupType Automatic\nGet-NetFirewallRule -Name OpenSSH-Server-In-TCP\nIf missing, create the inbound TCP rule for the configured SSH port. Inspect %ProgramData%\\ssh\\sshd_config, Match rules and effective AuthorizedKeysFile; custom configurations require manual review. Run the registration script as the destination login user; administrator accounts require elevation. No automatic elevation or reset of existing ACLs."
		plan.Enable += "\nIf the default port 22 rule is absent:\nNew-NetFirewallRule -Name OpenSSH-Server-In-TCP -DisplayName 'OpenSSH Server' -Enabled True -Direction Inbound -Protocol TCP -Action Allow -LocalPort 22"
		target := `Join-Path $RegistrationHome '.ssh'`
		filename := "authorized_keys"
		account := "standard"
		if windowsAccount == "administrator" {
			target, filename, account, plan.Shared = `$RegistrationServer`, "administrators_authorized_keys", "administrator", true
			plan.Target = `%ProgramData%\ssh\administrators_authorized_keys (shared by administrator logins)`
		}
		plan.Script = strings.NewReplacer("PUBLIC_KEY", key, "TARGET_DIRECTORY", target, "TARGET_FILENAME", filename, "ACCOUNT_TYPE", account).Replace(windowsRegistration)
	default:
		return RegistrationPlan{}, fmt.Errorf("select a confirmed destination OS before preparing commands")
	}
	return plan, nil
}

const posixRegistration = `set -eu
umask 077
directory="$HOME/.ssh"
authorized="$directory/authorized_keys"
check_path() {
    [ ! -L "$1" ] || { printf '%s\n' 'Symlink requires manual review' >&2; exit 1; }
    metadata=$(STAT_FORMAT "$1")
    owner=${metadata% *}
    mode=${metadata#* }
    [ "$owner" = "$(id -u)" ] && [ "$((0$mode & 022))" -eq 0 ] || { printf '%s\n' 'Ownership or permissions require manual review; nothing reset' >&2; exit 1; }
}
[ -d "$HOME" ] && [ ! -L "$HOME" ] || exit 1
check_path "$HOME"
if [ ! -e "$directory" ] && [ ! -L "$directory" ]; then mkdir "$directory"; fi
[ -d "$directory" ] || exit 1
check_path "$directory"
lock="$directory/ssh-keyflow-registration.lock"
mkdir "$lock" || { printf '%s\n' 'Registration locked; review owner before retry' >&2; exit 1; }
trap 'rmdir "$lock"' EXIT
trap 'exit 1' HUP INT TERM
if [ ! -e "$authorized" ] && [ ! -L "$authorized" ]; then (set -C; : > "$authorized"); fi
[ -f "$authorized" ] || exit 1
check_path "$authorized"
set +e
awk -v key_type='KEY_TYPE' -v key_blob='KEY_BLOB' '
function token(line,    position, character, quoted, escaped, result) {
    result=""; quoted=0; escaped=0
    sub(/^[ \t]+/, "", line)
    for (position=1; position<=length(line); position++) {
        character=substr(line,position,1)
        if (!quoted && character ~ /[ \t\r]/) break
        result=result character
        if (escaped) escaped=0
        else if (character=="\\" && quoted) escaped=1
        else if (character=="\"") quoted=!quoted
    }
    remainder=substr(line,position)
    return result
}
/^[ \t]*#/ { next }
{
    first=token($0); second=token(remainder); third=token(remainder)
    if ((first==key_type && second==key_blob) || (second==key_type && third==key_blob)) found=1
}
END { exit !found }
' "$authorized"
match_status=$?
set -e
if [ "$match_status" -eq 0 ]; then
    printf '%s\n' 'Already registered; unchanged'
elif [ "$match_status" -eq 1 ]; then
    newline='\n'
    carriage=$(printf '\r')
    case "$(tail -c 2 "$authorized")" in *"$carriage") newline='\r\n';; esac
    if [ -s "$authorized" ] && [ -n "$(tail -c 1 "$authorized")" ]; then printf "$newline" >> "$authorized"; fi
    printf "KEY_TYPE KEY_BLOB$newline" >> "$authorized"
    printf '%s\n' 'Public key appended; verify connection separately'
else
    printf '%s\n' 'requires review: unable to read authorized_keys' >&2
    exit 1
fi
`

const windowsRegistration = `param([string]$RegistrationHome = $HOME, [string]$RegistrationServer = (Join-Path $env:ProgramData 'ssh'))
$ErrorActionPreference = 'Stop'
$publicKey = 'PUBLIC_KEY'
$accountType = 'ACCOUNT_TYPE'
$identity = [Security.Principal.WindowsIdentity]::GetCurrent()
$administratorSID = 'S-1-5-32-544'
$systemSID = 'S-1-5-18'
$isAdministrator = @($identity.Groups | ForEach-Object { $_.Value }) -contains $administratorSID
if (($accountType -eq 'administrator') -ne $isAdministrator) { throw 'Account type mismatch; review target login and Match rules' }
$directory = TARGET_DIRECTORY
$authorized = Join-Path $directory 'TARGET_FILENAME'
$allowed = @($administratorSID, $systemSID)
if ($accountType -eq 'standard') { $allowed += $identity.User.Value }
Write-Output ('Registration target: ' + $authorized)
if ($accountType -eq 'administrator' -and (Read-Host 'This shared file affects administrator logins. Type yes to approve this exact target') -cne 'yes') { throw 'Cancelled' }
function Assert-PlainPath([string]$Path) {
    $current = $Path
    while ($current) {
        if (Test-Path -LiteralPath $current) {
            $item = Get-Item -LiteralPath $current -Force
            if ($item.Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'Reparse point requires manual review' }
        }
        $parent = Split-Path -Path $current -Parent
        if ($parent -eq $current) { break }
        $current = $parent
    }
}
function Assert-ACL([string]$Path) {
    $acl = Get-Acl -LiteralPath $Path
    $owner = $acl.GetOwner([Security.Principal.SecurityIdentifier]).Value
    if ($allowed -notcontains $owner) { throw 'Owner requires manual review; ACL unchanged' }
    foreach ($rule in $acl.GetAccessRules($true, $true, [Security.Principal.SecurityIdentifier])) {
        if ($rule.AccessControlType -eq 'Allow' -and $allowed -notcontains $rule.IdentityReference.Value) { throw 'ACL requires manual review; no ACL reset' }
    }
}
function Set-NewACL([string]$Path, [bool]$IsDirectory) {
    if ($IsDirectory) { $acl = New-Object Security.AccessControl.DirectorySecurity } else { $acl = New-Object Security.AccessControl.FileSecurity }
    $acl.SetAccessRuleProtection($true, $false)
    if ($accountType -eq 'administrator') { $acl.SetOwner((New-Object Security.Principal.SecurityIdentifier($administratorSID))) } else { $acl.SetOwner($identity.User) }
    foreach ($principal in $allowed) {
        $sid = New-Object Security.Principal.SecurityIdentifier($principal)
        $rule = New-Object Security.AccessControl.FileSystemAccessRule($sid, 'FullControl', 'Allow')
        $acl.AddAccessRule($rule)
    }
    Set-Acl -LiteralPath $Path -AclObject $acl
}
Assert-PlainPath $directory
if (-not (Test-Path -LiteralPath $directory)) {
    if ($accountType -eq 'administrator') { throw 'Server directory missing; configure OpenSSH first' }
    [IO.Directory]::CreateDirectory($directory) | Out-Null
    Set-NewACL $directory $true
}
Assert-ACL $directory
Assert-PlainPath $authorized
$existed = Test-Path -LiteralPath $authorized
if ($existed) { Assert-ACL $authorized; $mode = [IO.FileMode]::Open } else { $mode = [IO.FileMode]::CreateNew }
$stream = [IO.File]::Open($authorized, $mode, [IO.FileAccess]::ReadWrite, [IO.FileShare]::None)
try {
    if (-not $existed) { Set-NewACL $authorized $false }
    Assert-ACL $authorized
    $bytes = New-Object byte[] $stream.Length
    $offset = 0
    while ($offset -lt $bytes.Length) {
        $count = $stream.Read($bytes, $offset, $bytes.Length - $offset)
        if ($count -eq 0) { throw 'Incomplete file read; unchanged' }
        $offset += $count
    }
    $utf8 = New-Object Text.UTF8Encoding($false, $true)
    $text = $utf8.GetString($bytes).TrimStart([char]0xFEFF)
    if ($text.Contains([char]0)) { throw 'Unsupported encoding; review file; unchanged' }
    $parts = $publicKey.Split(' ')
    $pattern = '^[\t ]*(?:(?:"(?:[^"\\]|\\.)*"|[^\s"])+[\t ]+)?' + [regex]::Escape($parts[0]) + '[\t ]+' + [regex]::Escape($parts[1]) + '(?:[\t ]|$)'
    $found = $false
    foreach ($line in ($text -split '\r?\n')) {
        if ($line -notmatch '^[\t ]*#' -and $line -cmatch $pattern) { $found = $true; break }
    }
    if ($found) { Write-Output 'Already registered; unchanged' } else {
        $newline = [string][char]10
        if ($text.Contains(([string][char]13 + [char]10))) { $newline = [string][char]13 + [char]10 }
        $prefix = ''
        if ($bytes.Length -gt 0 -and $bytes[$bytes.Length - 1] -ne 10) { $prefix = $newline }
        $addition = $utf8.GetBytes($prefix + $publicKey + $newline)
        $stream.Position = $stream.Length
        $stream.Write($addition, 0, $addition.Length)
        $stream.Flush($true)
        Write-Output 'Public key appended; verify connection separately'
    }
} finally { $stream.Dispose() }
`
