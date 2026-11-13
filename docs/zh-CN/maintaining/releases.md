# 发布

简体中文 · [English](../../en/maintaining/releases.md) · [日本語](../../maintaining/releases.md)

[文档索引](../README.md) · [参与贡献](../../../CONTRIBUTING.zh-CN.md)

本页是分发的概要和检查的入口。批准范围和各阶段的完成条件以 [AGENTS.md](../../../AGENTS.md#tags-and-releases)（英语）为准；发布前还请确认[应用契约（中文概要）](../reference/application-contract.md)、作为正本的[应用契约](../../application-contract.md)（日语）和 [release.yml](../../../.github/workflows/release.yml)。为了合并仅修改文档的变更，不需要创建应用的发布标签。

## 开发与发布契约

Go 版本遵循 `go.mod` 的指定，打包测试使用 Python 3.12 的标准库。TUI 使用 Bubble Tea / Bubbles / Lip Gloss，CLI 使用 Cobra，本地密钥处理和状态管理由两者共用。

```sh
go test ./...
go vet ./...
python scripts/test_portable.py -v
python scripts/test_release.py -v
python scripts/test_installers.py -v
python scripts/test_installer_profiles.py -v
python scripts/package_release.py --version v0.0.0-check --output .tmp/dist-check
```

打包使用 `CGO_ENABLED=0`、`-buildvcs=false`、`-trimpath` 和 `-ldflags "-s -w -X main.version=<tag>"` 构建 6 个目标。开发版为 `dev`。上面的 `v0.0.0-check` 是用于本地验证的版本注入示例，不会发布。拒绝覆盖已有文件的输出目录。打包现有二进制文件时指定 `--binaries DIRECTORY`，从 `DIRECTORY/linux_amd64/ssh-keyflow` 等 6 个目标读取。

| 操作系统 | CPU | 资产名称 |
| --- | --- | --- |
| Linux | x64 / ARM64 | `ssh-keyflow_<tag>_linux_<amd64\|arm64>.tar.gz` |
| macOS | x64 / ARM64 | `ssh-keyflow_<tag>_darwin_<amd64\|arm64>.tar.gz` |
| Windows | x64 / ARM64 | `ssh-keyflow_<tag>_windows_<amd64\|arm64>.zip` |

归档内部为 `ssh-keyflow`（Windows 上为 `ssh-keyflow.exe`）和 `THIRD_PARTY_NOTICES.txt`。Unix 执行权限为 0755。顶层分发物只有 6 个归档和 `SHA256SUMS`，共 7 个。manifest 针对 6 个归档，按 64 位十六进制 SHA256、两个半角空格、基本文件名、LF 的格式记录。

CI 在 3 种操作系统上运行 Go 测试、Portable 生成物测试和安装程序测试，并确认 6 个目标的实际交叉构建和 Linux amd64 版的版本注入。推送 `v*` 标签后，在 CI 成功后将 7 个文件发布到 `GITHUB_REPOSITORY`。安装程序从 `main` 的源代码另行分发，不会更改 7 个发布分发物。预发布标签不会成为 latest 正式版。

发布按以下顺序进行：合并准备 PR，验证最终候选版本，创建标签并发布，验证发布后的分发物并合并记录 PR。准备记录使用已验证源代码的提交，最终标签的提交记录在标签、发布信息和发布后的记录中。发布后的记录使 main 领先于标签是正常的运作方式，仅修改记录不需要重新发布。各阶段的完成条件和批准范围请参阅 [AGENTS.md 的 Tags and releases](../../../AGENTS.md#tags-and-releases)（英语）。

发布作业跨标签串行执行，并在其中获取当前的 latest，按版本号的数值顺序比较。旧系列的修复版可以发布，但不会取代更新的 latest。latest 返回 404 时，只有在成功获取全部发布列表、并确认没有已发布的正式版时，才作为首次发布处理。如果认证、通信或 API 响应异常，或者存在正式版却无法获取 latest，则停止发布。`scripts/test_release.py` 在不进行外部发布的情况下验证这些判断和发布命令。

测试使用临时主目录、安装位置和分发物，不会更改真实用户的 SSH 数据、永久 PATH、配置文件或连接目标。交叉构建与在各操作系统 / CPU 上实机运行的证据要区分开来。已确认的范围请参阅 [docs/verification.md](../../verification.md)（日语），公开接口请参阅[应用契约（中文概要）](../reference/application-contract.md)。
