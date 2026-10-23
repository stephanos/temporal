import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
import sys
sys.path.insert(0, str(ROOT))
import driver


class DriverContract(unittest.TestCase):
    def test_complete_stage_and_subgate_set(self):
        mapping = driver.mapping()
        self.assertEqual(len(mapping), 96)
        for mode in ('serial', 'parallel'):
            self.assertEqual(driver.stage_commands(mode, '/config'), driver.stage_commands('serial', '/config'))
        for mutation in ('missing', 'duplicate', 'unexecuted'):
            rows = [dict(id='a', status='PASS', executed=True, exit=0)]
            if mutation == 'missing': rows = []
            if mutation == 'duplicate': rows *= 2
            if mutation == 'unexecuted': rows[0]['executed'] = False
            with self.subTest(mutation=mutation), self.assertRaises(driver.Refusal):
                driver.coverage(['a'], rows)

    def test_deadline_includes_send_preamble_final_and_suspension(self):
        self.assertEqual(driver.verdict(0, 100, 3700, 3600), 0)
        self.assertEqual(driver.verdict(0, 100, 3700.01, 3600), 3)
        self.assertEqual(driver.verdict(1, 100, 3800, 3600), 1)
        for limit in (0, -1, 3601, float('nan'), float('inf')):
            with self.subTest(limit=limit), self.assertRaises(driver.Refusal):
                driver.deadline(limit)

    def test_role_identity_and_refusal_are_not_success(self):
        expected = {'host': 'sha256:' + 'a' * 64}
        for role, observed in [('fixture', expected), ('release', {'host': 'bad'})]:
            with self.subTest(role=role), self.assertRaises(driver.Refusal):
                driver.bind_identity('release', expected, role, observed)
        for code in (1, 2, 3, 86, -9):
            with self.subTest(code=code), self.assertRaises(driver.Refusal):
                driver.completed(code, [0])

    def test_native_rust_digest_encoding_is_preserved(self):
        import hashlib
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'bytes'
            path.write_bytes(b'raw')
            self.assertEqual(driver.sha(path), hashlib.sha256(b'raw').hexdigest())
        identity = {'host': 'a' * 64, 'helper': 'b' * 64}
        driver.bind_identity('release', identity, 'release', identity)

    def test_source_guard_rejects_mutation_and_replaced_symlink(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / 'source'
            source.write_text('original')
            files = {str(source): driver.sha(source)}
            driver.check_files(files)
            source.write_text('changed')
            with self.assertRaises(driver.Refusal): driver.check_files(files)
            source.unlink()
            other = root / 'other'
            other.write_text('original')
            source.symlink_to(other)
            with self.assertRaises(driver.Refusal): driver.check_files(files)

    def test_manifest_cannot_earn_actual_gate(self):
        with self.assertRaises(driver.Refusal):
            driver.coverage(['a'], [{'id': 'a', 'status': 'PASS', 'executed': True, 'exit': 0, 'synthetic': True}])

    def test_timed_out_command_is_reaped_and_records_numeric_exit(self):
        import subprocess
        import sys
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with self.assertRaises(subprocess.TimeoutExpired):
                driver.command({'repo': str(root)}, root, 'timeout',
                               [sys.executable, '-c', 'import time; time.sleep(5)'], timeout=0.02)
            receipt = driver.read(root / 'timeout.command.json')
            self.assertLess(receipt['exit'], 0)
            self.assertEqual(receipt['observation'], 'inconclusive')

    def test_pending_actual_stage_writes_unexecuted_receipt(self):
        from unittest.mock import patch
        with tempfile.TemporaryDirectory() as directory:
            config = dict(root=directory, source_sha256='1'*64, identity={'host':'2'*64,'helper':'3'*64},
                          _config_sha256='4'*64, stages={'storage': {'adapter':'pending', 'reason':'actual producer absent'}})
            with patch.object(driver, 'guard', lambda config: None), \
                 patch('host_admission.admit'):
                self.assertEqual(driver.stage(config, 'storage'), 4)
            receipt = driver.read(Path(directory) / 'stages/storage/receipt.json')
            self.assertEqual((receipt['status'], receipt['exit'], receipt['subgates']), ('UNEXECUTED', 4, []))

    def test_frozen_resource_policy_preserved(self):
        stages = driver.scheduler.load_dag(driver.DAG)
        self.assertEqual(len(stages), 96)
        self.assertEqual(stages['busy-grouped'].cls, 'exclusive')
        self.assertEqual(stages['busy-boundaries'].cores, 12)
        self.assertIn('memguard', stages['runtime-31'].flags)
        self.assertFalse(driver.scheduler.can_start(stages['terminal-exit'], [stages['runtime-0']],
                                                  driver.scheduler.Policy(16, 32)))


if __name__ == '__main__':
    unittest.main()
