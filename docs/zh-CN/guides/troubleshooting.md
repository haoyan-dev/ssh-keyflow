# 故障排除

简体中文 · [English](../../en/guides/troubleshooting.md) · [日本語](../../guides/troubleshooting.md)

[文档索引](../README.md)

## 找不到命令

打开新的终端后重试。如果拒绝了添加到 PATH，请使用安装程序显示的程序完整路径。Windows PowerShell 和 PowerShell 7 的配置文件不同。请参阅[安装位置与 PATH 的说明](installation.md)。

## TUI 无法打开，或变更操作被拒绝

TUI 的输入和输出都需要交互式终端。变更操作也要求在真实终端中确认。请不要用通过管道传入的 `yes` 代替确认，而是在交互式终端中运行。在非终端环境中不带参数启动时会显示帮助。列表、公钥和指南的显示可以重定向。

## 无法复制公钥或指南

TUI 的复制功能需要支持 OSC52 的终端。如果终端不支持，请将以下命令的输出导入编辑器或操作系统的剪贴板。请将名称替换为你自己的管理名称、别名或固定 ID。

```sh
ssh-keyflow keys public --key laptop
ssh-keyflow hosts guide --host my-server --key laptop
```

公钥请保持为一行，不要把显示上的自动折行当作换行粘贴。指南中的步骤请按终端分别运行。

## 密钥创建中断

```sh
ssh-keyflow keys list
ssh-keyflow resume --id creation-example
```

将 `creation-example` 替换为列表中未完成的创建 ID。恢复时使用相同的 ID 和保存路径。不要删除部分生成的文件后重新创建，也不要用其他密钥覆盖同一位置；如果出现指纹或文件不一致，请检查其内容。

## 出现密钥、状态或锁的错误

请检查目标文件是否被移动或替换、权限是否正确，以及是否有其他进程在运行。应用不会自动修复不明的锁或无效的状态。如果有其他操作正在进行，请等待其结束。不要无条件删除锁或状态，请停止应用并参阅[数据与恢复](data-and-recovery.md)。存在未完成的旧密钥删除的密钥需要手动确认。

## 运行指南后仍无法 SSH 连接

关联、显示和复制都不能证明连接成功。请确认指南指定的操作系统、Windows 账户类型、粘贴位置和私钥路径。也请确认连接目标的 SSH 服务器、网络、实际的 `AuthorizedKeysFile` / `Match` 以及现有 config 的优先级。这些都不在应用自动验证的范围内。

如果反复要求输入口令，请检查指南中的 ssh-agent 步骤。agent 退出后，已加载的密钥不会保留。macOS 的钥匙串、Linux 的 agent 管理和 Windows 服务请按照各操作系统的步骤操作。

## 显示 Host key verification failed

对于已固定主机密钥的连接目标，固定密钥以外的密钥都会被拒绝。如果重装了连接目标或更新了主机密钥，请先在连接目标自身的终端中确认新的主机公钥，然后在真实终端中运行 `ssh-keyflow hosts pin --host <ID> --replace`。原因不明的变更可能是攻击，请不要替换。解除固定后，也请删除添加到 SSH config 中的 `HostKeyAlias`、`UserKnownHostsFile` 和 `StrictHostKeyChecking` 这 3 行。如果 `hosts list` 显示 `unknown`，请检查 `~/.ssh/ssh-keyflow/known_hosts`。应用不会改写无效的固定文件。

如果仅有公钥的密钥无法连接，请确认对应的私钥是否已加载到代理中。在 Windows 上出现 `UNPROTECTED PRIVATE KEY FILE` 警告，同样表示代理中没有该密钥。

## 安装程序拒绝更新或删除

没有所有权记录的现有文件，或安装后被修改的文件，不会被覆盖。手动替换二进制文件后，所有权哈希也会不再一致。请参阅[旧安装程序与手动维护](installation.md#v010-安装程序的用户)，并且不要删除其他文件或整个配置文件。

## 报告中应包含的信息

请按照[参与贡献](../../../CONTRIBUTING.zh-CN.md)中的报告步骤，记录版本、操作系统 / CPU、终端、运行的命令、预期结果、实际的诊断信息以及最小复现步骤。请勿包含私钥、口令或认证令牌，主机名、用户名、路径和状态内容也请在公开前检查。
