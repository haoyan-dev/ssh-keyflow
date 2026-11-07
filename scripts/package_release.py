import argparse
import hashlib
import os
from pathlib import Path
import re
import shutil
import subprocess
import tarfile
import tempfile
import zipfile


ROOT = Path(__file__).resolve().parents[1]
TARGETS = tuple((system, architecture) for system in ('linux', 'darwin', 'windows') for architecture in ('amd64', 'arm64'))


def main():
    parser = argparse.ArgumentParser(description='Build and package six SSH KeyFlow release targets.')
    parser.add_argument('--version', required=True)
    parser.add_argument('--output', type=Path, default=ROOT / 'dist')
    parser.add_argument('--binaries', type=Path, help='Package existing OS_ARCH binaries instead of building Go.')
    arguments = parser.parse_args()
    if not re.fullmatch(r'v[0-9]+\.[0-9]+\.[0-9]+(?:-[A-Za-z0-9.-]+)?', arguments.version):
        parser.error('Use a release tag such as v1.2.3.')
    notices = ROOT / 'THIRD_PARTY_NOTICES.txt'
    if not notices.is_file() or notices.is_symlink():
        parser.error('Missing regular THIRD_PARTY_NOTICES.txt.')
    output = arguments.output.resolve()
    if output.exists() and any(output.iterdir()):
        parser.error('Output directory must be empty; existing files are preserved.')
    output.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='keyflow-package-', dir=output.parent) as temporary:
        stage = Path(temporary)
        assets = stage / 'assets'
        assets.mkdir()
        for system, architecture in TARGETS:
            executable = 'ssh-keyflow.exe' if system == 'windows' else 'ssh-keyflow'
            target = system + '_' + architecture
            if arguments.binaries:
                binary = arguments.binaries / target / executable
            else:
                binary = stage / target / executable
                binary.parent.mkdir()
                environment = os.environ.copy()
                environment.update(GOOS=system, GOARCH=architecture, CGO_ENABLED='0')
                subprocess.run(['go', 'build', '-buildvcs=false', '-trimpath', '-ldflags', '-s -w -X main.version=' + arguments.version, '-o', str(binary), './cmd/ssh-keyflow'], cwd=ROOT, env=environment, check=True)
            if not binary.is_file() or binary.is_symlink():
                raise ValueError('Expected a regular binary: ' + str(binary))
            name = 'ssh-keyflow_' + arguments.version + '_' + target
            if system == 'windows':
                with zipfile.ZipFile(assets / (name + '.zip'), 'w', compression=zipfile.ZIP_DEFLATED) as bundle:
                    bundle.write(binary, executable)
                    bundle.write(notices, 'THIRD_PARTY_NOTICES.txt')
            else:
                with tarfile.open(assets / (name + '.tar.gz'), 'w:gz') as bundle:
                    entry = bundle.gettarinfo(str(binary), arcname=executable)
                    entry.mode = 0o755
                    entry.uid = entry.gid = 0
                    entry.uname = entry.gname = ''
                    with binary.open('rb') as source:
                        bundle.addfile(entry, source)
                    notice_entry = bundle.gettarinfo(str(notices), arcname='THIRD_PARTY_NOTICES.txt')
                    notice_entry.mode = 0o644
                    notice_entry.uid = notice_entry.gid = 0
                    notice_entry.uname = notice_entry.gname = ''
                    with notices.open('rb') as source:
                        bundle.addfile(notice_entry, source)
        manifest = ''.join(hashlib.sha256(asset.read_bytes()).hexdigest() + '  ' + asset.name + '\n' for asset in sorted(assets.iterdir()))
        (assets / 'SHA256SUMS').write_text(manifest, encoding='ascii', newline='\n')
        output.mkdir(exist_ok=True)
        for asset in assets.iterdir():
            shutil.move(str(asset), output / asset.name)
    print('Packaged six Portable targets and SHA256SUMS in ' + str(output))


if __name__ == '__main__':
    main()
