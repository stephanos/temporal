"""Fixed recipe negatives only; no guest or original qualification is executed."""
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
