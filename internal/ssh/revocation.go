package ssh

import "strings"

type RevocationPlan struct {
	Target        string
	Script        string
	InspectScript string
	Shared        bool
}

func StandardRevocation(system, windowsAccount string, public []byte) (RevocationPlan, error) {
	registration, err := StandardRegistration(system, windowsAccount, public)
	if err != nil {
		return RevocationPlan{}, err
	}
	fields := strings.Fields(string(public))
	plan := RevocationPlan{Target: registration.Target, Shared: registration.Shared}
	plan.Script = strings.NewReplacer("KEY_TYPE", fields[0], "KEY_BLOB", fields[1]).Replace(posixRevocation)
	if system == "windows" {
		directory, filename := `Join-Path $RevocationHome '.ssh'`, "authorized_keys"
		if plan.Shared {
			directory, filename = "$RevocationServer", "administrators_authorized_keys"
		}
		plan.Script = strings.NewReplacer("KEY_TYPE", fields[0], "KEY_BLOB", fields[1], "ACCOUNT_TYPE", windowsAccount, "TARGET_DIRECTORY", directory, "TARGET_FILENAME", filename).Replace(windowsRevocation)
	}
	plan.InspectScript = strings.ReplaceAll(plan.Script, "OPERATION_MODE", "inspect")
	plan.Script = strings.ReplaceAll(plan.Script, "OPERATION_MODE", "revoke")
	return plan, nil
}

const posixRevocation = `set -eu
command -v perl >/dev/null 2>&1 || { printf '%s\n' 'Perl is required; review manually; unchanged' >&2; exit 1; }
perl - 'KEY_TYPE' 'KEY_BLOB' 'OPERATION_MODE' <<'SSH_KEYFLOW_REVOCATION'
use strict;
use warnings;
use Fcntl qw(:DEFAULT :flock :mode);
use File::Basename qw(dirname);
use File::Temp qw(tempfile);
use Errno qw(ENOENT);
use IO::Handle;
my ($key_type, $key_blob, $operation_mode) = @ARGV;
sub plain_path {
    my ($path, $owned) = @_;
    my $current = $path;
    while (1) {
        my @metadata = lstat($current);
        die "Unreadable path requires manual review: $current\n" unless @metadata;
        die "Symlink requires manual review\n" if S_ISLNK($metadata[2]);
        die "Owner requires manual review\n" unless $metadata[4] == $< || (!$owned && $metadata[4] == 0);
        die "Permissions require manual review; unchanged\n" if ($metadata[2] & 0022) && !(!$owned && S_ISDIR($metadata[2]) && ($metadata[2] & 01000));
        my $parent = dirname($current);
        last if $parent eq $current;
        $current = $parent;
        $owned = 0;
    }
}
sub first_token {
    my ($remaining) = @_;
    $remaining =~ s/^[\t ]+//;
    my $token = '';
    my $quoted = 0;
    my $escaped = 0;
    while (length $remaining) {
        my $character = substr($remaining, 0, 1, '');
        if ($escaped) { $escaped = 0; }
        elsif ($quoted && ord($character) == 92) { $escaped = 1; }
        elsif ($character eq '"') { $quoted = !$quoted; }
        elsif (!$quoted && $character =~ /[\t ]/) { return ($token, $remaining); }
        $token .= $character;
    }
    die "Malformed quoted key options require manual review; unchanged\n" if $quoted || $escaped;
    return ($token, '');
}
sub matches_key {
    my ($line) = @_;
    $line =~ s/[\r\n]+$//;
    return 0 if $line =~ /^[\t ]*(?:#|$)/;
    my ($first, $remaining) = first_token($line);
    if ($first ne $key_type) { ($first, $remaining) = first_token($remaining); }
    return 0 unless $first eq $key_type;
    my ($blob, $comment) = first_token($remaining);
    return $blob eq $key_blob;
}
my $home = $ENV{HOME};
die "Absolute home required\n" unless defined($home) && $home =~ m{^(?:/|[A-Za-z]:/)} && $home !~ m{(?:^|/)\.\.?(?:/|$)};
plain_path($home, 1);
my $configuration = '/etc/ssh/sshd_config';
plain_path($configuration, 0);
die "Nonregular configuration\n" unless -f $configuration;
open(my $config_stream, '<', $configuration) or die "Cannot inspect sshd configuration; review manually\n";
while (my $directive = <$config_stream>) {
    $directive =~ s/^\xef\xbb\xbf// if $. == 1;
    die "Unsupported configuration encoding\n" if $directive =~ /\x00/;
    $directive =~ s/[\r\n]+$//;
    next if $directive =~ /^\s*(?:#|$)/;
    die "Configuration requires manual review; unchanged\n" if $directive =~ /^\s*(?:Include|Match)\b/i;
    if ($directive =~ /^\s*AuthorizedKeysFile\b/i) {
        die "Custom AuthorizedKeysFile requires manual review; unchanged\n" unless $directive =~ m{^\s*(?i:AuthorizedKeysFile)\s+\.ssh/authorized_keys\s*(?:#.*)?$};
    }
    if ($directive =~ /^\s*AuthorizedKeysCommand\b/i) {
        die "AuthorizedKeysCommand requires manual review; unchanged\n" unless $directive =~ /^\s*AuthorizedKeysCommand\s+none\s*(?:#.*)?$/i;
    }
}
close($config_stream) or die "Configuration read failed\n";
my $directory = "$home/.ssh";
if (!lstat($directory)) { die "Cannot inspect directory; review manually\n" unless $! == ENOENT; print "Already absent\n"; exit 0; }
plain_path($directory, 1);
die "Not a directory\n" unless -d $directory;
my $secondary = "$directory/authorized_keys2";
die "Secondary authorization source requires manual review; unchanged\n" if lstat($secondary);
die "Cannot inspect secondary authorization source\n" unless $! == ENOENT;
my $authorized = "$directory/authorized_keys";
if (!lstat($authorized)) { die "Cannot inspect file; review manually\n" unless $! == ENOENT; print "Already absent\n"; exit 0; }
plain_path($authorized, 1);
my @before = lstat($authorized);
die "Nonregular or hardlinked file requires manual review\n" unless S_ISREG($before[2]) && $before[3] == 1;
my $lock = "$directory/ssh-keyflow-registration.lock";
my $lock_owned = 0;
if ($operation_mode ne 'inspect') {
    mkdir($lock, 0700) or die "Operation locked; review before retry\n";
    $lock_owned = 1;
}
END { rmdir($lock) if defined($lock_owned) && $lock_owned; }
sysopen(my $stream, $authorized, O_RDONLY | O_NONBLOCK) or die "Cannot open authorized_keys: $!\n";
flock($stream, ($operation_mode eq 'inspect' ? LOCK_SH : LOCK_EX) | LOCK_NB) or die "File locked\n";
binmode($stream);
my @opened = stat($stream);
die "File changed; review manually\n" unless $opened[0] == $before[0] && $opened[1] == $before[1];
local $/;
my $bytes = <$stream>;
$bytes = '' unless defined($bytes);
my $original = $bytes;
die "Unsupported encoding; unchanged\n" if $bytes =~ /\x00/ || $bytes =~ /^(?:\xff\xfe|\xfe\xff)/;
my $bom = $bytes =~ s/^(\xef\xbb\xbf)// ? $1 : '';
my $retained = $bom;
my $removed = 0;
while ($bytes =~ /([^\r\n]*(?:\r\n|\r|\n)|[^\r\n]+\z)/g) {
    my $line = $1;
    if (matches_key($line)) { $removed++; } else { $retained .= $line; }
}
if (!$removed) { print "Already absent\n"; exit 0; }
if ($operation_mode eq 'inspect') { print "Public key present\n"; exit 0; }
my ($backup, $backup_path) = tempfile('authorized_keys.keyflow-backup-XXXXXXXX', DIR => $directory, UNLINK => 0);
binmode($backup);
system('cp', '-p', $authorized, $backup_path) == 0 or die "Cannot preserve authorization backup; unchanged\n";
seek($backup, 0, 0) or die "Cannot inspect backup; unchanged\n";
my $backup_bytes = <$backup>;
die "Backup differs; unchanged\n" unless defined($backup_bytes) && $backup_bytes eq $original;
$backup->sync() or die "Cannot sync backup; unchanged\n";
close($backup) or die "Cannot close backup; unchanged\n";
my ($staged, $staged_path) = tempfile('replacement-XXXXXXXX', DIR => $lock, UNLINK => 0);
binmode($staged);
system('cp', '-p', $backup_path, $staged_path) == 0 or die "Cannot preserve replacement metadata; unchanged\n";
seek($staged, 0, 0) or die "Cannot rewind replacement; unchanged\n";
print {$staged} $retained or die "Replacement write failed; unchanged\n";
$staged->flush() or die "Replacement flush failed; unchanged\n";
truncate($staged, length($retained)) or die "Replacement truncate failed; unchanged\n";
$staged->sync() or die "Replacement sync failed; unchanged\n";
my @staged_metadata = stat($staged);
die "Replacement metadata differs; unchanged\n" unless $staged_metadata[2] == $opened[2] && $staged_metadata[3] == 1 && $staged_metadata[4] == $opened[4] && $staged_metadata[5] == $opened[5];
sysopen(my $directory_stream, $directory, O_RDONLY) or die "Cannot open directory for durability; unchanged\n";
$directory_stream->sync() or die "Cannot sync backup directory; unchanged\n";
plain_path($authorized, 1);
my @current = lstat($authorized);
die "File changed; review manually\n" unless $current[0] == $opened[0] && $current[1] == $opened[1] && $current[2] == $opened[2] && $current[3] == 1 && $current[4] == $opened[4] && $current[5] == $opened[5] && $current[7] == $opened[7] && $current[9] == $opened[9] && $current[10] == $opened[10];
seek($stream, 0, 0) or die "Cannot recheck original; unchanged\n";
my $current_bytes = <$stream>;
die "File content changed; review manually\n" unless defined($current_bytes) && $current_bytes eq $original;
plain_path($staged_path, 1);
my @staged_current = lstat($staged_path);
die "Replacement changed; unchanged\n" unless $staged_current[0] == $staged_metadata[0] && $staged_current[1] == $staged_metadata[1] && $staged_current[3] == 1;
close($staged) or die "Cannot close replacement; unchanged\n";
rename($staged_path, $authorized) or die "Cannot replace authorization; backup retained\n";
$directory_stream->sync() or die "Cannot sync replacement directory; backup retained; result unknown\n";
close($directory_stream) or die "Cannot close directory; backup retained\n";
close($stream) or die "Close failed; result unknown\n";
print "Public key removed\n";
SSH_KEYFLOW_REVOCATION
`

const windowsRevocation = `param([string]$RevocationHome = $HOME, [string]$RevocationServer = (Join-Path $env:ProgramData 'ssh'))
$ErrorActionPreference = 'Stop'
$accountType = 'ACCOUNT_TYPE'
$readOnly = 'OPERATION_MODE' -eq 'inspect'
$keyType = 'KEY_TYPE'
$keyBlob = 'KEY_BLOB'
$identity = [Security.Principal.WindowsIdentity]::GetCurrent()
$administratorSID = 'S-1-5-32-544'
$systemSID = 'S-1-5-18'
$adminGroup = @($identity.Groups | ForEach-Object { $_.Value }) -contains $administratorSID
if (($accountType -eq 'administrator') -ne $adminGroup) { throw 'Account type mismatch; review target login and Match rules' }
if ($accountType -eq 'administrator' -and -not (New-Object Security.Principal.WindowsPrincipal($identity)).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) { throw 'Administrator target requires an already elevated session; no automatic elevation' }
$allowed = @($administratorSID, $systemSID)
if ($accountType -eq 'standard') { $allowed += $identity.User.Value }
function Assert-RevocationPath([string]$Path) {
    if ($Path -notmatch '^[A-Za-z]:[\\/]' -or $Path -match '(^|[\\/])\.\.?([\\/]|$)' -or $Path -match ':.*:') { throw 'Unsafe path; review manually' }
    $current = $Path
    while ($current) {
        try {
            $attributes = [IO.File]::GetAttributes($current)
            if ($attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'Reparse point requires manual review' }
        } catch {
            $failure = $_.Exception.GetBaseException()
            if ($failure -isnot [IO.FileNotFoundException] -and $failure -isnot [IO.DirectoryNotFoundException]) { throw }
        }
        $parent = Split-Path -Path $current -Parent
        if ($parent -eq $current) { break }
        $current = $parent
    }
}
function Assert-RevocationACL([string]$Path) {
    $acl = Get-Acl -LiteralPath $Path
    if ($allowed -notcontains $acl.GetOwner([Security.Principal.SecurityIdentifier]).Value) { throw 'Owner requires manual review; unchanged' }
    foreach ($rule in $acl.GetAccessRules($true, $true, [Security.Principal.SecurityIdentifier])) {
        if ($rule.AccessControlType -eq 'Allow' -and $allowed -notcontains $rule.IdentityReference.Value) { throw 'ACL requires manual review; no ACL reset' }
    }
}
function Get-RevocationToken([string]$Text) {
    $remaining = $Text.TrimStart([char[]]@([char]9, [char]32))
    $token = ''
    $quoted = $false
    $escaped = $false
    for ($position = 0; $position -lt $remaining.Length; $position++) {
        $character = $remaining[$position]
        if ($escaped) { $escaped = $false }
        elseif ($quoted -and $character -eq [char]92) { $escaped = $true }
        elseif ($character -eq [char]34) { $quoted = -not $quoted }
        elseif (-not $quoted -and ($character -eq [char]9 -or $character -eq [char]32)) { return @($token, $remaining.Substring($position + 1)) }
        $token += $character
    }
    if ($quoted -or $escaped) { throw 'Malformed quoted key options require manual review; unchanged' }
    return @($token, '')
}
function Test-RevocationIdentity([string]$Line) {
    $content = $Line.TrimEnd([char[]]@([char]13, [char]10))
    if ($content -match '^[\t ]*(#|$)') { return $false }
    $token = Get-RevocationToken $content
    if ($token[0] -cne $keyType) { $token = Get-RevocationToken $token[1] }
    if ($token[0] -cne $keyType) { return $false }
    $blob = Get-RevocationToken $token[1]
    return $blob[0] -ceq $keyBlob
}
if ($accountType -eq 'standard') {
    Assert-RevocationPath $RevocationHome
    Assert-RevocationACL $RevocationHome
}
Assert-RevocationPath $RevocationServer
Assert-RevocationACL $RevocationServer
$configuration = Join-Path $RevocationServer 'sshd_config'
Assert-RevocationPath $configuration
Assert-RevocationACL $configuration
$utf8 = New-Object Text.UTF8Encoding($false, $true)
$configText = $utf8.GetString([IO.File]::ReadAllBytes($configuration)).TrimStart([char]0xFEFF)
if ($configText.Contains([char]0)) { throw 'Unsupported configuration encoding; unchanged' }
$adminMatch = $false
$adminTarget = $false
foreach ($directive in ($configText -split '\r\n|\r|\n')) {
    if ($directive -match '^\s*(#|$)') { continue }
    if ($directive -match '^\s*Include\b') { throw 'Include requires manual review; unchanged' }
    if ($directive -match '^\s*Match\b') {
        if ($adminMatch -or $directive -notmatch '^\s*Match\s+Group\s+administrators\s*(#.*)?$') { throw 'Match requires manual review; unchanged' }
        $adminMatch = $true
        continue
    }
    if ($directive -match '^\s*AuthorizedKeysFile\b') {
        if ($adminMatch) {
            if ($directive -cnotmatch '^\s*AuthorizedKeysFile\s+__PROGRAMDATA__/ssh/administrators_authorized_keys\s*(#.*)?$') { throw 'Custom shared AuthorizedKeysFile requires manual review; unchanged' }
            $adminTarget = $true
        } elseif ($directive -cnotmatch '^\s*AuthorizedKeysFile\s+\.ssh/authorized_keys\s*(#.*)?$') { throw 'Custom AuthorizedKeysFile requires manual review; unchanged' }
    } elseif ($adminMatch) { throw 'Custom Match configuration requires manual review; unchanged' }
    if ($directive -match '^\s*AuthorizedKeysCommand\b' -and $directive -notmatch '^\s*AuthorizedKeysCommand\s+none\s*(#.*)?$') { throw 'AuthorizedKeysCommand requires manual review; unchanged' }
}
if ($accountType -eq 'administrator' -and -not $adminTarget) { throw 'Shared administrator target is not configured; review manually' }
$directory = TARGET_DIRECTORY
$authorized = Join-Path $directory 'TARGET_FILENAME'
Write-Output ('Revocation target: ' + $authorized)
if (-not $readOnly -and $accountType -eq 'administrator' -and (Read-Host 'This shared file affects administrator logins. Type yes to approve this exact target') -cne 'yes') { throw 'Cancelled' }
Assert-RevocationPath $directory
if (-not (Test-Path -LiteralPath $directory)) { Write-Output 'Already absent'; return }
Assert-RevocationACL $directory
Assert-RevocationPath $authorized
if (-not (Test-Path -LiteralPath $authorized)) { Write-Output 'Already absent'; return }
if ((Get-Item -LiteralPath $authorized -Force).PSIsContainer) { throw 'Nonregular target; unchanged' }
Assert-RevocationACL $authorized
if (-not ('SSHKeyFlowRevocationHandle' -as [type])) {
    Add-Type -TypeDefinition @'
using System;
using System.IO;
using System.Runtime.InteropServices;
using Microsoft.Win32.SafeHandles;
public static class SSHKeyFlowRevocationHandle {
    [StructLayout(LayoutKind.Sequential)]
    public struct FileInformation {
        public uint Attributes;
        public System.Runtime.InteropServices.ComTypes.FILETIME CreationTime;
        public System.Runtime.InteropServices.ComTypes.FILETIME AccessTime;
        public System.Runtime.InteropServices.ComTypes.FILETIME WriteTime;
        public uint VolumeSerial;
        public uint SizeHigh;
        public uint SizeLow;
        public uint Links;
        public uint IndexHigh;
        public uint IndexLow;
    }
    [DllImport("kernel32.dll", SetLastError = true)]
    private static extern bool GetFileInformationByHandle(SafeFileHandle handle, out FileInformation information);
    public static FileInformation Inspect(FileStream stream) {
        FileInformation information;
        if (!GetFileInformationByHandle(stream.SafeFileHandle, out information) || information.Links != 1)
            throw new IOException("Hardlinked or unverifiable target requires manual review");
        return information;
    }
    public static void AssertSame(FileInformation expected, FileStream stream) {
        FileInformation current = Inspect(stream);
        if (current.VolumeSerial != expected.VolumeSerial || current.IndexHigh != expected.IndexHigh || current.IndexLow != expected.IndexLow ||
            current.SizeHigh != expected.SizeHigh || current.SizeLow != expected.SizeLow || current.Attributes != expected.Attributes ||
            current.WriteTime.dwHighDateTime != expected.WriteTime.dwHighDateTime || current.WriteTime.dwLowDateTime != expected.WriteTime.dwLowDateTime)
            throw new IOException("Authorization target changed; unchanged");
    }
}
'@
}
$sharing = [IO.FileShare]::Read -bor [IO.FileShare]::Delete
$operationLock = $null
$stream = $null
try {
    if (-not $readOnly) {
        $lockPath = Join-Path $directory 'ssh-keyflow-revocation.lock'
        $operationLock = [IO.FileStream]::new($lockPath, [IO.FileMode]::CreateNew, [IO.FileAccess]::ReadWrite, [IO.FileShare]::None, 1, [IO.FileOptions]::DeleteOnClose)
    }
    $stream = [IO.File]::Open($authorized, [IO.FileMode]::Open, [IO.FileAccess]::Read, $sharing)
    $opened = [SSHKeyFlowRevocationHandle]::Inspect($stream)
    Assert-RevocationPath $authorized
    Assert-RevocationACL $authorized
    $originalACL = Get-Acl -LiteralPath $authorized
    $bytes = New-Object byte[] $stream.Length
    $offset = 0
    while ($offset -lt $bytes.Length) {
        $count = $stream.Read($bytes, $offset, $bytes.Length - $offset)
        if ($count -eq 0) { throw 'Incomplete file read; unchanged' }
        $offset += $count
    }
    $text = $utf8.GetString($bytes)
    if ($text.Contains([char]0)) { throw 'Unsupported encoding; unchanged' }
    $retained = New-Object Text.StringBuilder
    if ($text.Length -gt 0 -and $text[0] -eq [char]0xFEFF) { $null = $retained.Append([char]0xFEFF); $text = $text.Substring(1) }
    $removed = 0
    foreach ($line in [regex]::Matches($text, '[^\r\n]*(?:\r\n|\r|\n)|[^\r\n]+\z')) {
        if (Test-RevocationIdentity $line.Value) { $removed++ } else { $null = $retained.Append($line.Value) }
    }
    if ($removed -eq 0) { Write-Output 'Already absent'; return }
    if ($readOnly) { Write-Output 'Public key present'; return }
    $replacement = $utf8.GetBytes($retained.ToString())
    $backupPath = Join-Path $directory ('authorized_keys.keyflow-backup-' + [Guid]::NewGuid().ToString('N'))
    $stagedPath = Join-Path $directory ('authorized_keys.keyflow-stage-' + [Guid]::NewGuid().ToString('N'))
    $backup = [IO.File]::Open($backupPath, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
    try {
        Set-Acl -LiteralPath $backupPath -AclObject $originalACL
        $backup.Write($bytes, 0, $bytes.Length)
        $backup.Flush($true)
    } finally { $backup.Dispose() }
    if ((Get-Acl -LiteralPath $backupPath).Sddl -cne $originalACL.Sddl) { throw 'Backup ACL differs; unchanged' }
    $staged = [IO.File]::Open($stagedPath, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
    try {
        Set-Acl -LiteralPath $stagedPath -AclObject $originalACL
        $staged.Write($replacement, 0, $replacement.Length)
        $staged.Flush($true)
    } finally { $staged.Dispose() }
    Assert-RevocationPath $authorized
    Assert-RevocationACL $authorized
    if ((Get-Acl -LiteralPath $authorized).Sddl -cne $originalACL.Sddl -or (Get-Acl -LiteralPath $stagedPath).Sddl -cne $originalACL.Sddl) { throw 'Authorization ACL changed; unchanged' }
    $current = [IO.File]::Open($authorized, [IO.FileMode]::Open, [IO.FileAccess]::Read, $sharing)
    try {
        [SSHKeyFlowRevocationHandle]::AssertSame($opened, $current)
        $currentBytes = New-Object byte[] $current.Length
        $offset = 0
        while ($offset -lt $currentBytes.Length) {
            $count = $current.Read($currentBytes, $offset, $currentBytes.Length - $offset)
            if ($count -eq 0) { throw 'Incomplete file recheck; unchanged' }
            $offset += $count
        }
        if ([Convert]::ToBase64String($bytes) -cne [Convert]::ToBase64String($currentBytes)) { throw 'Authorization content changed; unchanged' }
        [IO.File]::Replace($stagedPath, $authorized, ($backupPath + '.replaced'), $false)
    } finally { $current.Dispose() }
    if ((Get-Acl -LiteralPath $authorized).Sddl -cne $originalACL.Sddl) { throw 'Replacement ACL differs; backup retained; review manually' }
    Write-Output 'Public key removed'
} finally {
    if ($null -ne $stream) { $stream.Dispose() }
    if ($null -ne $operationLock) { $operationLock.Dispose() }
}
`
