# 创建和接受机群名册

简体中文 · [English](../../en/guides/fleet.md) · [日本語](../../guides/fleet.md)

[文档索引](../README.md)

> 本页说明 `main` 上尚未发布的功能（机群同步第 1 阶段）。已发布的版本中不包含此功能。

机群是以同一机群根密钥作为信任起点固定下来的你自己的机器（节点）的集合。管理节点为名册（节点及其公钥的列表）签名，其他节点用已固定的根公钥验证以文件形式收到的名册，然后再接受它。第 1 阶段只到创建和分发名册为止，不写入 `authorized_keys`、known_hosts 或 SSH config。术语请参阅[术语表](../reference/glossary.md)。

## 前提

- 签名和验证需要 OpenSSH 8.4 或更高版本。使用 PATH 上的第一个 `ssh-keygen`，并用同一目录中的 `ssh -V` 确认版本。用 `fleet init --dry-run` 等 `--dry-run` 可以确认所用 `ssh-keygen` 的路径和版本。
- 更改状态的命令会先显示计划，然后在真实终端中要求输入 `yes`。加上 `--dry-run` 时只显示计划，不做任何更改。
- 在 `main` 上首次更改状态时（不限于机群命令，也包括创建密钥等操作），v0.11.0 的 `state.json`（模式 v3）会先保存为 `state.v3-<UTC时间>.json`，然后以 v4 保存。v0.11.0 及以前的版本无法读取 v4。请事先阅读[数据、迁移与恢复](data-and-recovery.md)。

## 1. 在每台机器上设置节点描述

连接源（`client`）是发起 SSH 连接的一方，连接目标（`remote`）是被连接的一方。连接源需要一个受管密钥（`keys create` 或 `keys import`）。

```sh
# 仅作为连接源的节点
ssh-keyflow node set --name laptop --roles client --key laptop

# 同时作为连接源和连接目标的节点
ssh-keyflow node set --name desk --roles client,remote --key desk --addresses desk.lan,192.0.2.10
ssh-keyflow node show
```

连接目标的主机公钥从 `sshd_config` 的 `HostKey` 行读取（默认在 Unix 上为 `/etc/ssh/sshd_config`，在 Windows 上为 `%ProgramData%\ssh\sshd_config`）。如果文件在其他位置，请指定 `--sshd-config`。端口不是 22 时指定 `--port`，与共同的登录账户不同时指定 `--login-account`。列表值以一个逗号分隔的值指定。

## 2. 在管理节点上创建机群

选择一台用来为名册签名的机器。

```sh
ssh-keyflow fleet init --login-account user --dry-run
ssh-keyflow fleet init --login-account user
```

`--login-account` 是所有连接目标共用的登录账户。机群根密钥创建在 `~/.ssh/ssh-keyflow/fleet/root_ed25519`，`ssh-keygen` 会直接要求输入口令。不能使用空口令。此密钥无法重新创建，请将整个 `~/.ssh/ssh-keyflow/` 保存到受访问限制的位置。

如果只有签名失败，重新运行同一命令会用已创建的密钥为版本 1 签名。更改管理节点自身的节点描述后，用 `fleet sign` 为下一个版本签名。

## 3. 将其他节点加入名册

在要加入的机器上写出节点描述文件，并复制到管理节点。

```sh
ssh-keyflow node export --out ./desk-node
```

在管理节点上添加，并为下一个版本签名。

```sh
ssh-keyflow fleet add ./desk-node
```

请将显示的名称、角色和指纹与要加入的机器上的 `node show` 核对后，再输入 `yes`。再次添加同一节点时，会并列显示新旧值，作为替换处理。

## 4. 分发并接受名册

在管理节点上写出名册，并复制到要接受的机器。

```sh
ssh-keyflow fleet export --out ./roster
```

会生成 `roster.json`、`roster.json.sig` 和 `fleet-root.pub` 这 3 个文件。`--out` 必须是不存在或为空的目录。在要接受的机器上运行：

```sh
ssh-keyflow fleet accept ./roster --dry-run
ssh-keyflow fleet accept ./roster
ssh-keyflow fleet show
```

首次接受时会显示根公钥的指纹和机群 ID。请确认它们与管理节点上 `fleet show` 的显示一致，再输入 `yes`。此操作会固定根公钥，此后只接受由已固定的根签名的名册。

`roster.json` 要按签名时的字节原样处理。如果经过 Git 的换行转换或用编辑器修改，会因不是规范形式的名册而被拒绝。

## 拒绝与警告

以下情况下，`fleet accept` 不做任何更改，并以退出码 1 结束。

- 根与已固定的根不同、签名不一致、`roster.json` 被修改
- 版本低于已接受的版本、版本相同但内容不同的名册、其他机群的名册
- 目录中有这 3 个文件以外的文件、名册不是规范形式

再次接受同一名册不会做任何更改。名册比 `--max-age`（默认 30 天）更旧，或创建时间超前 5 分钟以上时，会在标准错误中警告并继续处理。在管理节点上运行 `fleet sign` 会更新创建时间。

准确的规则、JSON 和状态格式请参阅[应用契约（中文概要）](../reference/application-contract.md#机群同步第-1-阶段尚未发布)，以及作为正本的[应用契约的机群同步第 1 阶段一节](../../application-contract.md#フリート同期フェーズ1未リリース)（日语）。
