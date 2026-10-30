"""Small producer-provenance controls; no actual proof file, archive or cache mutation."""
import hashlib
import json
import tempfile
import unittest
from pathlib import Path
import classify

HERE = Path(__file__).resolve().parent


class Controls(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(dir=HERE, prefix='safe-provenance-')
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.unit = '0123456789abcdef'
        self.artifact = self.root / 'debug/deps/libunit-0123456789abcdef.rlib'
        self.artifact.parent.mkdir(parents=True)

    def fixture(self, fingerprint_hash=None, output=None):
        dep = self.artifact.with_suffix('.d')
        dep.write_text(str(output or self.artifact) + ': /source/unit.rs\n')
        directory = self.root / 'debug/.fingerprint' / ('unit-' + (fingerprint_hash or self.unit))
        directory.mkdir(parents=True)
        fingerprint = directory / 'lib-unit.json'
        fingerprint.write_text(json.dumps(dict(rustc=1, target=2, profile=3, deps=[], local=[])))
        return classify.producer_indices(self.root, [dep, fingerprint])

    def test_suffix_and_magic_without_producer_do_not_dispose(self):
        self.assertIsNone(classify.compiled_provenance(self.artifact, b'!<arch>\n', self.root, {}, {}))
        self.assertIsNone(classify.compiled_provenance(self.artifact, b'text', self.root, {}, {}))

    def test_explicit_cargo_output_fingerprint_and_format_are_required(self):
        dependencies, fingerprints = self.fixture()
        result = classify.compiled_provenance(self.artifact, b'!<arch>\n', self.root, dependencies, fingerprints)
        self.assertEqual(result['unit_hash'], self.unit)
        self.assertEqual(len(result['dep_info']), 1)
        self.assertEqual(len(result['cargo_fingerprint']), 1)
        self.assertIsNone(classify.compiled_provenance(self.artifact, b'generated source', self.root, dependencies, fingerprints))

    def test_wrong_fingerprint_unit_does_not_dispose(self):
        dependencies, fingerprints = self.fixture(fingerprint_hash='fedcba9876543210')
        self.assertIsNone(classify.compiled_provenance(self.artifact, b'\x7fELF', self.root, dependencies, fingerprints))

    def test_outside_output_and_sibling_prefix_do_not_dispose(self):
        outside = self.root.parent / (self.root.name + '-sibling') / self.artifact.name
        dependencies, fingerprints = self.fixture(output=outside)
        self.assertFalse(classify.within(outside, self.root))
        self.assertIsNone(classify.compiled_provenance(self.artifact, b'rust\x00', self.root, dependencies, fingerprints))


if __name__ == '__main__':
    result = unittest.TextTestRunner(verbosity=2).run(unittest.defaultTestLoader.loadTestsFromTestCase(Controls))
    receipt = dict(success=result.wasSuccessful(), tests_run=result.testsRun, errors=len(result.errors), failures=len(result.failures),
                   input_sha256={name:hashlib.sha256((HERE/name).read_bytes()).hexdigest() for name in ['classify.py','supplemental.py','test_controls.py']},
                   scope='small synthetic producer metadata only; no actual archive/CAS/cleanup/build/Flow/Git')
    (HERE/'controls.json').write_text(json.dumps(receipt,indent=2)+'\n')
    raise SystemExit(not result.wasSuccessful())
