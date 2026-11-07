import base64
import os
from pathlib import Path
import shlex
import shutil
import stat
import subprocess
import unittest

try:
    from . import test_installers as installers
except ImportError:
    import test_installers as installers


class ProfileFixture(unittest.TestCase):
    """Compose the existing fixture without collecting its regression tests again."""

    fixture_type = None

    def setUp(self):
        self.case = self.fixture_type()
        self.addCleanup(self.case.doCleanups)
        self.case.setUp()

    def tearDown(self):
        self.case.tearDown()
        self.assertFalse(list(self.case.home.rglob('.ssh-keyflow-profile.*')))

    def snapshot(self):
        case = self.case
        if case.profile.is_dir():
            profile = ('directory', case.tree(case.profile))
        elif case.profile.exists():
            profile = ('file', case.profile.read_bytes(), stat.S_IMODE(case.profile.stat().st_mode))
        else:
            profile = None
        modes = {}
        if os.name != 'nt' and case.destination.exists():
            modes = {str(path.relative_to(case.destination)): stat.S_IMODE(path.stat().st_mode)
                     for path in case.destination.rglob('*')}
        return case.tree(case.destination), modes, profile

    def assert_snapshot(self, previous):
        self.assertEqual(self.snapshot(), previous)

    def assert_uninstalled(self):
        case = self.case
        self.assertFalse(case.binary.exists())
        self.assertFalse(case.notices.exists())
        self.assertFalse(case.receipt.exists())

    def powershell(self, command, executable=None):
        case = self.case
        environment = os.environ.copy()
        environment.update(HOME=str(case.home), USERPROFILE=str(case.home),
                           LOCALAPPDATA=str(case.home / 'local'))
        encoded = base64.b64encode(('$ErrorActionPreference = "Stop"; ' + command).encode('utf-16-le')).decode('ascii')
        return subprocess.run(
            [executable or installers.POWERSHELL, '-NoProfile', '-NonInteractive',
             '-ExecutionPolicy', 'Bypass', '-EncodedCommand', encoded],
            input='', capture_output=True, text=True, encoding='utf-8', errors='replace',
            env=environment, cwd=case.home, timeout=45, creationflags=subprocess.CREATE_NO_WINDOW)

    def windows_prelude(self, command):
        # Modify this fixture instance only; the shared helper and source stay untouched.
        self.case.windows_redirects = lambda: installers.InstallerTests.windows_redirects() + command

    def fail_shell_command(self, command, target):
        case = self.case
        native = subprocess.check_output(
            [installers.BASH, '--noprofile', '--norc', '-c', 'command -v ' + command],
            text=True, encoding='utf-8').strip()
        marker = case.home / 'profile failure checkpoint'
        wrapper = case.tools / command
        wrapper.write_text(
            '#!/bin/sh\nfor argument do\n    if [ "$argument" = ' +
            shlex.quote(installers.shell_path(target)) + ' ] && [ ! -e ' +
            shlex.quote(installers.shell_path(marker)) + ' ]; then\n        cat ' +
            shlex.quote(installers.shell_path(case.profile)) + ' > ' +
            shlex.quote(installers.shell_path(marker)) +
            '\n        exit 1\n    fi\ndone\nexec ' + shlex.quote(native) + ' "$@"\n',
            encoding='utf-8', newline='\n')
        wrapper.chmod(0o755)
        self.addCleanup(wrapper.unlink, missing_ok=True)
        return marker

    def fail_path_receipt(self):
        case = self.case
        if case.platform == 'shell':
            return self.fail_shell_command('cp', case.receipt / 'stanza'), None
        marker = case.home / 'profile failure checkpoint'
        blocked = Path(str(case.receipt) + '.tmp')
        # The first receipt precedes PATH changes. The second sees the new profile.
        self.windows_prelude(r"""
function ConvertTo-Json {
    [CmdletBinding()]
    param([Parameter(ValueFromPipeline=$true)]$InputObject)
    process {
        if ($InputObject.profile) {
            [IO.File]::Copy(""" + case.ps_quote(case.profile) + ', ' + case.ps_quote(marker) + r""")
            [IO.Directory]::CreateDirectory(""" + case.ps_quote(blocked) + r""") | Out-Null
        }
        Microsoft.PowerShell.Utility\ConvertTo-Json -InputObject $InputObject
    }
}
""")
        return marker, blocked

    def check_path_receipt_failure(self, upgrade=False, missing_profile=False):
        case = self.case
        if upgrade:
            case.assert_success(case.run_installer(version='v1.0.0'))
        if missing_profile:
            case.profile.unlink()
        previous = self.snapshot()
        marker, blocked = self.fail_path_receipt()
        result = case.run_installer(version='v2.0.0', extra=[case.consent], public_endpoints=True)
        case.assert_failure(result)
        self.assertTrue(marker.exists(), result.stdout + result.stderr)
        self.assertIn(b'SSH KeyFlow PATH', marker.read_bytes())
        if blocked is not None:
            self.assertTrue(blocked.is_dir(), result.stdout + result.stderr)
            blocked.rmdir()
        self.assert_snapshot(previous)
        if upgrade:
            case.assert_installed('v1.0.0')
            case.assert_success(case.run_installer(extra=[case.uninstall]))
            self.assert_uninstalled()

    def check_bad_profile_path(self, upgrade=False, read_only=False):
        case = self.case
        if upgrade:
            case.assert_success(case.run_installer(version='v1.0.0'))
        if read_only:
            original_mode = stat.S_IMODE(case.profile.stat().st_mode)
            case.profile.chmod(0o444)
            self.addCleanup(case.profile.chmod, original_mode)
            try:
                with case.profile.open('ab'):
                    pass
            except PermissionError:
                pass
            else:
                self.skipTest('This account/filesystem can write chmod(0444) files')
        else:
            case.profile.unlink()
            case.profile.mkdir()
        previous = self.snapshot()
        result = case.run_installer(version='v2.0.0', extra=[case.consent])
        case.assert_failure(result)
        self.assert_snapshot(previous)
        if upgrade:
            case.assert_installed('v1.0.0')
            case.assert_success(case.run_installer(extra=[case.uninstall]))
            self.assert_uninstalled()


class ProfileContract:
    def test_directory_profile_rolls_back_new_install(self):
        self.check_bad_profile_path()

    def test_directory_profile_rolls_back_upgrade(self):
        self.check_bad_profile_path(upgrade=True)

    def test_read_only_profile_rolls_back_new_install(self):
        self.check_bad_profile_path(read_only=True)

    def test_read_only_profile_rolls_back_upgrade(self):
        self.check_bad_profile_path(upgrade=True, read_only=True)

    def test_path_receipt_failure_restores_new_install_and_existing_profile(self):
        self.check_path_receipt_failure()

    def test_path_receipt_failure_removes_new_install_and_new_profile(self):
        self.check_path_receipt_failure(missing_profile=True)

    def test_path_receipt_failure_restores_upgrade_and_existing_profile(self):
        self.check_path_receipt_failure(upgrade=True)

    def test_path_receipt_failure_restores_upgrade_and_removes_new_profile(self):
        self.check_path_receipt_failure(upgrade=True, missing_profile=True)

    def check_profile_round_trip(self, original):
        case = self.case
        case.profile.write_bytes(original)
        if os.name != 'nt':
            case.profile.chmod(0o640)
        mode = stat.S_IMODE(case.profile.stat().st_mode)
        case.assert_success(case.run_installer(version='v1.0.0', extra=[case.consent]))
        self.assertEqual(stat.S_IMODE(case.profile.stat().st_mode), mode)
        installed = case.profile.read_bytes()
        case.assert_success(case.run_installer(version='v2.0.0', extra=[case.consent]))
        self.assertEqual(case.profile.read_bytes(), installed)
        case.assert_success(case.run_installer(extra=[case.uninstall]))
        self.assertEqual(case.profile.read_bytes(), original)
        self.assertEqual(stat.S_IMODE(case.profile.stat().st_mode), mode)
        self.assert_uninstalled()


@unittest.skipUnless(installers.BASH and Path(installers.BASH).is_file(), 'bash/Git Bash unavailable')
class ShellProfiles(ProfileContract, ProfileFixture):
    fixture_type = installers.ShellInstallers

    def test_profile_round_trip_preserves_non_utf8_bytes_and_mode(self):
        self.check_profile_round_trip(b'# existing \xff bytes\r\nwithout final newline')

    def test_profile_round_trip_preserves_empty_file_and_mode(self):
        self.check_profile_round_trip(b'')

    def test_profile_round_trip_preserves_newline_and_mode(self):
        self.check_profile_round_trip(b'# existing\n')

    def test_uninstall_profile_replacement_failure_preserves_installation(self):
        case = self.case
        case.assert_success(case.run_installer(extra=[case.consent]))
        previous = self.snapshot()
        marker = case.fail_next_move(case.profile)
        case.assert_failure(case.run_installer(extra=[case.uninstall]))
        self.assertTrue(marker.exists())
        self.assert_snapshot(previous)
        (case.tools / 'mv').unlink()
        case.assert_success(case.run_installer(extra=[case.uninstall]))
        self.assert_uninstalled()

    def test_uninstall_receipt_removal_failure_restores_profile_and_receipt(self):
        case = self.case
        case.assert_success(case.run_installer(extra=[case.consent]))
        previous = self.snapshot()
        marker = self.fail_shell_command('rm', case.receipt / 'profile')
        case.assert_failure(case.run_installer(extra=[case.uninstall]))
        self.assertTrue(marker.exists())
        self.assertNotIn(b'SSH KeyFlow PATH', marker.read_bytes())
        self.assert_snapshot(previous)
        (case.tools / 'rm').unlink()
        case.assert_success(case.run_installer(extra=[case.uninstall]))
        self.assert_uninstalled()

    def test_uninstall_binary_removal_failure_restores_profile_and_full_receipt(self):
        case = self.case
        case.assert_success(case.run_installer(extra=[case.consent]))
        previous = self.snapshot()
        marker = self.fail_shell_command('rm', case.binary)
        case.assert_failure(case.run_installer(extra=[case.uninstall]))
        self.assertTrue(marker.exists())
        self.assertNotIn(b'SSH KeyFlow PATH', marker.read_bytes())
        self.assert_snapshot(previous)
        (case.tools / 'rm').unlink()
        case.assert_installed()
        case.assert_success(case.run_installer(extra=[case.uninstall]))
        self.assert_uninstalled()


@unittest.skipUnless(os.name == 'nt' and installers.POWERSHELL, 'Native Windows installer')
class WindowsProfiles(ProfileContract, ProfileFixture):
    fixture_type = installers.WindowsInstallers

    def test_profile_round_trip_preserves_utf8_bom(self):
        self.check_profile_round_trip(b'\xef\xbb\xbf' + '# utf8 日本語\r\nwithout final newline'.encode('utf-8'))

    def test_profile_round_trip_preserves_utf16_le(self):
        self.check_profile_round_trip('# utf16 日本語\r\nwithout final newline'.encode('utf-16'))

    def test_profile_round_trip_preserves_utf16_be(self):
        self.check_profile_round_trip(b'\xfe\xff' + '# big endian 日本語\r\nno newline'.encode('utf-16-be'))

    def test_uninstall_locked_profile_preserves_installation(self):
        case = self.case
        case.assert_success(case.run_installer(extra=[case.consent]))
        previous = self.snapshot()
        case.assert_failure(case.run_installer(extra=[case.uninstall], lock_path=case.profile))
        self.assert_snapshot(previous)
        case.assert_success(case.run_installer(extra=[case.uninstall]))
        self.assert_uninstalled()

    def test_uninstall_receipt_write_failure_restores_profile_and_receipt(self):
        case = self.case
        case.assert_success(case.run_installer(extra=[case.consent]))
        previous = self.snapshot()
        marker = case.home / 'uninstall profile checkpoint'
        blocked = Path(str(case.receipt) + '.tmp')
        self.windows_prelude(r"""
function ConvertTo-Json {
    [CmdletBinding()]
    param([Parameter(ValueFromPipeline=$true)]$InputObject)
    process {
        [IO.File]::Copy(""" + case.ps_quote(case.profile) + ', ' + case.ps_quote(marker) + r""")
        [IO.Directory]::CreateDirectory(""" + case.ps_quote(blocked) + r""") | Out-Null
        Microsoft.PowerShell.Utility\ConvertTo-Json -InputObject $InputObject
    }
}
""")
        case.assert_failure(case.run_installer(extra=[case.uninstall], public_endpoints=True))
        self.assertTrue(marker.exists())
        self.assertNotIn(b'SSH KeyFlow PATH', marker.read_bytes())
        self.assertTrue(blocked.is_dir())
        blocked.rmdir()
        self.assert_snapshot(previous)
        case.assert_success(case.run_installer(extra=[case.uninstall]))
        self.assert_uninstalled()

    def test_uninstall_locked_binary_restores_profile_and_full_receipt(self):
        case = self.case
        case.assert_success(case.run_installer(extra=[case.consent]))
        previous = self.snapshot()
        case.assert_failure(case.run_installer(extra=[case.uninstall], lock_path=case.binary))
        self.assert_snapshot(previous)
        case.assert_installed()
        case.assert_success(case.run_installer(extra=[case.uninstall]))
        self.assert_uninstalled()

    def test_profile_acl_is_preserved_by_install_and_uninstall(self):
        case = self.case
        path = case.ps_quote(case.profile)
        result = self.powershell(
            '$acl = Get-Acl -LiteralPath ' + path +
            '; $acl.SetAccessRuleProtection($true, $true); Set-Acl -LiteralPath ' + path +
            ' -AclObject $acl; (Get-Acl -LiteralPath ' + path +
            ').GetSecurityDescriptorSddlForm([Security.AccessControl.AccessControlSections]::Access)')
        case.assert_success(result)
        original_acl = result.stdout.strip()
        self.assertIn('D:P', original_acl)
        for extra in ([case.consent], [case.uninstall]):
            case.assert_success(case.run_installer(extra=extra))
            result = self.powershell('(Get-Acl -LiteralPath ' + path +
                                     ').GetSecurityDescriptorSddlForm([Security.AccessControl.AccessControlSections]::Access)')
            case.assert_success(result)
            self.assertEqual(result.stdout.strip(), original_acl)
        self.assert_uninstalled()

    def test_iex_preserves_caller_variables_including_readonly_version(self):
        case = self.case
        executables = list(dict.fromkeys(filter(None, [installers.POWERSHELL, shutil.which('powershell')])))
        for index, executable in enumerate(executables):
            with self.subTest(powershell=executable):
                case.destination = case.home / 'local' / 'Programs' / 'ssh-keyflow'
                marker = case.home / ('caller preserved ' + str(index))
                source = case.releases / 'install.ps1'
                checks = r"""
if ($Version -cne 'caller readonly version') { throw 'Version changed' }
if (-not ((Get-Variable Version).Options -band [Management.Automation.ScopedItemOptions]::ReadOnly)) { throw 'Version lost ReadOnly' }
if ($Repository -cne 'caller/repository') { throw 'Repository changed' }
if ($InstallDir -cne 'caller install dir') { throw 'InstallDir changed' }
if ($ProfilePath -cne 'caller profile path') { throw 'ProfilePath changed' }
if ($Uninstall -ne $true -or $AddToPath -ne $true) { throw 'Caller switches changed' }
if ($ErrorActionPreference -ne 'Continue') { throw 'Caller error preference changed' }
if ((Get-Digest) -cne 'caller function') { throw 'Caller function changed' }
"""
                source.write_bytes((installers.ROOT / 'install.ps1').read_bytes().replace(b'\r\n', b'\n') +
                                   checks.encode('utf-8') +
                                   ('\n[IO.File]::WriteAllText(' + case.ps_quote(marker) + ', "preserved")\n').encode('utf-8'))
                self.windows_prelude(r"""
Set-Variable -Name Version -Value 'caller readonly version' -Option ReadOnly
$Repository = 'caller/repository'
$InstallDir = 'caller install dir'
$ProfilePath = 'caller profile path'
$Uninstall = $true
$AddToPath = $true
$ErrorActionPreference = 'Continue'
function Get-Digest { 'caller function' }
""")
                case.assert_success(case.run_installer(pipeline=True, public_endpoints=True, powershell=executable))
                self.assertEqual(marker.read_text(), 'preserved')
                case.assert_installed()
                self.assertEqual(case.profile.read_bytes(), b'export EXISTING=keep\n')
                case.assert_success(case.run_installer(extra=[case.uninstall], powershell=executable))
                self.assert_uninstalled()


if __name__ == '__main__':
    unittest.main()
