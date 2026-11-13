# 参与贡献

简体中文 · [English](CONTRIBUTING.en.md) · [日本語](CONTRIBUTING.md)

[README](README.zh-CN.md) · [文档索引](docs/zh-CN/README.md) · [工作规范](AGENTS.md)（英语）

欢迎提交缺陷修复、功能建议以及对教程和文档的改进。请明确一次变更要解决的问题，以及可以验证的完成条件。

## 报告与建议

请先在 [GitHub Issues](https://github.com/haoyan-dev/ssh-keyflow/issues) 中查看已有报告，并记录版本、操作系统 / CPU、终端、复现步骤、预期结果和实际结果。提出功能建议时，请说明想完成的工作以及现有功能的不足之处。请勿附上私钥、口令或认证令牌；日志、路径和连接目标信息也请在公开前检查。

实现规格和工单以[本地 Markdown](docs/agents/issue-tracker.md)（日语）管理。在 `.scratch/<feature>/spec.md` 中记录目的、范围、完成条件和决策，并从 `issues/01-<task>.md` 起按序号记录工作。请根据[状态标签](docs/agents/triage-labels.md)（英语）和依赖关系更新各工单的 `Status:` 与 `Blocked by:`。请与 GitHub 上的报告和 PR 互相引用。

## 开发环境

- Go 版本遵循 [go.mod](go.mod) 的指定。
- 验证脚本使用 Python 3.12 的标准库。
- 验证原生密钥操作需要准备 `ssh` 和 `ssh-keygen`。
- TUI 的手动确认在交互式终端中进行。

```sh
git clone https://github.com/haoyan-dev/ssh-keyflow.git
cd ssh-keyflow
go run ./cmd/ssh-keyflow --help
```

开发构建显示的版本为 `dev`。试用 TUI 或变更操作时，请准备一个专用的用户环境，避免与实际使用的密钥混在一起。自动测试使用临时主目录。

## 变更流程

1. 阅读 [AGENTS.md](AGENTS.md)（英语），确认工作树和分支。从最新的 `origin/main` 创建 `<type>/<short-kebab-case-description>` 形式的工作分支。外部贡献者可以从 fork 上的分支提交 PR。
2. 进行调研，并编写规格和工单。在设计或调查代码前，请先查阅[术语表](docs/zh-CN/reference/glossary.md)和[相关 ADR](docs/adr/README.md)（日语）。
3. 实现变更，并在同一 PR 中更新所需的文档。文档的放置位置和各文档负责的信息请参阅[文档维护](docs/zh-CN/maintaining/documentation.md)。
4. 运行受影响的检查，分别评审对规范的符合性和对规格的符合性，并确认评审意见已解决。
5. 在 PR 中记录问题、变更内容、已验证的完整提交 SHA、运行环境、结果和未验证范围。通过必需的 CI 和评审后再合并。

## 验证

文档的链接、标题引用、语言链接和翻译完整性：

```sh
python scripts/test_docs.py -v
python scripts/check_docs.py
```

涉及实现或分发的变更的检查：

```sh
go test ./...
go vet ./...
python scripts/test_portable.py -v
python scripts/test_release.py -v
python scripts/test_installers.py -v
python scripts/test_installer_profiles.py -v
```

仅修改文档时，进行链接检查并确认步骤和命令。修改 README、指南等需要翻译的页面时，请在同一 PR 中同时更新[英文版和中文版](docs/zh-CN/maintaining/documentation.md#各语言的文件布局)，或明确标注为待更新。修改实现、依赖、打包或安装程序时，请运行相应测试；如果打包输入发生变化，还需执行[发布指南](docs/zh-CN/maintaining/releases.md)中的 6 个目标构建。无论变更类型如何，都必须通过 CI 的必需检查。

需要长期参考的验证结果记录在 [docs/verification.md](docs/verification.md)（日语）中，包括目标 SHA、环境、结果和局限。请分别区分交叉构建、实机运行和实际 SSH 连接确认。

## 评审与发布

每个 PR 只针对一个目的，并说明用户可见的变更以及对兼容性和迁移的影响。仅修改文档的 PR 也需经评审后合并到 `main`。

应用的发布按照[发布指南](docs/zh-CN/maintaining/releases.md)，依次进行准备 PR、最终候选版本验证、标签与自动发布、分发物审计与记录 PR。推送标签会开始发布，因此请先确认 [AGENTS.md 的批准范围](AGENTS.md#tags-and-releases)（英语）。
