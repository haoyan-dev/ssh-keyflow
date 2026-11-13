# SSH KeyFlow 文档

简体中文 · [English](../en/README.md) · [日本語](../README.md)

[项目概览](../../README.zh-CN.md) · [参与贡献](../../CONTRIBUTING.zh-CN.md)

本目录中的文档对应 `main` 上的当前实现。若要了解已发布版本的行为，请参阅[对应标签](https://github.com/haoyan-dev/ssh-keyflow/tags)下的同名文档和[发布说明](https://github.com/haoyan-dev/ssh-keyflow/releases)。当前规格不会追溯适用于旧版本。

## 常用操作

| 想做的事 | 阅读页面 |
| --- | --- |
| 准备第一个密钥和连接目标，并完成连接设置 | [首次连接设置](tutorials/first-connection.md) |
| 安装、更新或删除程序 | [安装指南](guides/installation.md) |
| 查看 TUI 的操作按键和复制内容 | [TUI 操作](guides/terminal-ui.md) |
| 用 CLI 管理密钥、连接目标和关联 | [CLI 用法](guides/cli.md) |
| 了解状态的备份保存、迁移以及回退到旧版本时的注意事项 | [数据、迁移与恢复](guides/data-and-recovery.md) |
| 排查启动、复制、密钥和连接问题 | [故障排除](guides/troubleshooting.md) |

## 规格与背景

| 文档 | 作用 |
| --- | --- |
| [应用契约](reference/application-contract.md)（中文概要） | CLI、JSON、退出码、状态兼容性、安装与分发的公开契约 |
| [术语表](reference/glossary.md)（中文概要） | 密钥、连接目标、关联、未验证等领域术语 |
| [设计决策列表](../adr/README.md)（日语） | 已采纳的决策，以及更新或取代它们的决策 |
| [验证记录](../verification.md)（日语） | 按时间点记录的证据，包括目标提交、运行条件、结果和局限 |
| [第三方许可证的收集方法](../third-party-notices.md)（日语） | 分发物中包含的许可证文档及其收集依据 |

## 开发与维护

- [参与贡献](../../CONTRIBUTING.zh-CN.md)：环境搭建、变更流程、检查和评审。
- [文档维护](maintaining/documentation.md)：文档类型、各文档负责的信息、更新时的检查。
- [发布](maintaining/releases.md)：分发物、验证命令和发布流程的入口。
- [本地工单](../agents/issue-tracker.md)（日语）和[状态标签](../agents/triage-labels.md)（英语）：规格与工作的记录。
- [领域文档的使用方式](../agents/domain.md)（日语）：术语和设计决策的参考位置。
- [工作、分支与发布规范](../../AGENTS.md)（英语）：开发时的规则和批准范围。

## 过去的调查与评审

以下文档是记录当时的调查和验证。当前规格请参阅应用契约。

- [SSH 实现调查](../ssh-implementation-notes.md)（日语）
- [初始实现的最终评审](../final-review.md)（日语）
- [v0.2.0 评审](../v0.2.0-review.md)（日语）
- [版本管理策略审计](../versioning-policy-audit.md)（英语）

`.scratch/` 包含按功能整理的研究、规格、工单和工作记录。旧规格可能已被后续的 ADR 或当前契约取代。
