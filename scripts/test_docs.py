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
            redirect_stderr(io.StringIO()) as errors,
            redirect_stdout(io.StringIO()) as output,
        ):
            self.page.write_text("[x](missing.md)", encoding="utf-8")
            self.assertEqual(1, check_docs.main())
            self.assertIn("missing target", errors.getvalue())
            self.page.write_text("# Valid\n[x](#valid)", encoding="utf-8")
            self.assertEqual(0, check_docs.main())
            self.assertIn("1 maintained pages", output.getvalue())


if __name__ == "__main__":
    unittest.main()
