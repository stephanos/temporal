#!/usr/bin/env python3
"""Receipt admission checks; live owner equivalence runs in live-broker-owner-check.py."""
import importlib.util
import json
from pathlib import Path
import sys
import unittest

sys.dont_write_bytecode = True
path = Path(__file__).with_name('live-broker-observe.py')
spec = importlib.util.spec_from_file_location('observe', path)
observe = importlib.util.module_from_spec(spec)
spec.loader.exec_module(observe)


class ExactSelection(unittest.TestCase):
    def test_provider_observer_preserves_exact_original_admission_paths(self):
        self.assertTrue(hasattr(observe, 'instrument_provider'), 'missing provider observer')
        source = (Path(__file__).resolve().parents[2] / 'internal/backends/wasm/cluster/provider_test.go').read_bytes()
        observed = observe.instrument_provider(source)
        self.assertEqual(observed.count(b'providerObserveOpen(t, '), 3)
        self.assertIn(b'providerObserveOpen(t, reference, t.Context(), hard)', observed)
        self.assertEqual(observed.count(b't.Fatal('), source.count(b't.Fatal('))
        self.assertEqual(observed.count(b't.Fatalf('), source.count(b't.Fatalf('))
        self.assertEqual(observed.count(b'.Start('), source.count(b'.Start('))
        self.assertEqual(observed.count(b'.Wait('), source.count(b'.Wait('))

    def test_root_pass_does_not_admit_missing_or_skipped_cell(self):
        root = observe.ROOTS[37]
        cells = [f'{root}/{clock}/digest-{digest}' for clock in ('read-step', 'frozen') for digest in ('false', 'true')]
        passed = [dict(Test=name, Action='pass') for name in [root, *cells]]
        for replacement in (None, 'skip', 'fail'):
            with self.subTest(replacement=replacement):
                events = passed[:-1]
                if replacement:
                    events = [*events, dict(Test=cells[-1], Action=replacement)]
                with self.assertRaisesRegex(ValueError, 'required cell did not pass|failed or skipped subtest'):
                    observe.validate_events(b'\n'.join(json.dumps(event).encode() for event in events), [37])
        self.assertEqual(len(observe.validate_events(b'\n'.join(json.dumps(event).encode() for event in passed), [37])), 5)

    def test_observer_preserves_subtest_calls_and_assertions(self):
        source = (Path(__file__).resolve().parents[2] / 'internal/backends/wasm/wasi/integration_test.go').read_bytes()
        observed = observe.instrument(source)
        self.assertNotIn(b't.brokerObserveRun', observed)
        self.assertEqual(observed.count(b't.Fatal('), source.count(b't.Fatal('))
        self.assertEqual(observed.count(b't.Fatalf('), source.count(b't.Fatalf('))
        self.assertEqual(observed.count(b't.Run('), source.count(b't.Run('))
        self.assertGreater(observed.count(b'brokerObserveRun(t, '), 0)


class DiagnosticEncoding(unittest.TestCase):
    def test_expected_header_mutations_cannot_rewrite_actual_framing(self):
        path = Path(__file__).with_name('live-broker-owner-check.py')
        spec = importlib.util.spec_from_file_location('owner', path)
        owner = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(owner)
        self.assertTrue(hasattr(owner, 'require_diagnostic'), 'missing independent diagnostic header comparison')
        header = bytes.fromhex('474f4d4144444701' '00000001' '01000000'
                               '0000000000000100' '00000000000000a0' '0000000000000001' + '00' * 24)
        records = bytes(96)
        owner.require_diagnostic(records, header + records, 256)
        for offset in (0, 11, 12, 13, 23, 31, 39, 63):
            with self.subTest(offset=offset):
                changed = bytearray(header + records)
                changed[offset] ^= 1
                with self.assertRaises(AssertionError):
                    owner.require_diagnostic(records, bytes(changed), 256)


if __name__ == '__main__':
    unittest.main()
