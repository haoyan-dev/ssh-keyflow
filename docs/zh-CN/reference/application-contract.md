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
- 旧的 v0.1.0 安装程序会拒绝新归档；完好的 v0.1.0 安装请使用新的源代码分发安装程序。手动替换二进制文件后，更新或删除可能被拒绝。README 中说明了手动维护以及可选地手动清理 PATH 段落，不提供自动修复。
- CI 运行测试并进行 6 个目标的交叉构建，发布工作流在 CI 之后上传 7 个资产。基于测试夹具（fixture）的制品测试和交叉构建不能证明每个目标上的原生运行行为。

正本（该节原文为英语，以原文为准）：[Portable release](../../application-contract.md#portable-release)

## 当前功能范围（v0.5.0 及以后）

- SSH KeyFlow 专注于本地密钥管理和通过复制粘贴辅助连接设置。本契约取代旧规格中关于自动注册与取消注册、SSH 连接确认、设置编辑、密钥删除、编号菜单、Tasks 和递归设置搜索的描述。这是 v0.5.0 引入的有意不兼容变更，不追溯适用于 v0.4.0 及以前的二进制文件。分发、安装程序和发布流程保持不变。
- 公钥的部署和取消注册由用户手动完成。应用不运行也不生成 SSH 进程、远程认证或探测，以及用于安装、启动服务器或更改防火墙的脚本。根据 ADR 0002，应用会生成由用户粘贴到目标终端运行的公钥部署脚本和本地 config 追加脚本。新的密钥、关联和指南在注册和连接方面均为未验证，也不提供记录手动成功的操作。过去的成功记录作为历史保留，不解读为当前是否可连接。

正本：[現行機能の範囲（v0.5.0以降）](../../application-contract.md#現行機能の範囲v050以降)（日语）

## CLI

- 入口为 `cli.Run(ctx, args, cli.Options{Home, Version, Input, Output, Error, Runner, RunTUI}) int`，`cmd/ssh-keyflow` 返回退出码。Home 的解析推迟到需要的操作时进行。
- 命令分为 `keys list` / `create` / `import` / `public` / `remove`、`hosts list` / `add` / `associate` / `dissociate` / `remove` / `guide` / `pin` / `unpin`、`resume` 和 `tui`。`keys import` 省略 `--private` 时只导入公钥；`keys public` 不含文件中的注释；`hosts add` 需要现有密钥，`--port` 默认 22，Windows 需要 `--windows-account`；`hosts guide` 是只读的部署、取消注册和连接设置指南；`resume` 仅限未完成的本地 Creation ID；`tui` 的输入和输出都需要真实终端。各命令的必需和可选输入见正本中的命令表。
- `--os` 明确指定 `linux` / `macos` / `windows`。连接目标按固定 ID 或不区分大小写的别名解析，密钥按固定 ID 或唯一且完全一致的管理名称解析。含糊或不存在的对象会被拒绝。受管连接目标的别名重复会被拒绝，但不搜索 SSH config 中的别名。添加连接目标需要现有密钥，不会同时新建密钥。
- 不提供 `register`、`unregister`、`registrations list`、`keys delete`、`--new-key`、`--auth`、`--identity`、`--configure`。旧远程操作 ID 和删除 ID 的 `resume` 不会执行，而是要求手动确认。
- 根据 ADR 0004 增加了管理元数据的移除和解除关联。`remove-key` / `remove-host` / `dissociate` 是共享后端的 Operation，使用真实终端和完全一致的 `yes` 确认、在锁下重新解析对象、原子保存和上一状态备份。移除密钥或连接目标会去掉其全部关联，解除关联只去掉指定的一组。不存在的组合是参数错误。无需验证密钥文件或运行外部进程，也能整理文件缺失的密钥的管理信息。存在未完成旧密钥删除的密钥要求手动确认，并拒绝从管理中移除。
- 已移除的密钥和连接目标保存在 v2 的可选数组 `removed_keys` / `removed_hosts` 中。旧注册、创建结果、引导和设置历史的引用也会对归档 ID 进行验证，历史本身不被改写。当前关联只能引用受管的密钥和连接目标。归档中的指纹、路径和别名可以重复使用，重新导入和重新添加连接目标会创建新 ID。list、公钥、指南和选择候选不包括已移除的记录。不更改密钥文件、SSH 设置或远程注册，也不意味着吊销访问。旧二进制文件会拒绝包含归档数组的状态，因此降级需使用事先保存的状态。TUI 在列表或详情中用 `d` 从管理中移除，用 `u` 只将所选对象的现有关联作为候选进行解除，并交给现有的真实终端确认。
- 变更操作在初始化存储或运行进程之前要求真实终端输入，并向共享的 `app.Run` 传入显式 Request 和确认函数。只有去除首尾空白后完全一致的 `yes` 才会被接受。`ReadConfirmation` 逐字节读取一行，不预读后续的原生口令输入。不提供秘密值或无条件同意的标志。唯一例外是 `hosts pin --yes`：仅在未固定不同密钥时省略确认，并且不批准替换。
- 退出码：成功 0，执行失败或取消 1，参数或用法错误 2。含糊或不存在的对象也是参数错误，stderr 输出对应命令的 usage；运行时诊断不附带 usage。`--help` 和 `--version` 不解析 Home、不检查依赖命令、不保存。版本显示为 `SSH KeyFlow <version>`。
- 不带参数时，如果输入或输出任一方不是终端，则输出帮助并以 0 退出；两者都是真实终端时打开 TUI。显式 `tui` 不满足终端要求时返回 2。list、公钥和指南可以重定向，不探测可写性或 SSH 命令。
- list 的普通输出为表格。JSON 是以换行结尾的单个对象：`keys list` 返回 `keys` / `creations`，`hosts list` 返回 `hosts` / `associations` / `host_key_pins`。`host_key_pins` 为受管连接目标 ID → `[{type, fingerprint}]`，不包含已移除连接目标的固定；无法读取固定文件时省略，并在 stderr 输出原因。普通输出的 `HOST KEYS` 列显示 `pinned (N)` / `not pinned` / `unknown`。数组为空数组而非 null，字段名与保存模式一致。需要确认的信息只输出到 stderr，已验证的状态记录仍可读取。无效状态返回 1，不输出 JSON，并保留原文件。公钥和指南将用于复制的文本输出到 stdout。输出失败视为执行失败。

正本：[CLI](../../application-contract.md#cli)（日语）

## 共享后端与手动设置指南

- `app.Request` 的 Operation 为 `list` / `create` / `import` / `add` / `associate` / `dissociate` / `remove-key` / `remove-host` / `guide` / `public` / `resume` / `pin-host` / `unpin-host`。`app.Run` 要求显式 Request，不启动旧编号菜单。变更对象在锁下也会重新解析和验证。
- `app.Snapshot(Options)` 返回 `View{State, Review}`，不初始化存储、不探测依赖命令、不更改旧记录。`app.PublicKey(home, keyRef)` 和 `app.Guide(home, hostRef, keyRef)` 验证目标密钥的同一性后返回文本。`app.PrepareGuide` 经过相同验证后返回 `SetupGuide{Text, Steps}`，TUI 使用每个 `GuideStep` 的 `Title` / `Instructions` / `Script`。显示指南不以已保存的关联为前提，也不保存新的关联。
- `app.PrepareGuideWithOptions(home, hostRef, keyRef, GuideOptions)` 在相同验证之外接收仅用于指南的 `LocalOS` / `RemoteOS` / `WindowsAccount` / `IdentityFile`；`app.Request.GuideOptions` 将相同选择传给共享后端。操作系统为 `linux` / `macos` / `windows`，省略时使用运行应用的操作系统以及已保存的连接目标操作系统和账户。改为不同的连接目标操作系统时，不沿用原来的 Windows 账户类型。Windows 必须指定 standard / administrator，向非 Windows 目标指定账户会被拒绝。TUI 在选择连接目标和密钥后，在表单中显示并允许更改这些默认值，选择不会保存。受管私钥只读取格式头；带口令或无法判断时，加入 `AddKeysToAgent yes` 和面向本地操作系统的 ssh-agent 步骤（位于 config 追加与连接之间）。指定 `IdentityFile` 时不检查，按无法判断处理（ADR 0005）。
- 指南包含公钥的算法与 blob、指纹、部署位置、手动取消注册说明和本地文件路径。第一步是与所选连接目标操作系统和账户类型相应的公钥部署脚本：Linux/macOS 使用 POSIX shell，Windows 使用 PowerShell。保留现有密钥内容，已包含相同公钥时不追加；即使现有最后一行没有换行也会分开追加。拒绝符号链接和 reparse point。POSIX 确认登录用户名，Windows 标准账户确认用户名，Windows 管理员账户确认终端已提升权限。标准 OpenSSH 配置下的部署位置：
  - Linux / macOS：登录用户的 `~/.ssh/authorized_keys`；保留现有行，用户所有的 `.ssh` 为 700，文件为 600。
  - Windows 标准账户：`%USERPROFILE%\.ssh\authorized_keys`；保留现有行，写入权限限定为本人、SYSTEM 和 Administrators。
  - Windows 管理员账户：`%ProgramData%\ssh\administrators_authorized_keys`；由管理员登录共享，因此保留其他密钥，并确认 SYSTEM 和 Administrators 的 ACL。
- 自定义配置下由用户确认实际的 `AuthorizedKeysFile` / `Match`。前提是 SSH 服务器和网络事先可用。取消注册时保留其他有效的访问方式，只从同一文件中手动删除匹配的公钥行。不验证取消注册是否成功。
- 第二步是将 Host 片段追加到 `~/.ssh/config` 的脚本，片段包含客户端的 `HostName` / `User` / `Port` / `IdentityFile` / `IdentitiesOnly yes`；使用私钥路径，仅有公钥的密钥则使用已保存的公钥路径。按所选本地操作系统，Windows 使用 PowerShell，Linux/macOS 使用 POSIX shell。本地操作系统与运行应用的系统不同时，必须用 IdentityFile 指定对应私钥在该终端中的路径；接受所指定操作系统的绝对路径或 ~/path，拒绝控制字符、相对路径和其他操作系统的路径。不读取指定路径的文件，不传输密钥、不转换路径、不保证一致。仍维持受管原文件的同一性验证，不会因指定 IdentityFile 而将仅有公钥的密钥变为密钥对。保留现有内容；如果存在相同的显式别名（包括分组的 Host 行），则停止。不解析 Include、通配符、Match，也不保证优先级。PowerShell 要求现有文件为 UTF-8，在追加前拒绝 UTF-16 和无效的 UTF-8。标准账户的 Windows 公钥部署将 `.ssh` 目录和密钥文件的 DACL 限定为本人、SYSTEM 和 Administrators；管理员账户只将共享文件的 DACL 限定为 SYSTEM 和 Administrators。保留所有者和审计信息，并分别使用 Windows PowerShell 5.1 / PowerShell 7 的 .NET ACL API。
- 第三步是在运行前两步之后于本地终端使用的 `ssh 別名` 连接命令（`別名` 处为别名）。连接命令在 POSIX shell / PowerShell 中通用，不在命令参数中重复展开私钥路径。考虑到 OpenSSH 自身的展开，IdentityFile 中的 `%` 会被转义。包含 `${...}` 的路径无法保证按字面引用，因此保留公钥部署说明，注明原因并省略 config 追加和连接步骤。对于仅有公钥的密钥，将公钥文件用作 `IdentityFile`，使 OpenSSH 只从 SSH 代理（如 FIDO 密钥）使用对应的密钥，并提示需要将对应私钥加载到代理中；不加入 `AddKeysToAgent` 和 ssh-agent 步骤。本地操作系统与运行应用的系统不同时要求指定 `--identity-file`，未指定时只显示部署步骤。应用不运行任何脚本或命令。
- 应用本身不编辑、重新生成或删除普通的 `~/.ssh/config`、过去的专用 config、Include 和备份。只有用户运行复制的 config 追加脚本时，才会追加到普通 config。不提供递归搜索或通配符、Include 优先级的解析。粘贴位置以及与现有 Host 的重复和优先级由用户确认。现有 Include 启用过去专用 config 的情况也会原样保留。

正本：[共通バックエンドと手動ガイド](../../application-contract.md#共通バックエンドと手動ガイド)（日语）

## 主机密钥固定

- 根据 ADR 0006，将连接目标的主机公钥固定到应用所有的 `<Home>/.ssh/ssh-keyflow/known_hosts`。用户在连接目标自身的终端中运行 `cat /etc/ssh/ssh_host_*_key.pub`（POSIX），或在以管理员身份打开的 PowerShell 中运行 `Get-Content "$env:ProgramData\ssh\ssh_host_*_key.pub"`（Windows），然后粘贴输出。应用不打开网络连接，也不编辑 `~/.ssh/known_hosts`、SSH config 或 `sshd_config`。
- `hosts pin` 在真实终端中显示上述命令并读取粘贴内容直到空行；在非终端中读取标准输入或 `--file`。输入上限为 64 KiB、16 个密钥。只接受 `<type> <base64> [comment]` 格式、类型为 `ssh-ed25519` / `ecdsa-sha2-nistp256` / `ecdsa-sha2-nistp384` / `ecdsa-sha2-nistp521` / `ssh-rsa`、base64 规范且 blob 内算法一致的行。注释被丢弃，其他行作为忽略的行显示，不妨碍有效密钥。显示每个密钥的类型和 SHA256 指纹，完全一致的 `yes` 时保存。重新固定相同的一组密钥不会写入。非终端输入需要 `--yes`，`--yes` 仅在未固定不同密钥时省略确认。已固定不同密钥时，显示新旧指纹和警告，并要求 `--replace` 以及在真实终端中输入 `yes`。`hosts unpin` 在真实终端确认后只去除该连接目标的行。两者都只针对受管连接目标；从管理中移除连接目标会保留固定，因此移除确认中会提示先运行 `hosts unpin`。
- 固定文件每行一个密钥，格式为 `ssh-keyflow-<host-id> <type> <base64>`，以 LF 结尾。写入使用与状态相同的锁、原子替换、拒绝符号链接和权限检查，保持其他连接目标行的顺序和内容。读取是严格的：文件中包含无效行、CRLF、制表符、重复、注释或标记时，停止固定和解除，不自动修复。状态模式不变。
- 指南在生成时读取固定状态。仅当本地操作系统与运行应用的系统相同时：未固定则在部署步骤后加入连接目标上的“显示主机密钥”和本地的“固定主机密钥”（`ssh-keyflow hosts pin --host <id>`）步骤；已固定则在 Host 片段的 `IdentitiesOnly yes` 之后加入 `HostKeyAlias ssh-keyflow-<host-id>`、使用斜杠分隔并加引号的绝对路径的 `UserKnownHostsFile`，以及 `StrictHostKeyChecking yes`。路径使用与 IdentityFile 相同的转义，包含控制字符或 `${` 时不生成这些行。无法读取固定文件时显示 `Host keys: unknown`，不加入任何固定相关的行或步骤。未固定连接目标的片段不变。现有的 Host 需由用户手动添加这 3 行。

正本：[ホスト鍵の固定](../../application-contract.md#ホスト鍵の固定)（日语）

## TUI

- `tui.Run(ctx, app.Options)` 是 CLI 的 TUI 入口，以 Keys / Destinations 两个界面为中心。提供密钥创建和导入、使用现有密钥添加连接目标、关联与解除关联、从管理中移除，以及公钥和指南的显示与复制。新密钥单独创建。不提供旧的 Registrations / Tasks、远程认证和设置编辑表单，以及删除密钥文件的操作。
- 按键：Tab 切换界面，c/i 创建或导入密钥，a 添加连接目标，l 关联，u 解除现有关联，d 从管理中移除，p/g 显示公钥或指南，s 恢复所选的本地创建。`t` / `T` 固定 / 解除所选连接目标的主机密钥，交给原生终端进行粘贴和 `yes` 输入（固定时指定 Replace，存在不同密钥时显示新旧指纹）。连接目标详情显示 `Host keys: pinned (N)` / `not pinned` / `unknown`。y 对显示中的文本请求 OSC52 复制，终端不支持时引导使用 CLI 的公钥或指南输出。公钥保持为一行处理，不把 TUI 的折行作为换行复制。v 的技术详情以只读方式显示旧记录。最小终端为 32 列 × 12 行。按键操作的细节以界面内帮助为准。
- 指南从第一步开始显示，用 ← / → 或 `b` / `n` 切换。`y` 只复制所选的 `Script`，不混入说明文字或面向其他终端的脚本。每个步骤都标明本地 / 连接目标、操作系统、终端以及所需账户和权限。从 Help 返回时保持所选步骤。F5 重新加载时会使脚本失效（包括历史中的），切换步骤也不会恢复旧的复制内容。
- 使用与 CLI 相同的验证和保存处理，读取或显示指南不会启动执行。变更伴随对象和影响的确认。必要时将终端交给原生 `ssh-keygen`，口令不保留在 TUI 或状态中。失败、中断或取消不会显示为成功，也不视为已回滚保存的进度。

正本：[TUI](../../application-contract.md#tui)（日语）

## 保存模式与兼容性

- 存储区域为 `<Home>/.ssh/ssh-keyflow`，使用 `state.json`、生成密钥用的 `keys/`、滚动更新式的 `state.json.bak`、临时的 `state.lock`，以及首次固定时创建的主机密钥固定文件 `known_hosts`。没有状态时只返回 v2 空状态，读取时不创建文件。
- 读取 v1 和 v2。v1 仅读取时不会更改内容、版本或备份，首次成功保存状态时变为 v2。v2 的 `associations` 是 `{host_id, key_id}` 组合，引用有效的密钥和连接目标，不允许重复组合。不会将旧的 `registrations` / `history` 转换为关联，也不更改其注册或连接状态。
- 状态记录分为 Key、Host、Association 和 Creation，各自的保存字段见正本中的字段表。
- 必需数组为 `keys` / `hosts` / `registrations`。可选数组（如 `creations` / `associations` / `removed_keys` / `removed_hosts`）和旧版本的历史数组都会保留，完整列表见正本。维护过去的固定 ID、注册历史、设置摘要、中断意图、已删除密钥信息及其引用，不改写为新操作的成功或完成。没有专门查看旧历史的命令，但保存的数据会保留。v2 保存会重新格式化 JSON，因此保留过去记录的值与状态文件字节不变是两回事。
- 密钥 ID 为固定的随机 `key-<128-bit hex>`，指纹为 OpenSSH 公钥 blob 的 SHA256 格式，路径为绝对路径。导入保存 `origin: imported` 和非零的 UTC RFC3339Nano `imported_at`，创建保存 `origin: created` 和 `created_at`。私钥摘要用于在不要求口令的情况下检测替换，不保存私钥内容。新密钥的 `connection_check` 为 `unverified`，关联没有表示成功状态的字段。
- 无效 JSON 或带有尾随内容的 JSON、重复或未知字段、不支持的版本、缺少必需数组、无效元数据、重复的 ID / 别名 / 指纹 / 路径、矛盾或孤立的引用，在读取时被拒绝，不自动恢复或覆盖。旧远程操作和设置编辑的未完成记录为只读，不自动恢复。未完成的旧删除只使目标密钥不可用并要求手动确认，不会仅因该记录阻止其他密钥的操作。
- `state.json.bak` 是保存前一刻经过验证的 `state.json` 的精确字节。每次保存都会更新，因此在迁移后的同一操作中也可能被替换为 v2。它不是永久的 v1 迁移备份。**可能需要降级的用户，请在 v0.5.0 及以后版本中执行第一次变更操作之前，退出应用并另行保存 v1 状态。** 旧应用无法读取 v2，因此需要从保存的 v1 状态手动恢复，并核对迁移后新增的数据。读取时不创建迁移用副本，现有的密钥、config、Include 和备份也不会因迁移被改写。

正本：[保存スキーマと互換性](../../application-contract.md#保存スキーマと互換性)（日语）

## 本地密钥的安全条件与保存

- 创建使用原生 `ssh-keygen -t ed25519`，也允许空口令。生成位置为 `keys/id_ed25519_<安全化した管理名>_<UTC時刻>`（即安全化处理后的管理名称和 UTC 时间），冲突时加数字后缀。检查两个路径和已保存的预留，确认后再次检查。不覆盖或自动采用现有文件。
- 在产生副作用之前保存创建意图、固定 ID 和路径，并维护 `planned` / `pending` / `complete` 进度。恢复使用相同路径：没有文件则生成；只有私钥则用原生 `ssh-keygen -y` 导出，并以独占方式新建缺失的公钥；已有密钥对则不重新生成，而是核对。只有公钥、空文件或非普通文件、已记录的指纹或摘要发生变化、已观察到的文件消失时，作为需要确认的问题停止。不自动删除部分生成的文件；存在未完成的创建时，不新建其他密钥对。
- 导入也用 `ssh-keygen -l -E sha256` 验证明确指定的公钥；指定私钥时，要求与 `ssh-keygen -y` 得到的公钥指纹一致。不移动或改写原文件。保存前重新检查指纹和私钥摘要，不把缺失或替换视为成功。不提供删除本地文件或自动清理的 API。
- 整个变更由 `state.WithLock` 的排他 `state.lock` 保护，在确认、创建和保存期间持续持有。LockedStore 不在锁外使用，不嵌套加锁。不自动删除现有锁，只在变更时检查可写性。状态经验证后写入并同步到同一目录的临时文件，将上一个有效状态原子地保存到 `.bak` 后再替换 `state.json`。备份失败时不替换当前状态。用户在操作期间不要并行编辑目标密钥路径。

正本：[ローカル鍵の安全条件と保存](../../application-contract.md#ローカル鍵の安全条件と保存)（日语）

## 验证与发布

- 适用 README 中的 Go 测试和 vet、Portable / 安装程序测试以及 6 个目标的构建。测试使用临时主目录和替代进程，不更改真实用户的 SSH 数据、PATH、配置文件或连接目标。验证项包括：v1 读取不更改、v2 迁移与精确的上一状态备份、旧历史保留、只阻止受影响密钥的删除、读取不产生更改、密钥创建恢复与同一性验证。
- 结果、目标提交、环境和未验证范围记录在 `docs/verification.md` 中。过去自动注册与取消注册的测试记录只是历史，不是当前功能的契约或新实现的验证结果。交叉构建要与在各操作系统实机上运行 TUI 和原生密钥创建的证据区分开。显示指南并不验证实际的连接目标设置、注册、取消注册或连接成功。
- 发布使用指向已合并到 main 并经过验证的提交的附注 `vMAJOR.MINOR.PATCH` 标签。从合并准备 PR、验证最终候选版本、创建标签并发布，到合并发布后的验证记录 PR，其步骤、完成条件和批准范围以 [AGENTS.md 的 Tags and releases](../../../AGENTS.md#tags-and-releases)（英语）为准。准备记录引用已验证的源代码提交，最终标签的提交记录在标签、发布信息和发布后的记录中。即使因添加审计文档使 main 领先于标签，已发布的标签仍指向构建来源。推送 `v*` 标签会在 CI 之后启动自动发布。预发布不成为 latest，稳定版 latest 前进到更新的稳定版。已发布的标签和二进制资产不可变，修复以新版本发布。发布后确认分发物、校验和、内部版本、标签和源代码提交一致。开发构建仍显示 `dev`。

正本：[検証と公開](../../application-contract.md#検証と公開)（日语）
