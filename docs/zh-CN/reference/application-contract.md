# 应用契约（概要）

简体中文 · [English](../../en/reference/application-contract.md) · [日本語](../../application-contract.md)

> 本页是中文概要，只摘要说明契约的要点，不复制全部标志和模式字段。具有约束力的正本是[应用契约](../../application-contract.md)（主要以日语撰写，标题和前两节为英语），两者如有出入，以该页为准。

[文档索引](../README.md) · [术语表](glossary.md)

## 从源代码分发的一行命令安装程序

- `install.sh`（Linux/macOS 上的 POSIX sh）和 `install.ps1`（Windows PowerShell 5.1 或 PowerShell 7）从 `main` 单独分发，安装现有的 Portable 发布归档，不更改发布资产。默认获取稳定、非草稿、非预发布的最新版本；显式 `--version` / `-Version` 可以选择预发布版本。默认仓库为 `haoyan-dev/ssh-keyflow`，可用 `--repository` / `-Repository` 或 `SSH_KEYFLOW_REPOSITORY` 覆盖。
- 安装前要求归档恰好有一条匹配的 SHA256SUMS 校验和，并拒绝意外的归档条目；安装可执行文件和 `THIRD_PARTY_NOTICES.txt`。默认位置为 `$HOME/.local/bin` 和 `$env:LOCALAPPDATA/Programs/ssh-keyflow`。沿用 v0.1.0 的所有权记录；未拥有、已更改或缺失的已拥有文件会停止安装或删除，替换失败时恢复之前的状态。
- 更改 PATH 需要在终端中同意，或显式指定 `--add-to-path` / `-AddToPath`；通过管道传入的内容绝不被当作回答。卸载只删除安装程序所拥有的内容。SSH 密钥、状态、SSH 设置和远程注册绝不会被修改。更新或删除前必须退出应用。

正本（该节原文为英语，以原文为准）：[Source-hosted one-line installers](../../application-contract.md#source-hosted-one-line-installers)

## Portable 发布

- 发布恰好包含 6 个归档（Windows/Linux/macOS 的 amd64/arm64；Windows 为 zip，其余为 tar.gz）和 `SHA256SUMS`，共 7 个资产，不包含安装程序。每个归档只包含对应平台的可执行文件和 `THIRD_PARTY_NOTICES.txt`。由 `scripts/package_release.py --version <tag>` 构建。
- Portable 指可执行文件可自由放置。存储位置始终为 `<Home>/.ssh/ssh-keyflow`；移动、替换或删除可执行文件不会迁移数据，也不会更改 SSH 密钥、状态、config、远程注册、PATH、配置文件或安装程序记录。最终用户需要操作系统的 OpenSSH，不需要 Go。
- 旧的 v0.1.0 安装程序会拒绝新归档；完好的 v0.1.0 安装请使用新的源代码分发安装程序。手动替换二进制文件后，更新或删除可能被拒绝。[安装指南](../guides/installation.md)中说明了手动维护以及可选地手动清理 PATH 段落，不提供自动修复。
- CI 运行测试并进行 6 个目标的交叉构建，发布工作流在 CI 之后上传 7 个资产。基于测试夹具（fixture）的制品测试和交叉构建不能证明每个目标上的原生运行行为。

正本（该节原文为英语，以原文为准）：[Portable release](../../application-contract.md#portable-release)

## 当前功能范围（v0.11.0 及以后）

- SSH KeyFlow 是专注于管理本机 SSH 密钥对的 CLI。提供 Ed25519 密钥的创建、现有密钥对的导入、中断创建的恢复、从管理中移除、列表，以及可供复制的公钥显示。根据 ADR 0007，连接目标（`hosts`）与关联、手动设置指南、片段和 ssh-agent 步骤、主机密钥固定、终端 UI 以及仅导入公钥的功能已在 v0.11.0 中废弃。这是有意的不兼容变更，不追溯适用于 v0.10.0 及以前的二进制文件。分发、安装程序和发布流程保持不变。
- 应用不运行 SSH 进程、远程认证或探测、远程注册，也不生成脚本。不读取、编辑、重新生成或删除 `~/.ssh/config`、`~/.ssh/known_hosts`、Include 和过去的专用 config。v0.10.0 及以前创建的 `<Home>/.ssh/ssh-keyflow/known_hosts`（主机密钥固定文件）以及用户从指南追加的 SSH config 行也原样保留。将公钥部署到连接目标是用户的手动操作，复制 `keys public` 的输出来使用。
- `main` 包含 ADR 0008 的机群同步第 1 阶段（`node` / `fleet` 命令、状态模式 v4），作为尚未发布的功能。其范围和规则见[机群同步第 1 阶段](#机群同步第-1-阶段尚未发布)，不适用于已发布的版本。第 1 阶段同样不进行 SSH 连接或远程注册，也不写入 SSH 文件。

正本：[現行機能の範囲（v0.11.0以降）](../../application-contract.md#現行機能の範囲v0110以降)（日语）

## CLI

- 入口为 `cli.Run(ctx, args, cli.Options{Home, Version, Input, Output, Error, Runner}) int`，`cmd/ssh-keyflow` 返回退出码。Home 的解析推迟到需要的操作时进行。
- 命令为 `keys list` / `public` / `create` / `import` / `remove` 和 `resume`。`keys list` 可选 `--json`，显示受管密钥和未完成的本地创建；`keys public` 以一行显示公钥的算法和 blob，不含文件中的注释；`keys create` 创建 Ed25519 密钥对；`keys import` 必须指定 `--name`、`--public`、`--private`，验证现有密钥对后纳入管理，不移动文件；`keys remove` 从管理中去掉并保留密钥文件；`resume` 仅恢复未完成的本地创建。各命令的必需和可选输入见正本中的命令表。`main` 上的 `node` / `fleet` 命令见[机群同步第 1 阶段](#机群同步第-1-阶段尚未发布)。`keys remove` 会以执行失败（1）拒绝用作节点描述中连接源密钥的密钥，并提示先用 `node set` 更改。
- 密钥按固定 ID 或唯一且完全一致的管理名称解析。含糊或不存在的对象会被拒绝。不提供 `hosts`、`tui`、`register`、`unregister`、`registrations list`、`keys delete`、`--public-only`、`--new-key`、`--auth`、`--identity`、`--configure`，指定时为用法错误。
- 变更操作（create / import / remove / resume）在初始化存储或运行进程之前要求真实终端输入，并向共享的 `app.Run` 传入显式 Request 和确认函数。只有去除首尾空白后完全一致的 `yes` 才会被接受。`ReadConfirmation` 逐字节读取一行，不预读后续的原生口令输入。不提供秘密值或无条件同意的标志。
- 退出码：成功 0，执行失败或取消 1，参数或用法错误 2。含糊或不存在的对象也是参数错误，stderr 输出对应命令的 usage；运行时诊断不附带 usage。`--help` 和 `--version` 不解析 Home、不检查依赖命令、不保存。版本显示为 `SSH KeyFlow <version>`。不带参数启动时，无论终端类型如何，都输出帮助并以 0 退出。
- list 和公钥可以重定向，不探测可写性或 SSH 命令。list 的普通输出为表格。JSON 是以换行结尾的单个对象，只有 `keys` 和 `creations` 两个字段。数组为空数组而非 null，元素的字段名与保存模式一致。需要确认的信息和迁移预告只输出到 stderr，已验证的状态记录仍可读取。无效状态返回 1，不输出 JSON，并保留原文件。`keys public` 将末尾带换行的复制用文本输出到 stdout。输出失败视为执行失败。

正本：[CLI](../../application-contract.md#cli)（日语）

## 共享后端

- `app.Request{Operation, Name, PublicPath, PrivatePath, Key, Resume}` 的 Operation 为 `create` / `import` / `remove` / `resume`。未知的 Operation 是参数错误。`app.Run` 要求显式 Request，变更对象在锁下也会重新解析和验证。`remove` 不检查 `ssh-keygen`，也能整理密钥文件缺失的密钥的管理信息。
- `app.Snapshot(Options)` 返回 `View{State, Review}`，不初始化存储、不探测依赖命令、不进行迁移。Review 包含密钥文件的缺失或替换，以及旧模式的状态将在下一次变更时迁移的预告。`app.PublicKey(home, keyRef)` 重新检查目标密钥的指纹，返回 `<type> <base64>`。
- 从管理中移除在真实终端中以完全一致的 `yes` 确认后，只从状态中去除该密钥的记录。不更改密钥文件、SSH 设置或连接目标上的公钥注册，也不意味着吊销访问。不保留已移除的记录。同一密钥对可以通过重新导入以新的 ID 纳入管理。

正本：[共通バックエンド](../../application-contract.md#共通バックエンド)（日语）

## 保存模式与兼容性

- 存储区域为 `<Home>/.ssh/ssh-keyflow`，使用 `state.json`、生成密钥用的 `keys/`、滚动更新式的 `state.json.bak`、临时的 `state.lock`，以及从旧模式迁移时的 `state.v<N>-<UTC时间>.json`。没有状态时只返回空状态，读取时不创建文件。
- v0.11.0 唯一写入的模式是 v3，只包含 `version`、必需数组 `keys` 和可选数组 `creations`。`main`（第 1 阶段，尚未发布）只写入 v4。v4 在 v3 的字段之外增加可选的 `node` 和 `fleet`，`version` 为 4（见[状态模式 v4](#状态模式-v4)）。状态记录分为 Key 和 Creation，各自的保存字段见正本中的字段表。
- 密钥 ID 为固定的随机 `key-<128-bit hex>`，指纹为 OpenSSH 公钥 blob 的 SHA256 格式，路径为绝对路径。导入保存 `origin: imported` 和非零的 UTC RFC3339Nano `imported_at`，创建保存 `origin: created` 和 `created_at`。私钥摘要用于在不要求口令的情况下检测替换，不保存私钥内容。完成的创建在添加密钥的同一次保存中从 `creations` 去除。
- v3 的读取是严格的：无效 JSON 或带有尾随内容的 JSON、重复或仅大小写不同的字段、未知字段、不支持的版本、缺少 `keys`、无效元数据、重复的 ID / 指纹 / 路径，都会被拒绝，不自动恢复或覆盖。v4 的读取也遵循同样的规则。

正本：[保存スキーマと互換性](../../application-contract.md#保存スキーマと互換性)（日语）

### 从 v1 / v2 迁移

- v1 和 v2（v0.10.0 及以前）在读取时于内存中转换为 v3，仅读取不会更改文件。旧模式的读取同样严格，会拒绝未知字段、类型错误的数组、带后续内容的 JSON 以及缺少必需数组（`keys` / `hosts` / `registrations`）的文件。转换只沿用带有私钥路径的密钥和未完成的本地创建，不沿用以下内容：连接目标、关联和已归档的移除记录（`hosts` / `associations` / `removed_keys` / `removed_hosts`）；旧的注册、连接确认、设置、SSH config 编辑、引导和删除的记录与历史（`registrations` / `history` / `configs` / `config_edits` / `onboardings` / `deleted_keys` / `deletions`）；仅有公钥的密钥和存在未完成旧密钥删除的密钥；密钥的 `connection_check`、连接目标的 `config_key_id` 等旧字段。
- 预告按记录类型列出每种非空记录的数量。每个未完成的 SSH config 编辑都会单独列出路径，并提示手动检查该文件。`keys list` 在迁移前通过 stderr 输出此预告。变更命令（create / import / remove / resume）在获取锁之后、确认提示之前显示同样的预告，因此确认也包含对迁移的批准。首次成功保存状态时，先以独占方式将原 `state.json` 的字节写出到同一目录下新的 `state.v<N>-<UTC时间>.json`（冲突时加数字后缀），然后更新 `.bak` 并保存 v3。归档创建失败时不更改状态。如果之前的尝试已留下字节相同的归档，重试时会复用它，不会再增加相同的归档。归档不会被覆盖或自动删除，因此未沿用的记录也保留在磁盘上。
- 旧二进制文件无法读取 v3。降级时请退出应用，手动将归档恢复为 `state.json`。迁移后创建或导入的密钥不包含在恢复的状态中。

正本：[v1 / v2からの移行](../../application-contract.md#v1--v2からの移行)（日语）

## 机群同步第 1 阶段（尚未发布）

本节描述 `main` 上的实现，在包含第 1 阶段的 MINOR 版本发布之前，不属于任何已发布的版本。设计决策见 [ADR 0008](../../adr/0008-signed-fleet-roster.md)（日语）。第 1 阶段只涉及：将由机群根密钥签名的名册以文件形式交给其他节点，接收的节点用已固定的根公钥验证。不包括写入 `authorized_keys`、known_hosts 或 SSH config，通过网络加入或同步，吊销以及密钥轮换。

正本（整节）：[フリート同期フェーズ1（未リリース）](../../application-contract.md#フリート同期フェーズ1未リリース)（日语）

### 命令

- `node set` 创建或替换本机的节点描述，首次生成节点 ID 并在之后保留。必须指定 `--name` 和 `--roles`（`client` / `remote` 中的一个或多个，以一个逗号分隔的值指定）。连接源必须指定 `--key`（受管密钥，创建或导入均可）；连接目标必须指定 `--addresses`，可选 `--port`、`--login-account`、`--sshd-config`。`node show` 显示节点描述以及本机是否为管理节点；`node export --out <dir>` 写出节点描述文件 `node.json`。
- `fleet init --login-account` 在管理节点上创建并固定机群根密钥，为只包含本节点的名册版本 1 签名。`fleet sign` 用本机当前的节点描述为下一个版本签名（即使没有变化也更新 `created_at`）。`fleet add <dir>` 添加或替换 `<dir>/node.json` 中的节点并为下一个版本签名。
- `fleet show`（可选 `--json`、`--max-age <天数>`，1 至 36500，默认 30）显示已接受的名册、已固定根的指纹、版本和经过天数。`fleet export --out <dir>` 写出已接受的名册、签名和根公钥 3 个文件，非管理节点也可使用，不要求口令。`fleet accept <dir>` 验证名册，确认后接受，首次接受时固定根公钥。
- 列表值的标志（`--roles`、`--addresses`）重复指定是用法错误。节点名称、账户名称和地址的格式，各标志可用的角色，以及从 `sshd_config` 的 `HostKey` 行和 Include 读取主机公钥的规则见正本。没有 `HostKey` 行时读取默认目录（`/etc/ssh` 或 `%ProgramData%\ssh`，与 `--sshd-config` 的指定无关）中的 `ssh_host_{rsa,ecdsa,ed25519}_key.pub`，相对路径的 Include 也以默认目录为基准。不打开私有主机密钥。

正本：[コマンド](../../application-contract.md#コマンド)（日语）

### 终端、确认与退出码

- 退出码与 v0.11.0 相同：成功 0，执行失败或取消 1，用法错误 2。不存在或无法读取的 `<dir>`、错误的标志值，以及不存在或含糊的 `node set --key` 是用法错误。
- 更改状态的命令（`node set`、`fleet init`、`fleet sign`、`fleet add`、`fleet accept`）先显示计划，再在真实终端中要求完全一致的 `yes`，批准前不产生副作用。`--dry-run` 执行确认以外的全部计划（包括 `ssh-keygen` 的版本检查，以及 `fleet accept` 的签名验证和拒绝检查）并显示结果，不确认、不写入、不初始化存储；计划成功时以 0 结束，检查拒绝时以 1 结束，不需要真实终端。`fleet init`、`fleet sign`、`fleet add` 因需要为 `ssh-keygen` 输入口令，`node set` 因需要确认，在显示计划前要求真实终端；`fleet accept` 只在显示计划后需要确认时才要求真实终端，因此重新接受与已接受内容相同的名册时，无终端也会成功。
- 写出（`node export`、`fleet export`）不更改状态，因此不确认。`--out` 必须是不存在或为空的普通目录，非空或为链接时不写入任何内容而失败。文件以独占方式创建（模式 0644），中途失败时删除已写入的文件。读取操作不初始化存储，也不迁移状态，迁移预告和警告只输出到 stderr。

正本：[端末・確認・終了コード](../../application-contract.md#端末確認終了コード)（日语）

### 名册与文件

- 名册是包含 `format`（`ssh-keyflow-roster`）、`schema`（1）、`fleet_id`、`version`（每次签名加 1）、`created_at`（秒精度、带 `Z` 的 UTC）、共同的 `login_account` 以及按 `id` 排序的 `nodes`（1 至 1000 个）的 JSON 文档。公钥为不含注释的 `<type> <base64>`。
- 签名针对名册的字节本身，只接受规范形式：UTF-8（无 BOM）、LF 换行、末尾一个换行、无制表符、2 空格缩进、每行一个成员（`"name": value`）、上述字段顺序，且不超过 1 MiB。省略的可选字段不输出，而不是写为 `null`。字符串数组按字节顺序排列（`roles` 按 `client`、`remote` 的顺序）。应用不做规范化，因此 CRLF 或重新格式化会被拒绝。未知、重复或仅大小写不同的字段也会被拒绝。`schema` 为 2 或更高的名册会提示更新应用并失败。
- `fleet export` 写出 `roster.json`、`roster.json.sig`（命名空间为 `ssh-keyflow-roster` 的 `ssh-keygen -Y sign` 签名）和 `fleet-root.pub`（不含注释的一行 `ssh-ed25519 <base64>`）。`node export` 写出未签名的 `node.json`（`format` 为 `ssh-keyflow-node`，`schema` 1，`node` 1 个）。

正本：[名簿とファイル](../../application-contract.md#名簿とファイル)（日语）

### 管理节点的操作

- `fleet init` 要求先运行 `node set`，用原生 `ssh-keygen` 在 `~/.ssh/ssh-keyflow/fleet/root_ed25519`（及 `.pub`）创建带口令的 Ed25519 机群根密钥。拒绝空口令并丢弃所创建的密钥，不复用或替换现有文件。状态中只保存根密钥的路径和指纹，状态和输出都不包含私钥内容。固定密钥后版本 1 的签名失败时，重新运行 `fleet init` 会用同一密钥为版本 1 签名；未完成期间拒绝 `fleet sign` 和 `fleet accept`，`fleet show` 会警告。`fleet init` 只能运行一次，在已接受名册的节点上会被拒绝。
- `fleet sign` 和 `fleet add` 从已接受的名册创建下一个版本，由 `ssh-keygen` 直接在终端中要求口令，应用不设置 `SSH_ASKPASS`。签名后用已固定的根验证，再在一次保存中将签名、名册和 SHA-256 作为已接受内容保存；失败时不保存任何内容。
- `fleet add` 显示名称、角色和各密钥的指纹，要求与目标机器上的 `node show` 核对。已知 ID 作为替换处理，并列显示新旧值。拒绝管理节点自身的 ID、已被其他 ID 使用的名称以及与其他节点相同的密钥。第 1 阶段没有删除节点的功能。

正本：[管理ノードの操作](../../application-contract.md#管理ノードの操作)（日语）

### 接受规则

- `fleet accept` 在确认之前完成所有检查，任何失败都不更改状态。`<dir>` 只能包含上述 3 个普通文件。已有固定根时，`fleet-root.pub` 必须与其中之一一致；没有时作为首次固定的候选。签名用 `ssh-keygen -Y verify` 验证，临时文件只写入公开数据，放在系统临时目录下的新目录中，结束时删除。名册经严格解析并要求规范形式，已固定的机群要求 `fleet_id` 一致。
- 拒绝低于已接受版本的名册，以及版本相同但内容（SHA-256）不同的名册。版本和字节都相同时无需确认即成功，不写入任何内容。`created_at` 比 `--max-age` 更旧或超前 5 分钟以上时向 stderr 警告，不改变退出码。
- 显示版本、创建时间、节点数以及与已接受名册相比的变更（共同账户，添加、替换或删除的节点）。首次接受时还显示根公钥指纹和机群 ID，要求与管理节点上的 `fleet show` 核对，确认文字中也包含指纹。输入 `yes` 后，在一次保存中写入首次固定的根与机群 ID，以及已接受的版本、SHA-256、名册和签名。`fleet init` 未完成的管理节点不接受名册，而是提示重新运行 `fleet init`。

正本：[受け入れの規則](../../application-contract.md#受け入れの規則)（日语）

### 面向自动化的 JSON

- 向 stdout 输出以换行结尾的单个对象，警告不包含在 JSON 中。
- `node show --json` 为 `{"node": <节点>, "is_admin": <bool>}`。节点的字段名与名册中的节点相同，`client_keys` 输出连接源密钥的公钥。`node set` 之前为 `{"node":null,"is_admin":false}`。
- `fleet show --json` 为 `{"fleet": {"roots": [{"fingerprint": "SHA256:…"}], "sha256": "<名册字节的 SHA-256，十六进制>", "roster": <名册对象>}}`。`fleet_id`、`version` 和 `created_at` 从 `roster` 读取。第 1 阶段的 `roots` 始终只有 1 个元素。接受之前为 `{"fleet":null}`。

正本：[自動処理向けJSON](../../application-contract.md#自動処理向けjson)（日语）

### OpenSSH 要求

- 签名和验证需要 OpenSSH 8.4 或更高版本，用 PATH 上第一个 `ssh-keygen` 所在目录中的 `ssh -V` 确认版本，不满足时失败。一个命令中始终使用同一个 `ssh-keygen`，并在 `--dry-run` 输出以及签名或验证错误中显示其路径和版本。此要求只适用于 `fleet init`、`fleet sign`、`fleet add` 和 `fleet accept`，密钥管理命令的要求不变。

正本：[OpenSSHの要件](../../application-contract.md#opensshの要件)（日语）

### 状态模式 v4

- v4 在 v3 的 `version`、`keys`、`creations` 之外增加可选的 `node` 和 `fleet`。Node 和 Fleet 的保存字段见正本中的字段表。`client_key_ids` 是受管密钥的 ID，公钥本身从密钥文件读取。`accepted.roster` 原样保存已接受名册的字节，`fleet export` 写出它和签名。作为节点连接源密钥的密钥无法用 `keys remove` 移除。
- v3 的状态仅读取时不会被更改。`keys list`、`node show`、`fleet show`、写出以及变更命令的 `--dry-run` 会在 stderr 中预告下一次变更时将迁移到 v4，变更命令在确认前显示同样的预告。首次成功保存时，按与 v1 / v2 相同的规则先以独占方式将原 `state.json` 写出到 `state.v3-<UTC时间>.json`，再保存 v4，沿用 v3 的所有记录。`main` 也将 v1 / v2 的状态直接迁移到 v4，沿用范围与[从 v1 / v2 迁移](#从-v1--v2-迁移)相同。
- v0.11.0 及以前的版本无法读取 v4。回退时请退出应用，将当前的 `~/.ssh/ssh-keyflow/` 保存到其他位置，再将归档恢复为 `state.json`。迁移后的节点描述、机群、已接受的名册以及创建或导入的密钥不包含在恢复的状态中，`fleet/` 中的根密钥文件不再被状态引用；保留该文件时在 v4 中重新运行 `fleet init`，会因文件已存在而被拒绝。`state.json.bak` 是上一状态的滚动更新式备份，刚迁移后是 v3，但下一次变更会将其替换为 v4，因此回退时请使用归档。
- 机群根密钥的私钥无法重新创建，一旦丢失，就无法为同一机群的下一个版本签名。请在管理节点上将 `~/.ssh/ssh-keyflow/fleet/` 与状态一起保存到受访问限制的位置。第 1 阶段没有备用根密钥和根轮换。

正本：[状態スキーマv4](../../application-contract.md#状態スキーマv4)（日语）

## 本地密钥的安全条件与保存

- 创建使用原生 `ssh-keygen -t ed25519`，也允许空口令。生成位置为 `keys/id_ed25519_<安全化した管理名>_<UTC時刻>`（即安全化处理后的管理名称和 UTC 时间），冲突时加数字后缀。检查两个路径和已保存的预留，确认后再次检查。不覆盖或自动采用现有文件。
- 在产生副作用之前将创建意图、固定 ID 和路径保存为 `planned`，并以 `pending` 记录生成。恢复使用相同路径：没有文件则生成；只有私钥则用原生 `ssh-keygen -y` 导出，并以独占方式新建缺失的公钥；已有密钥对则不重新生成，而是核对。只有公钥、空文件或非普通文件、已记录的指纹或摘要发生变化、已观察到的文件消失时，作为需要确认的问题停止。不自动删除部分生成的文件；存在未完成的创建时，不新建其他密钥对。
- 导入用 `ssh-keygen -l -E sha256` 验证明确指定的公钥，并要求与从私钥通过 `ssh-keygen -y` 得到的公钥指纹一致。不移动或改写原文件。保存前重新检查指纹和私钥摘要，不把缺失或替换视为成功。不提供删除本地密钥文件或自动清理的 API。
- 整个变更由 `state.WithLock` 的排他 `state.lock` 保护，在确认、创建和保存期间持续持有。LockedStore 不在锁外使用，不嵌套加锁。不自动删除现有锁，只在变更时检查可写性。状态经验证后写入并同步到同一目录的临时文件，将上一个有效状态原子地保存到 `.bak` 后再替换 `state.json`。备份失败时不替换当前状态。用户在操作期间不要并行编辑目标密钥路径。

正本：[ローカル鍵の安全条件と保存](../../application-contract.md#ローカル鍵の安全条件と保存)（日语）

## 验证与发布

- 适用 CONTRIBUTING 中的 Go 测试和 vet、Portable / 安装程序测试以及 6 个目标的构建。测试使用临时主目录和替代进程，不更改真实用户的 SSH 数据、PATH、配置文件或连接目标。验证项包括：旧模式读取不更改及其预告、归档与缩减后的迁移、精确的上一状态备份、读取不产生更改、密钥创建恢复与同一性验证，以及已废弃命令的用法错误。
- 结果、目标提交、环境和未验证范围记录在 `docs/verification.md` 中。过去关于自动注册、指南、TUI 和主机密钥固定的验证记录只是历史，不是当前功能的契约。交叉构建要与在各操作系统实机上运行原生密钥创建的证据区分开。
- 发布使用指向已合并到 main 并经过验证的提交的附注 `vMAJOR.MINOR.PATCH` 标签。从合并准备 PR、验证最终候选版本、创建标签并发布，到合并发布后的验证记录 PR，其步骤、完成条件和批准范围以 [AGENTS.md 的 Tags and releases](../../../AGENTS.md#tags-and-releases)（英语）为准。准备记录引用已验证的源代码提交，最终标签的提交记录在标签、发布信息和发布后的记录中。即使因添加审计文档使 main 领先于标签，已发布的标签仍指向构建来源。推送 `v*` 标签会在 CI 之后启动自动发布。预发布不成为 latest，稳定版 latest 前进到更新的稳定版。已发布的标签和二进制资产不可变，修复以新版本发布。发布后确认分发物、校验和、内部版本、标签和源代码提交一致。开发构建仍显示 `dev`。

正本：[検証と公開](../../application-contract.md#検証と公開)（日语）
