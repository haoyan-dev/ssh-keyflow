# CLI 用法

简体中文 · [English](../../en/guides/cli.md) · [日本語](../../guides/cli.md)

[文档索引](../README.md)

## 命令示例

SSH KeyFlow 只通过 CLI 操作。各命令的详细说明请参阅 `--help`。不带参数启动时会显示帮助。

请将示例中的名称和路径替换为你自己的环境。以下是按用途分类的示例，并不是需要连续运行所有代码块的步骤。首次使用请参阅[教程](../tutorials/first-connection.md)。

### 帮助、版本与列表

```sh
ssh-keyflow --help
ssh-keyflow --version
ssh-keyflow keys list
ssh-keyflow keys list --json
```

### 创建、导入密钥与显示公钥

创建和导入按需选择其一。导入需要同时提供公钥和私钥的路径。

```sh
ssh-keyflow keys create --name laptop
ssh-keyflow keys import --name existing --public /absolute/path/id_ed25519.pub --private /absolute/path/id_ed25519
ssh-keyflow keys public --key laptop
```

`keys public` 将公钥以一行输出到标准输出，不包含文件中的注释。可用于重定向或通过管道传给剪贴板。

```sh
ssh-keyflow keys public --key laptop > laptop.pub
ssh-keyflow keys public --key laptop | pbcopy            # macOS
ssh-keyflow keys public --key laptop | Set-Clipboard     # Windows PowerShell
```

### 恢复中断的密钥创建

```sh
ssh-keyflow resume --id creation-example
```

请将 ID 替换为 `keys list` 中显示的未完成本地密钥创建 ID。

## 从管理中移除

```sh
ssh-keyflow keys remove --key existing
```

确认后将密钥从受管对象中去掉。不更改密钥文件、SSH 设置或连接目标上的公钥注册，也不会吊销 SSH 访问。找不到文件的密钥也可以从管理中移除。同一密钥对可以用 `keys import` 重新纳入管理（会分配新的 ID）。

在 `main` 上，用作节点描述中连接源密钥的密钥无法移除。请先用 `node set` 改为其他密钥。

密钥用固定 ID 或唯一的管理名称指定。不会自动选择含糊的名称。变更操作（create / import / remove / resume）需要可用于确认的真实终端。口令直接输入到操作系统的 `ssh-keygen` 中。`keys list` 和 `keys public` 可以重定向，并且只做读取，不会创建或迁移存储区域。

## 机群命令（尚未发布）

`main` 上有 `node` / `fleet` 命令，用于在节点之间以文件形式传递已签名的名册。已发布的版本中没有这些命令。

```sh
ssh-keyflow node set --name laptop --roles client --key laptop
ssh-keyflow node show --json
ssh-keyflow fleet show --json
```

步骤请参阅[创建和接受机群名册](fleet.md)。

## 已废弃的命令

v0.11.0 已废弃 `hosts`（add / list / remove / associate / dissociate / guide / pin / unpin）、`tui` 和 `keys import --public-only`。指定这些命令会产生用法错误（退出码 2）。公钥的部署和 SSH config 的设置请自行完成。步骤示例见[教程](../tutorials/first-connection.md)。

## JSON 与退出码

JSON 输出到标准输出，诊断信息输出到标准错误。`keys list --json` 只返回 `keys` 和 `creations`。`main` 上的 `node show --json` 返回 `node` 和 `is_admin`，`fleet show --json` 返回 `fleet`。退出码：成功为 0，执行失败或取消为 1，参数或用法错误为 2。

准确的输入、输出规格请参阅[应用契约（中文概要）](../reference/application-contract.md#cli)，以及作为正本的[应用契约的 CLI 一节](../../application-contract.md#cli)（日语）。
