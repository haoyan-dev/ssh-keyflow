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

正本：[現行機能の範囲（v0.11.0以降）](../../application-contract.md#現行機能の範囲v0110以降)（日语）

## CLI

- 入口为 `cli.Run(ctx, args, cli.Options{Home, Version, Input, Output, Error, Runner}) int`，`cmd/ssh-keyflow` 返回退出码。Home 的解析推迟到需要的操作时进行。
- 命令为 `keys list` / `public` / `create` / `import` / `remove` 和 `resume`。`keys list` 可选 `--json`，显示受管密钥和未完成的本地创建；`keys public` 以一行显示公钥的算法和 blob，不含文件中的注释；`keys create` 创建 Ed25519 密钥对；`keys import` 必须指定 `--name`、`--public`、`--private`，验证现有密钥对后纳入管理，不移动文件；`keys remove` 从管理中去掉并保留密钥文件；`resume` 仅恢复未完成的本地创建。各命令的必需和可选输入见正本中的命令表。
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

- 存储区域为 `<Home>/.ssh/ssh-keyflow`，使用 `state.json`、生成密钥用的 `keys/`、滚动更新式的 `state.json.bak`、临时的 `state.lock`，以及从旧模式迁移时的 `state.v<N>-<UTC时间>.json`。没有状态时只返回 v3 空状态，读取时不创建文件。
- v3 是本版本唯一写入的模式，只包含 `version`、必需数组 `keys` 和可选数组 `creations`。状态记录分为 Key 和 Creation，各自的保存字段见正本中的字段表。
- 密钥 ID 为固定的随机 `key-<128-bit hex>`，指纹为 OpenSSH 公钥 blob 的 SHA256 格式，路径为绝对路径。导入保存 `origin: imported` 和非零的 UTC RFC3339Nano `imported_at`，创建保存 `origin: created` 和 `created_at`。私钥摘要用于在不要求口令的情况下检测替换，不保存私钥内容。完成的创建在添加密钥的同一次保存中从 `creations` 去除。
- v3 的读取是严格的：无效 JSON 或带有尾随内容的 JSON、重复或仅大小写不同的字段、未知字段、不支持的版本、缺少 `keys`、无效元数据、重复的 ID / 指纹 / 路径，都会被拒绝，不自动恢复或覆盖。

正本：[保存スキーマと互換性](../../application-contract.md#保存スキーマと互換性)（日语）

### 从 v1 / v2 迁移

- v1 和 v2（v0.10.0 及以前）在读取时于内存中转换为 v3，仅读取不会更改文件。旧模式的读取同样严格，会拒绝未知字段、类型错误的数组、带后续内容的 JSON 以及缺少必需数组（`keys` / `hosts` / `registrations`）的文件。转换只沿用带有私钥路径的密钥和未完成的本地创建，不沿用以下内容：连接目标、关联和已归档的移除记录（`hosts` / `associations` / `removed_keys` / `removed_hosts`）；旧的注册、连接确认、设置、SSH config 编辑、引导和删除的记录与历史（`registrations` / `history` / `configs` / `config_edits` / `onboardings` / `deleted_keys` / `deletions`）；仅有公钥的密钥和存在未完成旧密钥删除的密钥；密钥的 `connection_check`、连接目标的 `config_key_id` 等旧字段。
- 预告按记录类型列出每种非空记录的数量。每个未完成的 SSH config 编辑都会单独列出路径，并提示手动检查该文件。`keys list` 在迁移前通过 stderr 输出此预告。变更命令（create / import / remove / resume）在获取锁之后、确认提示之前显示同样的预告，因此确认也包含对迁移的批准。首次成功保存状态时，先以独占方式将原 `state.json` 的字节写出到同一目录下新的 `state.v<N>-<UTC时间>.json`（冲突时加数字后缀），然后更新 `.bak` 并保存 v3。归档创建失败时不更改状态。如果之前的尝试已留下字节相同的归档，重试时会复用它，不会再增加相同的归档。归档不会被覆盖或自动删除，因此未沿用的记录也保留在磁盘上。
- 旧二进制文件无法读取 v3。降级时请退出应用，手动将归档恢复为 `state.json`。迁移后创建或导入的密钥不包含在恢复的状态中。

正本：[v1 / v2からの移行](../../application-contract.md#v1--v2からの移行)（日语）

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
