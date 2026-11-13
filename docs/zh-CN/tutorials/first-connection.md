# 首次连接设置

简体中文 · [English](../../en/tutorials/first-connection.md) · [日本語](../../tutorials/first-connection.md)

[文档索引](../README.md)

本教程将创建一个密钥，将它与连接目标关联，并显示手动设置指南。最后的 SSH 连接由你自己运行并确认。仅凭应用内保存的关联，连接目标上的设置不会发生任何变化。

## 事先准备

- [已安装的 SSH KeyFlow](../guides/installation.md) 和交互式终端。
- 本地终端上的 `ssh` 和 `ssh-keygen`。
- 一个 SSH 服务器可用、且你有权进行设置的连接目标。即使尚未注册公钥，也要能通过其他方式登录该机器。
- 连接目标的主机名、用户名和操作系统。请将下文中的 `server.example` 和 `alice` 替换为你自己的连接目标。

如果你从旧版本开始使用，请在执行变更操作前先阅读[状态的备份保存与迁移](../guides/data-and-recovery.md)。

## 1. 确认可以启动

在本地终端中运行。Windows 上也可以使用相同的命令。

```sh
ssh-keyflow --version
ssh-keyflow keys list
```

如果显示了版本并能读取密钥列表，即可进入下一步。没有密钥也没关系。开发构建的版本为 `dev`。仅读取列表不会创建或迁移存储区域。

## 2. 准备密钥

如需新密钥：

```sh
ssh-keyflow keys create --name laptop
```

检查显示的内容，并在确认提示中输入 `yes`。口令直接输入到操作系统的 `ssh-keygen` 中。不会覆盖已有路径。

如果使用现有密钥，请改为运行以下命令。将两个路径替换为你自己密钥对的绝对路径。在 Windows 上可使用 `C:\Users\alice\.ssh\id_ed25519` 等路径。

```sh
ssh-keyflow keys import --name laptop --public /absolute/path/id_ed25519.pub --private /absolute/path/id_ed25519
```

导入不会移动或改写原密钥文件。省略 `--private` 时只导入公钥。此时指南会将公钥文件用作 `IdentityFile`，并从 SSH 代理（如 FIDO 密钥）使用对应的私钥。

```sh
ssh-keyflow keys list
ssh-keyflow keys public --key laptop
```

确认管理名称 `laptop` 和公钥。如果存在多个相同的管理名称，请使用列表中的固定 ID。如果创建中断，请参阅[恢复指南](../guides/troubleshooting.md#密钥创建中断)。

## 3. 保存连接目标

以下是 Linux 连接目标的示例。`my-server` 是你自己选择的别名，请选择一个与现有 SSH config 不重复的名称。

```sh
ssh-keyflow hosts add --alias my-server --hostname server.example --user alice --os linux --key laptop
ssh-keyflow hosts list
```

确认后，连接目标以及它与密钥的关联会保存在本地。默认 SSH 端口为 22。其他端口用 `--port` 指定，macOS 连接目标用 `--os macos` 指定。Windows 连接目标需要 `--os windows --windows-account standard`，管理员账户则需要 `--windows-account administrator`。

## 4. 打开手动设置指南

```sh
ssh-keyflow hosts guide --host my-server --key laptop
```

输出会标明运行每个步骤的终端和脚本。仅显示指南不会更改任何设置。

1. 在**连接目标的终端**上，检查并运行公钥部署步骤。
2. 如要省去首次连接时的主机密钥确认，请完成接下来的两个步骤：在**连接目标的终端**上显示主机公钥（Windows 上使用管理员 PowerShell），然后在**本地终端**中将输出粘贴到 `ssh-keyflow hosts pin --host <ID>`。确认指纹并输入 `yes` 后，重新打开指南。config 追加步骤中会增加 3 行用于固定的设置，显示和固定步骤则会消失。
3. 在**本地终端**中，确认现有的 Host / Include / Match / 通配符及其优先级后，运行 config 追加步骤。如果已存在相同的显式别名，追加会停止。
4. 对于带口令或无法判断是否带口令的密钥，请阅读面向**本地终端**的 ssh-agent 步骤并完成所需设置。Windows 的服务设置需要管理员 PowerShell。在 Linux 上需要自行选择和设置 agent。
5. 在**本地终端**中运行最后的连接步骤，本例中为 `ssh my-server`。如果没有固定主机密钥，首次连接时请通过可信的方式核对连接目标的主机密钥。

如果要粘贴到与运行应用的操作系统不同的系统上，请先阅读 [CLI 中指南的操作系统指定](../guides/cli.md)。应用不会将密钥传输到其他机器，也不会推测路径的对应关系。

## 5. 结果与后续操作

在你自己的 SSH 终端中确认能够连接。SSH KeyFlow 不验证连接，也不提供记录成功的操作。如果无法连接，请参阅[故障排除](../guides/troubleshooting.md)。

如需在 TUI 中进行同样的管理，请启动 `ssh-keyflow` 并参阅 [TUI 操作](../guides/terminal-ui.md)。在指南中，只能用 `y` 复制当前显示的步骤。

解除关联和从管理中移除请参阅 [CLI 指南](../guides/cli.md)。这两种操作都不会吊销 SSH 访问。如果要从连接目标上移除公钥，请阅读手动设置指南中的取消注册说明，并在确保有其他登录方式后再操作。
