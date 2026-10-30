import errno
import json
import os
from pathlib import Path
import subprocess
import tempfile
import threading
import time
import unittest
from unittest.mock import patch
from census import apparent_bytes


class CensusTests(unittest.TestCase):
    def test_stable_apparent_bytes_match_du_with_hardlinks_and_no_symlink_following(self):
        with tempfile.TemporaryDirectory(dir='/tmp') as name:
            base = Path(name)
            root = base/'owned'; root.mkdir()
            outside = base/'outside'; outside.mkdir()
            (outside/'unowned').write_bytes(b'x'*1024*1024)
            (root/'subdir').mkdir()
            (root/'payload').write_bytes(b'x'*4097)
            os.link(root/'payload', root/'subdir/hardlink')
            (root/'escape').symlink_to(outside, target_is_directory=True)
            events = []
            expected = int(subprocess.check_output(['du','-sb','--',str(root)]).split()[0])
            self.assertEqual(apparent_bytes(root, lambda *args: events.append(args)), expected)
            self.assertEqual(events, [])
            self.assertEqual(apparent_bytes(root/'escape', lambda *args: events.append(args)),
                             os.lstat(root/'escape').st_size)
            self.assertEqual(events, [])

    def test_hardlink_dedup_is_per_root(self):
        with tempfile.TemporaryDirectory(dir='/tmp') as name:
            base = Path(name)
            first = base/'first'; first.mkdir()
            second = base/'second'; second.mkdir()
            (first/'data').write_bytes(b'x'*100)
            os.link(first/'data', second/'data')
            observed = sum(apparent_bytes(p, lambda *args: None) for p in [first, second])
            expected = sum(int(subprocess.check_output(['du','-sb','--',str(p)]).split()[0])
                           for p in [first,second])
            self.assertEqual(observed, expected)

    def test_concurrent_temporary_directory_creation_and_cleanup(self):
        with tempfile.TemporaryDirectory(dir='/tmp') as name:
            root = Path(name)
            temporary = root/'tmp'; temporary.mkdir()
            stop = threading.Event(); ready = threading.Event(); errors = []; events = []
            def writer():
                try:
                    while not stop.is_set():
                        with tempfile.TemporaryDirectory(prefix='TestPublicationNeverOverwritesinterrupted', dir=temporary) as directory:
                            for index in range(12):
                                Path(directory, str(index)).write_bytes(b'x'*128)
                            ready.set()
                except BaseException as error:
                    errors.append(error)
            worker = threading.Thread(target=writer); worker.start()
            self.assertTrue(ready.wait(5))
            scans = 0; start = time.monotonic()
            try:
                while scans < 500 or not events:
                    self.assertGreaterEqual(apparent_bytes(root, lambda *args: events.append(args)), 0)
                    scans += 1
                    self.assertLess(time.monotonic()-start, 15, 'concurrent census never completed')
            finally:
                stop.set(); worker.join(5)
            self.assertFalse(worker.is_alive())
            self.assertEqual(errors, [])
            self.assertGreater(len(events), 0)
            self.assertTrue(all(isinstance(event[3], FileNotFoundError) for event in events))
            self.assertTrue(all(event[2].startswith(os.fsencode(temporary)+b'/') for event in events))
            Path(__file__).with_name('concurrency-result.json').write_text(json.dumps({
                'scans':scans,'transients':len(events),'seconds':time.monotonic()-start,
                'sample':[{'operation':e[0],'root_path_hex':e[1].hex(),'path_hex':e[2].hex(),
                           'errno':e[3].errno} for e in events[:20]]}, indent=2)+'\n')

    def test_missing_raw_byte_path_is_logged_without_loss(self):
        with tempfile.TemporaryDirectory(dir='/tmp') as name:
            missing = os.fsencode(name)+b'/missing-\xff'
            events = []
            self.assertEqual(apparent_bytes(missing, lambda *args: events.append(args)), 0)
            self.assertEqual(events[0][2], missing)
            self.assertEqual(bytes.fromhex(events[0][2].hex()), missing)

    def test_permission_and_unexpected_io_errors_propagate(self):
        with tempfile.TemporaryDirectory(dir='/tmp') as name:
            for error in [PermissionError(errno.EACCES,'blocked'), OSError(errno.EIO,'io failure')]:
                with patch('census.os.scandir', side_effect=error):
                    with self.assertRaises(type(error)):
                        apparent_bytes(name, lambda *args: self.fail('unexpected error suppressed'))

    def test_missing_logger_failure_is_not_suppressed(self):
        def failed_logger(*_):
            raise FileNotFoundError(errno.ENOENT, 'logger unavailable')
        with self.assertRaisesRegex(FileNotFoundError, 'logger unavailable'):
            apparent_bytes('/tmp/umpire-no-such-census-root', failed_logger)


if __name__ == '__main__':
    unittest.main(verbosity=2)
