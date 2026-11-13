"""Regression checks for documentation link validation."""

from pathlib import Path
from contextlib import redirect_stderr, redirect_stdout
import io
import tempfile
import unittest
from unittest.mock import patch

import check_docs
from check_docs import anchors, check_page


class DocumentationChecks(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name).resolve()
        self.page = self.root / "README.md"

    def check(self, text):
        self.page.write_text(text, encoding="utf-8")
        return check_page(self.root, self.page)

    def test_relative_and_encoded_targets(self):
        (self.root / "my guide.md").write_text("# 日本語の見出し\n", encoding="utf-8")
        self.assertEqual([], self.check("[guide](my%20guide.md#日本語の見出し)"))

    def test_broken_path_and_fragment(self):
        errors = self.check("# Existing\n[a](missing.md)\n[b](#absent)")
        self.assertEqual(2, len(errors))
        self.assertIn("README.md:2: missing target", errors[0])
        self.assertIn("README.md:3: missing heading", errors[1])

    def test_fences_are_ignored_and_lines_preserved(self):
        errors = self.check("````md\n```\n[x](missing.md)\n````\n[x](other.md)")
        self.assertEqual(1, len(errors))
        self.assertIn("README.md:5:", errors[0])

    def test_duplicate_headings_and_punctuation(self):
        self.assertEqual(
            {"v010の利用者", "v010の利用者-1", "cli--json"},
            anchors("# v0.1.0の利用者\n# v0.1.0の利用者\n## CLI / JSON\n"),
        )

    def test_inline_code_examples_are_not_links(self):
        errors = self.check("Use `[English](…)` here.\n`a` [x](missing.md)")
        self.assertEqual(1, len(errors))
        self.assertIn("README.md:2: missing target", errors[0])

    def test_external_links_are_offline(self):
        self.assertEqual([], self.check("[web](https://example.invalid/x#missing)"))

    def test_path_cannot_escape_repository(self):
        self.assertIn("escapes repository", self.check("[x](../outside.md)")[0])

    def test_explicit_anchor_and_reference_links(self):
        self.assertEqual([], self.check('<a id="old"></a>\n[back][legacy]\n[legacy]: #old'))
        self.assertIn("undefined reference", self.check("[back][unknown]")[0])

    def test_html_anchor_and_heading_share_namespace(self):
        self.assertEqual({"old", "old-1"}, anchors('<a id="old"></a>\n# Old'))

    def test_anchors_inside_fenced_examples_do_not_exist(self):
        self.assertEqual({"real"}, anchors('# Real\n```html\n<a id="fake"></a>\n```'))

    def test_ci_exit_status_reports_broken_link(self):
        with (
            patch.object(check_docs, "ROOT", self.root),
            patch.object(check_docs, "PAGES", ("README.md",)),
            patch.object(check_docs, "DIRECTORIES", ()),
            patch.object(check_docs, "TRANSLATIONS", ()),
            redirect_stderr(io.StringIO()) as errors,
            redirect_stdout(io.StringIO()) as output,
        ):
            self.page.write_text("[x](missing.md)", encoding="utf-8")
            self.assertEqual(1, check_docs.main())
            self.assertIn("missing target", errors.getvalue())
            self.page.write_text("# Valid\n[x](#valid)", encoding="utf-8")
            self.assertEqual(0, check_docs.main())
            self.assertIn("1 maintained pages", output.getvalue())


SET = {"zh-CN": "docs/zh-CN/guide.md", "en": "docs/en/guide.md", "ja": "docs/guide.md"}
BARS = {
    "zh-CN": "简体中文 · [English](../en/guide.md) · [日本語](../guide.md)",
    "en": "[简体中文](../zh-CN/guide.md) · English · [日本語](../guide.md)",
    "ja": "[简体中文](zh-CN/guide.md) · [English](en/guide.md) · 日本語",
}


class LocaleChecks(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name).resolve()
        for code in BARS:
            self.write(SET[code], f"# Title\n\n{BARS[code]}\n\nBody\n")

    def write(self, path, text):
        (self.root / path).parent.mkdir(parents=True, exist_ok=True)
        (self.root / path).write_text(text, encoding="utf-8")

    def check(self, translations=(SET,)):
        return check_docs.check_translations(self.root, translations)

    def test_complete_set_passes(self):
        errors, stats = self.check()
        self.assertEqual([], errors)
        self.assertEqual({"sets": 1, "complete": 2, "fallback": 0, "stale": 0}, stats)

    def test_missing_translation_is_reported(self):
        (self.root / SET["en"]).unlink()
        errors, _ = self.check()
        self.assertIn("missing translation: docs/en/guide.md", errors[0])

    def test_missing_source_is_reported(self):
        (self.root / SET["ja"]).unlink()
        self.assertIn("missing source page: docs/guide.md", self.check()[0][0])

    def test_language_link_must_point_to_equivalent_page(self):
        self.write("docs/en/other.md", "# Other\n")
        self.write(SET["ja"], "# T\n\n[简体中文](zh-CN/guide.md) · [English](en/other.md) · 日本語\n")
        errors, _ = self.check()
        self.assertTrue(any("docs/guide.md:3: English link must point to docs/en/guide.md" in e for e in errors))
        self.assertTrue(any("undeclared translation page: docs/en/other.md" in e for e in errors))

    def test_language_link_rejects_fragment_and_missing_bar(self):
        self.write(SET["ja"], "# T\n\n[简体中文](zh-CN/guide.md#x) · [English](en/guide.md) · 日本語\n")
        self.write(SET["en"], "# T\n\nNo language links here.\n")
        errors, _ = self.check()
        self.assertTrue(any("简体中文 link must point" in e for e in errors))
        self.assertTrue(any("docs/en/guide.md:3: language links must list" in e for e in errors))

    def test_current_language_is_not_a_link(self):
        self.write(SET["en"], "# T\n\n[简体中文](../zh-CN/guide.md) · [English](guide.md) · [日本語](../guide.md)\n")
        self.assertIn("current language must be plain text English", self.check()[0][0])

    def test_declared_fallback_is_disclosed_and_counted(self):
        (self.root / SET["en"]).unlink()
        fallback = dict(SET, en=None)
        errors, _ = self.check((fallback,))
        self.assertEqual(2, len(errors))
        self.assertIn("expected language link [English → 日本語](docs/guide.md)", errors[0])
        self.write(SET["ja"], "# T\n\n[简体中文](zh-CN/guide.md) · [English → 日本語](guide.md) · 日本語\n")
        self.write(SET["zh-CN"], "# T\n\n简体中文 · [English → 日本語](../guide.md) · [日本語](../guide.md)\n")
        errors, stats = self.check((fallback,))
        self.assertEqual([], errors)
        self.assertEqual({"sets": 1, "complete": 1, "fallback": 1, "stale": 0}, stats)

    def test_stale_translation_needs_notice_and_is_not_complete(self):
        stale = f"# T\n\n{BARS['en']}\n\n{check_docs.STALE}\n\nBody\n"
        self.write(SET["en"], stale)
        errors, _ = self.check()
        self.assertIn("stale marker needs a '> ' notice", errors[0])
        self.write(SET["en"], stale.replace("Body", "> Out of date; see the Japanese page."))
        errors, stats = self.check()
        self.assertEqual([], errors)
        self.assertEqual(1, stats["stale"])
        self.assertEqual(1, stats["complete"])

    def test_locale_page_links_and_fragments_are_checked(self):
        page = self.root / SET["zh-CN"]
        self.write(SET["zh-CN"], f"# 标题\n\n{BARS['zh-CN']}\n\n[a](../guide.md#missing) [b](#标题)\n")
        errors = check_page(self.root, page)
        self.assertEqual(1, len(errors))
        self.assertIn("docs/zh-CN/guide.md:5: missing heading", errors[0].replace("\\", "/"))

    def test_repository_translation_table_is_consistent(self):
        codes = [code for code, _ in check_docs.LANGUAGES]
        for entry in check_docs.TRANSLATIONS:
            self.assertEqual(codes, list(entry))
            self.assertTrue(entry["ja"])


if __name__ == "__main__":
    unittest.main()
