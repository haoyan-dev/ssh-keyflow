# 故障排除

简体中文 · [English](../../en/guides/troubleshooting.md) · [日本語](../../guides/troubleshooting.md)

[文档索引](../README.md)

## 找不到命令

打开新的终端后重试。如果拒绝了添加到 PATH，请使用安装程序显示的程序完整路径。Windows PowerShell 和 PowerShell 7 的配置文件不同。请参阅[安装位置与 PATH 的说明](installation.md)。

## 变更操作被拒绝

创建、导入、移除和恢复要求在真实终端中确认。请不要用通过管道传入的 `yes` 代替确认，而是在交互式终端中运行。列表和公钥的显示可以重定向。

## 显示 `unknown command "hosts"` 等信息

`hosts`、`tui` 和 `--public-only` 已在 v0.11.0 中废弃。请参阅 [CLI 用法](cli.md#已废弃的命令)和[迁移说明](data-and-recovery.md#从-v0100-及更早版本迁移)。

## 想要复制公钥

将以下命令的输出导入剪贴板或编辑器。请将名称替换为你自己的管理名称或固定 ID。

```sh
ssh-keyflow keys public --key laptop
```

公钥请保持为一行，不要把显示上的自动折行当作换行粘贴。

## 密钥创建中断

```sh
ssh-keyflow keys list
ssh-keyflow resume --id creation-example
```

将 `creation-example` 替换为列表中未完成的创建 ID。恢复时使用相同的 ID 和保存路径。不要删除部分生成的文件后重新创建，也不要用其他密钥覆盖同一位置；如果出现指纹或文件不一致，请检查其内容。

## 出现密钥、状态或锁的错误

请检查目标文件是否被移动或替换、权限是否正确，以及是否有其他进程在运行。应用不会自动修复不明的锁或无效的状态。如果有其他操作正在进行，请等待其结束。不要无条件删除锁或状态，请停止应用并参阅[数据与恢复](data-and-recovery.md)。

## 显示 `state.json uses schema v1` / `v2`

这是旧版本的状态。下一次变更操作会归档原文件，并只沿用密钥对。请先确认显示的“Not carried over”内容，然后再操作。详情见[迁移说明](data-and-recovery.md#从-v0100-及更早版本迁移)。

## 部署公钥后仍无法 SSH 连接

SSH KeyFlow 不验证连接。请确认连接目标的 SSH 服务器、网络、登录用户、实际的 `AuthorizedKeysFile` / `Match`、`authorized_keys` 及其父目录的权限，以及本地 SSH config 中的 `IdentityFile`。Windows 管理员账户使用 `%ProgramData%\ssh\administrators_authorized_keys`。

如果反复要求输入口令，请将密钥加载到操作系统的 ssh-agent 中（`ssh-add <私钥路径>`）。agent 退出后，已加载的密钥不会保留。

## 安装程序拒绝更新或删除

没有所有权记录的现有文件，或安装后被修改的文件，不会被覆盖。手动替换二进制文件后，所有权哈希也会不再一致。请参阅[旧安装程序与手动维护](installation.md#v010-安装程序的用户)，并且不要删除其他文件或整个配置文件。

## 报告中应包含的信息

请按照[参与贡献](../../../CONTRIBUTING.zh-CN.md)中的报告步骤，记录版本、操作系统 / CPU、终端、运行的命令、预期结果、实际的诊断信息以及最小复现步骤。请勿包含私钥、口令或认证令牌，主机名、用户名、路径和状态内容也请在公开前检查。
