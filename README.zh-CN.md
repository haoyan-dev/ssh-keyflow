# SSH KeyFlow

简体中文 · [English](README.en.md) · [日本語](README.md)

一款用 Go 编写的 TUI / CLI 工具，用于管理本地 SSH 密钥和连接目标，并通过复制粘贴的方式辅助部署公钥和设置连接。

[首次使用](docs/zh-CN/tutorials/first-connection.md) · [文档](docs/zh-CN/README.md) · [参与贡献](CONTRIBUTING.zh-CN.md) · [发布](https://github.com/haoyan-dev/ssh-keyflow/releases)

## 功能

- 创建 Ed25519 密钥，导入现有密钥对或公钥，恢复中断的密钥创建。
- 通过 Keys / Destinations 的 TUI 和 CLI 管理密钥、连接目标及关联。
- 显示和复制面向 Linux / macOS / Windows 的公钥部署、SSH config 追加和连接步骤。
- 粘贴在连接目标上显示的主机公钥，将其固定。这样可以省去首次连接时的确认，并拒绝其他主机密钥。
- 设置通过公钥文件选择并使用 SSH 代理中的私钥（如 FIDO 密钥）。

应用本身不执行 SSH 连接或远程注册。关联只是本地的管理信息，不代表连接成功。设置由用户在目标终端上运行生成的脚本完成。从管理中移除与删除密钥文件、吊销 SSH 访问是不同的操作。

## 一行命令安装

支持 Windows / Linux / macOS 的 amd64（x64）/ arm64。不需要 Go 和管理员权限，但需要操作系统自带的 `ssh` 和 `ssh-keygen`。

macOS / Linux：

```sh
curl -fsSL https://raw.githubusercontent.com/haoyan-dev/ssh-keyflow/main/install.sh | sh
```

Windows（PowerShell）：

```powershell
irm https://raw.githubusercontent.com/haoyan-dev/ssh-keyflow/main/install.ps1 | iex
```

脚本从 `main` 获取，验证最新正式版本及其校验和后进行安装。如需先检查内容，请先保存脚本并阅读后再运行。仅当你在确认提示中同意时才会添加到 PATH。详情请参阅[安装、更新与卸载](docs/zh-CN/guides/installation.md)。

## 启动与操作

在新的终端中运行以下命令。如果没有添加到 PATH，请使用安装程序显示的完整路径。

```sh
ssh-keyflow --version
ssh-keyflow
```

TUI 需要在交互式终端中使用。准备第一个密钥和连接目标的步骤见[首次连接设置](docs/zh-CN/tutorials/first-connection.md)，操作按键见 [TUI 操作](docs/zh-CN/guides/terminal-ui.md)，命令示例见 [CLI 用法](docs/zh-CN/guides/cli.md)。

## 文档

| 目的 | 阅读页面 |
| --- | --- |
| 准备第一个密钥和连接目标 | [教程](docs/zh-CN/tutorials/first-connection.md) |
| 查找操作步骤和问题处理方法 | [文档索引](docs/zh-CN/README.md) |
| 确认 CLI、JSON 和保存状态的约定 | [应用契约](docs/zh-CN/reference/application-contract.md) |
| 参与修复或改进文档 | [参与贡献](CONTRIBUTING.zh-CN.md) |

## 安全条件与数据

管理信息保存在 `~/.ssh/ssh-keyflow/state.json`，生成的密钥保存在同一目录的 `keys/` 中，固定的主机公钥保存在 `known_hosts` 中。从旧版本迁移或降级前，请先阅读[数据、迁移与恢复](docs/zh-CN/guides/data-and-recovery.md)。`state.json.bak` 是保存上一状态的滚动更新式备份，并不是用于回退到旧版本的永久备份。

## 获取并运行 Portable 发行包

可以使用[发布](https://github.com/haoyan-dev/ssh-keyflow/releases)页面上按操作系统 / CPU 区分的归档文件和 `SHA256SUMS`。请参阅[验证与解压步骤](docs/zh-CN/guides/installation.md#获取并运行-portable-发行包)。

## 手动更新 Portable、PATH 与删除程序

请参阅[手动更新、PATH 设置与删除](docs/zh-CN/guides/installation.md#手动更新-portablepath-与删除程序)。即使移动了可执行文件，数据仍保存在主目录下。

## v0.1.0 安装程序的用户

请参阅[从旧安装程序迁移](docs/zh-CN/guides/installation.md#v010-安装程序的用户)。其中说明了所有权记录、PATH 和手动替换的处理方式。

## 开发与发布契约

开发环境和检查请参阅[参与贡献](CONTRIBUTING.zh-CN.md)，分发与发布请参阅[发布指南](docs/zh-CN/maintaining/releases.md)。发布的批准范围和完成条件以 [AGENTS.md](AGENTS.md#tags-and-releases)（英语）为准。

依赖库的许可证和版权声明见 [THIRD_PARTY_NOTICES.txt](THIRD_PARTY_NOTICES.txt)。
