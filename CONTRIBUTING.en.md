# Contributing

[简体中文](CONTRIBUTING.zh-CN.md) · English · [日本語](CONTRIBUTING.md)

[README](README.en.md) · [Documentation index](docs/en/README.md) · [Working policy](AGENTS.md)

We welcome bug fixes, feature proposals, and improvements to tutorials and documentation. Make clear which problem a single change solves and what verifiable completion criteria it has.

## Reports and proposals

Check existing reports in [GitHub Issues](https://github.com/haoyan-dev/ssh-keyflow/issues), and record the version, OS / CPU, terminal, steps to reproduce, and the expected and actual results. For a feature proposal, describe the task you want to accomplish and what the existing features lack. Do not attach private keys, passphrases or authentication tokens, and review logs, paths and destination details before publishing them.

Implementation specs and tickets are managed as [local Markdown](docs/agents/issue-tracker.md) (Japanese). Record the goal, scope, completion criteria and decisions in `.scratch/<feature>/spec.md`, and the work in sequentially numbered files starting at `issues/01-<task>.md`. Keep each ticket's `Status:` and `Blocked by:` in line with the [status labels](docs/agents/triage-labels.md) (Japanese) and its dependencies. Cross-reference these with reports and PRs on GitHub.

## Development environment

- Use the Go version specified in [go.mod](go.mod).
- The verification scripts use the Python 3.12 standard library.
- Have `ssh` and `ssh-keygen` available to verify native key operations.
- Do manual TUI checks in an interactive terminal.

```sh
git clone https://github.com/haoyan-dev/ssh-keyflow.git
cd ssh-keyflow
go run ./cmd/ssh-keyflow --help
```

Development builds report the version `dev`. When trying the TUI or changing operations, prepare a dedicated user environment that does not mix with the keys you actually use. Automated tests use a temporary home.

## How to make a change

1. Read [AGENTS.md](AGENTS.md) and check the working tree and branch. Create a working branch named `<type>/<short-kebab-case-description>` from the latest `origin/main`. External contributors can open a PR from a branch on their fork.
2. Investigate, then write a spec and tickets. Before design or code investigation, check the [glossary](docs/en/reference/glossary.md) and the [related ADRs](docs/adr/README.md) (Japanese).
3. Implement the change and update the necessary documentation in the same PR. For where documents go and what information they own, see [Maintaining documentation](docs/en/maintaining/documentation.md).
4. Run the affected checks, and review conformance to the standards and conformance to the spec separately. Confirm that review findings are resolved.
5. Record the problem, the changes, the full commit SHA you verified, the environment, the results and what was not verified in the PR. Merge only after the required CI and reviews pass.

## Verification

Documentation links, heading references, language links and translation coverage:

```sh
python scripts/test_docs.py -v
python scripts/check_docs.py
```

Checks for changes that affect the implementation or distribution:

```sh
go test ./...
go vet ./...
python scripts/test_portable.py -v
python scripts/test_release.py -v
python scripts/test_installers.py -v
python scripts/test_installer_profiles.py -v
```

For documentation-only changes, run the link check and verify the procedures and commands. If you change a translated page such as the README or a guide, update the [English and Chinese versions](docs/en/maintaining/documentation.md#language-layout) in the same PR, or explicitly mark them as awaiting update. If you change the implementation, dependencies, packaging or installers, run the corresponding tests, and if the packaging inputs change, also apply the six-target build from the [release guide](docs/en/maintaining/releases.md). Required CI checks must pass regardless of the type of change.

Record verification results that will be referred to later in [docs/verification.md](docs/verification.md) (Japanese), with the target SHA, environment, results and limits. Keep cross-builds, runs on real hardware and actual SSH connection checks distinct from each other.

## Review and publishing

Keep each PR to one purpose, and describe user-visible changes and the impact on compatibility and migration. Documentation-only PRs are also merged into `main` after review.

To publish the app, follow the [release guide](docs/en/maintaining/releases.md) in this order: preparation PR, verification of the final candidate, tag and automatic publication, then the distribution audit and record PR. Pushing a tag starts publication, so check the [approval scope in AGENTS.md](AGENTS.md#tags-and-releases).
