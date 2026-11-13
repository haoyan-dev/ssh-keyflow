# Maintaining documentation

[简体中文](../../zh-CN/maintaining/documentation.md) · English · [日本語](../../maintaining/documentation.md)

[Documentation index](../README.md) · [Contributing](../../../CONTRIBUTING.en.md)

Documentation is kept as Markdown readable on GitHub and managed in the same PR as the source. The supported languages are Simplified Chinese (`zh-CN`), English (`en`) and Japanese (`ja`). Commands, identifiers and screen labels use the implementation's spelling.

## Language support requirements

The README, documentation index, tutorials, how-to and troubleshooting guides, contributing guide, and documentation maintenance and publishing guides are provided in all three languages. The public specification must also be reachable from each language, while the authoritative contract and terminology are maintained in one place. Past research, tickets, ADRs and verification records do not need to be translated retroactively.

- Put `简体中文` / `English` / `日本語` language links in the README and each page so readers can move to the page with the same content. On an untranslated page, state what is missing and which language is the fallback.
- Keep links to the existing root paths and published headings. The Japanese version is the translation source and stays at the existing paths.
- Keep prerequisites, examples, safety conditions, and migration and downgrade limits aligned across languages. Do not translate commands, flags, JSON keys, paths or product identifiers.
- A PR that changes content also updates the corresponding translations. Untranslated or awaiting-update pages are explicitly marked and tracked; do not present the docs as fully supported in all languages while something is missing.
- CI checks links, headings, cross-language correspondence and translation coverage for all three languages. Terminology and technical accuracy of translations are reviewed per language.

These requirements cover documentation only; they do not include translating the app's UI or CLI messages.

## Language layout

| Japanese (translation source) | English | 简体中文 |
| --- | --- | --- |
| `README.md` / `CONTRIBUTING.md` | `README.en.md` / `CONTRIBUTING.en.md` | `README.zh-CN.md` / `CONTRIBUTING.zh-CN.md` |
| `docs/<path>` (index, tutorials, guides, maintaining) | `docs/en/<path>` | `docs/zh-CN/<path>` |
| `docs/application-contract.md` (authoritative) | `docs/en/reference/application-contract.md` (overview) | `docs/zh-CN/reference/application-contract.md` (overview) |
| `GLOSSARY.md` (authoritative) | `docs/en/reference/glossary.md` (overview) | `docs/zh-CN/reference/glossary.md` (overview) |

The English and Chinese versions of the contract and glossary are overviews. They link to the sections of the authoritative source and state that the Japanese source wins if they differ. Terms are shown together with the Japanese spelling used in the authoritative source. ADRs, verification records, past research and reviews, `AGENTS.md`, agent operation documents and third-party license reconciliation are not translated. In the translated indexes, links to them indicate the original language.

Put the language links right after each page title in the form `[简体中文](…) · [English](…) · 日本語`, and do not link the current language. Links point to the page with the same content and carry no heading fragment. For a page that cannot be translated, set that language to `None` in `TRANSLATIONS` in `scripts/check_docs.py`, and link from the other language versions to the Japanese version like `[English → 日本語](…)`. If a translation has not caught up with changes to the Japanese version, put `<!-- translation: stale -->` after the title and language links, followed by a `> ` notice written in that language. The checker reports complete, awaiting-update (stale) and fallback counts separately.

## Placement and information ownership

| Location | Readers and role | When to update |
| --- | --- | --- |
| `README.md` | First-time visitors. Purpose, installation, which page to read next | Changes to the product scope, installation entry point or main pages |
| `docs/README.md` | People looking for a procedure. Index of tasks and documents | Pages added, moved or retired |
| `docs/tutorials/` | First-time users. Ordered steps to one result | Changes to first-time operations |
| `docs/guides/` | Users with a goal. Recurring tasks and troubleshooting | Changes to procedures or constraints |
| `docs/application-contract.md` | Users and automation authors. Exact promises of the public interface | Changes to the CLI, JSON, compatibility or distribution requirements |
| `GLOSSARY.md` / `docs/adr/` | Designers. Terms and the reasons for decisions | Terms or design decisions added or updated |
| `CONTRIBUTING.md` / `docs/maintaining/` | Contributors and maintainers. Development and publishing | Changes to checks, reviews or maintenance procedures |
| `*.en.md` / `*.zh-CN.md` / `docs/en/` / `docs/zh-CN/` | English and Chinese readers. Translations equivalent to the Japanese pages, and overviews of the contract and glossary | Changes to the corresponding Japanese page |
| `docs/verification.md` | Reviewers and maintainers. Point-in-time verification evidence | Completion of verification or release audits |
| `.scratch/<feature>/` | Workers. Research, specs, tickets and work records | Per-feature investigation and progress |

`application-contract.md` is authoritative for the contract, the glossary for terms, and `AGENTS.md` for publishing rules. Guides show the examples needed for a task and do not duplicate every flag or the schema. Verification records and past reviews reflect the time they were recorded and do not substitute for the current contract.

## Writing pages

- Start with the purpose and a link back to the documentation index.
- In a tutorial, state the prerequisites, the terminal to operate on, the values to replace, the steps, the expected result and the next task.
- In a guide, state the conditions it applies to, the task, how to verify it and recovery caveats where needed.
- Specify a language such as `sh` / `powershell` for code blocks. Do not mix shell prompt characters into commands.
- Make examples match real commands. If a pinned version is an example of how to specify a version, say so.
- Never publish private keys or credentials. Distinguish generating or displaying from running, removal from management from revoking access, and building from verifying on real hardware.
- Use relative paths for internal links, and check existing links to a page when you move it. Where possible, leave a pointer for published headings.

## Checking a PR

1. Confirm that the new page can be reached from the [index](../README.md).
2. Run `python scripts/test_docs.py` and `python scripts/check_docs.py` to verify link targets, headings, language links and translation coverage. These checks do not use the network and do not check external URL responses or translation quality.
3. In a Markdown preview, check headings, tables, code blocks and navigation between pages. Check command examples against the contract, `--help` and the affected tests.
4. In the standards review, check placement, language, terminology, information ownership and existing links. In the spec review, check separately the ticket's completion criteria and whether users can complete the procedure. If translations changed, compare terminology, command examples, safety conditions and compatibility promises against the Japanese version for each language.
5. Record the verified SHA, environment, results and limits in the PR, and do not treat unverified items as successful.

The checker covers the README, CONTRIBUTING, the glossary, AGENTS, and the index, tutorials, guides, maintaining, ADRs, agent operation documents and the application contract, together with their English and Chinese versions. For historical documents and `.scratch/`, it checks that link targets exist, but old content is not recursively rewritten to match the current specification. When you add a new set of documents or a page to be translated, also update the targets and `TRANSLATIONS` in `scripts/check_docs.py`.

Documentation-only changes are published as normal reviewed PRs. Application versions, tags and distributions follow the [existing publishing procedure](releases.md).
