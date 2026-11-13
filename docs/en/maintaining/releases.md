# Releases and publishing

[简体中文](../../zh-CN/maintaining/releases.md) · English · [日本語](../../maintaining/releases.md)

[Documentation index](../README.md) · [Contributing](../../../CONTRIBUTING.en.md)

This page is an overview of distribution and an entry point to the checks. [AGENTS.md](../../../AGENTS.md#tags-and-releases) is authoritative for approval scope and the completion criteria of each stage; before publishing, also check the [application contract](../reference/application-contract.md) (overview) and its [authoritative version](../../application-contract.md) (Japanese), and [release.yml](../../../.github/workflows/release.yml). You do not need to create an application release tag to merge documentation-only changes.

## Development and release contract

Go follows the version in `go.mod`, and the packaging tests use the Python 3.12 standard library. The TUI uses Bubble Tea / Bubbles / Lip Gloss and the CLI uses Cobra, sharing local key handling and state management.

```sh
go test ./...
go vet ./...
python scripts/test_portable.py -v
python scripts/test_release.py -v
python scripts/test_installers.py -v
python scripts/test_installer_profiles.py -v
python scripts/package_release.py --version v0.0.0-check --output .tmp/dist-check
```

Packaging builds six targets with `CGO_ENABLED=0`, `-buildvcs=false`, `-trimpath` and `-ldflags "-s -w -X main.version=<tag>"`. Development builds are `dev`. The `v0.0.0-check` above is an example of version injection for local verification and does not publish anything. Overwriting an output directory that already contains files is refused. To package existing binaries, specify `--binaries DIRECTORY`; the six targets are read from `DIRECTORY/linux_amd64/ssh-keyflow` and so on.

| OS | CPU | Asset name |
| --- | --- | --- |
| Linux | x64 / ARM64 | `ssh-keyflow_<tag>_linux_<amd64\|arm64>.tar.gz` |
| macOS | x64 / ARM64 | `ssh-keyflow_<tag>_darwin_<amd64\|arm64>.tar.gz` |
| Windows | x64 / ARM64 | `ssh-keyflow_<tag>_windows_<amd64\|arm64>.zip` |

Each archive contains `ssh-keyflow` (`ssh-keyflow.exe` on Windows) and `THIRD_PARTY_NOTICES.txt`. The Unix executable mode is 0755. The top-level distribution consists of exactly seven assets: six archives and `SHA256SUMS`. The manifest records, for each of the six archives, the 64-digit SHA256, two ASCII spaces, the base name and LF.

CI runs the Go tests, Portable artifact tests and installer tests on three OSes, and checks actual cross-builds of the six targets and version injection for the Linux amd64 build. Pushing a `v*` tag publishes the seven assets to `GITHUB_REPOSITORY` after CI succeeds. The installers are served separately from the source on `main` and do not change the seven release assets. A prerelease tag is never made the latest stable release.

A release proceeds in this order: merge the preparation PR, verify the final candidate, create and publish the tag, then verify the published distribution and merge the record PR. The preparation record uses the verified source commit, and the final tag's commit is kept in the tag, the release information and the post-release record. It is normal for `main` to move ahead of the tag with the post-release record, and a record-only change does not require a new release. For the completion criteria and approval scope of each stage, see [Tags and releases in AGENTS.md](../../../AGENTS.md#tags-and-releases).

The publish job is serialized across tags; within it, it fetches the current latest release and compares versions in numeric order. A fix release for an older series can be published, but it does not replace a newer latest. If latest returns 404, it is treated as a first publication only when listing all releases succeeds and confirms that no stable release has been published. Publication stops on authentication, network or API response errors, or when latest cannot be fetched although a stable release exists. `scripts/test_release.py` verifies these decisions and the publish commands without publishing externally.

Tests use a temporary home, install location and distribution, and do not change the real user's SSH data, persistent PATH, profiles or destinations. Cross-builds are kept distinct from evidence of running on real hardware for each OS / CPU. For the verified scope, see [docs/verification.md](../../verification.md) (Japanese); for the public interface, see [docs/application-contract.md](../../application-contract.md) (Japanese, authoritative) or its [English overview](../reference/application-contract.md).
