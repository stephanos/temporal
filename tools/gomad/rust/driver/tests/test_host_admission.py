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

    def test_mount_escape_and_longest_mount_match(self):
        (self.parent / 'with space').mkdir()
        mount = str(self.parent / 'with space').replace(' ', r'\040')
        mounts = '1 0 0:1 / / rw - ext4 /dev/root rw\n2 1 0:2 / ' + mount + ' rw - fuse.sshfs source rw\n'
        kind, _ = admission.filesystem(self.parent / 'with space', mounts)
        self.assertEqual(kind, 'fuse.sshfs')

    def test_real_kernel_facts_without_dispatch(self):
        if not Path('/proc/self/mountinfo').exists():
            self.skipTest('Linux mountinfo is the original supported host contract')
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
