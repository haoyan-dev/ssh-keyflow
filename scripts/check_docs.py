"""Check repository-local links in maintained Markdown, without network access.

Supports inline links/images, reference links, ATX headings and explicit HTML
anchors used by these docs. This is not a general Markdown renderer.

Also checks locale coverage: every set in TRANSLATIONS has a Japanese source
page, each declared translation exists, every page of the set has a language
link line directly after its title that links the equivalent pages, and no
locale page exists outside TRANSLATIONS. A translation declared as None is a
disclosed gap: the other pages link the Japanese source as `<Language> → 日本語`.
A translation marked stale must show a notice right after the marker.
"""

from pathlib import Path
import re
import sys
import unicodedata
from urllib.parse import unquote, urlsplit


ROOT = Path(__file__).resolve().parents[1]
PAGES = (
    "README.md", "CONTRIBUTING.md", "AGENTS.md", "GLOSSARY.md",
    "docs/README.md", "docs/application-contract.md", "docs/third-party-notices.md",
)
DIRECTORIES = (
    "docs/tutorials", "docs/guides", "docs/maintaining", "docs/adr", "docs/agents",
    "docs/en", "docs/zh-CN",
)
LANGUAGES = (("zh-CN", "简体中文"), ("en", "English"), ("ja", "日本語"))
SOURCE = "ja"
LOCALE_DIRECTORIES = ("docs/zh-CN", "docs/en")
STALE = "<!-- translation: stale -->"


def _root_page(name):
    stem = name.removesuffix(".md")
    return {"zh-CN": f"{stem}.zh-CN.md", "en": f"{stem}.en.md", "ja": name}


def _docs_page(name, translated=None):
    translated = translated or name
    return {"zh-CN": f"docs/zh-CN/{translated}", "en": f"docs/en/{translated}", "ja": f"docs/{name}"}


TRANSLATIONS = (
    _root_page("README.md"),
    _root_page("CONTRIBUTING.md"),
    _docs_page("README.md"),
    _docs_page("tutorials/first-connection.md"),
    _docs_page("guides/installation.md"),
    _docs_page("guides/terminal-ui.md"),
    _docs_page("guides/cli.md"),
    _docs_page("guides/data-and-recovery.md"),
    _docs_page("guides/troubleshooting.md"),
    _docs_page("maintaining/documentation.md"),
    _docs_page("maintaining/releases.md"),
    _docs_page("application-contract.md", "reference/application-contract.md"),
    {"zh-CN": "docs/zh-CN/reference/glossary.md", "en": "docs/en/reference/glossary.md", "ja": "GLOSSARY.md"},
)
LANGUAGE_LINK = re.compile(r"^\[([^\]]+)\]\(([^)\s]+)\)$")
INLINE_LINK = re.compile(r"!?\[[^\]\n]*\]\(\s*(<[^>\n]*>|[^\s)]+)(?:\s+\"[^\"]*\")?\s*\)")
REFERENCE = re.compile(r"^ {0,3}\[([^\]]+)\]:\s*<?([^\s>]+)>?", re.M)
REFERENCE_LINK = re.compile(r"!?\[([^\]\n]+)\]\[([^\]\n]*)\]")


def prose(text):
    """Remove fenced examples while keeping source line numbers."""
    lines = []
    fence = None
    for line in text.splitlines():
        marker = re.match(r"^ {0,3}(`{3,}|~{3,})", line)
        if marker:
            run = marker[1]
            if fence is None:
                fence = run
            elif run[0] == fence[0] and len(run) >= len(fence) and not line[marker.end():].strip():
                fence = None
            lines.append("")
        else:
            lines.append("" if fence else line)
    return "\n".join(lines)


def anchors(text):
    text = prose(text)
    result = set(re.findall(r'<(?:a|span)\b[^>]*\b(?:id|name)=["\']([^"\']+)["\']', text))
    for line in text.splitlines():
        heading = re.match(r"^ {0,3}#{1,6}\s+(.+?)\s*#*\s*$", line)
        if not heading:
            continue
        title = re.sub(r"<[^>]*>", "", heading[1]).lower()
        slug = "".join(
            c for c in title
            if c in "-_" or not unicodedata.category(c).startswith(("P", "S"))
        ).replace(" ", "-")
        candidate = slug
        suffix = 0
        while candidate in result:
            suffix += 1
            candidate = f"{slug}-{suffix}"
        result.add(candidate)
    return result


def check_page(root, page):
    text = prose(page.read_text(encoding="utf-8"))
    # Link syntax inside inline code is an example, not a link; keep offsets.
    text = re.sub(r"`[^`\n]+`", lambda m: " " * len(m[0]), text)
    references = {m[1].casefold(): m[2] for m in REFERENCE.finditer(text)}
    links = [(m.start(), m[1].strip("<>")) for m in INLINE_LINK.finditer(text)]
    errors = []
    for match in REFERENCE_LINK.finditer(text):
        label = (match[2] or match[1]).casefold()
        if label not in references:
            errors.append(f"{page.relative_to(root)}: undefined reference [{label}]")
        else:
            links.append((match.start(), references[label]))
    for position, destination in links:
        url = urlsplit(destination)
        if url.scheme or url.netloc:
            continue
        target = (page.parent / unquote(url.path)).resolve() if url.path else page
        location = f"{page.relative_to(root)}:{text[:position].count(chr(10)) + 1}"
        if not target.is_relative_to(root):
            errors.append(f"{location}: link escapes repository: {destination}")
        elif not target.exists():
            errors.append(f"{location}: missing target: {destination}")
        elif url.fragment and target.suffix.lower() == ".md":
            if unquote(url.fragment) not in anchors(target.read_text(encoding="utf-8")):
                errors.append(f"{location}: missing heading: {destination}")
    return errors


def check_language_links(root, page, current, entry):
    """Check the language link line that must follow the page title."""
    lines = prose(page.read_text(encoding="utf-8")).splitlines()
    title = next((i for i, line in enumerate(lines) if re.match(r"^#\s", line)), None)
    bar = None if title is None else next(
        (i for i in range(title + 1, len(lines)) if lines[i].strip()), None)
    rel = page.relative_to(root).as_posix()
    if bar is None:
        return [f"{rel}: missing language links after the title"]
    location = f"{rel}:{bar + 1}"
    tokens = lines[bar].strip().split(" · ")
    if len(tokens) != len(LANGUAGES):
        return [f"{location}: language links must list 简体中文 · English · 日本語"]
    source_label = dict(LANGUAGES)[SOURCE]
    errors = []
    for token, (code, label) in zip(tokens, LANGUAGES):
        if code == current:
            if token != label:
                errors.append(f"{location}: current language must be plain text {label}")
            continue
        target = entry[code]
        expected_label = label if target else f"{label} → {source_label}"
        expected = target or entry[SOURCE]
        match = LANGUAGE_LINK.match(token)
        if not match or match[1] != expected_label:
            errors.append(f"{location}: expected language link [{expected_label}]({expected})")
            continue
        url = urlsplit(match[2])
        resolved = (page.parent / unquote(url.path)).resolve()
        if url.scheme or url.netloc or url.fragment or resolved != (root / expected).resolve():
            errors.append(f"{location}: {label} link must point to {expected}")
    return errors


def check_stale(root, page):
    """Return (is_stale, errors); a stale marker needs a visible notice."""
    lines = prose(page.read_text(encoding="utf-8")).splitlines()
    marker = next((i for i, line in enumerate(lines) if line.strip() == STALE), None)
    if marker is None:
        return False, []
    notice = next((line for line in lines[marker + 1:] if line.strip()), "")
    if notice.startswith("> "):
        return True, []
    return True, [f"{page.relative_to(root).as_posix()}:{marker + 1}: stale marker needs a '> ' notice"]


def check_translations(root, translations):
    """Return (errors, stats) for locale coverage and language links."""
    errors = []
    stats = {"sets": len(translations), "complete": 0, "fallback": 0, "stale": 0}
    declared = {path for entry in translations for path in entry.values() if path}
    for entry in translations:
        if not (root / entry[SOURCE]).is_file():
            errors.append(f"missing source page: {entry[SOURCE]}")
            continue
        for code, _ in LANGUAGES:
            path = entry[code]
            if path is None:
                stats["fallback"] += 1
                continue
            page = root / path
            if not page.is_file():
                errors.append(f"missing translation: {path} ({code} for {entry[SOURCE]})")
                continue
            if code != SOURCE:
                stale, problems = check_stale(root, page)
                errors.extend(problems)
                stats["stale" if stale else "complete"] += 1
            errors.extend(check_language_links(root, page, code, entry))
    locale_pages = [p for d in LOCALE_DIRECTORIES for p in (root / d).rglob("*.md")]
    locale_pages += [p for code, _ in LANGUAGES for p in root.glob(f"*.{code}.md")]
    for page in sorted(set(locale_pages)):
        if page.relative_to(root).as_posix() not in declared:
            errors.append(f"undeclared translation page: {page.relative_to(root).as_posix()}")
    return errors, stats


def main():
    for stream in (sys.stdout, sys.stderr):
        stream.reconfigure(encoding="utf-8")
    pages = [ROOT / name for name in PAGES]
    pages += [page for name in DIRECTORIES for page in (ROOT / name).rglob("*.md")]
    pages += [ROOT / path for entry in TRANSLATIONS for path in entry.values() if path]
    errors = []
    for page in sorted(set(pages)):
        if page.is_file():
            errors.extend(check_page(ROOT, page))
        elif page.relative_to(ROOT).as_posix() in PAGES:
            errors.append(f"missing maintained page: {page.relative_to(ROOT)}")
    problems, stats = check_translations(ROOT, TRANSLATIONS)
    errors += problems
    if errors:
        print("\n".join(errors), file=sys.stderr)
        return 1
    count = len({page for page in pages if page.is_file()})
    print(
        f"Documentation links passed ({count} maintained pages; offline). "
        f"Locales zh-CN/en/ja: {stats['sets']} page sets, {stats['complete']} complete "
        f"translations, {stats['stale']} stale, {stats['fallback']} Japanese fallbacks."
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
