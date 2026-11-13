# Install, update and uninstall

[简体中文](../../zh-CN/guides/installation.md) · English · [日本語](../../guides/installation.md)

[Documentation index](../README.md)

Supported targets are amd64 / arm64 on Windows / Linux / macOS. Have the OS OpenSSH (`ssh` and `ssh-keygen`) available. The pinned versions below are examples of how to specify a version. Normally, install the stable release without specifying a version, and read [Data and recovery](data-and-recovery.md) before going back to an older version.

## One-line install

The following commands install the latest stable release.

**macOS / Linux**

```sh
curl -fsSL https://raw.githubusercontent.com/haoyan-dev/ssh-keyflow/main/install.sh | sh
```

**Windows (PowerShell)**

```powershell
irm https://raw.githubusercontent.com/haoyan-dev/ssh-keyflow/main/install.ps1 | iex
```

The installer detects the OS and CPU (x64 / ARM64) and downloads the latest stable release and `SHA256SUMS`. After verifying the checksum and the archive contents, it places the executable and the license document in a per-user location. Go and administrator rights are not required. The OS `ssh` and `ssh-keygen` are needed separately. If you want to review the downloaded script first, save it and read it before running it.

The default install location is `~/.local/bin` on macOS/Linux and `%LOCALAPPDATA%\Programs\ssh-keyflow` on Windows. Adding to PATH is confirmed in an interactive terminal and saved to your shell profile only if you approve. Open a new terminal and run:

```sh
ssh-keyflow --version
ssh-keyflow
```

If you declined the PATH change, or ran the installer non-interactively, you can launch the executable using the full path the installer prints. On Windows, the saved PATH applies only to the profile of the PowerShell that ran the installer. In a different kind of PowerShell or in Command Prompt, use the manual PATH instructions that are printed.

Running the same command again updates the installation. Exit SSH KeyFlow before updating. The installer manages only the executable, the license document and its own PATH stanza. It does not change keys, state, SSH configuration or public key registrations on destinations. Existing files without an ownership record, and files that have been modified, are not overwritten.

To specify a version or install location, or to uninstall, save the script and run it.

```sh
# macOS / Linux: select v0.2.1 and explicitly add to PATH
curl -fsSLo install.sh https://raw.githubusercontent.com/haoyan-dev/ssh-keyflow/main/install.sh
sh install.sh --version v0.2.1 --add-to-path
# After exiting the app, remove only the owned executable, license document and PATH stanza
sh install.sh --uninstall
```

```powershell
# Windows: select v0.2.1 and explicitly add to PATH
Invoke-WebRequest -Uri https://raw.githubusercontent.com/haoyan-dev/ssh-keyflow/main/install.ps1 -OutFile install.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\install.ps1 -Version v0.2.1 -AddToPath
# Remove after exiting the app
powershell -NoProfile -ExecutionPolicy Bypass -File .\install.ps1 -Uninstall
```

The Windows example above sets the execution policy for that process only. To manage the installation with PowerShell 7, replace `powershell` with `pwsh`. Specify the install location with `--install-dir /absolute/path` / `-InstallDir C:\absolute\path`, and the target profile with `--profile /absolute/path` / `-ProfilePath C:\absolute\profile.ps1`. If you used a custom install location, pass the same option when updating and removing. To use a different repository, you can specify `--repository OWNER/REPO` / `-Repository OWNER/REPO`.

## Getting and running the Portable release

Archives are provided for amd64 (x64) / arm64 on Windows / Linux / macOS. Windows uses zip; Linux/macOS use tar.gz. Neither Go nor an installer is required; the OS `ssh` and `ssh-keygen` are required. Use the TUI in an interactive terminal.

Downloads are on [GitHub Releases](https://github.com/haoyan-dev/ssh-keyflow/releases). Of the six archives and `SHA256SUMS` for the target tag, download the archive for your OS / CPU and the manifest.

Example: to use v0.2.0 on Linux amd64, verify and extract as follows.

```sh
sha256sum --ignore-missing -c SHA256SUMS
mkdir -p ./ssh-keyflow-portable
tar -xzf ssh-keyflow_v0.2.0_linux_amd64.tar.gz -C ./ssh-keyflow-portable
./ssh-keyflow-portable/ssh-keyflow --version
./ssh-keyflow-portable/ssh-keyflow
```

On macOS, compare the result of `shasum -a 256 ssh-keyflow_v0.2.0_darwin_arm64.tar.gz` or similar against the single matching line in the manifest. In Windows PowerShell, compare as follows.

```powershell
Get-FileHash .\ssh-keyflow_v0.2.0_windows_amd64.zip -Algorithm SHA256
Get-Content .\SHA256SUMS
# Extract only after the hash matches the single line for the target file name
Expand-Archive .\ssh-keyflow_v0.2.0_windows_amd64.zip .\ssh-keyflow-portable
& .\ssh-keyflow-portable\ssh-keyflow.exe --version
& .\ssh-keyflow-portable\ssh-keyflow.exe
```

SHA256 checks integrity; it is not signature verification. Each archive contains the executable and `THIRD_PARTY_NOTICES.txt`. Choose any location you can write to, and keep the bundled license document when you move or share the distribution. Portable means you can place the executable wherever you like; it does not include any mechanism for carrying keys or management data between users.

## Manual Portable updates, PATH and removal

Before updating, exit the app, verify the checksum of the new distribution, and extract it into a separate directory. Then replace the existing executable and bundled license document with the new ones and check `--version`. Even after you move the executable, it keeps using the state and keys under your home directory as before. Previously generated settings also remain as they are.

Set PATH yourself only if you need it. For the current session only, run `export PATH="/absolute/path/ssh-keyflow-portable:$PATH"` in a POSIX shell or `$env:PATH = 'C:\absolute\path\ssh-keyflow-portable;' + $env:PATH` in PowerShell. To make it persistent, edit your shell profile or the OS PATH settings yourself. The executable never changes PATH, profiles or the registry automatically.

To remove the executable, exit the app and delete the executable and the bundled license document. Remove any PATH entry you added yourself if needed. Keys, state, SSH configuration, registration history and public key registrations on destinations are kept. Removing the executable is separate from unregistering keys or deleting local keys.

## Users of the v0.1.0 installer

The old default install location is `~/.local/bin` on Linux/macOS and `%LOCALAPPDATA%\Programs\ssh-keyflow` on Windows. If you have not changed the executable or the old ownership record, you can update with the new installer. It keeps the existing PATH stanza and also starts managing the bundled license document. The v0.2.0 archives are not supported by the old installer, which expects exactly one executable in the archive, so do not reuse the old script.

After a manual replacement, the executable's hash no longer matches the old ownership record, and the old script may refuse to update or remove it. From then on, update and remove manually. The old ownership records (`.ssh-keyflow-install/` on POSIX, `.ssh-keyflow-install.json` on Windows) and the existing PATH are not migrated or edited automatically. You can leave the records as they are.

To remove the old PATH stanza, check the profile indicated by the ownership record and the dedicated `SSH KeyFlow PATH begin` / `SSH KeyFlow PATH end` range, and edit only the necessary range by hand. Do not delete the whole profile or your SSH data. On Windows, the old PATH lives in the profile of the PowerShell that ran the installer, not in the registry, and Windows PowerShell and PowerShell 7 use different profiles.
