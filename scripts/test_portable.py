import hashlib
import os
import platform
import shutil
from pathlib import Path
import subprocess
import sys
import tarfile
import tempfile
import unittest
import zipfile

ROOT = Path(__file__).resolve().parents[1]
TARGETS = ('linux_amd64', 'linux_arm64', 'darwin_amd64', 'darwin_arm64', 'windows_amd64', 'windows_arm64')


class PortableReleaseTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix='keyflow-portable-')
        self.addCleanup(self.temporary.cleanup)
        self.base = Path(self.temporary.name)
        self.binaries = self.base / 'binaries'
        for target in TARGETS:
            binary = self.binaries / target / ('ssh-keyflow.exe' if target.startswith('windows') else 'ssh-keyflow')
            binary.parent.mkdir(parents=True)
            binary.write_bytes(b'portable fixture\n')
        self.output = self.base / 'dist'

    def package(self, *extra):
        return subprocess.run([sys.executable, str(ROOT / 'scripts/package_release.py'), '--version', 'v0.2.0-test', '--binaries', str(self.binaries), '--output', str(self.output), *extra], capture_output=True, text=True)

    def test_six_archives_contain_binary_and_notices_with_valid_checksums(self):
        result = self.package()
        self.assertEqual(result.returncode, 0, result.stderr)
        names = {f'ssh-keyflow_v0.2.0-test_{target}' + ('.zip' if target.startswith('windows') else '.tar.gz') for target in TARGETS}
        self.assertEqual({p.name for p in self.output.iterdir()}, names | {'SHA256SUMS'})
        manifest = (self.output / 'SHA256SUMS').read_bytes()
        self.assertNotIn(b'\r', manifest)
        lines = manifest.decode('ascii').splitlines()
        self.assertEqual(len(lines), 6)
        self.assertEqual({line[66:] for line in lines}, names)
        for line in lines:
            digest, name = line.split('  ')
            self.assertEqual(hashlib.sha256((self.output / name).read_bytes()).hexdigest(), digest)
        notices = (ROOT / 'THIRD_PARTY_NOTICES.txt').read_bytes()
        for archive in self.output.iterdir():
            if archive.suffix == '.zip':
                with zipfile.ZipFile(archive) as bundle:
                    self.assertEqual(set(bundle.namelist()), {'ssh-keyflow.exe', 'THIRD_PARTY_NOTICES.txt'})
                    self.assertEqual(bundle.read('ssh-keyflow.exe'), b'portable fixture\n')
                    self.assertEqual(bundle.read('THIRD_PARTY_NOTICES.txt'), notices)
            elif archive.name.endswith('.tar.gz'):
                with tarfile.open(archive) as bundle:
                    self.assertEqual(set(bundle.getnames()), {'ssh-keyflow', 'THIRD_PARTY_NOTICES.txt'})
                    self.assertEqual(bundle.extractfile('ssh-keyflow').read(), b'portable fixture\n')
                    self.assertEqual(bundle.extractfile('THIRD_PARTY_NOTICES.txt').read(), notices)
                    self.assertEqual(bundle.getmember('ssh-keyflow').mode, 0o755)
                    self.assertEqual(bundle.getmember('THIRD_PARTY_NOTICES.txt').mode, 0o644)
                    self.assertTrue(all(member.isfile() and member.uid == member.gid == 0 for member in bundle.getmembers()))

    def test_missing_binary_publishes_nothing_and_preserves_sources(self):
        missing = self.binaries / 'windows_arm64' / 'ssh-keyflow.exe'
        missing.unlink()
        before = {p: p.read_bytes() for p in self.binaries.rglob('*') if p.is_file()}
        result = self.package()
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.output.exists())
        self.assertEqual({p: p.read_bytes() for p in before}, before)

    def test_invalid_version_and_nonempty_output_preserve_existing_assets(self):
        self.output.mkdir()
        existing = self.output / 'keep.txt'
        existing.write_bytes(b'existing release')
        for extra in ((), ('--version', '../bad')):
            with self.subTest(extra=extra):
                result = self.package(*extra)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(existing.read_bytes(), b'existing release')
                self.assertEqual(list(self.output.iterdir()), [existing])

    def test_missing_notices_fail_without_publishing_assets(self):
        isolated = self.base / 'source'
        (isolated / 'scripts').mkdir(parents=True)
        script = isolated / 'scripts/package_release.py'
        shutil.copyfile(ROOT / 'scripts/package_release.py', script)
        result = subprocess.run([sys.executable, str(script), '--version', 'v0.2.0', '--binaries', str(self.binaries), '--output', str(self.output)], capture_output=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.output.exists())

    def test_native_portable_version_and_replacement_preserve_user_data(self):
        system = {'Windows': 'windows', 'Linux': 'linux', 'Darwin': 'darwin'}[platform.system()]
        architecture = {'amd64': 'amd64', 'x86_64': 'amd64', 'arm64': 'arm64', 'aarch64': 'arm64'}[platform.machine().lower()]
        executable = 'ssh-keyflow.exe' if system == 'windows' else 'ssh-keyflow'
        native = self.binaries / (system + '_' + architecture) / executable
        native.unlink()
        environment = os.environ.copy()
        environment['CGO_ENABLED'] = '0'
        subprocess.run(['go', 'build', '-buildvcs=false', '-trimpath', '-ldflags', '-X main.version=v0.2.0-test', '-o', str(native), './cmd/ssh-keyflow'], cwd=ROOT, env=environment, check=True)
        result = self.package()
        self.assertEqual(result.returncode, 0, result.stderr)
        install = self.base / 'portable'
        install.mkdir()
        archive = self.output / ('ssh-keyflow_v0.2.0-test_' + system + '_' + architecture + ('.zip' if system == 'windows' else '.tar.gz'))
        if system == 'windows':
            with zipfile.ZipFile(archive) as bundle:
                bundle.extractall(install)
        else:
            with tarfile.open(archive) as bundle:
                bundle.extractall(install, filter='data')
        home = self.base / 'home'
        files = {
            '.ssh/ssh-keyflow/state.json': b'{"version":1,"keys":[],"hosts":[],"registrations":[]}',
            '.ssh/ssh-keyflow/keys/retain': b'key fixture, not a real secret',
            '.ssh/config': b'# existing SSH config\n',
            '.ssh/ssh-keyflow/config': b'# existing managed config\n',
            'profile': b'# existing PATH profile\n',
            '.ssh-keyflow-install.json': b'legacy ownership receipt',
        }
        for name, content in files.items():
            file = home / name
            file.parent.mkdir(parents=True, exist_ok=True)
            file.write_bytes(content)
        environment.update(HOME=str(home), USERPROFILE=str(home))
        binary = install / executable
        for arguments in (['--version'], ['--help'], ['keys', 'list', '--json']):
            run = subprocess.run([str(binary), *arguments], env=environment, capture_output=True, text=True)
            self.assertEqual(run.returncode, 0, run.stderr)
            if arguments == ['--version']:
                self.assertIn('v0.2.0-test', run.stdout)
        moved = self.base / 'moved'
        moved.mkdir()
        binary = Path(shutil.move(binary, moved / executable))
        replacement = moved / (executable + '.new')
        shutil.copyfile(native, replacement)
        replacement.chmod(0o755)
        os.replace(replacement, binary)
        run = subprocess.run([str(binary), 'hosts', 'list', '--json'], env=environment, capture_output=True)
        self.assertEqual(run.returncode, 0, run.stderr)
        binary.unlink()
        self.assertEqual({str(p.relative_to(home)).replace(os.sep, '/'): p.read_bytes() for p in home.rglob('*') if p.is_file()}, files)


if __name__ == '__main__':
    unittest.main()
