# CLI 用法

简体中文 · [English](../../en/guides/cli.md) · [日本語](../../guides/cli.md)

[文档索引](../README.md)

## 命令示例

在 CLI 中也可以直接指定相同的操作。各命令的详细说明请参阅 `--help`。

请将示例中的名称、主机名和路径替换为你自己的环境。以下是按用途分类的示例，并不是需要连续运行所有代码块的步骤。首次使用请参阅[教程](../tutorials/first-connection.md)。

### 帮助、版本与列表

```sh
ssh-keyflow --help
ssh-keyflow --version
ssh-keyflow keys list --json
ssh-keyflow hosts list --json
ssh-keyflow tui
```

### 创建、导入密钥与显示公钥

创建和导入按需选择其一。`existing` 是要导入的公钥的管理名称。

```sh
ssh-keyflow keys create --name laptop
ssh-keyflow keys import --name existing --public /absolute/path/id_ed25519.pub
ssh-keyflow keys public --key laptop
```

### 连接目标与关联

`laptop` 和 `existing` 指定已创建或已导入的密钥。

```sh
ssh-keyflow hosts add --alias my-server --hostname server.example --user alice --os linux --key laptop
ssh-keyflow hosts associate --host my-server --key existing
```

### 手动设置指南

```sh
ssh-keyflow hosts guide --host my-server --key laptop
ssh-keyflow hosts guide --host my-server --key laptop --local-os linux --identity-file /home/alice/.ssh/id_ed25519 --remote-os windows --windows-account standard
```

### 固定主机密钥

```sh
ssh-keyflow hosts pin --host my-server
ssh-keyflow hosts pin --host my-server --file host-keys.txt --yes
ssh-keyflow hosts pin --host my-server --replace
ssh-keyflow hosts unpin --host my-server
```

`hosts pin` 会显示需要在连接目标上运行的命令。粘贴在连接目标自身终端中输出的主机公钥，然后在空行处按 Enter（Windows 上在以管理员身份打开的 PowerShell 中显示）。确认每个密钥的类型和 SHA256 指纹后输入 `yes`，即会保存到 `~/.ssh/ssh-keyflow/known_hosts`。`--file` 读取保存了输出的文件；来自管道或文件的输入需要 `--yes`。如果已固定了不同的密钥，`--yes` 不会替换它。只有在连接目标自身上确认其主机密钥确实已更改时，才在真实终端中指定 `--replace`。`hosts unpin` 会在真实终端中确认后解除固定。添加到 SSH config 中用于固定的 3 行需要你自己删除。

固定后再次运行 `hosts guide`，Host 片段中会增加 `HostKeyAlias`、`UserKnownHostsFile` 和 `StrictHostKeyChecking yes`。对于 SSH config 中已有的 Host，请自行添加这 3 行。`hosts list` 在 `HOST KEYS` 列中显示固定状态。从管理中移除连接目标后固定仍会保留，如不再需要，请在移除前运行 `hosts unpin`。

### 恢复中断的密钥创建

```sh
ssh-keyflow resume --id creation-example
```

在 `keys import` 中添加 `--private /absolute/path/id_ed25519` 会验证并导入密钥对；省略则只导入公钥。请将路径替换为你环境中的绝对路径。`hosts add` 的 `--port` 默认为 22，`--os` 为 `linux` / `macos` / `windows`。Windows 还需指定 `--windows-account standard` 或 `--windows-account administrator`。`resume` 的 ID 请替换为 `keys list` 中显示的未完成本地密钥创建 ID。

## 从管理中移除与解除关联

请根据要整理的对象进行选择。以下也是相互独立的操作示例。

```sh
ssh-keyflow hosts dissociate --host my-server --key existing
ssh-keyflow hosts remove --host my-server
ssh-keyflow keys remove --key existing
```

`keys remove` / `hosts remove` 会在确认后从管理中移除对象及其全部关联。`hosts dissociate` 只解除指定组合的关联，密钥和连接目标仍会保留。在 TUI 中，于列表或详情中按 `d`；解除关联时按 `u` 并选择现有的组合。这些操作都不会更改密钥文件、SSH 设置或连接目标上的公钥注册，也不会吊销 SSH 访问。找不到文件的密钥也可以从管理中移除。如果存在未完成的旧密钥删除，则需要手动确认。为保留创建结果和旧历史，已移除的元数据会保存在状态中的 `removed_keys` / `removed_hosts` 里；重新导入或添加同一别名的连接目标时会分配新的 ID。旧二进制文件无法读取包含这些数组的状态，因此回退到旧版本时，请使用事先另行保存的状态。

连接目标用固定 ID 或别名指定，密钥用固定 ID 或唯一的管理名称指定。不会自动选择含糊的密钥名称。`hosts guide` 是明确指定连接目标和密钥的只读操作，不会保存关联。变更操作需要可用于确认的真实终端。口令直接输入到操作系统的 `ssh-keygen` 中。`list`、`keys public` 和 `hosts guide` 可以重定向，并且只做读取，不会创建或迁移存储区域。

## 指南的操作系统与私钥路径

`hosts guide` 的 `--local-os` 和 `--remote-os` 可以指定 `linux` / `macos` / `windows`。`--windows-account` 用于 Windows 连接目标。省略时使用运行应用的操作系统，以及已保存的连接目标操作系统和账户。`--identity-file` 是从本地终端看到的私钥路径；对于密钥对，如果选择了与运行应用的系统不同的本地操作系统，则必须指定。请使用所指定操作系统的绝对路径或 `~/path`。仅有公钥的密钥会将已保存的公钥路径用作 `IdentityFile`，因此本地操作系统与运行应用的系统不同时，请在 `--identity-file` 中指定从该终端看到的公钥路径。与主机密钥固定相关的步骤和行，仅在本地操作系统与运行应用的系统相同时显示。从 Windows 版粘贴到 WSL 时，请选择 `--local-os linux`，并指定 WSL 可见的对应密钥路径。应用不会验证或传输该路径上的密钥，也不会推测驱动器的对应关系。这些指定只作用于指南，不会更改已保存的连接目标或受管密钥的路径。

## JSON 与退出码

JSON 输出到标准输出，诊断信息输出到标准错误。`keys list --json` 返回 `keys` 和 `creations`，`hosts list --json` 返回 `hosts`、`associations` 和 `host_key_pins`（按连接目标 ID 列出的类型和指纹）。无法读取固定文件时会省略 `host_key_pins`，并在标准错误中输出原因。退出码：成功为 0，执行失败或取消为 1，参数或用法错误为 2。在非终端环境中不带参数启动时，会显示帮助并以 0 退出。

如需按步骤试用，请参阅[首次连接设置](../tutorials/first-connection.md)；准确的输入、输出规格请参阅[应用契约](../reference/application-contract.md#cli)。
