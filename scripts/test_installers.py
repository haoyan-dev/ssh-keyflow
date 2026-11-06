import base64
import functools
import hashlib
import http.server
import io
import json
import os
import shlex
from pathlib import Path
import shutil
import subprocess
import sys
import tarfile
import tempfile
import threading
import unittest
import zipfile


ROOT = Path(__file__).resolve().parents[1]
BASH = shutil.which('bash') if os.name != 'nt' else 'C:/Program Files/Git/bin/bash.exe'


def shell_path(path):
    value = str(path).replace('\\', '/')
    return '/' + value[0].lower() + value[2:] if len(value) > 1 and value[1] == ':' else value


class QuietHandler(http.server.SimpleHTTPRequestHandler):
    def log_message(self, format, *args):
        pass


class Installers(unittest.TestCase):
    def setUp(self):
        temporary_root = ROOT / '.tmp' / 'installer-tests'
        temporary_root.mkdir(parents=True, exist_ok=True)
        self.temporary = tempfile.TemporaryDirectory(prefix='keyflow installer ', dir=temporary_root)
        self.addCleanup(self.temporary.cleanup)
        self.home = Path(self.temporary.name) / 'home space'
        self.home.mkdir()
        self.destination = self.home / 'bin space'
        self.profile = self.home / 'test profile'
        self.profile.write_bytes(b'export EXISTING=keep\n')
        self.data = self.home / '.ssh' / 'ssh-keyflow'
        self.data.mkdir(parents=True)
        (self.data / 'state.json').write_bytes(b'preserve state')
        (self.data / 'private-key').write_bytes(b'preserve key')
        (self.data / 'config').write_bytes(b'preserve managed config')
        (self.home / '.ssh' / 'config').write_bytes(b'Host existing\n  HostName unchanged\n')
        self.ssh_snapshot = {path.relative_to(self.home): path.read_bytes() for path in (self.home / '.ssh').rglob('*') if path.is_file()}
        self.releases = Path(self.temporary.name) / 'releases'
        self.releases.mkdir()
        self.fixture('v1.0.0')
        self.fixture('v2.0.0')
        latest = self.releases / 'api' / 'releases'
        latest.mkdir(parents=True)
        (latest / 'latest').write_text(json.dumps({'tag_name': 'v1.0.0', 'prerelease': False, 'draft': False}))
        handler = functools.partial(QuietHandler, directory=str(self.releases))
        self.server = http.server.ThreadingHTTPServer(('127.0.0.1', 0), handler)
        self.addCleanup(self.server.server_close)
        self.addCleanup(self.server.shutdown)
        threading.Thread(target=self.server.serve_forever, daemon=True).start()
        self.base = 'http://127.0.0.1:' + str(self.server.server_port)

    def fixture(self, version):
        release = self.releases / version
        release.mkdir(exist_ok=True)
        checksums = []
        for system in ('linux', 'darwin', 'windows'):
            for architecture in ('amd64', 'arm64'):
                executable = 'ssh-keyflow.exe' if system == 'windows' else 'ssh-keyflow'
                content = (system + '/' + architecture + '/' + version + '\n').encode()
                name = 'ssh-keyflow_' + version + '_' + system + '_' + architecture
                archive = release / (name + ('.zip' if system == 'windows' else '.tar.gz'))
                if system == 'windows':
                    with zipfile.ZipFile(archive, 'w') as bundle:
                        bundle.writestr(executable, content)
                else:
                    with tarfile.open(archive, 'w:gz') as bundle:
                        entry = tarfile.TarInfo(executable)
                        entry.size = len(content)
                        entry.mode = 0o755
                        bundle.addfile(entry, io.BytesIO(content))
                checksums.append(hashlib.sha256(archive.read_bytes()).hexdigest() + '  ' + archive.name)
        (release / 'SHA256SUMS').write_text('\n'.join(checksums) + '\n')

    def run_installer(self, platform='shell', version=None, extra=(), answer='n\n', system='Linux', cpu='x86_64', repository='fixture/ssh-keyflow', powershell='pwsh'):
        environment = os.environ.copy()
        environment['SSH_KEYFLOW_RELEASE_URL'] = self.base
        environment['SSH_KEYFLOW_API_URL'] = self.base + '/api'
        environment['SSH_KEYFLOW_REPOSITORY'] = repository
        if platform == 'shell':
            tools = self.home / 'tools'
            tools.mkdir(exist_ok=True)
            (tools / 'uname').write_text('#!/bin/sh\ncase "$1" in\n-s) echo ' + system + ';;\n-m) echo ' + cpu + ';;\nesac\n')
            (tools / 'uname').chmod(0o755)
            environment['HOME'] = shell_path(self.home)
            environment['PATH'] = str(tools) + os.pathsep + environment['PATH']
            environment['SSH_KEYFLOW_TEST_TOOLS'] = shell_path(tools)
            installer = self.home / 'install.sh'
            installer.write_bytes((ROOT / 'install.sh').read_bytes().replace(b'\r\n', b'\n'))
            arguments = [BASH, '-c', 'PATH="$SSH_KEYFLOW_TEST_TOOLS:$PATH"; export PATH; exec "$0" "$@"', getattr(self, 'shell', 'sh')]
            arguments += [shell_path(installer), '--install-dir', shell_path(self.destination), '--profile', shell_path(self.profile)]
            if version:
                arguments += ['--version', version]
        else:
            environment['PROCESSOR_ARCHITECTURE'] = 'ARM64' if cpu == 'arm64' else 'AMD64'
            environment.pop('PROCESSOR_ARCHITEW6432', None)
            environment['USERPROFILE'] = str(self.home)
            environment['HOME'] = str(self.home)
            environment['LOCALAPPDATA'] = str(self.home / 'local')
            arguments = [powershell, '-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', str(ROOT / 'install.ps1'), '-InstallDir', str(self.destination), '-ProfilePath', str(self.profile)]
            if version:
                arguments += ['-Version', version]
        return subprocess.run(arguments + list(extra), input=answer, text=True, encoding='utf-8', errors='replace', capture_output=True, env=environment, timeout=30)

    def tearDown(self):
        current = {path.relative_to(self.home): path.read_bytes() for path in (self.home / '.ssh').rglob('*') if path.is_file()}
        self.assertEqual(current, self.ssh_snapshot)

    def assert_success(self, result):
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_shell_installs_latest_stable_fixture(self):
        self.assert_success(self.run_installer())
        self.assertEqual((self.destination / 'ssh-keyflow').read_bytes(), b'linux/amd64/v1.0.0\n')
        self.assertEqual(self.profile.read_bytes(), b'export EXISTING=keep\n')

    def test_shell_update_and_uninstall_preserve_data_and_unowned_files(self):
        self.assert_success(self.run_installer(version='v1.0.0'))
        (self.destination / 'other-tool').write_bytes(b'not ours')
        self.assert_success(self.run_installer(version='v2.0.0'))
        self.assertEqual((self.destination / 'ssh-keyflow').read_bytes(), b'linux/amd64/v2.0.0\n')
        self.assert_success(self.run_installer(extra=['--uninstall']))
        self.assertFalse((self.destination / 'ssh-keyflow').exists())
        self.assertEqual((self.destination / 'other-tool').read_bytes(), b'not ours')
        self.assertEqual((self.data / 'state.json').read_bytes(), b'preserve state')
        self.assertEqual((self.data / 'private-key').read_bytes(), b'preserve key')

    def test_shell_path_requires_confirmation_and_removes_only_exact_owned_stanza(self):
        self.assert_success(self.run_installer(answer='n\n'))
        self.assertEqual(self.profile.read_bytes(), b'export EXISTING=keep\n')
        self.assert_success(self.run_installer(answer='y\n'))
        installed_profile = self.profile.read_bytes()
        self.assertIn(b'SSH KeyFlow PATH', installed_profile)
        self.assert_success(self.run_installer(answer='y\n'))
        self.assertEqual(self.profile.read_bytes(), installed_profile)
        with self.profile.open('ab') as profile:
            profile.write(b'export AFTER=keep\n')
        self.assert_success(self.run_installer(extra=['--uninstall']))
        self.assertEqual(self.profile.read_bytes(), b'export EXISTING=keep\nexport AFTER=keep\n')

    @unittest.skipUnless(os.name == 'nt', 'Native Windows installer')
    def test_windows_installs_latest_stable_fixture(self):
        self.assert_success(self.run_installer(platform='windows'))
        self.assertEqual((self.destination / 'ssh-keyflow.exe').read_bytes(), b'windows/amd64/v1.0.0\n')
        self.assertEqual(self.profile.read_bytes(), b'export EXISTING=keep\n')

    @unittest.skipUnless(os.name == 'nt', 'Native Windows installer')
    def test_windows_update_uninstall_and_owned_path_preserve_data(self):
        self.assert_success(self.run_installer(platform='windows', version='v1.0.0', answer='y\n'))
        original = self.profile.read_bytes()
        self.assertIn(b'SSH KeyFlow PATH', original)
        self.assert_success(self.run_installer(platform='windows', version='v2.0.0', answer='y\n'))
        self.assertEqual((self.destination / 'ssh-keyflow.exe').read_bytes(), b'windows/amd64/v2.0.0\n')
        self.assertEqual(self.profile.read_bytes(), original)
        with self.profile.open('ab') as profile:
            profile.write(b'after=keep\n')
        self.assert_success(self.run_installer(platform='windows', extra=['-Uninstall']))
        self.assertFalse((self.destination / 'ssh-keyflow.exe').exists())
        self.assertEqual(self.profile.read_bytes(), b'export EXISTING=keep\nafter=keep\n')
        self.assertEqual((self.data / 'state.json').read_bytes(), b'preserve state')
        self.assertEqual((self.data / 'private-key').read_bytes(), b'preserve key')

    def test_latest_rejects_prerelease_metadata(self):
        latest = self.releases / 'api' / 'releases' / 'latest'
        latest.write_text(json.dumps({'tag_name': 'v1.0.0', 'prerelease': True, 'draft': False}))
        for platform in self.platforms():
            with self.subTest(platform=platform):
                result = self.run_installer(platform=platform)
                self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
                self.assertIn('stable', result.stdout + result.stderr)
                self.assertFalse((self.destination / self.executable(platform)).exists())

    def test_latest_rejects_prerelease_tag_even_when_marked_stable(self):
        self.fixture('v1.0.0-rc1')
        (self.releases / 'api' / 'releases' / 'latest').write_text(json.dumps({'tag_name': 'v1.0.0-rc1', 'prerelease': False, 'draft': False}))
        for platform in self.platforms():
            with self.subTest(platform=platform):
                result = self.run_installer(platform=platform)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn('stable', result.stdout + result.stderr)
                self.assertFalse((self.destination / self.executable(platform)).exists())

    def test_shell_refuses_hardlink_archive_with_valid_checksum(self):
        archive = self.releases / 'v1.0.0' / 'ssh-keyflow_v1.0.0_linux_amd64.tar.gz'
        with tarfile.open(archive, 'w:gz') as bundle:
            entry = tarfile.TarInfo('ssh-keyflow')
            entry.type = tarfile.LNKTYPE
            entry.linkname = 'SHA256SUMS'
            bundle.addfile(entry)
        manifest = hashlib.sha256(archive.read_bytes()).hexdigest() + '  ' + archive.name + '\n'
        (archive.parent / 'SHA256SUMS').write_text(manifest)
        result = self.run_installer(version='v1.0.0')
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse((self.destination / 'ssh-keyflow').exists())

    def test_shell_profile_can_be_sourced_twice_without_path_duplicates(self):
        self.assert_success(self.run_installer(answer='y\n'))
        result = subprocess.run([BASH, '--noprofile', '--norc', '-c', '. "$1"; . "$1"; printf "%s" "$PATH"', 'profile-test', shell_path(self.profile)], capture_output=True, text=True, timeout=10)
        self.assert_success(result)
        self.assertEqual(result.stdout.split(':').count(shell_path(self.destination)), 1)

    def platforms(self):
        return ('shell', 'windows') if os.name == 'nt' else ('shell',)

    def executable(self, platform):
        return 'ssh-keyflow.exe' if platform == 'windows' else 'ssh-keyflow'

    def test_checksum_failure_preserves_previous_install_and_profile(self):
        for platform in self.platforms():
            with self.subTest(platform=platform):
                self.assert_success(self.run_installer(platform=platform, version='v1.0.0'))
                binary = self.destination / self.executable(platform)
                previous = binary.read_bytes()
                for archive in (self.releases / 'v2.0.0').glob('*.zip' if platform == 'windows' else '*.tar.gz'):
                    archive.write_bytes(b'corrupt archive')
                result = self.run_installer(platform=platform, version='v2.0.0', answer='y\n')
                self.assertNotEqual(result.returncode, 0)
                self.assertIn('SHA256', result.stdout + result.stderr)
                self.assertEqual(binary.read_bytes(), previous)
                self.assertEqual(self.profile.read_bytes(), b'export EXISTING=keep\n')
                self.assert_success(self.run_installer(platform=platform, extra=['-Uninstall'] if platform == 'windows' else ['--uninstall']))

    def test_missing_or_duplicate_manifest_entry_refuses_install(self):
        for manifest_text in ('', '0' * 64 + '  wrong-asset\n'):
            (self.releases / 'v1.0.0' / 'SHA256SUMS').write_text(manifest_text)
            for platform in self.platforms():
                with self.subTest(platform=platform, manifest=manifest_text):
                    result = self.run_installer(platform=platform, version='v1.0.0')
                    self.assertNotEqual(result.returncode, 0)
                    self.assertIn('manifest', result.stdout + result.stderr)
                    self.assertFalse((self.destination / self.executable(platform)).exists())
        self.fixture('v1.0.0')
        manifest = self.releases / 'v1.0.0' / 'SHA256SUMS'
        manifest.write_text(manifest.read_text() * 2)
        for platform in self.platforms():
            result = self.run_installer(platform=platform, version='v1.0.0')
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('manifest', result.stdout + result.stderr)

    def test_unowned_or_changed_binary_is_preserved(self):
        self.destination.mkdir()
        for platform in self.platforms():
            with self.subTest(platform=platform):
                binary = self.destination / self.executable(platform)
                binary.write_bytes(b'user binary')
                for extra in ([], ['-Uninstall'] if platform == 'windows' else ['--uninstall']):
                    result = self.run_installer(platform=platform, version='v1.0.0', extra=extra)
                    self.assertNotEqual(result.returncode, 0)
                    self.assertIn('unowned or changed', result.stdout + result.stderr)
                    self.assertEqual(binary.read_bytes(), b'user binary')
                binary.unlink()
                self.assert_success(self.run_installer(platform=platform, version='v1.0.0'))
                binary.write_bytes(b'changed owned binary')
                for extra in ([], ['-Uninstall'] if platform == 'windows' else ['--uninstall']):
                    result = self.run_installer(platform=platform, version='v2.0.0', extra=extra)
                    self.assertNotEqual(result.returncode, 0)
                    self.assertIn('unowned or changed', result.stdout + result.stderr)
                    self.assertEqual(binary.read_bytes(), b'changed owned binary')

    def test_changed_owned_path_stanza_is_preserved(self):
        for platform in self.platforms():
            with self.subTest(platform=platform):
                self.profile.write_bytes(b'keep\n')
                self.assert_success(self.run_installer(platform=platform, answer='y\n'))
                changed = self.profile.read_bytes().replace(b'PATH end', b'PATH edited')
                self.profile.write_bytes(changed)
                result = self.run_installer(platform=platform, extra=['-Uninstall'] if platform == 'windows' else ['--uninstall'])
                self.assertNotEqual(result.returncode, 0)
                self.assertIn('changed or duplicated', result.stdout + result.stderr)
                self.assertEqual(self.profile.read_bytes(), changed)
                self.assertTrue((self.destination / self.executable(platform)).exists())

    def test_repository_is_required_even_with_fixture_endpoints(self):
        for platform in self.platforms():
            result = self.run_installer(platform=platform, repository='')
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('OWNER/REPO', result.stdout + result.stderr)
            self.assertFalse((self.destination / self.executable(platform)).exists())

    def test_shell_selects_four_unix_targets_and_rejects_unknown_cpu(self):
        for system in ('Linux', 'Darwin'):
            for cpu in ('x86_64', 'aarch64'):
                with self.subTest(system=system, cpu=cpu):
                    self.assert_success(self.run_installer(version='v2.0.0', system=system, cpu=cpu))
                    expected = (('linux' if system == 'Linux' else 'darwin') + '/' + ('amd64' if cpu == 'x86_64' else 'arm64') + '/v2.0.0\n').encode()
                    self.assertEqual((self.destination / 'ssh-keyflow').read_bytes(), expected)
        result = self.run_installer(system='Linux', cpu='riscv64')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('Unsupported CPU', result.stderr)

    @unittest.skipUnless(os.name == 'nt', 'Native Windows installer')
    def test_windows_arm64_and_powershell_51(self):
        self.assert_success(self.run_installer(platform='windows', version='v2.0.0', cpu='arm64', powershell='powershell'))
        self.assertEqual((self.destination / 'ssh-keyflow.exe').read_bytes(), b'windows/arm64/v2.0.0\n')
        self.assert_success(self.run_installer(platform='windows', extra=['-Uninstall'], powershell='powershell'))

    def test_packaging_produces_six_verified_archives(self):
        binaries = self.home / 'built'
        for system in ('linux', 'darwin', 'windows'):
            for architecture in ('amd64', 'arm64'):
                target = binaries / (system + '_' + architecture)
                target.mkdir(parents=True)
                (target / ('ssh-keyflow.exe' if system == 'windows' else 'ssh-keyflow')).write_bytes(b'fixture binary')
        output = self.home / 'dist'
        result = subprocess.run([sys.executable, str(ROOT / 'scripts' / 'package_release.py'), '--version', 'v3.0.0', '--binaries', str(binaries), '--output', str(output)], capture_output=True, text=True, timeout=30)
        self.assert_success(result)
        archives = list(output.glob('*.tar.gz')) + list(output.glob('*.zip'))
        self.assertEqual(len(archives), 6)
        manifest = (output / 'SHA256SUMS').read_text()
        for archive in archives:
            self.assertIn(hashlib.sha256(archive.read_bytes()).hexdigest() + '  ' + archive.name + '\n', manifest)
            if archive.suffix == '.zip':
                with zipfile.ZipFile(archive) as bundle:
                    self.assertEqual(bundle.namelist(), ['ssh-keyflow.exe'])
                    self.assertEqual(bundle.read('ssh-keyflow.exe'), b'fixture binary')
            else:
                with tarfile.open(archive) as bundle:
                    self.assertEqual(bundle.getnames(), ['ssh-keyflow'])
                    self.assertEqual(bundle.extractfile('ssh-keyflow').read(), b'fixture binary')

    def test_shell_receipt_write_failure_rolls_back_new_install(self):
        tools = self.home / 'tools'
        tools.mkdir()
        wrapper = tools / 'mv'
        native_mv = subprocess.check_output([BASH, '-c', 'command -v mv'], text=True).strip()
        wrapper.write_text('#!/bin/sh\ncase "$*" in *"/.ssh-keyflow-install/hash") exit 1;; esac\nexec ' + shlex.quote(native_mv) + ' "$@"\n')
        wrapper.chmod(0o755)
        result = self.run_installer(version='v1.0.0')
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse((self.destination / 'ssh-keyflow').exists())
        self.assertFalse((self.destination / '.ssh-keyflow-install').exists(), result.stdout + result.stderr)

    def test_truncated_path_stanza_is_not_removed(self):
        for platform in self.platforms():
            with self.subTest(platform=platform):
                self.profile.write_bytes(b'keep without trailing newline')
                self.assert_success(self.run_installer(platform=platform, answer='y\n'))
                changed = self.profile.read_bytes()[:-1]
                self.profile.write_bytes(changed)
                result = self.run_installer(platform=platform, extra=['-Uninstall'] if platform == 'windows' else ['--uninstall'])
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(self.profile.read_bytes(), changed)

    def test_paths_with_quotes_and_unicode_preserve_profile_bytes(self):
        for platform in self.platforms():
            with self.subTest(platform=platform):
                self.destination = self.home / "日本語's bin"
                self.profile = self.home / (platform + " 日本語's profile")
                original = b'original without final newline'
                self.profile.write_bytes(original)
                self.assert_success(self.run_installer(platform=platform, answer='y\n'))
                if platform == 'shell':
                    result = subprocess.run([BASH, '--noprofile', '--norc', '-c', '. "$1"; . "$1"; printf "%s" "$PATH"', 'profile-test', shell_path(self.profile)], capture_output=True, text=True, encoding='utf-8', timeout=10)
                    self.assert_success(result)
                    self.assertEqual(result.stdout.split(':').count(shell_path(self.destination)), 1)
                result = self.run_installer(platform=platform, extra=['-Uninstall'] if platform == 'windows' else ['--uninstall'])
                self.assert_success(result)
                self.assertEqual(self.profile.read_bytes(), original, result.stdout + result.stderr)

    def test_dash_unicode_profile_preserves_original_bytes(self):
        if not shutil.which('dash') and not Path('C:/Program Files/Git/usr/bin/dash.exe').exists():
            self.skipTest('dash unavailable')
        self.shell = 'dash'
        self.test_paths_with_quotes_and_unicode_preserve_profile_bytes()

    def test_shell_receipt_write_failure_preserves_previous_version(self):
        self.assert_success(self.run_installer(version='v1.0.0'))
        wrapper = self.home / 'tools' / 'mv'
        native_mv = subprocess.check_output([BASH, '-c', 'command -v mv'], text=True).strip()
        wrapper.write_text('#!/bin/sh\ncase "$*" in *"/.ssh-keyflow-install/hash") exit 1;; esac\nexec ' + shlex.quote(native_mv) + ' "$@"\n')
        wrapper.chmod(0o755)
        result = self.run_installer(version='v2.0.0')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual((self.destination / 'ssh-keyflow').read_bytes(), b'linux/amd64/v1.0.0\n')
        self.assert_success(self.run_installer(extra=['--uninstall']))

    @unittest.skipUnless(os.name == 'nt', 'Native Windows installer')
    def test_windows_receipt_write_failure_preserves_previous_version(self):
        self.assert_success(self.run_installer(platform='windows', version='v1.0.0'))
        (self.destination / '.ssh-keyflow-install.json.tmp').mkdir()
        result = self.run_installer(platform='windows', version='v2.0.0')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual((self.destination / 'ssh-keyflow.exe').read_bytes(), b'windows/amd64/v1.0.0\n')
        self.assert_success(self.run_installer(platform='windows', extra=['-Uninstall']))

    @unittest.skipUnless(os.name == 'nt', 'Native Windows installer')
    def test_powershell_51_preserves_utf16_profile(self):
        original = 'existing = 日本語\r\n'.encode('utf-16')
        self.profile.write_bytes(original)
        self.assert_success(self.run_installer(platform='windows', answer='y\n', powershell='powershell'))
        self.assertIn('SSH KeyFlow PATH', self.profile.read_bytes().decode('utf-16'))
        self.assert_success(self.run_installer(platform='windows', extra=['-Uninstall'], powershell='powershell'))
        self.assertEqual(self.profile.read_bytes(), original)

    @unittest.skipUnless(os.name == 'nt', 'Native Windows installer')
    def test_windows_profile_can_be_sourced_twice_without_path_duplicates(self):
        self.profile = self.home / 'test profile.ps1'
        self.profile.write_bytes(b'$existing = "keep"\n')
        self.assert_success(self.run_installer(platform='windows', answer='y\n'))
        environment = os.environ.copy()
        environment['SSH_KEYFLOW_TEST_PROFILE'] = str(self.profile)
        environment['SSH_KEYFLOW_TEST_DESTINATION'] = str(self.destination)
        command = '$ErrorActionPreference = "Stop"; . $env:SSH_KEYFLOW_TEST_PROFILE; . $env:SSH_KEYFLOW_TEST_PROFILE; @($env:PATH -split ";" | Where-Object { $_ -eq $env:SSH_KEYFLOW_TEST_DESTINATION }).Count'
        encoded_command = base64.b64encode(command.encode('utf-16-le')).decode('ascii')
        result = subprocess.run(['pwsh', '-NoProfile', '-NonInteractive', '-ExecutionPolicy', 'Bypass', '-EncodedCommand', encoded_command], input='', capture_output=True, text=True, encoding='utf-8', env=environment, timeout=30)
        self.assert_success(result)
        self.assertEqual(result.stdout.strip(), '1', result.stdout + result.stderr)


if __name__ == '__main__':
    unittest.main()
