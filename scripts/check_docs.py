"""Check repository-local links in maintained Markdown, without network access.

Supports inline links/images, reference links, ATX headings and explicit HTML
anchors used by these docs. This is not a general Markdown renderer.
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
)
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


def main():
    pages = [ROOT / name for name in PAGES]
    pages += [page for name in DIRECTORIES for page in (ROOT / name).rglob("*.md")]
    errors = []
    for page in sorted(set(pages)):
        if not page.is_file():
            errors.append(f"missing maintained page: {page.relative_to(ROOT)}")
        else:
            errors.extend(check_page(ROOT, page))
    if errors:
        print("\n".join(errors), file=sys.stderr)
        return 1
    print(f"Documentation links passed ({len(set(pages))} maintained pages; offline).")
    return 0


if __name__ == "__main__":
    sys.exit(main())
