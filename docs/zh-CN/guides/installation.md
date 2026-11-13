# 安装、更新与卸载

简体中文 · [English](../../en/guides/installation.md) · [日本語](../../guides/installation.md)

[文档索引](../README.md)

支持 Windows / Linux / macOS 的 amd64 / arm64。请准备操作系统自带的 OpenSSH（`ssh` 和 `ssh-keygen`）。下文中的固定版本只是演示指定方式的示例。通常无需指定版本即可获取正式版本；回退到旧版本前，请先阅读[数据与恢复](data-and-recovery.md)。

## 一行命令安装

使用以下命令即可安装最新正式版本。

**macOS / Linux**

```sh
curl -fsSL https://raw.githubusercontent.com/haoyan-dev/ssh-keyflow/main/install.sh | sh
```

**Windows（PowerShell）**

```powershell
irm https://raw.githubusercontent.com/haoyan-dev/ssh-keyflow/main/install.ps1 | iex
```

安装程序会判断操作系统和 CPU（x64 / ARM64），获取最新正式版本和 `SHA256SUMS`。验证校验和与归档内容后，将程序本体和许可证文档安装到用户专用的位置。不需要 Go 和管理员权限，但需要另外准备操作系统的 `ssh` 和 `ssh-keygen`。如需先检查要获取的脚本，请先保存并阅读其内容后再运行。

默认安装位置为 macOS/Linux 上的 `~/.local/bin`，Windows 上的 `%LOCALAPPDATA%\Programs\ssh-keyflow`。是否添加到 PATH 会在交互式终端中询问，仅在你同意时才写入 shell 的配置文件（profile）。请打开新的终端并运行以下命令。

```sh
ssh-keyflow --version
ssh-keyflow
```

如果拒绝添加到 PATH，或在非交互环境中运行，可以使用安装程序显示的程序完整路径启动。Windows 上的 PATH 保存只适用于运行安装程序的那种 PowerShell 的配置文件。在其他种类的 PowerShell 或命令提示符中，请按照显示的 PATH 手动设置说明操作。

再次运行同一命令即可更新。更新前请退出 SSH KeyFlow。安装程序只管理程序本体、许可证文档以及它自己追加的 PATH 设置，不会更改密钥、状态、SSH 设置或连接目标上的公钥注册。没有所有权记录的现有文件或已被修改的文件不会被覆盖。

如需指定版本或安装位置，或要卸载，请保存脚本后再运行。

```sh
# macOS / Linux：指定 v0.2.1，并显式添加到 PATH
curl -fsSLo install.sh https://raw.githubusercontent.com/haoyan-dev/ssh-keyflow/main/install.sh
sh install.sh --version v0.2.1 --add-to-path
# 先退出程序，然后只删除安装程序所拥有的程序本体、许可证文档和 PATH 追加内容
sh install.sh --uninstall
```

```powershell
# Windows：指定 v0.2.1，并显式添加到 PATH
Invoke-WebRequest -Uri https://raw.githubusercontent.com/haoyan-dev/ssh-keyflow/main/install.ps1 -OutFile install.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\install.ps1 -Version v0.2.1 -AddToPath
# 先退出程序，然后删除
powershell -NoProfile -ExecutionPolicy Bypass -File .\install.ps1 -Uninstall
```

上面的 Windows 示例只为该进程指定执行策略。如果使用 PowerShell 7 管理，请将 `powershell` 替换为 `pwsh`。安装位置用 `--install-dir /absolute/path` / `-InstallDir C:\absolute\path` 指定，目标配置文件用 `--profile /absolute/path` / `-ProfilePath C:\absolute\profile.ps1` 指定。如果使用了自定义安装位置，更新和删除时也请使用相同的指定。如需使用其他仓库，可以指定 `--repository OWNER/REPO` / `-Repository OWNER/REPO`。

## 获取并运行 Portable 发行包

提供适用于 Windows / Linux / macOS 的 amd64（x64）/ arm64 归档。Windows 为 zip，Linux/macOS 为 tar.gz。不需要 Go 或安装程序，但需要操作系统的 `ssh` 和 `ssh-keygen`。TUI 需要在交互式终端中使用。

分发来源为 [GitHub Releases](https://github.com/haoyan-dev/ssh-keyflow/releases)。在对应标签的 6 个归档和 `SHA256SUMS` 中，请获取适用于你的操作系统 / CPU 的归档和 manifest。

示例：在 Linux amd64 上使用 v0.2.0 时，按如下方式验证并解压。

```sh
sha256sum --ignore-missing -c SHA256SUMS
mkdir -p ./ssh-keyflow-portable
tar -xzf ssh-keyflow_v0.2.0_linux_amd64.tar.gz -C ./ssh-keyflow-portable
./ssh-keyflow-portable/ssh-keyflow --version
./ssh-keyflow-portable/ssh-keyflow
```

在 macOS 上，请将 `shasum -a 256 ssh-keyflow_v0.2.0_darwin_arm64.tar.gz` 等命令的结果与 manifest 中对应的唯一一行核对。在 Windows PowerShell 中按如下方式核对。

```powershell
Get-FileHash .\ssh-keyflow_v0.2.0_windows_amd64.zip -Algorithm SHA256
Get-Content .\SHA256SUMS
# 确认哈希与目标文件名对应的唯一一行一致后再解压
Expand-Archive .\ssh-keyflow_v0.2.0_windows_amd64.zip .\ssh-keyflow-portable
& .\ssh-keyflow-portable\ssh-keyflow.exe --version
& .\ssh-keyflow-portable\ssh-keyflow.exe
```

SHA256 用于确认完整性，并不是签名验证。每个归档都包含可执行文件和 `THIRD_PARTY_NOTICES.txt`。请选择用户可写入的任意位置放置程序；移动或共享分发物时，请一并保留附带的许可证文档。Portable 指的是可执行文件可以自由放置，并不包含在用户之间携带密钥或管理数据的机制。

## 手动更新 Portable、PATH 与删除程序

更新前请退出应用，验证新分发物的校验和，并解压到另一个目录。然后用新文件替换现有的可执行文件和附带的许可证文档，并确认 `--version`。即使移动了程序本体，它仍使用原来主目录下的状态和密钥。过去生成的设置也会原样保留。

仅在需要时自行设置 PATH。如果只对当前会话生效，在 POSIX shell 中运行 `export PATH="/absolute/path/ssh-keyflow-portable:$PATH"`，在 PowerShell 中运行 `$env:PATH = 'C:\absolute\path\ssh-keyflow-portable;' + $env:PATH`。如需永久生效，请自行编辑所用 shell 的配置文件或操作系统的 PATH 设置。程序本体不会自动更改 PATH、配置文件或注册表。

删除程序本体时，请先退出应用，然后删除可执行文件和附带的许可证文档。你自己添加的 PATH 请根据需要移除。密钥、状态、SSH 设置、注册历史以及连接目标上的公钥注册都会保留。删除程序本体与取消注册或删除本地密钥是不同的操作。

## v0.1.0 安装程序的用户

旧版的默认安装位置为 Linux/macOS 上的 `~/.local/bin`，Windows 上的 `%LOCALAPPDATA%\Programs\ssh-keyflow`。如果程序本体和旧的所有权记录没有被修改，可以用新的安装程序更新。新安装程序会保留现有的 PATH 追加内容，并开始管理附带的许可证文档。v0.2.0 及以后的归档不兼容只要求一个可执行文件的旧安装程序，因此请不要重复使用旧脚本。

手动替换后，程序本体的哈希与旧的所有权记录不再一致，旧脚本的更新和删除也可能被拒绝。此后请手动进行更新和删除。旧的所有权记录（POSIX 上的 `.ssh-keyflow-install/`，Windows 上的 `.ssh-keyflow-install.json`）和现有 PATH 不会被自动迁移或自动编辑。这些记录可以原样保留。

如需移除旧的 PATH 追加内容，请确认所有权记录所指的配置文件以及专用的 `SSH KeyFlow PATH begin` / `SSH KeyFlow PATH end` 范围，只手动编辑必要的部分。不要删除整个配置文件或 SSH 数据。Windows 上的旧 PATH 位于运行安装程序的 PowerShell 的配置文件中，而不是注册表中；Windows PowerShell 和 PowerShell 7 的目标文件不同。
