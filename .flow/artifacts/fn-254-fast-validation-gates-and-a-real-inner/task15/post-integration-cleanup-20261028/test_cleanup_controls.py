"""Read-only admission helpers exercised against tiny disposable fixtures; no real cleanup commands."""
import hashlib
import json
import os
import stat
import tempfile
import unittest
from pathlib import Path
import cleanup

HERE = Path(__file__).resolve().parent


class Controls(unittest.TestCase):
    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory(dir=HERE, prefix='safe-cleanup-admission-')
        self.addCleanup(self.scratch.cleanup)
        self.root = Path(self.scratch.name)
        self.path = self.root/'original'
        self.path.write_bytes(b'original')
        info = self.path.stat()
        self.row = dict(path='original', source=str(self.path), sha256=hashlib.sha256(b'original').hexdigest(),
                        device=info.st_dev, inode=info.st_ino, size=info.st_size, mode=oct(stat.S_IMODE(info.st_mode)),
                        disposition='supplemental-custody-required-preserve-until-verified')

    def test_exact_original_and_removed_subset_are_admitted(self):
        self.assertEqual(cleanup.verify_remaining(self.root, [self.row], True)['files'], 1)
        self.path.unlink()
        self.assertEqual(cleanup.verify_remaining(self.root, [self.row])['files'], 0)
        with self.assertRaisesRegex(AssertionError, 'full private membership changed'):
            cleanup.verify_remaining(self.root, [self.row], True)

    def test_new_file_refuses_remainder_removal(self):
        (self.root/'new').write_bytes(b'new')
        with self.assertRaisesRegex(AssertionError, 'new file not admitted'):
            cleanup.verify_remaining(self.root, [self.row])

    def test_changed_inode_mode_and_bytes_are_refused(self):
        self.path.write_bytes(b'altered!')
        with self.assertRaisesRegex(AssertionError, 'hash changed'):
            cleanup.verify_remaining(self.root, [self.row])
        self.path.write_bytes(b'original')
        os.chmod(self.path, 0o700)
        with self.assertRaisesRegex(AssertionError, 'metadata changed'):
            cleanup.verify_remaining(self.root, [self.row])
        replacement = self.root/'replacement'
        replacement.write_bytes(b'original')
        os.chmod(replacement, int(self.row['mode'], 8))
        replacement.replace(self.path)
        self.assertNotEqual(self.path.stat().st_ino, self.row['inode'])
        with self.assertRaisesRegex(AssertionError, 'metadata changed'):
            cleanup.verify_remaining(self.root, [self.row])

    def test_symlink_and_unknown_process_visibility_are_refused(self):
        (self.root/'link').symlink_to(self.path)
        with self.assertRaisesRegex(AssertionError, 'symlink/special'):
            cleanup.verify_remaining(self.root, [self.row])
        class Inspector:
            @staticmethod
            def ownership(_):
                return dict(matches=[], unreadable=[dict(pid=999999, uid=0, comm='unknown')])
        with self.assertRaisesRegex(AssertionError, 'unrecognized unreadable process'):
            cleanup.ownership(Inspector, True)
        class Active:
            @staticmethod
            def ownership(_):
                return dict(matches=[dict(pid=1, matches=['fd:9'])], unreadable=[])
        with self.assertRaisesRegex(AssertionError, 'active owners'):
            cleanup.ownership(Active, True)


if __name__ == '__main__':
    result = unittest.TextTestRunner(verbosity=2).run(unittest.defaultTestLoader.loadTestsFromTestCase(Controls))
    output = dict(success=result.wasSuccessful(), tests_run=result.testsRun, failures=len(result.failures), errors=len(result.errors),
                  cleanup_script_sha256=hashlib.sha256((HERE/'cleanup.py').read_bytes()).hexdigest(),
                  actual_cleanup_executed=False, actual_cargo_executed=False,
                  scope='tiny disposable fixture admission and mocked ownership only')
    (HERE/'cleanup-controls.json').write_text(json.dumps(output, indent=2)+'\n')
    raise SystemExit(not result.wasSuccessful())
