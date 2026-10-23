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

    def test_old_stress_cycle_refused_even_when_its_file_is_sealed(self):
        with tempfile.TemporaryDirectory() as temporary:
            path = Path(temporary) / 'cycle.json'
            config = dict(repo='/repo', root='/new-attempt', source_manifest='/manifest', source_sha256='1'*64,
                sent_epoch=2, deadline_s=3600, identity=dict(host='2'*64, helper='3'*64), binary_sha256='4'*64,
                stress={}, stress_cycle=str(path), files={'/static': '5'*64}, public_verbs=dict(bindings='/bindings'))
            fields = ('repo', 'root', 'source_manifest', 'source_sha256', 'sent_epoch', 'deadline_s', 'identity', 'binary_sha256', 'stress')
            plan = {key: config[key] for key in fields}
            plan.update(schema='gomad.stress-cycle/v1', files={'/static': '5'*64})
            path.write_text(json.dumps(plan))
            config['files'][str(path)] = sha(path)
            reader.stress_cycle(config, {'stress-cycle': path})
            with self.assertRaisesRegex(Refusal, 'another stress cycle'):
                reader.stress_cycle(config, {'stress-cycle': path.with_name('borrowed.json')})
            plan['files'] = {}
            path.write_text(json.dumps(plan))
            config['files'][str(path)] = sha(path)
            with self.assertRaisesRegex(Refusal, 'inputs differ'):
                reader.stress_cycle(config, {'stress-cycle': path})
            plan['files'] = {'/static': '5'*64}
            plan['root'] = '/old-attempt'
            path.write_text(json.dumps(plan))
            config['files'][str(path)] = sha(path)
            with self.assertRaisesRegex(Refusal, 'old/foreign'):
                reader.stress_cycle(config, {'stress-cycle': path})

    def test_arbitrary_public_binding_fields_refused(self):
        with self.assertRaisesRegex(Refusal, 'only the sealed binding'):
            reader.binding(dict(public_verbs=dict(bindings='/any', argv=['true'])))


if __name__ == '__main__':
    unittest.main()
