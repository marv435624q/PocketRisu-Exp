"""Pure verifier contract/privacy tests; never execute a candidate or service."""
import importlib.util
import json
import os
from pathlib import Path
import signal
import unittest
from unittest.mock import MagicMock, call, patch

spec = importlib.util.spec_from_file_location('verifier', Path(__file__).with_name('verify-tsnet-cert7.py'))
verifier = importlib.util.module_from_spec(spec)
spec.loader.exec_module(verifier)


class VerifierContractTests(unittest.TestCase):
    def test_status_keeps_only_allowed_fields(self):
        private = 'private-value-sentinel'
        row = verifier.sanitized_status({'status': 'https-error', 'backendState': 'Running', 'registered': True,
            'httpsStage': 'certificate', 'networkClass': 'timeout',
            'error': 'SetDNS ' + private + ': zero serverNoiseKey', 'authUrl': private, 'url': private,
            'nodeKey': private, 'email': private, 'ip': private})
        self.assertEqual(row, {'status': 'https-error', 'backendState': 'Running', 'registered': True,
                               'httpsStage': 'certificate', 'networkClass': 'timeout', 'errorCode': 'zero-noise-key'})
        self.assertNotIn(private, json.dumps(row))
        rejected = verifier.sanitized_status({'status': 'https-error', 'backendState': 'Running',
                                              'registered': True, 'httpsStage': private, 'networkClass': private})
        self.assertNotIn('httpsStage', rejected)
        self.assertNotIn('networkClass', rejected)
        self.assertIsNone(verifier.sanitized_status({'status': private}))

    def test_https_requires_trust_and_successful_http(self):
        self.assertTrue(verifier.sanitized_https({'success': True, 'tlsVerified': True, 'httpStatus': 401})['success'])
        for value in ({'success': True, 'tlsVerified': False, 'httpStatus': 200},
                      {'success': True, 'tlsVerified': True, 'httpStatus': 502},
                      {'success': True, 'tlsVerified': True, 'httpStatus': True},
                      {'success': False, 'tlsVerified': True, 'httpStatus': 200, 'errorCode': 'private-value-sentinel'}):
            row = verifier.sanitized_https(value)
            self.assertFalse(row['success'])
            self.assertNotIn('private-value-sentinel', json.dumps(row))

    def test_registration_and_timeout_are_safe_error_classes(self):
        self.assertEqual(verifier.classify_error('existing Tailscale registration required; private-value-sentinel'), 'registration-required')
        self.assertEqual(verifier.classify_error('context deadline exceeded private-value-sentinel'), 'deadline')
        self.assertEqual(verifier.classify_error('private-value-sentinel'), 'other')

    def test_progress_is_allowlisted_and_private_values_are_dropped(self):
        private = 'private-value-sentinel'
        self.assertEqual(verifier.sanitized_progress({'stage': 'order', 'state': 'done', 'elapsedMs': 1234,
                                                       'domain': private, 'url': private}),
                         {'stage': 'order', 'state': 'done', 'elapsedMs': 1234})
        self.assertEqual(verifier.sanitized_progress({'stage': 'set-dns-post', 'state': 'start', 'elapsedMs': 0,
                                                       'domain': private, 'url': private}),
                         {'stage': 'set-dns-post', 'state': 'start', 'elapsedMs': 0})
        self.assertEqual(verifier.sanitized_progress({'stage': 'noise-handshake-443', 'state': 'done', 'elapsedMs': 321,
                                                       'domain': private, 'url': private}),
                         {'stage': 'noise-handshake-443', 'state': 'done', 'elapsedMs': 321})
        for value in ({'stage': private, 'state': 'done', 'elapsedMs': 1},
                      {'stage': 'order', 'state': private, 'elapsedMs': 1},
                      {'stage': 'order', 'state': 'done', 'elapsedMs': True},
                      {'stage': 'order', 'state': 'done', 'elapsedMs': 300001}):
            self.assertIsNone(verifier.sanitized_progress(value))

    def test_intermediate_failure_does_not_poison_later_success(self):
        statuses, progress = [], []
        final, terminal = verifier.consume_child_json(
            b'{"type":"status","status":"https-error","backendState":"Running","registered":true,"httpsStage":"certificate","error":"context deadline exceeded"}',
            statuses, progress)
        self.assertIsNone(final)
        self.assertIsNone(terminal)
        final, terminal = verifier.consume_child_json(
            b'{"type":"https-verification","success":true,"tlsVerified":true,"httpStatus":200}',
            statuses, progress)
        self.assertIsNone(terminal)
        self.assertTrue(verifier.successful_https_exit(0, final))

    def test_final_json_without_newline_uses_same_parser(self):
        statuses, progress = [], []
        pending = (b'{"type":"https-progress","stage":"order","state":"done","elapsedMs":12}\n'
                   b'{"type":"https-verification","success":true,"tlsVerified":true,"httpStatus":204}')
        pending, final, terminal = verifier.consume_pending_lines(pending, statuses, progress, None, flush=False)
        self.assertIsNone(terminal)
        self.assertIsNone(final)
        self.assertEqual(progress, [{"stage": "order", "state": "done", "elapsedMs": 12}])
        self.assertTrue(pending)
        pending, final, terminal = verifier.consume_pending_lines(pending, statuses, progress, final, flush=True)
        self.assertEqual(pending, b"")
        self.assertIsNone(terminal)
        self.assertTrue(verifier.successful_https_exit(0, final))

    def test_children_do_not_inherit_tailscale_environment(self):
        variables = {'TS_AUTHKEY': 'secret', 'TS_AUTH_KEY': 'secret', 'TS_CLIENT_SECRET': 'secret',
                     'TS_CONTROL_URL': 'private', 'TSNET_FORCE_LOGIN': '1', 'PATH': '/bin'}
        with patch.dict(os.environ, variables, clear=True):
            expected = {'PATH': '/bin'}
            self.assertEqual(verifier.child_environment(), expected)
            completed = MagicMock(returncode=0, stdout=b'')
            with patch.object(verifier.subprocess, 'run', return_value=completed) as run:
                verifier.single_probe(['--help'], 0)
                self.assertEqual(run.call_args.kwargs['env'], expected)
                verifier.build_info({'sourceId': 'a' * 64, 'candidate': 'x', 'longVersion': 'x'})
                self.assertEqual(run.call_args.kwargs['env'], expected)
            with patch.object(verifier, 'preflight', return_value=True), \
                 patch.object(verifier.subprocess, 'Popen', side_effect=OSError) as popen:
                verifier.online(Path('/nonexistent/tailscale-state'), 'test-host', 30)
                self.assertEqual(popen.call_args.kwargs['env'], expected)

    def test_exited_leader_still_cleans_descendant_group(self):
        proc = MagicMock(pid=1234)
        proc.poll.return_value = 0
        alive = {'value': True}
        def killpg(pid, sig):
            self.assertEqual(pid, 1234)
            if not alive['value']:
                raise ProcessLookupError()
            if sig == signal.SIGTERM:
                alive['value'] = False
        with patch.object(verifier.os, 'killpg', side_effect=killpg) as kill:
            self.assertTrue(verifier.stop_process(proc))
            self.assertIn(call(1234, signal.SIGTERM), kill.call_args_list)
            proc.stdin.write.assert_not_called()

    def test_group_kill_error_is_reported_without_unbounded_wait(self):
        proc = MagicMock(pid=1234)
        proc.poll.return_value = 0
        def killpg(_pid, sig):
            if sig == signal.SIGKILL:
                raise PermissionError()
        with patch.object(verifier.os, 'killpg', side_effect=killpg), \
             patch.object(verifier.time, 'monotonic', side_effect=[0, 6, 6]):
            self.assertFalse(verifier.stop_process(proc))
            proc.wait.assert_not_called()


if __name__ == '__main__':
    unittest.main()
