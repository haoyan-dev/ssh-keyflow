import base64
import functools
import hashlib
import http.server
import io
import json
import os
from pathlib import Path
import shlex
import shutil
import stat
import subprocess
import tarfile
import tempfile
import threading
import unittest
import warnings
import zipfile


ROOT = Path(__file__).resolve().parents[1]
BASH = str(Path(os.environ.get('ProgramFiles', 'C:/Program Files')) / 'Git/bin/bash.exe') if os.name == 'nt' else shutil.which('bash')
POWERSHELL = shutil.which('pwsh') or shutil.which('powershell')
NOTICES = 'THIRD_PARTY_NOTICES.txt'
DEFAULT_REPOSITORY = 'haoyan-dev/ssh-keyflow'


def shell_path(path):
    value = str(path).replace('\\', '/')
    return '/' + value[0].lower() + value[2:] if len(value) > 1 and value[1] == ':' else value


def digest(content):
    return hashlib.sha256(content).hexdigest()


class QuietHandler(http.server.SimpleHTTPRequestHandler):
    def log_message(self, format, *args):
        pass


class InstallerTests:
    """Common contract, run independently against each native installer."""

    platform = None

    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix='keyflow installer ')
        self.addCleanup(self.temporary.cleanup)
        self.home = Path(self.temporary.name) / 'home space'
        self.home.mkdir()
        self.home = self.home.resolve()
        self.destination = self.home / 'bin space'
        self.profile = self.home / 'test profile'
        self.profile.write_bytes(b'export EXISTING=keep\n')
        self.data = self.home / '.ssh' / 'ssh-keyflow'
        self.data.mkdir(parents=True)
        (self.data / 'state.json').write_bytes(b'preserve state')
        (self.data / 'private-key').write_bytes(b'preserve key')
        (self.data / 'config').write_bytes(b'preserve managed config')
        (self.home / '.ssh' / 'config').write_bytes(b'Host existing\n  HostName unchanged\n')
        self.ssh_snapshot = self.tree(self.home / '.ssh')
        self.tools = self.home / 'tools'
        self.tools.mkdir()
        self.releases = Path(self.temporary.name) / 'releases'
        self.releases.mkdir()
        self.fixture('v1.0.0')
        self.fixture('v2.0.0')
        self.latest = self.releases / 'api' / 'releases' / 'latest'
        self.latest.parent.mkdir(parents=True)
        self.latest.write_text(json.dumps({'tag_name': 'v1.0.0', 'prerelease': False, 'draft': False}))
        for name in ('install.sh', 'install.ps1'):
            (self.releases / name).write_bytes((ROOT / name).read_bytes().replace(b'\r\n', b'\n'))
        handler = functools.partial(QuietHandler, directory=str(self.releases))
        self.server = http.server.ThreadingHTTPServer(('127.0.0.1', 0), handler)
        self.addCleanup(self.server.server_close)
        self.addCleanup(self.server.shutdown)
        threading.Thread(target=self.server.serve_forever, daemon=True).start()
        self.base = 'http://127.0.0.1:' + str(self.server.server_port)

    def tearDown(self):
        self.assertEqual(self.tree(self.home / '.ssh'), self.ssh_snapshot)
        if self.destination.exists():
            self.assertFalse(list(self.destination.glob('.ssh-keyflow-stage.*')))
            self.assertFalse((self.destination / '.ssh-keyflow-lock').exists())

    @staticmethod
    def tree(root):
        if not root.exists():
            return {}
        return {str(path.relative_to(root)): path.read_bytes() if path.is_file() else None
                for path in root.rglob('*')}

    @property
    def executable(self):
        return 'ssh-keyflow.exe' if self.platform == 'windows' else 'ssh-keyflow'

    @property
    def binary(self):
        return self.destination / self.executable

    @property
    def notices(self):
        return self.destination / NOTICES

    @property
    def receipt(self):
        return self.destination / ('.ssh-keyflow-install.json' if self.platform == 'windows' else '.ssh-keyflow-install')

    @property
    def consent(self):
        return '-AddToPath' if self.platform == 'windows' else '--add-to-path'

    @property
    def uninstall(self):
        return '-Uninstall' if self.platform == 'windows' else '--uninstall'

    def fixture(self, version):
        release = self.releases / version
        release.mkdir(exist_ok=True)
        for system in ('linux', 'darwin', 'windows'):
            for architecture in ('amd64', 'arm64'):
                executable = 'ssh-keyflow.exe' if system == 'windows' else 'ssh-keyflow'
                content = (system + '/' + architecture + '/' + version + '\n').encode()
                name = 'ssh-keyflow_' + version + '_' + system + '_' + architecture
                archive = release / (name + ('.zip' if system == 'windows' else '.tar.gz'))
                members = [(executable, content, 'file'), (NOTICES, self.notice_content(version), 'file')]
                self.write_archive(archive, members)
        self.manifest(release)

    @staticmethod
    def notice_content(version):
        return ('Third-party notices for ' + version + '\n').encode()

    @staticmethod
    def write_archive(archive, members):
        if archive.suffix == '.zip':
            modes = {'file': stat.S_IFREG, 'symlink': stat.S_IFLNK, 'directory': stat.S_IFDIR,
                     'fifo': stat.S_IFIFO, 'character': stat.S_IFCHR, 'block': stat.S_IFBLK,
                     'socket': stat.S_IFSOCK}
            with warnings.catch_warnings():
                warnings.simplefilter('ignore', UserWarning)  # Duplicate names are intentional.
                with zipfile.ZipFile(archive, 'w') as bundle:
                    for name, content, kind in members:
                        entry = zipfile.ZipInfo(name)
                        entry.create_system = 3
                        entry.external_attr = (modes[kind] | 0o755) << 16
                        if kind == 'directory':
                            entry.external_attr |= 0x10
                        bundle.writestr(entry, content)
        else:
            types = {'file': tarfile.REGTYPE, 'symlink': tarfile.SYMTYPE,
                     'hardlink': tarfile.LNKTYPE, 'directory': tarfile.DIRTYPE,
                     'fifo': tarfile.FIFOTYPE, 'character': tarfile.CHRTYPE, 'block': tarfile.BLKTYPE}
            with tarfile.open(archive, 'w:gz') as bundle:
                for name, content, kind in members:
                    entry = tarfile.TarInfo(name)
                    entry.type = types[kind]
                    entry.mode = 0o755 if name == 'ssh-keyflow' else 0o644
                    if kind == 'file':
                        entry.size = len(content)
                    if kind in ('symlink', 'hardlink'):
                        entry.linkname = '../escape'
                    bundle.addfile(entry, io.BytesIO(content) if kind == 'file' else None)

    @staticmethod
    def manifest(release):
        archives = sorted(list(release.glob('*.zip')) + list(release.glob('*.tar.gz')))
        (release / 'SHA256SUMS').write_text(
            ''.join(digest(path.read_bytes()) + '  ' + path.name + '\n' for path in archives),
            encoding='ascii', newline='\n')

    def archive(self, version='v2.0.0'):
        target = 'windows_amd64.zip' if self.platform == 'windows' else 'linux_amd64.tar.gz'
        return self.releases / version / ('ssh-keyflow_' + version + '_' + target)

    def run_installer(self, version=None, extra=(), answer='n\n', system='Linux', cpu='x86_64',
                      repository='fixture/ssh-keyflow', powershell=None, pipeline=False,
                      public_endpoints=False, lock_path=None):
        environment = os.environ.copy()
        for name in ('SSH_KEYFLOW_RELEASE_URL', 'SSH_KEYFLOW_API_URL', 'SSH_KEYFLOW_REPOSITORY'):
            environment.pop(name, None)
        if not public_endpoints:
            environment.update(SSH_KEYFLOW_RELEASE_URL=self.base, SSH_KEYFLOW_API_URL=self.base + '/api')
        if repository is not None:
            environment['SSH_KEYFLOW_REPOSITORY'] = repository
        environment.update(
            HOME=str(self.home), USERPROFILE=str(self.home), LOCALAPPDATA=str(self.home / 'local'),
            XDG_CONFIG_HOME=str(self.home / '.config'), CURL_HOME=str(self.home),
            SSH_KEYFLOW_TEST_BASE=self.base,
            SSH_KEYFLOW_TEST_URLS=str(self.home / 'requested urls'),
            SSH_KEYFLOW_TEST_PROFILE=str(self.profile),
            SSH_KEYFLOW_TEST_SOURCE=self.base + '/install.' + ('ps1' if self.platform == 'windows' else 'sh'),
            NO_PROXY='*', no_proxy='*',
        )
        if self.platform == 'shell':
            (self.tools / 'uname').write_text(
                '#!/bin/sh\ncase "$1" in\n-s) echo ' + shlex.quote(system) +
                ';;\n-m) echo ' + shlex.quote(cpu) + ';;\nesac\n', encoding='utf-8', newline='\n')
            (self.tools / 'uname').chmod(0o755)
            environment.update(HOME=shell_path(self.home), SHELL='/bin/sh',
                               SSH_KEYFLOW_TEST_TOOLS=shell_path(self.tools),
                               SSH_KEYFLOW_TEST_URLS=shell_path(self.home / 'requested urls'))
            installer = self.home / 'install.sh'
            installer.write_bytes((ROOT / 'install.sh').read_bytes().replace(b'\r\n', b'\n'))
            if public_endpoints:
                self.redirect_shell_downloads()
            prefix = 'PATH="$SSH_KEYFLOW_TEST_TOOLS:$PATH"; export PATH; '
            if pipeline:
                command = prefix + 'set -o pipefail; curl -fsSL "$SSH_KEYFLOW_TEST_SOURCE" | sh'
                arguments = [BASH, '--noprofile', '--norc', '-c', command]
            else:
                arguments = [BASH, '--noprofile', '--norc', '-c', prefix + 'exec "$0" "$@"',
                             getattr(self, 'shell', 'sh'), shell_path(installer),
                             '--install-dir', shell_path(self.destination), '--profile', shell_path(self.profile)]
                if version:
                    arguments += ['--version', version]
                arguments += list(extra)
        else:
            environment['PROCESSOR_ARCHITECTURE'] = 'ARM64' if cpu == 'arm64' else 'AMD64'
            environment.pop('PROCESSOR_ARCHITEW6432', None)
            command = self.windows_redirects() if public_endpoints else ''
            if lock_path:
                environment['SSH_KEYFLOW_TEST_LOCK'] = str(lock_path)
                command += '$held = [IO.File]::Open($env:SSH_KEYFLOW_TEST_LOCK, "Open", "Read", "Read"); '
            if pipeline:
                command += ('$PROFILE = [pscustomobject]@{CurrentUserAllHosts=$env:SSH_KEYFLOW_TEST_PROFILE}; '
                            'irm $env:SSH_KEYFLOW_TEST_SOURCE | iex')
            else:
                parameters = ['-InstallDir', str(self.destination), '-ProfilePath', str(self.profile)]
                if version:
                    parameters += ['-Version', version]
                parameters += list(extra)
                command += '& ' + self.ps_quote(ROOT / 'install.ps1') + ' ' + ' '.join(
                    value if value.startswith('-') else self.ps_quote(value) for value in parameters)
            encoded = base64.b64encode(command.encode('utf-16-le')).decode('ascii')
            arguments = [powershell or POWERSHELL, '-NoProfile', '-ExecutionPolicy', 'Bypass',
                         '-EncodedCommand', encoded]
        # Detach from any controlling terminal: tests may never prompt on the real tty.
        options = {'creationflags': subprocess.CREATE_NO_WINDOW} if os.name == 'nt' else {'start_new_session': True}
        return subprocess.run(arguments, input=answer, text=True, encoding='utf-8', errors='replace',
                              capture_output=True, env=environment, cwd=self.home, timeout=45, **options)

    @staticmethod
    def ps_quote(value):
        return "'" + str(value).replace("'", "''") + "'"

    def redirect_shell_downloads(self):
        native_curl = subprocess.check_output(
            [BASH, '--noprofile', '--norc', '-c', 'command -v curl'], text=True).strip()
        wrapper = self.tools / 'curl'
        wrapper.write_text("""#!/bin/sh
count=$#
while [ "$count" -gt 0 ]; do
    argument=$1; shift; count=$((count - 1))
    case "$argument" in
        https://github.com/*/releases/download/*)
            printf '%s\\n' "$argument" >> "$SSH_KEYFLOW_TEST_URLS"
            suffix=${argument#https://github.com/}
            argument=$SSH_KEYFLOW_TEST_BASE/${suffix#*/releases/download/} ;;
        https://api.github.com/repos/*/releases/latest)
            printf '%s\\n' "$argument" >> "$SSH_KEYFLOW_TEST_URLS"
            argument=$SSH_KEYFLOW_TEST_BASE/api/releases/latest ;;
        http://127.0.0.1:*) ;;
        http://*|https://*) exit 97 ;;
    esac
    set -- "$@" "$argument"
done
exec """ + shlex.quote(native_curl) + ' "$@"\n', encoding='utf-8', newline='\n')
        wrapper.chmod(0o755)

    @staticmethod
    def windows_redirects():
        # Module-qualified calls cannot recurse into these local wrappers.
        return r"""
function Convert-FixtureUri([string]$Uri) {
    if ($Uri -like 'http://127.0.0.1:*') { return $Uri }
    [IO.File]::AppendAllText($env:SSH_KEYFLOW_TEST_URLS, $Uri + "`n")
    if ($Uri -match '^https://github.com/[^/]+/[^/]+/releases/download/(.+)$') {
        return $env:SSH_KEYFLOW_TEST_BASE + '/' + $Matches[1]
    }
    if ($Uri -match '^https://api.github.com/repos/[^/]+/[^/]+/releases/latest$') {
        return $env:SSH_KEYFLOW_TEST_BASE + '/api/releases/latest'
    }
    throw "Non-fixture request: $Uri"
}
function Invoke-WebRequest {
    param([string]$Uri, [string]$OutFile, [switch]$UseBasicParsing)
    $PSBoundParameters['Uri'] = Convert-FixtureUri $Uri
    Microsoft.PowerShell.Utility\Invoke-WebRequest @PSBoundParameters
}
function Invoke-RestMethod {
    param([string]$Uri)
    Microsoft.PowerShell.Utility\Invoke-RestMethod -Uri (Convert-FixtureUri $Uri)
}
"""

    def assert_success(self, result):
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def assert_failure(self, result):
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)

    def snapshot(self):
        return self.tree(self.destination), self.profile.read_bytes()

    def assert_snapshot(self, previous):
        self.assertEqual(self.snapshot(), previous)

    def assert_installed(self, version='v1.0.0', system=None, architecture='amd64'):
        system = system or ('windows' if self.platform == 'windows' else 'linux')
        self.assertEqual(self.binary.read_bytes(), (system + '/' + architecture + '/' + version + '\n').encode())
        self.assertEqual(self.notices.read_bytes(), self.notice_content(version))
        if self.platform == 'windows':
            receipt = json.loads(self.receipt.read_text(encoding='utf-8-sig'))
            self.assertEqual(receipt['hash'], digest(self.binary.read_bytes()))
            self.assertEqual(receipt['noticesHash'], digest(self.notices.read_bytes()))
        else:
            self.assertEqual((self.receipt / 'hash').read_text().strip(), digest(self.binary.read_bytes()))
            self.assertEqual((self.receipt / 'notices-hash').read_text().strip(), digest(self.notices.read_bytes()))

    def make_legacy(self):
        self.notices.unlink()
        if self.platform == 'windows':
            receipt = json.loads(self.receipt.read_text(encoding='utf-8-sig'))
            del receipt['noticesHash']
            self.receipt.write_text(json.dumps(receipt), encoding='utf-8')
        else:
            (self.receipt / 'notices-hash').unlink()

    def test_installs_latest_stable_fixture_and_records_both_hashes(self):
        self.assert_success(self.run_installer())
        self.assert_installed()
        self.assertEqual(self.profile.read_bytes(), b'export EXISTING=keep\n')

    def test_update_and_uninstall_preserve_data_and_unowned_files(self):
        self.assert_success(self.run_installer(version='v1.0.0'))
        (self.destination / 'other-tool').write_bytes(b'not ours')
        self.assert_success(self.run_installer(version='v2.0.0'))
        self.assert_installed('v2.0.0')
        self.assert_success(self.run_installer(extra=[self.uninstall]))
        self.assertFalse(self.binary.exists())
        self.assertFalse(self.notices.exists())
        self.assertFalse(self.receipt.exists())
        self.assertEqual(self.tree(self.destination), {'other-tool': b'not ours'})

    def test_explicit_path_consent_is_idempotent_and_uninstall_removes_only_owned_stanza(self):
        self.assert_success(self.run_installer())
        self.assertEqual(self.profile.read_bytes(), b'export EXISTING=keep\n')
        self.assert_success(self.run_installer(extra=[self.consent], answer=''))
        installed_profile = self.profile.read_bytes()
        self.assertIn(b'SSH KeyFlow PATH', installed_profile)
        self.assert_success(self.run_installer(version='v2.0.0', extra=[self.consent], answer=''))
        self.assertEqual(self.profile.read_bytes(), installed_profile)
        with self.profile.open('ab') as profile:
            profile.write(b'after=keep\n')
        self.assert_success(self.run_installer(extra=[self.uninstall]))
        self.assertEqual(self.profile.read_bytes(), b'export EXISTING=keep\nafter=keep\n')

    def test_latest_rejects_draft_prerelease_and_unstable_tag(self):
        for changes in ({'prerelease': True}, {'draft': True}, {'tag_name': 'v1.0.0-rc1'}):
            with self.subTest(changes=changes):
                metadata = {'tag_name': 'v1.0.0', 'prerelease': False, 'draft': False}
                metadata.update(changes)
                self.latest.write_text(json.dumps(metadata))
                result = self.run_installer()
                self.assert_failure(result)
                self.assertIn('stable', result.stdout + result.stderr)
                self.assertFalse(self.binary.exists())
                self.assertFalse(self.notices.exists())

    def test_checksum_failure_preserves_previous_install_and_profile(self):
        self.assert_success(self.run_installer(version='v1.0.0', extra=[self.consent]))
        previous = self.snapshot()
        self.archive().write_bytes(b'corrupt archive')
        result = self.run_installer(version='v2.0.0', extra=[self.consent])
        self.assert_failure(result)
        self.assertIn('SHA256', result.stdout + result.stderr)
        self.assert_snapshot(previous)

    def test_missing_invalid_or_duplicate_manifest_entry_refuses_install(self):
        manifest = self.releases / 'v1.0.0' / 'SHA256SUMS'
        valid = manifest.read_text()
        invalid = valid.replace(digest(self.archive('v1.0.0').read_bytes()), 'z' * 64)
        for text in ('', '0' * 64 + '  wrong-asset\n', valid * 2, invalid, valid + invalid):
            with self.subTest(manifest=text):
                manifest.write_text(text)
                result = self.run_installer(version='v1.0.0')
                self.assert_failure(result)
                self.assertIn('manifest', result.stdout + result.stderr)
                self.assertFalse(self.binary.exists())
                self.assertFalse(self.notices.exists())

    def test_unowned_binary_or_notices_are_preserved(self):
        self.destination.mkdir()
        for path in (self.binary, self.notices):
            with self.subTest(path=path.name):
                path.write_bytes(b'user file')
                previous = self.snapshot()
                for extra in ([], [self.uninstall]):
                    self.assert_failure(self.run_installer(version='v1.0.0', extra=extra))
                    self.assert_snapshot(previous)
                path.unlink()

    def test_changed_owned_binary_or_notices_block_update_and_uninstall(self):
        for name in (self.executable, NOTICES):
            with self.subTest(name=name):
                self.assert_success(self.run_installer(version='v1.0.0', extra=[self.consent]))
                path = self.destination / name
                original = path.read_bytes()
                path.write_bytes(b'changed owned file')
                previous = self.snapshot()
                for extra in ([], [self.uninstall]):
                    self.assert_failure(self.run_installer(version='v2.0.0', extra=extra))
                    self.assert_snapshot(previous)
                path.write_bytes(original)
                self.assert_success(self.run_installer(extra=[self.uninstall]))

    def test_missing_owned_notices_block_update_and_uninstall(self):
        self.assert_success(self.run_installer())
        self.notices.unlink()
        previous = self.snapshot()
        for extra in ([], [self.uninstall]):
            self.assert_failure(self.run_installer(version='v2.0.0', extra=extra))
            self.assert_snapshot(previous)

    def test_legacy_receipt_upgrades_with_no_unowned_notice(self):
        self.assert_success(self.run_installer(version='v1.0.0', extra=[self.consent]))
        self.make_legacy()
        original_profile = self.profile.read_bytes()
        self.assert_success(self.run_installer(version='v2.0.0'))
        self.assert_installed('v2.0.0')
        self.assertEqual(self.profile.read_bytes(), original_profile)
        self.assert_success(self.run_installer(extra=[self.uninstall]))
        self.assertEqual(self.profile.read_bytes(), b'export EXISTING=keep\n')

    def test_legacy_receipt_cannot_claim_existing_notices(self):
        self.assert_success(self.run_installer())
        self.make_legacy()
        # Even byte-identical notices are unowned without a notice hash.
        for content in (self.notice_content('v1.0.0'), b'user notices'):
            with self.subTest(content=content):
                self.notices.write_bytes(content)
                previous = self.snapshot()
                for extra in ([], [self.uninstall]):
                    self.assert_failure(self.run_installer(version='v2.0.0', extra=extra))
                    self.assert_snapshot(previous)

    def test_legacy_receipt_can_uninstall_without_notices(self):
        self.assert_success(self.run_installer(extra=[self.consent]))
        self.make_legacy()
        self.assert_success(self.run_installer(extra=[self.uninstall]))
        self.assertFalse(self.binary.exists())
        self.assertFalse(self.receipt.exists())
        self.assertEqual(self.profile.read_bytes(), b'export EXISTING=keep\n')

    def test_changed_duplicated_or_truncated_path_stanza_is_preserved(self):
        self.assert_success(self.run_installer(extra=[self.consent]))
        installed = self.profile.read_bytes()
        for changed in (installed.replace(b'PATH end', b'PATH edited'), installed + installed, installed[:-1]):
            with self.subTest(profile=changed):
                self.profile.write_bytes(changed)
                previous = self.snapshot()
                self.assert_failure(self.run_installer(extra=[self.uninstall]))
                self.assert_snapshot(previous)

    def test_paths_with_quotes_and_unicode_preserve_profile_bytes(self):
        self.destination = self.home / "日本語's bin"
        self.profile = self.home / "日本語's profile"
        original = b'original without final newline'
        self.profile.write_bytes(original)
        self.assert_success(self.run_installer(extra=[self.consent], answer=''))
        self.assert_installed()
        self.assert_success(self.run_installer(extra=[self.uninstall]))
        self.assertEqual(self.profile.read_bytes(), original)

    def test_repository_defaults_when_environment_unset_or_empty(self):
        for repository in (None, ''):
            with self.subTest(repository=repository):
                log = self.home / 'requested urls'
                log.write_text('')
                self.assert_success(self.run_installer(repository=repository, public_endpoints=True))
                self.assert_installed()
                urls = log.read_text().splitlines()
                self.assertIn('https://api.github.com/repos/' + DEFAULT_REPOSITORY + '/releases/latest', urls)
                asset = self.archive('v1.0.0').name
                prefix = 'https://github.com/' + DEFAULT_REPOSITORY + '/releases/download/v1.0.0/'
                self.assertIn(prefix + asset, urls)
                self.assertIn(prefix + 'SHA256SUMS', urls)

    def test_repository_override_is_used_and_validated(self):
        option = '-Repository' if self.platform == 'windows' else '--repository'
        self.assert_success(self.run_installer(repository='invalid environment', extra=[option, 'other-owner/other-repo'],
                                               public_endpoints=True))
        urls = (self.home / 'requested urls').read_text().splitlines()
        self.assertTrue(any(url.startswith('https://github.com/other-owner/other-repo/releases/download/') for url in urls))
        previous = self.snapshot()
        for invalid in ('', 'missing-owner', '../owner/repo', 'owner/repo?x=1', 'owner/repo/extra', 'owner repo/name'):
            with self.subTest(repository=invalid):
                self.assert_failure(self.run_installer(extra=[option, invalid]))
                self.assert_snapshot(previous)
        self.assert_failure(self.run_installer(repository='invalid environment'))
        self.assert_snapshot(previous)

    def test_pipeline_defaults_install_without_changing_profile(self):
        self.destination = (self.home / 'local' / 'Programs' / 'ssh-keyflow' if self.platform == 'windows'
                            else self.home / '.local' / 'bin')
        if self.platform == 'shell':
            self.profile = self.home / '.profile'
            self.profile.write_bytes(b'export EXISTING=keep\n')
            source = self.releases / 'install.sh'
            source.write_bytes(
                b'read() { : > "$HOME/unexpected source read"; command read "$@"; }\n' +
                source.read_bytes() + b'\nprintf "%s" completed > "$HOME/pipeline sentinel"\n')
        self.assert_success(self.run_installer(pipeline=True, repository=None, public_endpoints=True))
        self.assert_installed()
        self.assertEqual(self.profile.read_bytes(), b'export EXISTING=keep\n')
        if self.platform == 'shell':
            self.assertEqual((self.home / 'pipeline sentinel').read_bytes(), b'completed')
            self.assertFalse((self.home / 'unexpected source read').exists())
        urls = (self.home / 'requested urls').read_text()
        self.assertIn('https://api.github.com/repos/' + DEFAULT_REPOSITORY + '/releases/latest', urls)

    def test_invalid_archive_with_valid_checksum_preserves_previous_install(self):
        self.assert_success(self.run_installer(version='v1.0.0', extra=[self.consent]))
        previous = self.snapshot()
        archive = self.archive()
        archive.write_bytes(b'this is not an archive')
        self.manifest(archive.parent)
        self.assert_failure(self.run_installer(version='v2.0.0'))
        self.assert_snapshot(previous)

    def test_bad_archive_members_and_types_preserve_previous_install(self):
        self.assert_success(self.run_installer(version='v1.0.0', extra=[self.consent]))
        previous = self.snapshot()
        binary = (self.executable, b'replacement', 'file')
        notices = (NOTICES, b'replacement notices', 'file')
        cases = [
            ('missing notices', [binary]), ('missing binary', [notices]),
            ('extra member', [binary, notices, ('extra', b'bad', 'file')]),
            ('duplicate binary', [binary, binary, notices]),
            ('duplicate notices', [binary, notices, notices]),
        ]
        absolute = str(self.home / 'escape') if self.platform == 'windows' else shell_path(self.home / 'escape')
        for name in ('../escape', '../../escape', absolute, 'nested/' + NOTICES,
                     './' + NOTICES, '..\\escape'):
            cases.append((name, [binary, (name, b'bad', 'file')]))
        kinds = ['symlink', 'directory', 'fifo', 'character', 'block']
        kinds += ['socket'] if self.platform == 'windows' else ['hardlink']
        for kind in kinds:
            cases.append(('binary ' + kind, [(self.executable, b'../escape', kind), notices]))
            cases.append(('notices ' + kind, [binary, (NOTICES, b'../escape', kind)]))
        for label, members in cases:
            with self.subTest(archive=label):
                archive = self.archive()
                self.write_archive(archive, members)
                self.manifest(archive.parent)  # Valid checksum: rejection must inspect the archive.
                self.assert_failure(self.run_installer(version='v2.0.0', extra=[self.consent]))
                self.assert_snapshot(previous)
                self.assertFalse((self.home / 'escape').exists())
                self.assertFalse((Path(self.temporary.name) / 'escape').exists())


@unittest.skipUnless(BASH and Path(BASH).is_file(), 'bash/Git Bash unavailable')
class ShellInstallers(InstallerTests, unittest.TestCase):
    platform = 'shell'

    def test_nonterminal_does_not_read_stdin_as_path_consent(self):
        result = self.run_installer(answer='y\n')
        self.assert_success(result)
        self.assert_installed()
        self.assertEqual(self.profile.read_bytes(), b'export EXISTING=keep\n')

    def test_profile_can_be_sourced_twice_without_path_duplicates(self):
        self.destination = self.home / "日本語's bin"
        self.assert_success(self.run_installer(extra=[self.consent]))
        result = subprocess.run(
            [BASH, '--noprofile', '--norc', '-c', '. "$1"; . "$1"; printf "%s" "$PATH"',
             'profile-test', shell_path(self.profile)], capture_output=True, text=True, encoding='utf-8', timeout=10)
        self.assert_success(result)
        self.assertEqual(result.stdout.split(':').count(shell_path(self.destination)), 1)

    def test_selects_four_unix_targets_and_rejects_unknown_os_or_cpu(self):
        for system in ('Linux', 'Darwin'):
            for cpu in ('x86_64', 'aarch64'):
                with self.subTest(system=system, cpu=cpu):
                    self.assert_success(self.run_installer(version='v2.0.0', system=system, cpu=cpu))
                    self.assert_installed('v2.0.0', system=system.lower(),
                                          architecture='amd64' if cpu == 'x86_64' else 'arm64')
        previous = self.snapshot()
        for system, cpu in (('Linux', 'riscv64'), ('FreeBSD', 'x86_64')):
            self.assert_failure(self.run_installer(system=system, cpu=cpu))
            self.assert_snapshot(previous)

    def test_dash_unicode_profile_preserves_original_bytes(self):
        result = subprocess.run([BASH, '--noprofile', '--norc', '-c', 'command -v dash'],
                                capture_output=True, text=True, timeout=10)
        if result.returncode:
            self.skipTest('dash unavailable')
        self.shell = 'dash'
        self.test_paths_with_quotes_and_unicode_preserve_profile_bytes()

    def fail_next_move(self, destination):
        native_mv = subprocess.check_output(
            [BASH, '--noprofile', '--norc', '-c', 'command -v mv'], text=True).strip()
        marker = self.home / 'injected failure'
        marker.unlink(missing_ok=True)
        wrapper = self.tools / 'mv'
        # Fail exactly once so recovery moves are permitted; assert the hook fired.
        wrapper.write_text(
            '#!/bin/sh\nfor target do :; done\nif [ "$target" = ' + shlex.quote(shell_path(destination)) +
            ' ] && [ ! -e ' + shlex.quote(shell_path(marker)) + ' ]; then\n    : > ' +
            shlex.quote(shell_path(marker)) + '\n    exit 1\nfi\nexec ' + shlex.quote(native_mv) + ' "$@"\n',
            encoding='utf-8', newline='\n')
        wrapper.chmod(0o755)
        return marker

    def test_notice_and_receipt_write_failures_roll_back_new_install_and_upgrade(self):
        for upgrade in (False, True):
            for name in (NOTICES, '.ssh-keyflow-install/hash', '.ssh-keyflow-install/notices-hash'):
                with self.subTest(upgrade=upgrade, destination=name):
                    if upgrade:
                        self.assert_success(self.run_installer(version='v1.0.0', extra=[self.consent]))
                    previous = self.snapshot()
                    marker = self.fail_next_move(self.destination / name)
                    result = self.run_installer(version='v2.0.0', extra=[self.consent])
                    self.assert_failure(result)
                    self.assertTrue(marker.exists(), result.stdout + result.stderr)
                    self.assert_snapshot(previous)
                    (self.tools / 'mv').unlink()
                    if upgrade:
                        self.assert_installed('v1.0.0')
                        self.assert_success(self.run_installer(extra=[self.uninstall]))
                        self.assertFalse(self.notices.exists())
                        self.assertFalse(self.receipt.exists())


@unittest.skipUnless(os.name == 'nt' and POWERSHELL, 'Native Windows installer')
class WindowsInstallers(InstallerTests, unittest.TestCase):
    platform = 'windows'

    def test_redirected_stdin_does_not_consent_to_path_changes(self):
        self.assert_success(self.run_installer(answer='y\n'))
        self.assert_installed()
        self.assertEqual(self.profile.read_bytes(), b'export EXISTING=keep\n')

    def test_arm64_and_powershell_51(self):
        self.assert_success(self.run_installer(version='v2.0.0', cpu='arm64', powershell='powershell'))
        self.assert_installed('v2.0.0', architecture='arm64')
        self.assert_success(self.run_installer(extra=[self.uninstall], powershell='powershell'))

    def test_powershell_51_pipeline_defaults(self):
        self.destination = self.home / 'local' / 'Programs' / 'ssh-keyflow'
        self.assert_success(self.run_installer(pipeline=True, repository='', public_endpoints=True, powershell='powershell'))
        self.assert_installed()
        self.assertEqual(self.profile.read_bytes(), b'export EXISTING=keep\n')

    def test_notice_write_failure_preserves_previous_version(self):
        self.assert_success(self.run_installer(version='v1.0.0', extra=[self.consent]))
        previous = self.snapshot()
        self.assert_failure(self.run_installer(version='v2.0.0', lock_path=self.notices, extra=[self.consent]))
        self.assert_snapshot(previous)
        self.assert_installed('v1.0.0')
        self.assert_success(self.run_installer(extra=[self.uninstall]))

    def test_receipt_write_failure_rolls_back_new_install_and_upgrade(self):
        for upgrade in (False, True):
            with self.subTest(upgrade=upgrade):
                if upgrade:
                    self.assert_success(self.run_installer(version='v1.0.0', extra=[self.consent]))
                self.destination.mkdir(exist_ok=True)
                blocked = self.destination / '.ssh-keyflow-install.json.tmp'
                blocked.mkdir()
                previous = self.snapshot()
                self.assert_failure(self.run_installer(version='v2.0.0', extra=[self.consent]))
                self.assert_snapshot(previous)
                # This pre-existing obstacle belongs to the fixture, not the installer.
                blocked.rmdir()
                if upgrade:
                    self.assert_installed('v1.0.0')
                    self.assert_success(self.run_installer(extra=[self.uninstall]))
                else:
                    self.assertFalse(self.binary.exists())
                    self.assertFalse(self.notices.exists())
                    self.assertFalse(self.receipt.exists())

    def test_powershell_51_preserves_utf16_profile(self):
        original = 'existing = 日本語\r\n'.encode('utf-16')
        self.profile.write_bytes(original)
        self.assert_success(self.run_installer(extra=[self.consent], answer='', powershell='powershell'))
        self.assertIn('SSH KeyFlow PATH', self.profile.read_bytes().decode('utf-16'))
        self.assert_success(self.run_installer(extra=[self.uninstall], powershell='powershell'))
        self.assertEqual(self.profile.read_bytes(), original)

    def test_profile_can_be_sourced_twice_without_path_duplicates(self):
        self.profile = self.home / 'test profile.ps1'
        self.profile.write_bytes(b'$existing = "keep"\n')
        self.assert_success(self.run_installer(extra=[self.consent]))
        command = ('$ErrorActionPreference = "Stop"; . ' + self.ps_quote(self.profile) +
                   '; . ' + self.ps_quote(self.profile) + '; @($env:PATH -split ";" | Where-Object { $_ -eq ' +
                   self.ps_quote(self.destination) + ' }).Count')
        encoded = base64.b64encode(command.encode('utf-16-le')).decode('ascii')
        result = subprocess.run(
            [POWERSHELL, '-NoProfile', '-NonInteractive', '-ExecutionPolicy', 'Bypass', '-EncodedCommand', encoded],
            input='', capture_output=True, text=True, encoding='utf-8', timeout=30)
        self.assert_success(result)
        self.assertEqual(result.stdout.strip(), '1', result.stdout + result.stderr)


if __name__ == '__main__':
    unittest.main()
