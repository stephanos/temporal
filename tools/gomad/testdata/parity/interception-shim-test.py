#!/usr/bin/env python3
"""Offline checker/byte-capture regressions, with no real Go or Gomad execution."""
import copy
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

HERE = Path(__file__).resolve().parent
loader = SourceFileLoader('shim_acceptance', str(HERE / 'interception-shim-check.py'))
check = module_from_spec(spec_from_loader(loader.name, loader))
loader.exec_module(check)


class ByteCapture(unittest.TestCase):
    def test_non_utf8_empty_and_shell_metacharacters_round_trip_without_shell(self):
        with tempfile.TemporaryDirectory(prefix='interception-shim-offline-') as directory:
            root = Path(directory)
            check.program(root / 'go', 'compiler')
            check.program(root / 'child', 'host-child')
            env = dict(os.environb, PATH=os.fsencode(root) + b':' + os.environb[b'PATH'],
                       INTERCEPTION_CAPTURE_ROOT=os.fsencode(root), LC_ALL=b'C.UTF-8',
                       INTERCEPTION_INVALID_ENV=b'\xff\x80 env value')
            # Dict keyword arguments above are text; convert the entire map to bytes.
            env = {os.fsencode(key): os.fsencode(value) for key, value in env.items()}
            result = subprocess.run([os.fsencode(root / 'child'), b'list', *check.RAW_ARGS], env=env,
                                    stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            self.assertEqual(result.returncode, 0, result.stderr)
            received = [json.loads(path.read_bytes()) for path in root.glob('compiler-*.json')]
            self.assertEqual(len(received), 1)
            self.assertEqual(received[0]['argv'], [value.hex() for value in [b'list', *check.RAW_ARGS]])
            self.assertEqual(received[0]['environment'], json.loads((root / 'inherited.json').read_bytes())['environment'])
            self.assertEqual(received[0]['executable'], str(root / 'go'))

    def test_fixture_goos_uses_host_not_guest(self):
        with tempfile.TemporaryDirectory(prefix='interception-shim-offline-') as directory:
            root = Path(directory)
            check.program(root / 'go', 'compiler')
            result = subprocess.run([root / 'go', 'env', 'GOOS'], env=dict(os.environ, INTERCEPTION_CAPTURE_ROOT=str(root)),
                                    stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout, (os.uname().sysname.lower() + '\n').encode())

    def test_source_case_set_covers_host_refusals_boundaries_and_cache(self):
        self.assertEqual(len(check.CASES), len(check.CORPUS['cases']))
        for command in ('env','list','mod','generate','vet','tool','version','install','clean','run','build'):
            self.assertIn(('unmatched-' if command in ('run','build') else 'host-') + command, check.CASES)
        for flag in ('race','fuzz','cover','coverprofile','coverpkg','msan','asan'):
            for channel in ('argv','goflags'):
                self.assertIn('refuse-' + flag + '-' + channel, check.CASES)
        for name in ('boundary-args','boundary-double-args','boundary-terminator','guest-lookalikes',
                     'goflags-quoted','repeated-effective-flags','empty-effective-exec','external-launcher',
                     'exec-true-count0','multi-package-json','auto-schema-mutation','auto-schema-missing',
                     'auto-changed-executable','concurrent-identities','explicit-excluded-tag'):
            self.assertTrue(check.CASES[name]['requires_real_inputs'])


class DriverFlags(unittest.TestCase):
    def test_effective_empty_exec_and_count_preserve_last_value_before_guest_boundary(self):
        flags = check.driver_flags(['-exec=false','-exec=gomad run-guest','-count=2',
                                    '-args','-exec=guest-only','-count=guest-only'])
        self.assertEqual(flags, dict(exec='gomad run-guest', count='2'))
        self.assertEqual(check.driver_flags(['-exec=false','-exec=']), dict(exec=''))
        for token in ('-args','--args','--'):
            self.assertEqual(check.driver_flags(['-exec','gomad run-guest',token,'-exec=wrong']),
                             dict(exec='gomad run-guest'))


class InstallationMapping(unittest.TestCase):
    def fixture(self):
        bindings = [['go.temporal.io/server/tests/gomadfunctional', 'a' * 64]]
        origin = dict(invocation=dict(installation_sha256=check.sha(check.encode(bindings))))
        provenance = dict(inputs=dict(shared_driver_installations=bindings))
        return provenance, origin, copy.deepcopy(bindings)

    def test_correct_retained_mapping_binds_aggregate_and_actual_manifest(self):
        provenance, origin, expected = self.fixture()
        check.authenticate_installations(provenance, origin, expected)

    def test_self_consistent_false_mapping_cannot_borrow_supplied_manifest(self):
        provenance, origin, expected = self.fixture()
        provenance['inputs']['shared_driver_installations'][0][1] = 'b' * 64
        origin['invocation']['installation_sha256'] = check.sha(check.encode(provenance['inputs']['shared_driver_installations']))
        with self.assertRaisesRegex(ValueError, 'exact_installation_mapping'):
            check.authenticate_installations(provenance, origin, expected)

    def test_duplicate_unknown_and_commitment_changes_are_refused(self):
        for mode in ('duplicate','unknown','commitment'):
            provenance, origin, expected = self.fixture()
            if mode == 'duplicate':
                provenance['inputs']['shared_driver_installations'] *= 2
            elif mode == 'unknown':
                provenance['inputs']['shared_driver_installations'].append(['other/package','b' * 64])
            else:
                origin['invocation']['installation_sha256'] = 'f' * 64
            with self.subTest(mode=mode), self.assertRaises(ValueError):
                check.authenticate_installations(provenance, origin, expected)


if __name__ == '__main__':
    unittest.main()
