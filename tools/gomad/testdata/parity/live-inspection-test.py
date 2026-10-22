#!/usr/bin/env python3
import importlib.util
from pathlib import Path
import struct
import tempfile
from types import SimpleNamespace
import unittest


class BoundaryTests(unittest.TestCase):
    def setUp(self):
        path = Path(__file__).with_name('live-inspection-check.py')
        spec = importlib.util.spec_from_file_location('live_inspection', path)
        self.module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.module)
        self.assertTrue(hasattr(self.module, "boundaries"), "missing retained boundary decoder")

    def test_decodes_every_boundary_and_rejects_trailing_bytes(self):
        node = b'a'
        handle = struct.pack('<I', len(node)) + node + struct.pack('<Q', 2)
        cut = handle + struct.pack('<7Q', 0, 0, 0, 0, 0, 0, 0) + struct.pack('<I', 0) + bytes(224)
        frame = handle + struct.pack('<6Q', 3, 4, 5, 6, 7, 8) + struct.pack('<I', 1) + cut
        raw = b'GMWIB\x01\0\0' + struct.pack('<I', 2) + (struct.pack('<I', len(frame)) + frame) * 2
        expected = dict(node='a', incarnation=2, decision=3, trace=4, order=5, time=6, world_bytes=7, lifecycle_bytes=8, cuts=[dict(instance='a', incarnation=2, choice_records=0, choice_bytes=0, wasi_bytes=0, network_bytes=0, storage_bytes=0, stdout_bytes=0, stderr_bytes=0)])
        self.assertEqual(self.module.boundaries(raw), [expected, expected])
        with self.assertRaises((AssertionError, ValueError)):
            self.module.boundaries(raw + b'\0')

    def test_failed_command_retains_numeric_exit_and_raw_logs(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            lane = self.module.Lane(SimpleNamespace(binary=Path('/bin/false'), work=root))
            with self.assertRaises(AssertionError):
                lane.run('refused', root / 'artifact', [])
            self.assertTrue((root / 'commands.json').exists(), 'failed command receipt missing')
            self.assertEqual(lane.rows[0]['status'], 1)
            self.assertTrue((root / 'refused.stdout').exists())
            self.assertTrue((root / 'refused.stderr').exists())

    def test_network_cuts_require_zero_target_and_full_predecessor(self):
        cuts = [dict(instance='a', incarnation=1, network_bytes=12),
                dict(instance='b', incarnation=1, network_bytes=0)]
        self.module.network_cuts(cuts, {('a', 1): 12, ('b', 1): 10}, ('b', 1))
        cuts[0]['network_bytes'] = 11
        with self.assertRaises(AssertionError):
            self.module.network_cuts(cuts, {('a', 1): 12, ('b', 1): 10}, ('b', 1))


if __name__ == '__main__':
    unittest.main()
