"""Negative reader fixtures; these do not execute qualification lanes."""
import json
from pathlib import Path
import sys
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import final_public_verbs as reader
from driver import Refusal, sha


class PublicVerbReaderTests(unittest.TestCase):
    def test_foreign_candidate_aggregate_refused(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            bindings = root / 'bindings.json'
            bindings.write_text('{"trees":{}}')
            config = dict(repo=temporary, public_verbs=dict(bindings=str(bindings)))
            (root / 'public-verbs').mkdir()
            for suffix in ('command', 'process'):
                (root / ('public-verbs.' + suffix + '.json')).write_text('{}')
            (root / 'public-verbs/all-verb-result.json').write_text(json.dumps(dict(
                schema='gomad.all-verb-parity/v1', status='passed', bindings_sha256='0' * 64)))
            with patch.object(reader.live, 'completed_command'), self.assertRaisesRegex(Refusal, 'foreign/incomplete'):
                reader.authenticate(config, root, [], {}, SimpleNamespace())

    def test_missing_raw_goldens_refused(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            bindings = root / 'bindings.json'
            bindings.write_text('{"trees":{}}')
            config = dict(public_verbs=dict(bindings=str(bindings)), binary=str(bindings), binary_sha256=sha(bindings))
            lane = SimpleNamespace(CORPUS=root, load_bindings=lambda *_: ({'release': bindings}, {}))
            with patch.object(reader, 'orchestrator', return_value=lane), patch.object(reader.live, 'sealed_input'):
                with self.assertRaisesRegex(Refusal, 'missing raw golden corpus'):
                    reader.binding(config)

    def test_alternate_release_refused(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            bindings = root / 'bindings.json'
            bindings.write_text('{"trees":{}}')
            config = dict(public_verbs=dict(bindings=str(bindings)), binary='/other/release', binary_sha256=sha(bindings))
            lane = SimpleNamespace(load_bindings=lambda *_: ({'release': bindings}, {}))
            with patch.object(reader, 'orchestrator', return_value=lane), patch.object(reader.live, 'sealed_input'):
                with self.assertRaisesRegex(Refusal, 'differs from frozen candidate'):
                    reader.binding(config)

    def test_missing_and_duplicate_fixed_commands_refused(self):
        for names in ([], ['foundation'], ['foundation', 'foundation']):
            with self.subTest(names=names), tempfile.TemporaryDirectory() as temporary:
                work = Path(temporary)
                (work / 'commands.json').write_text(json.dumps([dict(name=name) for name in names]))
                with self.assertRaisesRegex(Refusal, 'missing/duplicate/reordered'):
                    reader.authenticate_lanes({}, work, SimpleNamespace())

    def test_arbitrary_public_binding_fields_refused(self):
        with self.assertRaisesRegex(Refusal, 'only the sealed binding'):
            reader.binding(dict(public_verbs=dict(bindings='/any', argv=['true'])))


if __name__ == '__main__':
    unittest.main()
