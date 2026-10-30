"""Four tiny synthetic retention refusal controls; never inspect proof originals or create CAS."""
import hashlib
import io
import json
import os
import tarfile
import tempfile
import unittest
from pathlib import Path

import collect
import verify

HERE = Path(__file__).resolve().parent


def row(name='data', data=b'good', mode=0o644):
    return dict(path=name, source='/unused', size=len(data), mode=oct(mode),
                sha256=hashlib.sha256(data).hexdigest(), roles=['synthetic fixture only'])


def fake_archive(path, entries):
    with tarfile.open(path, 'w:gz') as archive:
        for name, data, mode in entries:
            info = tarfile.TarInfo(name)
            info.size, info.mode = len(data), mode
            archive.addfile(info, io.BytesIO(data))


class Controls(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory(dir=HERE, prefix='safe-controls-')
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)

    def test_corrupt_member_refused(self):
        path = self.root / 'fake.tar.gz'
        fake_archive(path, [('data', b'evil', 0o644)])
        with self.assertRaisesRegex(AssertionError, 'member hash mismatch'):
            verify.verify_payload(path, [row()])
        fake_archive(path, [('data', b'good', 0o755)])
        with self.assertRaisesRegex(AssertionError, 'member mode mismatch'):
            verify.verify_payload(path, [row()])
        fake_archive(path, [('data', b'longer', 0o644)])
        with self.assertRaisesRegex(AssertionError, 'member size mismatch'):
            verify.verify_payload(path, [row()])

    def test_missing_member_refused(self):
        path = self.root / 'fake.tar.gz'
        fake_archive(path, [('data', b'good', 0o644)])
        with self.assertRaisesRegex(AssertionError, 'missing or extra archive member'):
            verify.verify_payload(path, [row(), row('missing')])
        self.assertEqual(verify.verify_payload(path, [row()]), 1)

    def test_path_escape_refused(self):
        path = self.root / 'fake.tar.gz'
        fake_archive(path, [('../escape', b'good', 0o644)])
        with self.assertRaisesRegex(AssertionError, 'unsafe archive path'):
            verify.verify_payload(path, [row()])
        for name in ['../escape', '/absolute', 'a/../escape', 'a//empty', './dot', 'a\\escape']:
            value = row(name)
            value['suggested_member'] = value.pop('path')
            with self.subTest(name=name), self.assertRaisesRegex(AssertionError, 'unsafe member name'):
                collect.rows_from({'required_retention_members': [value]})
        self.assertFalse((self.root.parent / 'escape').exists())

    def test_modified_input_refused(self):
        path = self.root / 'source'
        path.write_bytes(b'good')
        os.chmod(path, 0o644)
        value = row()
        value['source'] = str(path)
        with collect.checked_stream(value)[0] as stream:
            self.assertEqual(stream.read(), b'good')
        path.write_bytes(b'evil')
        with self.assertRaisesRegex(AssertionError, 'modified input hash'):
            collect.checked_stream(value)
        path.write_bytes(b'good')
        os.chmod(path, 0o755)
        with self.assertRaisesRegex(AssertionError, 'modified input mode'):
            collect.checked_stream(value)


if __name__ == '__main__':
    suite = unittest.defaultTestLoader.loadTestsFromTestCase(Controls)
    result = unittest.TextTestRunner(verbosity=2).run(suite)
    inputs = {name: hashlib.sha256((HERE/name).read_bytes()).hexdigest()
              for name in ['collect.py', 'verify.py', 'test_controls.py']}
    receipt = dict(success=result.wasSuccessful(), tests_run=result.testsRun,
                   failures=len(result.failures), errors=len(result.errors), inputs_sha256=inputs,
                   scope='tiny synthetic files only; no private proof read/collection/CAS write',
                   proof_collection_executed=False, binaries_copied=False, cleanup_authorized=False)
    (HERE/'controls.json').write_text(json.dumps(receipt, indent=2)+'\n')
    raise SystemExit(not result.wasSuccessful())
