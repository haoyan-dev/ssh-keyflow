package app

import (
	"fmt"
	"regexp"
	"strings"

	"ssh-keyflow/internal/state"
)

const powershellReadUTF8 = `    $utf8 = [Text.UTF8Encoding]::new($false, $true)
    $existing = ''
    if (Test-Path -LiteralPath $file) {
        $existing = $utf8.GetString([IO.File]::ReadAllBytes($file)).TrimStart([char]0xFEFF)
        if ($existing.IndexOf([char]0) -ge 0) { throw 'Expected UTF-8 text; review this file encoding first.' }
    }
`

func posixPlacement(host state.Host, public string) GuideStep {
	return GuideStep{
		Title:        fmt.Sprintf("DESTINATION %s — %s POSIX shell", host.Hostname, host.OS),
		Instructions: fmt.Sprintf("Sign in as %s on the destination. Paste this entire script into sh, bash or zsh. It appends the public key to ~/.ssh/authorized_keys and sets permissions.", host.User),
		Script: fmt.Sprintf(`(
set -eu
if [ "$(id -un)" != '%s' ]; then
    printf 'Sign in as the destination login user first.\n' >&2
    exit 1
fi
umask 077
dir="$HOME/.ssh"
file="$dir/authorized_keys"
key='%s'
[ ! -L "$dir" ] && [ ! -L "$file" ] || { printf 'Refusing a symlink.\n' >&2; exit 1; }
[ ! -e "$file" ] || [ -f "$file" ] || { printf 'Expected a regular file.\n' >&2; exit 1; }
mkdir -p "$dir"
chmod 700 "$dir"
touch "$file"
chmod 600 "$file"
if grep -F -- "$key" "$file" >/dev/null; then
    printf 'Public key is already present.\n'
else
    result=$?
    [ "$result" -eq 1 ] || exit "$result"
    printf '\n%%s\n' "$key" >> "$file"
    printf 'Public key added.\n'
fi
)
`, host.User, public),
	}
}

func windowsPlacement(host state.Host, public string) GuideStep {
	admin := host.WindowsAccount == "administrator"
	instructions := fmt.Sprintf("Sign in as %s on the destination and use PowerShell. Target: %%USERPROFILE%%\\.ssh\\authorized_keys. The script restricts access to this user, SYSTEM and Administrators.", host.User)
	location := `$dir = Join-Path $env:USERPROFILE '.ssh'
$file = Join-Path $dir 'authorized_keys'`
	identityCheck := fmt.Sprintf(`$identity = [Security.Principal.WindowsIdentity]::GetCurrent()
if ($env:USERNAME -ine '%s' -and $identity.Name -ine '%s') {
    throw 'Sign in as the destination login user first.'
}`, host.User, host.User)
	principals := "@($identity.User.Value, 'S-1-5-18', 'S-1-5-32-544')"
	directoryACL := `    $directoryAcl = Get-Acl -LiteralPath $dir
    $directoryAcl.SetAccessRuleProtection($true, $false)
    foreach ($rule in @($directoryAcl.Access)) { $directoryAcl.RemoveAccessRuleSpecific($rule) }
    foreach ($sid in $principals) {
        $rule = [Security.AccessControl.FileSystemAccessRule]::new([Security.Principal.SecurityIdentifier]::new($sid), 'FullControl', 'ContainerInherit, ObjectInherit', 'None', 'Allow')
        $directoryAcl.AddAccessRule($rule)
    }
    if ($PSVersionTable.PSEdition -eq 'Core') {
        [IO.FileSystemAclExtensions]::SetAccessControl([IO.DirectoryInfo]::new($dir), $directoryAcl)
    } else {
        [IO.Directory]::SetAccessControl($dir, $directoryAcl)
    }
`
	if admin {
		instructions = "Open PowerShell as Administrator on the destination. Target: %ProgramData%\\ssh\\administrators_authorized_keys, shared by administrator logins. The script restricts the file to SYSTEM and Administrators."
		location = `$dir = Join-Path $env:ProgramData 'ssh'
$file = Join-Path $dir 'administrators_authorized_keys'`
		identityCheck = `$identity = [Security.Principal.WindowsIdentity]::GetCurrent()
if (-not ([Security.Principal.WindowsPrincipal]::new($identity)).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    throw 'Open PowerShell as Administrator first.'
}`
		principals = "@('S-1-5-18', 'S-1-5-32-544')"
		directoryACL = ""
	}
	return GuideStep{
		Title:        "DESTINATION " + host.Hostname + " — Windows PowerShell (" + host.WindowsAccount + ")",
		Instructions: instructions,
		Script: fmt.Sprintf(`& {
    $ErrorActionPreference = 'Stop'
%s
%s
    $key = '%s'
    foreach ($path in @($dir, $file)) {
        if ((Test-Path -LiteralPath $path) -and ((Get-Item -Force -LiteralPath $path).Attributes -band [IO.FileAttributes]::ReparsePoint)) {
            throw 'Refusing a reparse point.'
        }
    }
%s    [IO.Directory]::CreateDirectory($dir) | Out-Null
    $principals = %s
%s    [IO.File]::Open($file, [IO.FileMode]::OpenOrCreate, [IO.FileAccess]::Write, [IO.FileShare]::None).Dispose()
    $acl = Get-Acl -LiteralPath $file
    $acl.SetAccessRuleProtection($true, $false)
    foreach ($rule in @($acl.Access)) { $acl.RemoveAccessRuleSpecific($rule) }
    foreach ($sid in $principals) {
        $acl.AddAccessRule([Security.AccessControl.FileSystemAccessRule]::new([Security.Principal.SecurityIdentifier]::new($sid), 'FullControl', 'Allow'))
    }
    if ($PSVersionTable.PSEdition -eq 'Core') {
        [IO.FileSystemAclExtensions]::SetAccessControl([IO.FileInfo]::new($file), $acl)
    } else {
        [IO.File]::SetAccessControl($file, $acl)
    }
    if ($existing.Contains($key)) {
        Write-Output 'Public key is already present.'
    } else {
        [IO.File]::AppendAllText($file, [Environment]::NewLine + $key + [Environment]::NewLine, [Text.UTF8Encoding]::new($false))
        Write-Output 'Public key added.'
    }
}
`, identityCheck, location, public, powershellReadUTF8, principals, directoryACL),
	}
}

func clientSetup(host state.Host, config, platform string) GuideStep {
	step := GuideStep{
		Title:        fmt.Sprintf("LOCAL computer — %s POSIX shell", platform),
		Instructions: "Run in the selected local terminal, where the matching private key is accessible. Appends the Host snippet to ~/.ssh/config; preserves existing content. Review existing Host, wildcard, Match and Include rules first: the script only detects literal aliases in this file and cannot resolve precedence or included files. It stops if that alias already exists.",
	}
	if platform == "windows" {
		step.Title = "LOCAL computer — Windows PowerShell"
		step.Script = fmt.Sprintf(`& {
    $ErrorActionPreference = 'Stop'
    $dir = Join-Path $HOME '.ssh'
    $file = Join-Path $dir 'config'
    $snippet = @'
%s
'@
    foreach ($path in @($dir, $file)) {
        if ((Test-Path -LiteralPath $path) -and ((Get-Item -Force -LiteralPath $path).Attributes -band [IO.FileAttributes]::ReparsePoint)) {
            throw 'Refusing a reparse point.'
        }
    }
%s    if ([regex]::IsMatch($existing, '(?im)^[ \t]*Host(?:[ \t]+|[ \t]*=[ \t]*)(?:[^ \t\r\n]+[ \t]+)*"?%s"?(?:[ \t\r\n]|$)')) {
        throw 'Host alias already exists. Review the existing SSH config first.'
    }
    [IO.Directory]::CreateDirectory($dir) | Out-Null
    [IO.File]::AppendAllText($file, [Environment]::NewLine + $snippet + [Environment]::NewLine, [Text.UTF8Encoding]::new($false))
    Write-Output 'SSH config snippet added. You can now run the connection step.'
}
`, strings.TrimSuffix(config, "\n"), powershellReadUTF8, regexp.QuoteMeta(host.Alias))
	} else {
		step.Script = fmt.Sprintf(`(
set -eu
umask 077
dir="$HOME/.ssh"
file="$dir/config"
[ ! -L "$dir" ] && [ ! -L "$file" ] || { printf 'Refusing a symlink.\n' >&2; exit 1; }
[ ! -e "$file" ] || [ -f "$file" ] || { printf 'Expected a regular file.\n' >&2; exit 1; }
mkdir -p "$dir"
if [ -f "$file" ]; then
    if awk -v alias='%s' '{ sub(/^[ \t]*[Hh][Oo][Ss][Tt][ \t]*=[ \t]*/, "Host ") } tolower($1) == "host" { for (i=2; i<=NF; i++) { token=$i; gsub(/^"|"$/, "", token); if (tolower(token) == tolower(alias)) found=1 } } END { exit !found }' "$file"; then
        printf 'Host alias already exists. Review the existing SSH config first.\n' >&2
        exit 1
    else
        result=$?
        [ "$result" -eq 1 ] || exit "$result"
    fi
fi
chmod 700 "$dir"
printf '\n' >> "$file"
cat >> "$file" <<'SSH_KEYFLOW_CONFIG'
%sSSH_KEYFLOW_CONFIG
chmod 600 "$file"
printf 'SSH config snippet added. You can now run the connection step.\n'
)
`, host.Alias, config)
	}
	return step
}
