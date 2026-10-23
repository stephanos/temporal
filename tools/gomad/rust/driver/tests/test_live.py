"""Receiver-only fixtures: never runtime/live qualification evidence."""
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import driver


class LiveReader(unittest.TestCase):
    def test_exact_owned_requirements_and_no_manifest_only_success(self):
        import live
        owned = live.required()
        self.assertEqual(len(owned), 124)
        self.assertEqual(len(set(owned)), 124)
        self.assertIn('test-wasm/complete-behavioral-reach', owned)
        for config in ({}, {'live_corpus': {'roots': list(range(1, 40)), 'status': 'PASS'}}):
            with self.subTest(config=config), self.assertRaises(driver.Refusal):
                live.read_corpus(config, Path('/unexecuted'))

    def test_process_journal_requires_actual_completion_and_exact_argv(self):
        import live
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            argv = ['/python', '/reader.py']
            command = dict(argv=argv, cwd='/repo', pid=42, exit=0, observation='completed', start_ns=1, end_ns=2)
            process = dict(argv=argv, pid=42, active=False, exit=0, observation='completed', start_ns=1, end_ns=2)
            for mutation in ('missing', 'pid', 'argv', 'active', 'inconclusive', 'exit', 'zero-time'):
                with self.subTest(mutation=mutation):
                    c, p = dict(command), dict(process)
                    if mutation == 'missing':
                        with self.assertRaises(driver.Refusal): live.completed_command(c, None, argv, '/repo')
                        continue
                    if mutation == 'pid': p['pid'] = 43
                    if mutation == 'argv': p['argv'] = ['/other']
                    if mutation == 'active': p['active'] = True
                    if mutation == 'inconclusive': p['observation'] = 'inconclusive'
                    if mutation == 'exit': c['exit'] = p['exit'] = 1
                    if mutation == 'zero-time': c['end_ns'] = p['end_ns'] = 1
                    with self.assertRaises(driver.Refusal): live.completed_command(c, p, argv, '/repo')
            live.completed_command(command, process, argv, '/repo')

    def test_admission_requires_both_original_encodings_every_cell_and_no_writes(self):
        import live
        variants = live.tampers()
        operations = ('verify', 'storage-selections', 'reopen', 'replay')
        rows = [dict(variant=v, operation=o, status=2, executable_writes=0) for v in variants for o in operations]
        for mutation in ('missing', 'duplicate', 'success', 'writes', 'unknown'):
            changed = [dict(row) for row in rows]
            if mutation == 'missing': changed.pop()
            if mutation == 'duplicate': changed[-1] = dict(changed[0])
            if mutation == 'success': changed[0]['status'] = 0
            if mutation == 'writes': changed[0]['executable_writes'] = 1
            if mutation == 'unknown': changed[0]['variant'] = 'unknown'
            with self.subTest(mutation=mutation), self.assertRaises(driver.Refusal): live.admission_rows(changed)
        live.admission_rows(rows)

    def test_private35_refuses_declared_iterations_without_actual_raw_comparisons(self):
        import live
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            driver.write(root / 'result.json', dict(schema='gomad.live-cluster-shutdown-reinsertion/v1', conditions='ordinary', unchanged_guest=True,
                iterations=[dict(iteration=n, live_verified=True, completed=2, reverse_insertion=bool(n % 2), world_bytes=1, network_bytes=1) for n in range(16)]))
            with self.assertRaises((driver.Refusal, OSError)): live.private35(root, 'test result: ok. 1 passed; 0 failed; 0 ignored;')

    def test_reach_rejects_unmapped_original_subtests_and_unknown_ignored_tests(self):
        import live
        original = {'wasi/original_test.go#TestOriginal', 'wasi/original_test.go#TestOriginal/subcase'}
        for entries in ({}, {next(iter(original)): {'evidence': ['transferred-live/TestAlias']}},
                        {key: {'evidence': ['unexecuted']} for key in original}):
            with self.subTest(entries=entries), self.assertRaises(driver.Refusal):
                live.reach_coverage(original, entries, {'transferred-live/TestOriginal'})
        live.reach_coverage(original, {key: {'evidence': ['transferred-live/TestOriginal']} for key in original},
                            {'transferred-live/TestOriginal'})
