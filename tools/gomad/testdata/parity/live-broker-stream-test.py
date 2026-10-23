#!/usr/bin/env python3
"""Offline original root36 row controls; requires authenticated V6 reference input."""
from copy import deepcopy
import hashlib
import importlib.util
import json
import os
import tempfile
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location('cluster_check', Path(__file__).with_name('live-broker-cluster-check.py'))
check = importlib.util.module_from_spec(spec)
spec.loader.exec_module(check)


class IndependentStreams(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        root = Path(os.environ['GOMAD_LIVE_BROKER_REFERENCE'])
        raw = (root / 'observation-index.json').read_bytes()
        assert hashlib.sha256(raw).hexdigest() == '0522dffcc1c717749778907002f11055d77c0ac1a89a72a7d68b7dce51c611ba'
        index = json.loads(raw)
        cls.rows = []
        for name, descriptor in index['files'].items():
            if not name.startswith('cluster-'):
                continue
            raw = (root / 'observations' / name).read_bytes()
            assert len(raw) == descriptor['bytes'] and hashlib.sha256(raw).hexdigest() == descriptor['sha256']
            row = json.loads(raw)
            if row['test'].startswith(check.observe.ROOTS[36] + '/'):
                cls.rows.append(row)
        assert len(cls.rows) == 18

    def test_original_record_and_forced_replay_controls_include_link_before_start(self):
        modes, leading_links = [], 0
        for row in self.rows:
            with self.subTest(ordinal=row['ordinal'], test=row['test']):
                starts = [step for step in row['steps'] if step['action'] == 'start']
                mode = starts[0]['runtime']['Choice']['Mode']
                self.assertEqual([step['node'] for step in starts], ['a', 'b'])
                self.assertEqual([step['runtime']['Choice']['Mode'] for step in starts], [mode, mode])
                self.assertIn(mode, (1, 2))
                self.assertEqual(check.independent_stream_recording(row), mode == 1)
                modes.append(mode)
                leading_links += row['steps'][0]['action'] == 'link'
        self.assertEqual(modes.count(1), 9)
        self.assertEqual(modes.count(2), 9)
        self.assertEqual(leading_links, 6)

    def test_original_replay_controls_still_force_both_actual_guest_traces(self):
        replay_rows = [row for row in self.rows if not check.independent_stream_recording(row)]
        self.assertEqual(len(replay_rows), 9)
        with tempfile.TemporaryDirectory() as directory:
            for row in replay_rows:
                with self.subTest(ordinal=row['ordinal']):
                    payload = check.raw_input(row, dict(configuration={}, modules={}), Path(directory), row['ordinal'])
                    self.assertEqual(set(payload['forcing']), {'a/1', 'b/1'})
                    for step in [step for step in row['steps'] if step['action'] == 'start']:
                        state = next(state for state in row['states'] if state['handle'] == dict(ID=step['node'], Incarnation=1))
                        choice = step['runtime']['Choice']
                        self.assertEqual(choice['Mode'], 2)
                        forced = payload['forcing'][step['node'] + '/1']
                        self.assertEqual(forced['mode'], 2)
                        self.assertEqual(forced['identity'], choice['ExecutionIdentity'])
                        expected = check.b64(choice.get('Trace') or state['result']['ChoiceTrace']['Bytes'])
                        self.assertNotEqual(expected, b'')
                        self.assertEqual(Path(forced['trace_path']).read_bytes(), expected)

    def test_missing_ambiguous_and_changed_runtime_controls_fail_closed(self):
        original = next(row for row in self.rows if row['steps'][0]['action'] == 'link'
                        and next(step for step in row['steps'] if step['action'] == 'start')['runtime']['Choice']['Mode'] == 1)
        for mutation in ('missing-a', 'duplicate-a', 'missing-runtime', 'nil-runtime', 'missing-choice', 'nil-choice', 'missing-mode', 'unknown-mode', 'mixed-mode', 'wrong-incarnation'):
            with self.subTest(mutation=mutation):
                row = deepcopy(original)
                a, b = [step for step in row['steps'] if step['action'] == 'start']
                if mutation == 'missing-a': row['steps'].remove(a)
                elif mutation == 'duplicate-a': row['steps'].append(deepcopy(a))
                elif mutation == 'missing-runtime': del a['runtime']
                elif mutation == 'nil-runtime': a['runtime'] = None
                elif mutation == 'missing-choice': del a['runtime']['Choice']
                elif mutation == 'nil-choice': a['runtime']['Choice'] = None
                elif mutation == 'missing-mode': del a['runtime']['Choice']['Mode']
                elif mutation == 'unknown-mode': a['runtime']['Choice']['Mode'] = 3
                elif mutation == 'mixed-mode': b['runtime']['Choice']['Mode'] = 2
                elif mutation == 'wrong-incarnation': a['incarnation'] = 2
                with self.assertRaises(AssertionError):
                    check.independent_stream_recording(row)


if __name__ == '__main__':
    unittest.main()
