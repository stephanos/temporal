import hashlib
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import source_custody
import workloads


def sha(body):
    return hashlib.sha256(body).hexdigest()


class SourceCustodyTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.base = Path(self.temporary.name)
        self.repo = self.base/'repo'
        self.root = self.base/'producer'
        self.repo.mkdir()
        self.root.mkdir(mode=0o700)
        self.originals = {'go.mod': b'module go.temporal.io/server\n\ngo 1.27.1\n',
                          'go.sum': b'original sums\n', 'common/a.go': b'package common\n',
                          'common/build/build.go': b'package build\n',
                          'internal/evidence/fixture.go': b'package evidence\n',
                          'internal/testdata/target': b'tracked fixture\n',
                          str(workloads.WASM/'testdata/temporal_storage/recovery_test.go'): b'package temporal\n'}
        self.files = {}
        for relative, body in self.originals.items():
            path = self.repo/relative
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(body)
            self.files[str(path)] = sha(body)
        self.manifest = self.base/'source.sha256'
        self.manifest.write_text(''.join(value+'  '+key+'\n' for key, value in sorted(self.files.items())))
        self.files[str(self.manifest)] = sha(self.manifest.read_bytes())

    def stage(self):
        return source_custody.stage(self.root, self.repo, self.files, 'recovery', self.manifest)

    def verify(self, source, working):
        return source_custody.verify(self.root, self.repo, self.files, 'recovery', source, working,
                                     frozen_source_manifest=self.manifest)

    def test_snapshot_preserves_frozen_source_and_normal_module_and_reuses(self):
        source, working, originals = self.stage()
        self.assertFalse((self.repo/'.tmp').exists())
        self.assertEqual((Path(working)/'go.mod').read_bytes(), self.originals['go.mod'])
        for name in ('common/build/build.go', 'internal/evidence/fixture.go', 'internal/testdata/target'):
            self.assertEqual((Path(working)/name).read_bytes(), self.originals[name])
        self.assertEqual((Path(working)/source/'recovery_test.go').read_bytes(), b'package temporal\n')
        pins = self.verify(source, working)
        record = json.loads((self.root/'source-custody.json').read_bytes())
        self.assertEqual(record['copied_files'], len(self.originals)+1)
        self.assertEqual(record['copied_bytes'], sum(map(len, self.originals.values()))+len(b'package temporal\n'))
        self.assertEqual(len(pins), len(self.originals)+1)
        self.assertEqual(self.stage(), (source, working, originals))
        for name in pins:
            self.assertEqual(Path(name).stat().st_mode & 0o777, 0o600)

    def test_mutation_missing_extra_symlink_and_provenance_refused(self):
        for kind in ('source', 'derived', 'manifest', 'missing', 'extra', 'extra-directory', 'symlink', 'hardlink', 'receipt', 'foreign', 'missing-original'):
            with self.subTest(kind=kind):
                self.setUp()
                source, working, _ = self.stage()
                target = Path(working)/'common/a.go'
                if kind == 'source': (self.repo/'common/a.go').write_bytes(b'changed')
                elif kind == 'derived': target.write_bytes(b'changed')
                elif kind == 'manifest': self.manifest.write_bytes(b'changed')
                elif kind == 'missing': target.unlink()
                elif kind == 'extra': (Path(working)/'extra').write_bytes(b'extra')
                elif kind == 'extra-directory': (Path(working)/'extra').mkdir()
                elif kind == 'missing-original': (self.repo/'common/a.go').unlink()
                elif kind == 'symlink':
                    target.unlink()
                    target.symlink_to(self.repo/'common/a.go')
                elif kind == 'hardlink':
                    target.unlink()
                    target.hardlink_to(self.repo/'common/a.go')
                elif kind == 'receipt':
                    receipt = self.root/'source-custody.json'
                    value = json.loads(receipt.read_bytes())
                    value['copied_bytes'] += 1
                    receipt.write_text(json.dumps(value))
                else: working = self.base/'foreign'
                with self.assertRaises((ValueError, OSError)):
                    self.verify(source, working)

    def test_forged_manifest_choice_and_foreign_reuse_refused(self):
        source, working, _ = self.stage()
        other = self.base/'other.sha256'
        other.write_bytes(self.manifest.read_bytes())
        self.files[str(other)] = sha(other.read_bytes())
        receipt = self.root/'source-custody.json'
        value = json.loads(receipt.read_bytes())
        value['source_manifest'] = str(other)
        receipt.write_bytes(workloads.canonical(value)+b'\n')
        with self.assertRaises(ValueError): self.verify(source, working)
        foreign = self.base/'foreign'
        foreign.mkdir()
        (foreign/'source-custody.json').write_bytes(receipt.read_bytes())
        (foreign/'source-custody.json').chmod(0o600)
        with self.assertRaises(ValueError):
            source_custody.verify(foreign, self.repo, self.files, 'recovery', source, working,
                                  frozen_source_manifest=self.manifest)

    def test_derived_owner_changes_are_refused(self):
        source, working, _ = self.stage()
        with patch.object(source_custody.os, 'geteuid', return_value=source_custody.os.geteuid()+1):
            with self.assertRaises(ValueError): self.verify(source, working)

    def test_root_must_remain_a_private_directory(self):
        source, working, _ = self.stage()
        self.root.chmod(0o777)
        with self.assertRaisesRegex(ValueError, 'private directory'):
            self.verify(source, working)
        self.root.chmod(0o700)
        ordinary_file = self.base/'not-a-directory'
        ordinary_file.write_bytes(b'file')
        with self.assertRaisesRegex(ValueError, 'private directory'):
            source_custody.recipe(ordinary_file, self.repo, self.files, 'recovery', self.manifest)

    def test_unknown_directory_is_refused_before_traversal(self):
        source, working, _ = self.stage()
        unexpected = Path(working)/'unexpected'
        unexpected.mkdir(mode=0o700)
        (unexpected/'nested').mkdir(mode=0o700)
        original = source_custody.os.scandir
        scanned = []
        def observe(path):
            scanned.append(Path(path))
            self.assertNotEqual(Path(path), unexpected)
            return original(path)
        with patch.object(source_custody.os, 'scandir', side_effect=observe):
            with self.assertRaisesRegex(ValueError, 'unexpected derived directory'):
                self.verify(source, working)
        self.assertNotIn(unexpected, scanned)

    def test_descriptor_read_refuses_growth_and_replacement(self):
        for replacement in (False, True):
            with self.subTest(replacement=replacement):
                source = self.base/'growing'
                source.write_bytes(b'initial')
                original = source_custody.os.fstat
                calls = []
                def observe(descriptor):
                    info = original(descriptor)
                    if not calls:
                        if replacement:
                            source.unlink()
                            source.write_bytes(b'initial')
                        else:
                            with source.open('ab') as stream:
                                stream.write(b'x'*64)
                    calls.append(info)
                    return info
                with patch.object(source_custody, 'MAX_BYTES', 16):
                    with patch.object(source_custody.os, 'fstat', side_effect=observe):
                        with self.assertRaises(ValueError): source_custody.regular(source)

    def test_collision_and_bounds_refused_before_copy(self):
        (self.root/'source-module').mkdir()
        with self.assertRaises(ValueError): self.stage()
        (self.root/'source-module').rmdir()
        with patch.object(source_custody, 'MAX_BYTES', 1), self.assertRaises(ValueError): self.stage()
        self.assertFalse((self.root/'source-module').exists())
        with patch.object(source_custody, 'MAX_FILES', 1), self.assertRaises(ValueError): self.stage()

    def test_retry_stages_every_original_and_preserves_fixed_key(self):
        names = ['temporal_retry_test.go', 'extra.go']
        for name in names:
            original = self.repo/workloads.WASM/'testdata/network_streams'/name
            original.parent.mkdir(parents=True, exist_ok=True)
            original.write_bytes(('package retry // '+name+'\n').encode())
            self.files[str(original)] = sha(original.read_bytes())
        originals = {name: value for name, value in self.files.items() if name != str(self.manifest)}
        self.manifest.write_text(''.join(value+'  '+name+'\n' for name, value in sorted(originals.items())))
        self.files[str(self.manifest)] = sha(self.manifest.read_bytes())
        source, working, pins = source_custody.stage(self.root, self.repo, self.files, 'retry', self.manifest)
        self.assertEqual(sorted(path.name for path in (Path(working)/source).iterdir()), sorted(names))
        self.assertEqual(len(pins), 2)
        source_custody.verify(self.root, self.repo, self.files, 'retry', source, working,
                              frozen_source_manifest=self.manifest)

    def test_unsealed_original_and_symlink_original_refused(self):
        self.files.pop(str(self.repo/'common/a.go'))
        with self.assertRaises(ValueError): self.stage()
        self.files[str(self.repo/'common/a.go')] = sha(b'package common\n')
        original = self.repo/'common/a.go'
        original.unlink()
        original.symlink_to(self.repo/'go.mod')
        with self.assertRaises(ValueError): self.stage()


if __name__ == '__main__':
    unittest.main()
