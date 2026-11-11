import itertools
import json
from pathlib import Path
import subprocess
import unittest
from unittest.mock import patch

import publish_release


def release(tag, prerelease=False, draft=False):
    return {'tag_name': tag, 'prerelease': prerelease, 'draft': draft}


class FakeGitHub:
    def __init__(self, latest=None, pages=None, status=None):
        self.latest = latest
        self.pages = pages if pages is not None else [[]]
        self.status = status
        self.commands = []
        self.list_failure = False

    def run(self, command, **kwargs):
        self.commands.append(command)
        if command[1] == 'api':
            if command[2].endswith('/latest'):
                status = self.status or (200 if self.latest else 404)
                body = self.latest if status == 200 else {'message': 'lookup failed'}
                return subprocess.CompletedProcess(command, int(status != 200),
                                                   f'HTTP/2.0 {status}\nContent-Type: application/json\n\n'
                                                   + json.dumps(body), '')
            return subprocess.CompletedProcess(command, int(self.list_failure), json.dumps(self.pages), '')
        if '--latest=true' in command:
            self.latest = release(command[3])
        return subprocess.CompletedProcess(command, 0, '', '')


class ReleaseTests(unittest.TestCase):
    def publish(self, github, tag='v1.9.1'):
        with patch.object(publish_release.subprocess, 'run', side_effect=github.run):
            publish_release.publish('example/project', tag, [Path('dist/archive.zip')])
        return github.commands[-1]

    def test_older_maintenance_does_not_replace_newer_latest(self):
        github = FakeGitHub(release('v1.10.0'))
        command = self.publish(github)
        self.assertIn('--latest=false', command)
        self.assertEqual(github.latest['tag_name'], 'v1.10.0')
        self.assertEqual(command[:4], ['gh', 'release', 'create', 'v1.9.1'])
        self.assertEqual(command[command.index('--repo') + 1], 'example/project')
        self.assertIn('--verify-tag', command)
        self.assertIn('--generate-notes', command)
        self.assertIn('dist/archive.zip', [arg.replace('\\', '/') for arg in command])

    def test_newer_numeric_version_advances_latest(self):
        github = FakeGitHub(release('v1.9.9'))
        self.assertIn('--latest=true', self.publish(github, 'v1.10.0'))
        self.assertEqual(github.latest['tag_name'], 'v1.10.0')

    def test_equal_version_does_not_request_latest_or_mutate_existing_release(self):
        github = FakeGitHub(release('v1.9.1'))
        command = self.publish(github)
        self.assertIn('--latest=false', command)
        self.assertEqual(command[2], 'create')  # Never edit/delete/re-upload an existing release.

    def test_first_release_requires_successful_empty_listing(self):
        github = FakeGitHub()
        self.assertIn('--latest=true', self.publish(github, 'v0.1.0'))
        self.assertIn('--paginate', github.commands[1])
        self.assertIn('--slurp', github.commands[1])

    def test_first_stable_after_only_previews_and_drafts(self):
        github = FakeGitHub(pages=[[release('v1.0.0-rc.1', prerelease=True)],
                                  [release('v2.0.0', draft=True)]])
        self.assertIn('--latest=true', self.publish(github))

    def test_prerelease_never_looks_up_or_advances_latest(self):
        github = FakeGitHub(release('v1.10.0'), status=500)
        command = self.publish(github, 'v2.0.0-rc.1')
        self.assertIn('--prerelease', command)
        self.assertIn('--latest=false', command)
        self.assertEqual(len(github.commands), 1)

    def test_lookup_failures_never_publish(self):
        for status in (401, 403, 429, 500, 503):
            with self.subTest(status=status):
                github = FakeGitHub(status=status)
                with self.assertRaises(publish_release.ReleaseError):
                    self.publish(github)
                self.assertTrue(all(command[1] == 'api' for command in github.commands))

    def test_404_listing_failure_never_publishes(self):
        github = FakeGitHub()
        github.list_failure = True
        with self.assertRaises(publish_release.ReleaseError):
            self.publish(github)
        self.assertEqual(len(github.commands), 2)

    def test_404_with_stable_on_later_page_never_publishes(self):
        github = FakeGitHub(pages=[[release('v2.0.0-rc.1', prerelease=True)],
                                  [release('v1.10.0')]])
        with self.assertRaises(publish_release.ReleaseError):
            self.publish(github)
        self.assertEqual(len(github.commands), 2)

    def test_invalid_current_latest_never_publishes(self):
        for value in (release('latest'), release('v1.0.0', draft=True),
                      release('v1.0.0-rc.1', prerelease=True), {}, []):
            with self.subTest(value=value):
                github = FakeGitHub(value, status=200)
                with self.assertRaises(publish_release.ReleaseError):
                    self.publish(github)
                self.assertEqual(len(github.commands), 1)

    def test_malformed_or_transport_response_never_publishes(self):
        for output in ('', 'not json', 'HTTP/2.0 200\nContent-Type: application/json\n\nnot json',
                       'HTTP/2.0 200\nContent-Type: application/json\n\nnull'):
            with self.subTest(output=output):
                with patch.object(publish_release.subprocess, 'run',
                                  return_value=subprocess.CompletedProcess([], 0, output, '')) as run:
                    with self.assertRaises(publish_release.ReleaseError):
                        publish_release.publish('example/project', 'v1.0.0', [Path('asset')])
                    self.assertEqual(run.call_count, 1)

    def test_transport_error_never_publishes(self):
        with patch.object(publish_release.subprocess, 'run',
                          return_value=subprocess.CompletedProcess([], 1, '', 'network failure')) as run:
            with self.assertRaises(publish_release.ReleaseError):
                publish_release.publish('example/project', 'v1.0.0', [Path('asset')])
            self.assertEqual(run.call_count, 1)

    def test_invalid_listing_never_publishes(self):
        for pages in ([], {}, [None], [[{}]]):
            with self.subTest(pages=pages):
                github = FakeGitHub(pages=pages)
                with self.assertRaises(publish_release.ReleaseError):
                    self.publish(github)
                self.assertEqual(len(github.commands), 2)

    def test_crlf_http_headers(self):
        github = FakeGitHub(release('v1.10.0'))
        original = github.run

        def crlf(command, **kwargs):
            result = original(command, **kwargs)
            result.stdout = result.stdout.replace('\n', '\r\n')
            return result

        with patch.object(publish_release.subprocess, 'run', side_effect=crlf):
            publish_release.publish('example/project', 'v1.9.1', [Path('asset')])
        self.assertEqual(github.latest['tag_name'], 'v1.10.0')

    def test_invalid_candidates_fail_before_api_calls(self):
        for tag in ('v01.2.3', 'v1.2', 'v1.2.3-01', 'v1.2.3-', 'latest', 'v1.2.3+build'):
            with self.subTest(tag=tag):
                github = FakeGitHub()
                with self.assertRaises(publish_release.ReleaseError):
                    self.publish(github, tag)
                self.assertEqual(github.commands, [])

    def test_queued_publications_preserve_maximum_in_every_order(self):
        # The workflow serializes lookup + creation. Exercise all possible
        # queue orders; each queued publication must look up the updated state.
        tags = ('v1.9.1', 'v1.10.0', 'v2.0.0-rc.1')
        for order in itertools.permutations(tags):
            with self.subTest(order=order):
                github = FakeGitHub(release('v1.9.0'))
                for tag in order:
                    self.publish(github, tag)
                self.assertEqual(github.latest['tag_name'], 'v1.10.0')

    def test_creation_failure_is_not_retried(self):
        github = FakeGitHub(release('v1.10.0'))
        original = github.run

        def fail_create(command, **kwargs):
            if command[1] == 'release':
                github.commands.append(command)
                raise subprocess.CalledProcessError(1, command)
            return original(command, **kwargs)

        with patch.object(publish_release.subprocess, 'run', side_effect=fail_create):
            with self.assertRaises(subprocess.CalledProcessError):
                publish_release.publish('example/project', 'v1.9.1', [Path('asset')])
        self.assertEqual(len(github.commands), 2)


if __name__ == '__main__':
    unittest.main()
