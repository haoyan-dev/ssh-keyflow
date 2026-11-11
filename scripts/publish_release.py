"""Publish built assets; caller must serialize this operation across release tags."""

import argparse
import json
from pathlib import Path
import re
import subprocess
import sys


class ReleaseError(ValueError):
    pass


def version(tag):
    """Return numeric stable precedence and whether this tag is a preview."""
    if not isinstance(tag, str):
        raise ReleaseError('Release tag must be a string.')
    match = re.fullmatch(r'v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)'
                         r'(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?', tag)
    if not match:
        raise ReleaseError('Expected a release tag such as v1.2.3 or v1.2.3-rc.1.')
    preview = match[4]
    if preview and any(part.isdigit() and len(part) > 1 and part[0] == '0'
                       for part in preview.split('.')):
        raise ReleaseError('Numeric prerelease identifiers must not have leading zeroes.')
    return tuple(int(match[index]) for index in (1, 2, 3)), preview is not None


def release_record(value):
    if (not isinstance(value, dict) or not isinstance(value.get('tag_name'), str)
            or type(value.get('draft')) is not bool
            or type(value.get('prerelease')) is not bool):
        raise ReleaseError('GitHub returned an invalid release record.')
    return value


def decode_json(text):
    try:
        return json.loads(text)
    except json.JSONDecodeError as error:
        raise ReleaseError('GitHub returned invalid JSON; publication stopped.') from error


def current_latest(repository):
    endpoint = f'repos/{repository}/releases'
    result = subprocess.run(['gh', 'api', endpoint + '/latest', '--include'],
                            capture_output=True, text=True, check=False)
    # --include exposes the HTTP status, so a network/auth failure cannot be
    # confused with an absent latest release by matching an error message.
    response = re.fullmatch(r'HTTP/\S+ ([0-9]{3})[^\n]*\n.*?\n\n(.*)',
                            result.stdout.replace('\r\n', '\n'), re.DOTALL)
    if response is None:
        raise ReleaseError('Latest lookup did not return an HTTP response; publication stopped.')
    status = int(response[1])
    if status == 200 and result.returncode == 0:
        latest = release_record(decode_json(response[2]))
        if latest['draft'] or latest['prerelease']:
            raise ReleaseError('Latest release is not a published stable release.')
        numeric, preview = version(latest['tag_name'])
        if preview:
            raise ReleaseError('Latest release has a prerelease tag.')
        return numeric
    if status != 404:
        raise ReleaseError(f'Latest lookup failed (HTTP {status}); publication stopped.')

    # A 404 can also hide an inaccessible repository. Require a successful
    # paginated listing with no published stable releases before bootstrapping.
    result = subprocess.run(['gh', 'api', endpoint, '--paginate', '--slurp'],
                            capture_output=True, text=True, check=False)
    if result.returncode:
        raise ReleaseError('Could not verify absence of stable releases; publication stopped.')
    pages = decode_json(result.stdout)
    if not isinstance(pages, list) or not pages or any(not isinstance(page, list) for page in pages):
        raise ReleaseError('GitHub returned an invalid release listing.')
    for page in pages:
        for item in page:
            item = release_record(item)
            if not item['draft'] and not item['prerelease']:
                raise ReleaseError('Latest is missing despite a published stable release; publication stopped.')
    return None


def publish(repository, tag, assets):
    numeric, preview = version(tag)
    if not re.fullmatch(r'[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+', repository):
        raise ReleaseError('Expected an explicit OWNER/REPO repository.')
    if not assets:
        raise ReleaseError('No release assets were found.')
    latest = None if preview else current_latest(repository)
    advance = not preview and (latest is None or numeric > latest)
    command = ['gh', 'release', 'create', tag, *(str(asset) for asset in assets),
               '--repo', repository, '--verify-tag', '--generate-notes',
               '--latest=true' if advance else '--latest=false']
    if preview:
        command.append('--prerelease')
    subprocess.run(command, check=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--repository', required=True)
    parser.add_argument('--version', required=True)
    parser.add_argument('--assets', type=Path, default=Path('dist'))
    arguments = parser.parse_args()
    try:
        publish(arguments.repository, arguments.version,
                sorted(path for path in arguments.assets.glob('*') if path.is_file()))
    except (ReleaseError, subprocess.CalledProcessError, OSError) as error:
        print(f'Release failed: {error}', file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())
