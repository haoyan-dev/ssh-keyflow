[CmdletBinding()]
param(
    [string]$Repository = $env:SSH_KEYFLOW_REPOSITORY,
    [string]$Version,
    [string]$InstallDir = (Join-Path $env:LOCALAPPDATA 'Programs\ssh-keyflow'),
    [string]$ProfilePath = $PROFILE.CurrentUserAllHosts,
    [switch]$Uninstall
)
Set-StrictMode -Version 2.0
$ErrorActionPreference = 'Stop'
function Get-Digest([string]$Path) {
    $algorithm = [Security.Cryptography.SHA256]::Create()
    $stream = [IO.File]::OpenRead($Path)
    try { return [BitConverter]::ToString($algorithm.ComputeHash($stream)).Replace('-', '').ToLowerInvariant() }
    finally { $stream.Dispose(); $algorithm.Dispose() }
}
function Assert-RegularPath([string]$Path) {
    if (Test-Path -LiteralPath $Path) {
        if ((Get-Item -LiteralPath $Path -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw "Refusing reparse point: $Path" }
    }
}
function Save-Receipt($Receipt, [string]$Path) {
    $temporaryReceipt = $Path + '.tmp'
    [IO.File]::WriteAllText($temporaryReceipt, ($Receipt | ConvertTo-Json), (New-Object Text.UTF8Encoding($false)))
    if ([IO.File]::Exists($Path)) { [IO.File]::Replace($temporaryReceipt, $Path, [NullString]::Value) }
    else { [IO.File]::Move($temporaryReceipt, $Path) }
}
$stage = $null
$lock = $null
$lockPath = $null
try {
    if ($env:OS -ne 'Windows_NT') { throw 'Unsupported OS; use install.sh on Linux/macOS.' }
    if (-not [IO.Path]::IsPathRooted($InstallDir)) { throw 'Install directory must be absolute.' }
    $InstallDir = [IO.Path]::GetFullPath($InstallDir)
    Assert-RegularPath $InstallDir
    [IO.Directory]::CreateDirectory($InstallDir) | Out-Null
    $binary = Join-Path $InstallDir 'ssh-keyflow.exe'
    $receiptPath = Join-Path $InstallDir '.ssh-keyflow-install.json'
    $lockPath = Join-Path $InstallDir '.ssh-keyflow-lock'
    Assert-RegularPath $binary
    Assert-RegularPath $receiptPath
    $lock = [IO.File]::Open($lockPath, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
    $receipt = $null
    if (Test-Path -LiteralPath $receiptPath) { $receipt = [IO.File]::ReadAllText($receiptPath) | ConvertFrom-Json }
    if (Test-Path -LiteralPath $binary) {
        if (-not $receipt -or (Get-Digest $binary) -ne $receipt.hash) { throw 'Existing binary is unowned or changed; preserved.' }
    } elseif ($receipt) { throw 'Installation receipt exists without binary; inspect before retrying.' }
    if ($Uninstall) {
        if ($receipt) {
            if ($receipt.profile) {
                Assert-RegularPath $receipt.profile
                $encoding = [Text.Encoding]::GetEncoding(28591)
                $profileBytes = [IO.File]::ReadAllBytes($receipt.profile)
                $profileText = $encoding.GetString($profileBytes)
                $stanza = $encoding.GetString([Convert]::FromBase64String($receipt.stanza))
                $offset = $profileText.IndexOf($stanza, [StringComparison]::Ordinal)
                if ($offset -lt 0 -or $profileText.IndexOf($stanza, $offset + $stanza.Length, [StringComparison]::Ordinal) -ge 0) { throw 'Owned PATH stanza was changed or duplicated; profile and installation preserved.' }
                [IO.File]::WriteAllBytes($receipt.profile, $encoding.GetBytes($profileText.Remove($offset, $stanza.Length)))
            }
            Remove-Item -LiteralPath $binary
            Remove-Item -LiteralPath $receiptPath
        }
        Write-Output 'Uninstalled owned executable; SSH keys, state, configuration and remote registrations retained.'
        return
    }
    if ($Repository -notmatch '^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$') { throw 'Specify -Repository OWNER/REPO or SSH_KEYFLOW_REPOSITORY.' }
    $architecture = $env:PROCESSOR_ARCHITEW6432
    if (-not $architecture) { $architecture = $env:PROCESSOR_ARCHITECTURE }
    switch ($architecture) {
        'AMD64' { $architecture = 'amd64' }
        'ARM64' { $architecture = 'arm64' }
        default { throw 'Unsupported CPU; supported: x64 and ARM64.' }
    }
    $releaseUrl = $env:SSH_KEYFLOW_RELEASE_URL
    if (-not $releaseUrl) { $releaseUrl = "https://github.com/$Repository/releases/download" }
    $apiUrl = $env:SSH_KEYFLOW_API_URL
    if (-not $apiUrl) { $apiUrl = "https://api.github.com/repos/$Repository" }
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    if (-not $Version) {
        $latest = Invoke-RestMethod -Uri "$apiUrl/releases/latest"
        if ($latest.prerelease -or $latest.draft) { throw 'Latest response is not a stable release.' }
        $Version = $latest.tag_name
        if ($Version -notmatch '^v[0-9]+\.[0-9]+\.[0-9]+$') { throw 'Latest tag is not stable.' }
    }
    if ($Version -notmatch '^v[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?$') { throw 'Version must be a release tag such as v1.2.3.' }
    $stage = Join-Path $InstallDir ('.ssh-keyflow-stage.' + [Guid]::NewGuid().ToString('N'))
    [IO.Directory]::CreateDirectory($stage) | Out-Null
    $asset = "ssh-keyflow_${Version}_windows_${architecture}.zip"
    $archive = Join-Path $stage $asset
    $manifest = Join-Path $stage 'SHA256SUMS'
    Invoke-WebRequest -UseBasicParsing -Uri "$releaseUrl/$Version/$asset" -OutFile $archive
    Invoke-WebRequest -UseBasicParsing -Uri "$releaseUrl/$Version/SHA256SUMS" -OutFile $manifest
    $entries = @([IO.File]::ReadAllLines($manifest) | Where-Object { $_ -match ('^[a-fA-F0-9]{64}  ' + [regex]::Escape($asset) + '$') })
    if ($entries.Count -ne 1) { throw 'Missing or ambiguous SHA256 manifest entry.' }
    if ((Get-Digest $archive) -ne $entries[0].Substring(0, 64)) { throw 'SHA256 mismatch.' }
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $bundle = [IO.Compression.ZipFile]::OpenRead($archive)
    try {
        if ($bundle.Entries.Count -ne 1 -or $bundle.Entries[0].FullName -cne 'ssh-keyflow.exe') { throw 'Unexpected archive members.' }
        $candidate = Join-Path $stage 'ssh-keyflow.exe'
        [IO.Compression.ZipFileExtensions]::ExtractToFile($bundle.Entries[0], $candidate)
    } finally { $bundle.Dispose() }
    $hash = Get-Digest $candidate
    $previous = Join-Path $stage 'previous.exe'
    $replaced = $false
    try {
        if ([IO.File]::Exists($binary)) { [IO.File]::Replace($candidate, $binary, $previous) }
        else { [IO.File]::Move($candidate, $binary) }
        $replaced = $true
        $newReceipt = [pscustomobject]@{ hash = $hash; profile = $null; stanza = $null }
        if ($receipt) { $newReceipt.profile = $receipt.profile; $newReceipt.stanza = $receipt.stanza }
        Save-Receipt $newReceipt $receiptPath
        $receipt = $newReceipt
    } catch {
        if ($replaced) {
            if ([IO.File]::Exists($previous)) { [IO.File]::Replace($previous, $binary, [NullString]::Value) }
            else { Remove-Item -LiteralPath $binary }
        }
        throw
    }
    Write-Output "Installed $Version to $binary"
    $quotedDir = $InstallDir.Replace("'", "''")
    Write-Output ('Manual PATH command: $env:PATH = ''' + $quotedDir + ';'' + $env:PATH')
    if (-not $receipt.profile -and ($env:PATH -split ';') -notcontains $InstallDir) {
        $ProfilePath = [IO.Path]::GetFullPath($ProfilePath)
        Assert-RegularPath $ProfilePath
        $encodedDir = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($InstallDir))
        $decode = "[Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('$encodedDir'))"
        $stanza = "`n`$null = 'SSH KeyFlow PATH begin'`nif ((`$env:PATH -split ';') -notcontains ($decode)) { `$env:PATH = ($decode) + ';' + `$env:PATH }`n`$null = 'SSH KeyFlow PATH end'`n"
        Write-Output "Proposed addition to ${ProfilePath}:"
        Write-Output $stanza
        $answer = Read-Host 'Persist this PATH addition? [y/N]'
        if ($answer -match '^(y|yes)$') {
            $profileBytes = [byte[]]@()
            if ([IO.File]::Exists($ProfilePath)) { $profileBytes = [IO.File]::ReadAllBytes($ProfilePath) }
            $encoding = New-Object Text.UTF8Encoding($false)
            if ($profileBytes.Length -ge 2 -and $profileBytes[0] -eq 255 -and $profileBytes[1] -eq 254) { $encoding = [Text.Encoding]::Unicode }
            if ($profileBytes.Length -ge 2 -and $profileBytes[0] -eq 254 -and $profileBytes[1] -eq 255) { $encoding = [Text.Encoding]::BigEndianUnicode }
            if ($encoding.GetString($profileBytes).Contains('SSH KeyFlow PATH begin')) { throw 'Unowned PATH marker exists; use manual guidance.' }
            [IO.Directory]::CreateDirectory([IO.Path]::GetDirectoryName($ProfilePath)) | Out-Null
            $stanzaBytes = $encoding.GetBytes($stanza)
            $receipt.profile = $ProfilePath
            $receipt.stanza = [Convert]::ToBase64String($stanzaBytes)
            Save-Receipt $receipt $receiptPath
            [IO.File]::WriteAllBytes($ProfilePath, [byte[]]($profileBytes + $stanzaBytes))
            Write-Output 'PATH saved for PowerShell. Open a new PowerShell session.'
        } else { Write-Output 'Profile unchanged. Use the manual PATH command when needed.' }
    }
} catch {
    Write-Error $_ -ErrorAction Continue
    exit 1
} finally {
    if ($stage -and [IO.Directory]::Exists($stage)) {
        $resolvedStage = [IO.Path]::GetFullPath($stage)
        if ([IO.Path]::GetDirectoryName($resolvedStage) -ne $InstallDir -or -not [IO.Path]::GetFileName($resolvedStage).StartsWith('.ssh-keyflow-stage.')) { throw 'Unsafe staging cleanup path.' }
        Remove-Item -LiteralPath $resolvedStage -Recurse -Force
    }
    if ($lock) { $lock.Dispose(); Remove-Item -LiteralPath $lockPath }
}
