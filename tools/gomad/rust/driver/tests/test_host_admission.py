from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import driver
import host_admission as admission


class HostAdmission(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.parent = Path(self.temp.name).resolve()
        self.root = self.parent / 'evidence'
        self.root.mkdir()
        self.scratch = self.parent / 'scratch'
        self.config = dict(root=str(self.root), scratch=str(self.scratch),
                           temporary_paths=dict(TMPDIR=str(self.scratch / 'tmp'),
                                                GOTMPDIR=str(self.scratch / 'go-tmp')),
                           _config_sha256='1'*64, sent_epoch=123, environment={})

    def facts(self, path):
        return dict(path=str(path), filesystem='ext4', mount='/',
                    available_bytes=admission.MIN_FREE_BYTES)

    def test_exact_original_boundary_and_temporary_environment(self):
        receipt = self.root / 'host-admission.json'
        admission.admit(self.config, receipt, initial=True, fact=self.facts)
        row = driver.read(receipt)
        self.assertEqual(row['status'], 'PASS')
        self.assertEqual(len(row['volumes']), 2)
        self.assertEqual(row['sent_epoch'], 123)
        self.assertTrue(all(Path(p).is_dir() for p in self.config['temporary_paths'].values()))
        self.assertEqual(driver.clean_env(self.config)['TMPDIR'], str(self.scratch / 'tmp'))
        admission.admit(self.config, self.root / 'stage-admission.json', initial=False, fact=self.facts)

    def test_both_volumes_require_twelve_gib_before_scratch_creation(self):
        for bad in (self.root, self.scratch.parent):
            with self.subTest(volume=bad):
                def facts(path):
                    row = self.facts(path)
                    if path == bad:
                        row['available_bytes'] -= 1
                    return row
                receipt = self.root / ('refusal-' + bad.name + '.json')
                with self.assertRaisesRegex(driver.Refusal, 'less than 12 GiB'):
                    admission.admit(self.config, receipt, initial=True, fact=facts)
                self.assertFalse(self.scratch.exists())
                self.assertEqual(driver.read(receipt)['status'], 'REFUSED')
                self.assertEqual(len(driver.read(receipt)['volumes']), 2)

    def test_original_forbidden_filesystems(self):
        for kind in ('fuseblk', 'fuse.sshfs', 'myfuse', 'virtiofs'):
            def facts(path):
                return dict(self.facts(path), filesystem=kind if path == self.scratch.parent else 'ext4')
            receipt = self.root / (kind + '.json')
            with self.subTest(kind=kind), self.assertRaisesRegex(driver.Refusal, 'filesystem refused'):
                admission.admit(self.config, receipt, initial=True, fact=facts)
            self.assertFalse(self.scratch.exists())

    def test_missing_explicit_or_overlapping_paths_refuse(self):
        for key, value in (('temporary_paths', {}), ('scratch', str(self.root / 'scratch'))):
            with self.subTest(key=key), self.assertRaises(driver.Refusal):
                admission.bindings(dict(self.config, **{key: value}))
        with self.assertRaises(driver.Refusal):
            admission.bindings(dict(self.config, environment={'TMPDIR': '/tmp'}))

    def test_direct_stage_requires_matching_initial_admission(self):
        with self.assertRaises(driver.Refusal):
            admission.admit(self.config, self.root / 'stage.json', initial=False, fact=self.facts)
        self.assertFalse(self.scratch.exists())

    def test_serving_filesystem_uses_actual_stat_not_stacked_mount_inference(self):
        # The previous max(depth,type,mount) implementation falsely chose xfs
        # over a later virtiofs overmount at the same mountpoint. The actual
        # stat probe is authoritative even with hidden/stacked mount entries.
        import subprocess
        observed = subprocess.CompletedProcess([], 0, stdout='virtiofs\n', stderr='')
        with patch.object(admission.subprocess, 'run', return_value=observed) as probe:
            self.assertEqual(admission.host_fact(self.parent)['filesystem'], 'virtiofs')
        probe.assert_called_once_with(['/usr/bin/stat', '-f', '-c', '%T', str(self.parent)],
                                      check=True, capture_output=True, text=True,
                                      env={'LC_ALL': 'C'}, timeout=5)
        def facts(path):
            return dict(self.facts(path), filesystem='virtiofs')
        with self.assertRaisesRegex(driver.Refusal, 'filesystem refused'):
            admission.admit(self.config, self.root / 'stacked.json', initial=True, fact=facts)
        self.assertFalse(self.scratch.exists())

    def test_probe_errors_fail_closed_with_refusal_receipt(self):
        import subprocess
        for index, error in enumerate((OSError('probe unavailable'), subprocess.TimeoutExpired('stat', 5),
                                       subprocess.CalledProcessError(1, 'stat'))):
            with patch.object(admission.subprocess, 'run', side_effect=error), self.assertRaises(driver.Refusal):
                admission.admit(self.config, self.root / (str(index) + '.json'), initial=True)
            self.assertEqual(driver.read(self.root / (str(index) + '.json'))['status'], 'REFUSED')
            self.assertFalse(self.scratch.exists())
        with patch.object(admission.subprocess, 'run', return_value=subprocess.CompletedProcess([], 0, stdout='')):
            with self.assertRaises(driver.Refusal):
                admission.filesystem_type(self.parent)

    def test_real_kernel_facts_without_dispatch(self):
        row = admission.host_fact(self.parent)
        self.assertGreaterEqual(row['available_bytes'], 0)
        self.assertTrue(row['filesystem'])

    def test_run_refuses_before_source_or_candidate_commands(self):
        config = dict(self.config, root=str(self.parent / 'new-evidence'), deadline_s=3600,
                      identity=dict(host='1'*64, helper='2'*64))
        with patch.object(admission, 'admit', side_effect=driver.Refusal('resource refusal')) as admit, \
             patch.object(driver, 'guard', side_effect=AssertionError('source command before admission')):
            self.assertEqual(driver.run(config, self.parent / 'config.json', 'parallel'), 2)
        admit.assert_called_once()
        self.assertEqual(driver.read(Path(config['root']) / 'outcome.json')['reason'], 'resource refusal')


if __name__ == '__main__':
    unittest.main()
