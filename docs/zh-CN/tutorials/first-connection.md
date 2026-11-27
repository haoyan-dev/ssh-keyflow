# 首次连接设置

简体中文 · [English](../../en/tutorials/first-connection.md) · [日本語](../../tutorials/first-connection.md)

[文档索引](../README.md)

本教程将准备一个密钥，由你自己将其公钥部署到连接目标，然后确认 SSH 连接。SSH KeyFlow 只负责创建或导入密钥以及显示公钥，不会设置连接目标，也不会进行 SSH 连接。

## 事先准备

- [已安装的 SSH KeyFlow](../guides/installation.md) 和交互式终端。
- 本地终端上的 `ssh` 和 `ssh-keygen`。
- 一个 SSH 服务器可用、且你有权进行设置的连接目标。即使尚未注册公钥，也要能通过其他方式登录该机器。
- 连接目标的主机名和用户名。请将下文中的 `server.example` 和 `alice` 替换为你自己的连接目标。

如果你从旧版本开始使用，请在执行变更操作前先阅读[状态的迁移](../guides/data-and-recovery.md#从-v0100-及更早版本迁移)。

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

如果使用现有密钥对，请改为运行以下命令。将两个路径替换为你自己密钥对的绝对路径。在 Windows 上可使用 `C:\Users\alice\.ssh\id_ed25519` 等路径。

```sh
ssh-keyflow keys import --name laptop --public /absolute/path/id_ed25519.pub --private /absolute/path/id_ed25519
```

导入不会移动或改写原密钥文件。

```sh
ssh-keyflow keys list
```

确认管理名称 `laptop`、指纹和公钥路径。私钥路径可以通过 `keys list --json` 的 `private_path` 确认。如果创建中断，请参阅[恢复指南](../guides/troubleshooting.md#密钥创建中断)。

## 3. 将公钥部署到连接目标

在本地终端中显示公钥，并保持为一行进行复制。

```sh
ssh-keyflow keys public --key laptop
```

在**连接目标的终端**（通过其他方式登录的终端）上，将其追加到登录用户的 `authorized_keys`。Linux / macOS 连接目标的操作如下。请将 `<公钥>` 替换为复制的那一行。

```sh
mkdir -p ~/.ssh && chmod 700 ~/.ssh
printf '%s\n' '<公钥>' >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
```

对于 Windows 连接目标，标准账户追加到 `%USERPROFILE%\.ssh\authorized_keys`，管理员账户追加到 `%ProgramData%\ssh\administrators_authorized_keys`。管理员用的文件需要将写入权限限定为 SYSTEM 和 Administrators。请不要删除已有的行。

## 4. 连接

在本地终端中指定私钥进行连接。`<私钥路径>` 是在第 2 步中确认的路径。

```sh
ssh -i <私钥路径> -o IdentitiesOnly=yes alice@server.example
```

首次连接时，请通过可信的方式核对连接目标的主机密钥指纹，然后再输入 `yes`。如要省去每次的指定，请自行在 `~/.ssh/config` 中添加如下 Host 块。请确认它不与现有的 Host / Include / Match 重复。

```text
Host my-server
  HostName server.example
  User alice
  IdentityFile <私钥路径>
  IdentitiesOnly yes
```

## 5. 结果与后续操作

在你自己的 SSH 终端中确认能够连接。SSH KeyFlow 不验证连接，也不提供记录成功的操作。如果无法连接，请参阅[故障排除](../guides/troubleshooting.md)。

从管理中移除请参阅 [CLI 指南](../guides/cli.md#从管理中移除)。该操作不会吊销 SSH 访问。如果要从连接目标上移除公钥，请在确保有其他登录方式后，自行从 `authorized_keys` 中删除对应的那一行。
