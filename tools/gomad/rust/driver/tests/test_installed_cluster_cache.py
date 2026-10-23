"""Dependency cache custody counterexamples; no compiler or guest execution."""
import base64
import hashlib
from pathlib import Path
import sys
import tempfile
import unittest
import zipfile

sys.dont_write_bytecode = True
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import installed_cluster_cache as cache
from driver import Refusal


class CacheInputs(unittest.TestCase):
    def fixture(self, root, module=cache.MODULE, version=cache.VERSION):
        prefix = module + '@' + version
        payloads = {'go.mod': ('module ' + module + '\n').encode(), 'unix/file.go': b'package unix\n'}
        directory = root / prefix
        metadata = root / 'cache/download' / module / '@v'
        metadata.mkdir(parents=True)
        archive = metadata / (version + '.zip')
        with zipfile.ZipFile(archive, 'w') as output:
            for name, raw in payloads.items():
                file = directory / name
                file.parent.mkdir(parents=True, exist_ok=True)
                file.write_bytes(raw)
                output.writestr(prefix + '/' + name, raw)
        text = ''.join(hashlib.sha256(raw).hexdigest() + '  ' + prefix + '/' + name + '\n'
                       for name, raw in sorted(payloads.items()))
        digest = 'h1:' + base64.b64encode(hashlib.sha256(text.encode()).digest()).decode()
        (metadata / (version + '.ziphash')).write_text(digest)
        (metadata / (version + '.mod')).write_bytes(payloads['go.mod'])
        (metadata / (version + '.info')).write_text('{"Version":"' + version + '"}')
        (metadata / (version + '.lock')).write_bytes(b'')
        sums = root / 'go.sum'
        mod_text = hashlib.sha256(payloads['go.mod']).hexdigest() + '  go.mod\n'
        mod_digest = 'h1:' + base64.b64encode(hashlib.sha256(mod_text.encode()).digest()).decode()
        sums.write_text(module + ' ' + version + ' ' + digest + '\n'
            + module + ' ' + version + '/go.mod ' + mod_digest + '\n')
        return directory, metadata, sums

    def test_go_mod_checksum_is_authenticated_separately_from_archive_checksum(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            _, _, sums = self.fixture(root)
            sums.write_text(sums.read_text().splitlines()[0] + '\n')
            with self.assertRaises(Refusal):
                cache.verify(root, sums)

    def test_exact_archive_source_and_go_sum_snapshot(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            _, _, sums = self.fixture(root / 'source')
            expected = cache.verify(root / 'source', sums)
            cache.snapshot(root / 'source', root / 'private', sums)
            self.assertEqual(cache.verify(root / 'private', sums), expected)

    def test_changed_extracted_bytes_extra_source_checksum_or_missing_meta_refuse(self):
        for mutation in ('bytes', 'extra', 'sum', 'meta', 'alias'):
            with self.subTest(mutation=mutation), tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary)
                directory, metadata, sums = self.fixture(root)
                if mutation == 'bytes': (directory / 'unix/file.go').write_bytes(b'changed')
                elif mutation == 'extra': (directory / 'extra.go').write_bytes(b'extra')
                elif mutation == 'sum': sums.write_text('')
                elif mutation == 'meta': (metadata / (cache.VERSION + '.mod')).unlink()
                else:
                    (directory / 'alias').symlink_to(directory / 'go.mod')
                with self.assertRaises((Refusal, OSError)):
                    cache.verify(root, sums)


if __name__ == '__main__':
    unittest.main()
