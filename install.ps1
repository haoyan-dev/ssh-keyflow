& {
Set-StrictMode -Version 2.0
$ErrorActionPreference = 'Stop'
$Repository = if ($env:SSH_KEYFLOW_REPOSITORY) { $env:SSH_KEYFLOW_REPOSITORY } else { 'haoyan-dev/ssh-keyflow' }
$Version = ''
$InstallDir = Join-Path $env:LOCALAPPDATA 'Programs\ssh-keyflow'
$ProfilePath = $PROFILE.CurrentUserAllHosts
$Uninstall = $false
$AddToPath = $false
for ($argumentIndex = 0; $argumentIndex -lt $args.Count; $argumentIndex++) {
    switch -Regex ([string]$args[$argumentIndex]) {
        '^-(Repository|Version|InstallDir|ProfilePath)$' {
            $parameterName = $Matches[1]
            $argumentIndex++
            if ($argumentIndex -ge $args.Count) { throw "Missing value for -$parameterName" }
            Set-Variable -Name $parameterName -Value ([string]$args[$argumentIndex]) -Scope Local
        }
        '^-(Uninstall|AddToPath)$' { Set-Variable -Name $Matches[1] -Value $true -Scope Local }
        default { throw "Unknown argument: $($args[$argumentIndex])" }
    }
}
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
    if (Test-Path -LiteralPath $temporaryReceipt) { throw 'Temporary receipt already exists; inspect before retrying.' }
    [IO.File]::WriteAllText($temporaryReceipt, ($Receipt | ConvertTo-Json), (New-Object Text.UTF8Encoding($false)))
    if ([IO.File]::Exists($Path)) { [IO.File]::Replace($temporaryReceipt, $Path, [NullString]::Value) }
    else { [IO.File]::Move($temporaryReceipt, $Path) }
}
function Save-OwnedProfile([string]$Path, [byte[]]$Bytes, $NewReceipt, [string]$ReceiptPath) {
    $parent = [IO.Path]::GetDirectoryName($Path)
    [IO.Directory]::CreateDirectory($parent) | Out-Null
    $profileStage = Join-Path $parent ('.ssh-keyflow-profile.' + [Guid]::NewGuid().ToString('N'))
    [IO.Directory]::CreateDirectory($profileStage) | Out-Null
    $candidateProfile = Join-Path $profileStage 'replacement'
    $previousProfile = Join-Path $profileStage 'previous'
    $profileReplaced = $false
    $keepRecovery = $false
    try {
        [IO.File]::WriteAllBytes($candidateProfile, $Bytes)
        if ([IO.File]::Exists($Path)) { [IO.File]::Replace($candidateProfile, $Path, $previousProfile) }
        else { [IO.File]::Move($candidateProfile, $Path) }
        $profileReplaced = $true
        Save-Receipt $NewReceipt $ReceiptPath
    } catch {
        if ($profileReplaced) {
            $keepRecovery = $true
            if ([IO.File]::Exists($previousProfile)) { [IO.File]::Replace($previousProfile, $Path, [NullString]::Value) }
            else { Remove-Item -LiteralPath $Path }
            $keepRecovery = $false
        }
        throw
    } finally {
        if (-not $keepRecovery) {
            $resolvedProfileStage = [IO.Path]::GetFullPath($profileStage)
            if ([IO.Path]::GetDirectoryName($resolvedProfileStage) -ne $parent -or -not [IO.Path]::GetFileName($resolvedProfileStage).StartsWith('.ssh-keyflow-profile.')) { throw 'Unsafe profile staging cleanup path.' }
            Remove-Item -LiteralPath $resolvedProfileStage -Recurse -Force
        } else { Write-Warning "Profile recovery files retained at $profileStage" }
    }
}
$stage = $null
$lock = $null
$lockPath = $null
$keepRecovery = $false
try {
    if ($env:OS -ne 'Windows_NT') { throw 'Unsupported OS; use install.sh on Linux/macOS.' }
    if (-not [IO.Path]::IsPathRooted($InstallDir)) { throw 'Install directory must be absolute.' }
    $InstallDir = [IO.Path]::GetFullPath($InstallDir)
    Assert-RegularPath $InstallDir
    [IO.Directory]::CreateDirectory($InstallDir) | Out-Null
    $binary = Join-Path $InstallDir 'ssh-keyflow.exe'
    $notice = Join-Path $InstallDir 'THIRD_PARTY_NOTICES.txt'
    $receiptPath = Join-Path $InstallDir '.ssh-keyflow-install.json'
    $lockPath = Join-Path $InstallDir '.ssh-keyflow-lock'
    Assert-RegularPath $binary
    Assert-RegularPath $notice
    Assert-RegularPath $receiptPath
    $lock = [IO.File]::Open($lockPath, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
    $receipt = $null
    if (Test-Path -LiteralPath $receiptPath) { $receipt = [IO.File]::ReadAllText($receiptPath) | ConvertFrom-Json }
    if (Test-Path -LiteralPath $binary) {
        if (-not $receipt -or (Get-Digest $binary) -ne $receipt.hash) { throw 'Existing binary is unowned or changed; preserved.' }
    } elseif ($receipt) { throw 'Installation receipt exists without binary; inspect before retrying.' }
    $noticesHash = $null
    if ($receipt -and $receipt.PSObject.Properties['noticesHash']) { $noticesHash = $receipt.noticesHash }
    if (Test-Path -LiteralPath $notice) {
        if (-not $noticesHash -or (Get-Digest $notice) -ne $noticesHash) { throw 'Existing notices are unowned or changed; preserved.' }
    } elseif ($noticesHash) { throw 'Owned notices are missing; inspect before retrying.' }
    if ($Uninstall) {
        if ($receipt) {
            $stage = Join-Path $InstallDir ('.ssh-keyflow-stage.' + [Guid]::NewGuid().ToString('N'))
            [IO.Directory]::CreateDirectory($stage) | Out-Null
            $previous = Join-Path $stage 'previous.exe'
            $previousNotice = Join-Path $stage 'previous-notices'
            $previousReceipt = Join-Path $stage 'previous-receipt'
            [IO.File]::Copy($binary, $previous)
            if ($noticesHash) { [IO.File]::Copy($notice, $previousNotice) }
            [IO.File]::Copy($receiptPath, $previousReceipt)
            $profileRemoved = $false
            $binaryRemoved = $false
            $noticeRemoved = $false
            try {
                if ($receipt.profile) {
                    Assert-RegularPath $receipt.profile
                    $encoding = [Text.Encoding]::GetEncoding(28591)
                    $profileBytes = [IO.File]::ReadAllBytes($receipt.profile)
                    $profileText = $encoding.GetString($profileBytes)
                    $stanza = $encoding.GetString([Convert]::FromBase64String($receipt.stanza))
                    $offset = $profileText.IndexOf($stanza, [StringComparison]::Ordinal)
                    if ($offset -lt 0 -or $profileText.IndexOf($stanza, $offset + $stanza.Length, [StringComparison]::Ordinal) -ge 0) { throw 'Owned PATH stanza was changed or duplicated; profile and installation preserved.' }
                    $newReceipt = $receipt.PSObject.Copy()
                    $newReceipt.profile = $null
                    $newReceipt.stanza = $null
                    Save-OwnedProfile $receipt.profile $encoding.GetBytes($profileText.Remove($offset, $stanza.Length)) $newReceipt $receiptPath
                    $profileRemoved = $true
                }
                Remove-Item -LiteralPath $binary
                $binaryRemoved = $true
                if ($noticesHash) { Remove-Item -LiteralPath $notice; $noticeRemoved = $true }
                Remove-Item -LiteralPath $receiptPath
            } catch {
                $keepRecovery = $true
                if ($binaryRemoved) { [IO.File]::Move($previous, $binary) }
                if ($noticeRemoved) { [IO.File]::Move($previousNotice, $notice) }
                if ($profileRemoved) { Save-OwnedProfile $receipt.profile $profileBytes $receipt $receiptPath }
                if ([IO.File]::Exists($receiptPath)) { [IO.File]::Replace($previousReceipt, $receiptPath, [NullString]::Value) }
                else { [IO.File]::Move($previousReceipt, $receiptPath) }
                $keepRecovery = $false
                throw
            }
        }
        Write-Output 'Uninstalled owned executable and notices; SSH keys, state, configuration and remote registrations retained.'
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
    $entries = @([IO.File]::ReadAllLines($manifest) | Where-Object { $_ -match ('(?:^|\s)' + [regex]::Escape($asset) + '(?:\s|$)') })
    if ($entries.Count -ne 1 -or $entries[0] -notmatch ('^[a-fA-F0-9]{64}  ' + [regex]::Escape($asset) + '$')) { throw 'Missing or ambiguous SHA256 manifest entry.' }
    if ((Get-Digest $archive) -ne $entries[0].Substring(0, 64)) { throw 'SHA256 mismatch.' }
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $bundle = [IO.Compression.ZipFile]::OpenRead($archive)
    try {
        $names = @($bundle.Entries | ForEach-Object { $_.FullName })
        if ($names.Count -ne 2 -or $names -cnotcontains 'THIRD_PARTY_NOTICES.txt' -or $names -cnotcontains 'ssh-keyflow.exe') { throw 'Unexpected archive members.' }
        foreach ($entry in $bundle.Entries) {
            $unixType = ($entry.ExternalAttributes -shr 16) -band 61440
            if (($unixType -ne 0 -and $unixType -ne 32768) -or ($entry.ExternalAttributes -band 16)) { throw 'Expected regular archive members.' }
            [IO.Compression.ZipFileExtensions]::ExtractToFile($entry, (Join-Path $stage $entry.FullName))
        }
    } finally { $bundle.Dispose() }
    $candidate = Join-Path $stage 'ssh-keyflow.exe'
    $candidateNotice = Join-Path $stage 'THIRD_PARTY_NOTICES.txt'
    $hash = Get-Digest $candidate
    $newNoticesHash = Get-Digest $candidateNotice
    $previous = Join-Path $stage 'previous.exe'
    $previousNotice = Join-Path $stage 'previous-notices'
    $previousReceipt = Join-Path $stage 'previous-receipt'
    if ([IO.File]::Exists($receiptPath)) { [IO.File]::Copy($receiptPath, $previousReceipt) }
    $replaced = $false
    $noticeReplaced = $false
    try {
        if ([IO.File]::Exists($binary)) { [IO.File]::Replace($candidate, $binary, $previous) }
        else { [IO.File]::Move($candidate, $binary) }
        $replaced = $true
        if ([IO.File]::Exists($notice)) { [IO.File]::Replace($candidateNotice, $notice, $previousNotice) }
        else { [IO.File]::Move($candidateNotice, $notice) }
        $noticeReplaced = $true
        $newReceipt = [pscustomobject]@{ hash = $hash; noticesHash = $newNoticesHash; profile = $null; stanza = $null }
        if ($receipt) { $newReceipt.profile = $receipt.profile; $newReceipt.stanza = $receipt.stanza }
        Save-Receipt $newReceipt $receiptPath
        $receipt = $newReceipt
        $quotedDir = $InstallDir.Replace("'", "''")
        if (-not $receipt.profile -and ($env:PATH -split ';') -notcontains $InstallDir) {
            $ProfilePath = [IO.Path]::GetFullPath($ProfilePath)
            Assert-RegularPath $ProfilePath
            $encodedDir = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($InstallDir))
            $decode = "[Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('$encodedDir'))"
            $stanza = "`n`$null = 'SSH KeyFlow PATH begin'`nif ((`$env:PATH -split ';') -notcontains ($decode)) { `$env:PATH = ($decode) + ';' + `$env:PATH }`n`$null = 'SSH KeyFlow PATH end'`n"
            Write-Output "Proposed addition to ${ProfilePath}:"
            Write-Output $stanza
            $answer = 'no'
            if ($AddToPath) { $answer = 'yes' }
            elseif (-not [Console]::IsInputRedirected) {
                try { $answer = Read-Host 'Persist this PATH addition? [y/N]' }
                catch [Management.Automation.PSInvalidOperationException] { $answer = 'no' }
            }
            if ($answer -match '^(y|yes)$') {
                $profileBytes = [byte[]]@()
                if ([IO.File]::Exists($ProfilePath)) { $profileBytes = [IO.File]::ReadAllBytes($ProfilePath) }
                $encoding = New-Object Text.UTF8Encoding($false)
                if ($profileBytes.Length -ge 2 -and $profileBytes[0] -eq 255 -and $profileBytes[1] -eq 254) { $encoding = [Text.Encoding]::Unicode }
                if ($profileBytes.Length -ge 2 -and $profileBytes[0] -eq 254 -and $profileBytes[1] -eq 255) { $encoding = [Text.Encoding]::BigEndianUnicode }
                if ($encoding.GetString($profileBytes).Contains('SSH KeyFlow PATH begin')) { throw 'Unowned PATH marker exists; use manual guidance.' }
                $stanzaBytes = $encoding.GetBytes($stanza)
                $newReceipt = $receipt.PSObject.Copy()
                $newReceipt.profile = $ProfilePath
                $newReceipt.stanza = [Convert]::ToBase64String($stanzaBytes)
                Save-OwnedProfile $ProfilePath ([byte[]]($profileBytes + $stanzaBytes)) $newReceipt $receiptPath
                $receipt = $newReceipt
                Write-Output 'PATH saved for PowerShell. Open a new PowerShell session.'
            } else { Write-Output 'Profile unchanged. Use the manual PATH command when needed.' }
        }
    } catch {
        $keepRecovery = $true
        if ($noticeReplaced) {
            if ([IO.File]::Exists($previousNotice)) { [IO.File]::Replace($previousNotice, $notice, [NullString]::Value) }
            else { Remove-Item -LiteralPath $notice }
        }
        if ($replaced) {
            if ([IO.File]::Exists($previous)) { [IO.File]::Replace($previous, $binary, [NullString]::Value) }
            else { Remove-Item -LiteralPath $binary }
        }
        if ([IO.File]::Exists($previousReceipt)) { [IO.File]::Replace($previousReceipt, $receiptPath, [NullString]::Value) }
        elseif ([IO.File]::Exists($receiptPath)) { Remove-Item -LiteralPath $receiptPath }
        $keepRecovery = $false
        throw
    }
    Write-Output "Installed $Version to $binary"
    Write-Output ('Run: & ''' + $binary.Replace("'", "''") + '''')
    Write-Output ('Manual PATH command: $env:PATH = ''' + $quotedDir + ';'' + $env:PATH')
} catch {
    throw
} finally {
    if ($keepRecovery) { Write-Warning "Installation recovery files retained at $stage" }
    elseif ($stage -and [IO.Directory]::Exists($stage)) {
        $resolvedStage = [IO.Path]::GetFullPath($stage)
        if ([IO.Path]::GetDirectoryName($resolvedStage) -ne $InstallDir -or -not [IO.Path]::GetFileName($resolvedStage).StartsWith('.ssh-keyflow-stage.')) { throw 'Unsafe staging cleanup path.' }
        Remove-Item -LiteralPath $resolvedStage -Recurse -Force
    }
    if ($lock) { $lock.Dispose(); Remove-Item -LiteralPath $lockPath }
}
} @args
