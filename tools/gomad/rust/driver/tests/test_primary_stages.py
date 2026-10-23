"""Fixed recipe negatives only; no guest or original qualification is executed."""
import copy
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import primary_stages as primary
from driver import Refusal


class PrimaryRecipes(unittest.TestCase):
    def config(self, root, template):
        return dict(root=str(root), python='/sealed/python', _config_path='/sealed/driver.json',
            binary='/sealed/release', repo='/sealed/repo', source_manifest='/sealed/source.sha256',
            live_corpus=dict(families={
                'cluster': dict(role='host-fixture', argv=['/sealed/python', 'cluster', '--input', str(template)]),
                'broker': dict(role='host-fixture', argv=['/sealed/python', 'broker', '--observation', '/sealed/observation'])}))

    def test_grouped_has_ten_distinct_actual_invocations(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            template = root / 'template.json'
            template.write_text(json.dumps(dict(modules={'target/module.wasm': '/sealed/module.wasm'})))
            config = self.config(root, template)
            with patch.object(primary.live, 'sealed_input'), patch.object(primary.live, 'role', return_value=dict(binary='/sealed/fixture')):
                rows = primary.recipes(config, 'busy-grouped', root / 'stages/busy-grouped')
            self.assertEqual(len(rows), 10)
            self.assertEqual(len({tuple(row['argv']) for row in rows}), 10)
            self.assertTrue(all(row['argv'][-2:] == ['--cell', 'grouped'] for row in rows))

    def test_broker_stages_select_distinct_original_roots(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            config = self.config(root, root / 'unused')
            with patch.object(primary.live, 'sealed_input'), patch.object(primary.live, 'role', return_value=dict(binary='/sealed/fixture')):
                for label, roots in [('busy-boundaries', '33,34'), ('network-streams', '36')]:
                    row = primary.recipes(config, label, root / 'stages' / label)
                    self.assertEqual(row[0]['argv'][-2:], ['--roots', roots])

    def test_artifact_and_operator_use_global_retained_owner_once(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            config = self.config(root, root / 'unused')
            with patch.object(primary.live, 'sealed_input'):
                for label, section in [('artifact', 'retained'), ('operator', 'operator')]:
                    rows = primary.recipes(config, label, root / 'stages' / label)
                    self.assertEqual(len(rows), 1)
                    self.assertEqual(rows[0]['argv'][-4:], ['--section', section, '--root', str(root)])
                    self.assertNotIn('--read-only', rows[0]['argv'])

    def test_conformance_reader_rejects_wrong_features_profile_and_missing_binary(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            binary = root / 'primary/target/release/gomad-wasmhost'
            binary.parent.mkdir(parents=True)
            binary.write_bytes(b'synthetic artifact')
            artifact = dict(reason='compiler-artifact', executable=str(binary),
                target=dict(name='gomad-wasmhost', kind=['bin'], src_path='/repo/tools/gomad/rust/wasmhost/src/main.rs'),
                features=['conformance', 'reference-helper'], profile=dict(opt_level='3', debug_assertions=False, test=False))
            recipe = [dict(argv=['/cargo', '--features', 'conformance'])]
            output = root / 'primary-final.stdout'
            with patch.object(primary, 'recipes', return_value=recipe):
                for mutation in ('feature', 'profile', 'source', 'duplicate', 'unfinished', 'missing-binary'):
                    changed = copy.deepcopy(artifact)
                    if mutation == 'feature':
                        changed['features'] = ['reference-helper']
                    if mutation == 'profile':
                        changed['profile']['debug_assertions'] = True
                    if mutation == 'source':
                        changed['target']['src_path'] = '/other/main.rs'
                    rows = [changed, dict(reason='build-finished', success=mutation != 'unfinished')]
                    if mutation == 'duplicate':
                        rows.insert(0, changed)
                    if mutation == 'missing-binary':
                        binary.unlink()
                    output.write_text('\n'.join(json.dumps(row) for row in rows))
                    with self.subTest(mutation=mutation), self.assertRaises(Refusal):
                        primary.conformance(dict(repo='/repo'), root)

    def test_foreign_stage_root_refused(self):
        with self.assertRaisesRegex(Refusal, 'foreign primary'):
            primary.recipes(dict(root='/attempt'), 'storage', Path('/other'))

    def test_unimplemented_broad_roots_refuse(self):
        for label in primary.MISSING:
            with self.subTest(label=label), self.assertRaisesRegex(Refusal, 'unknown primary stage'):
                primary.recipes({}, label, Path('/attempt'))

    def test_caller_command_not_a_primary_control(self):
        import adapters
        config = dict(stages={'storage': dict(adapter='primary', argv=['true'])})
        with self.assertRaisesRegex(Refusal, 'caller controls'):
            adapters.execute_primary(config, Path('/attempt'), 'storage')


if __name__ == '__main__':
    unittest.main()
