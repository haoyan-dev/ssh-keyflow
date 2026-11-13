# SSH KeyFlow documentation

[简体中文](../zh-CN/README.md) · English · [日本語](../README.md)

[Project overview](../../README.en.md) · [Contributing](../../CONTRIBUTING.en.md)

This directory documents the current implementation on `main`. To find out how a released version behaves, read the same documents at the [corresponding tag](https://github.com/haoyan-dev/ssh-keyflow/tags) and the [release notes](https://github.com/haoyan-dev/ssh-keyflow/releases). The current specification does not apply retroactively to older versions.

## Common tasks

| What you want to do | Page |
| --- | --- |
| Prepare your first key and destination and set up the connection | [Your first connection setup](tutorials/first-connection.md) |
| Install, update or remove the executable | [Installation guide](guides/installation.md) |
| Check TUI key bindings and where copies go | [Using the TUI](guides/terminal-ui.md) |
| Manage keys, destinations and associations with the CLI | [Using the CLI](guides/cli.md) |
| Back up state, migrate, and check caveats for returning to an older version | [Data, migration and recovery](guides/data-and-recovery.md) |
| Investigate launch, copy, key and connection problems | [Troubleshooting](guides/troubleshooting.md) |

## Specifications and background

| Document | Role |
| --- | --- |
| [Application contract (overview)](reference/application-contract.md) | Public contract for the CLI, JSON, exit codes, state compatibility, installation and distribution |
| [Glossary (overview)](reference/glossary.md) | Domain terms such as key, destination, association and unverified |
| [Index of design decisions](../adr/README.md) (Japanese) | Adopted decisions and the decisions that update or supersede them |
| [Verification records](../verification.md) (Japanese) | Point-in-time evidence including target commits, conditions, results and limits |
| [How third-party licenses are collected](../third-party-notices.md) (Japanese) | License documents included in distributions and the basis for collecting them |

## Development and maintenance

- [Contributing](../../CONTRIBUTING.en.md): environment setup, how to make changes, checks and review.
- [Maintaining documentation](maintaining/documentation.md): document types, owned information and checks when updating.
- [Releases and publishing](maintaining/releases.md): entry point for distributions, verification commands and publishing steps.
- [Local tickets](../agents/issue-tracker.md) (Japanese) and [status labels](../agents/triage-labels.md) (Japanese): records of specs and work.
- [Domain document practices](../agents/domain.md) (Japanese): where to find terms and design decisions.
- [Work, branch and release policy](../../AGENTS.md): rules and approval scope during development.

## Past investigations and reviews

The following documents are investigations and verifications as of the time they were recorded. For the current specification, see the application contract.

- [SSH implementation notes](../ssh-implementation-notes.md) (Japanese)
- [Final review of the initial implementation](../final-review.md) (Japanese)
- [v0.2.0 review](../v0.2.0-review.md) (Japanese)
- [Versioning policy audit](../versioning-policy-audit.md) (Japanese)

`.scratch/` holds per-feature research, specs, tickets and work records (Japanese). Older specs may have been superseded by later ADRs or the current contract.
