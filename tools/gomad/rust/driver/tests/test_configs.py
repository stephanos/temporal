from pathlib import Path
import sys
import unittest

HERE = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(HERE))


class ConfigurationContract(unittest.TestCase):
    def test_fixture_and_original_workflow_specs_use_unchanged_bounds(self):
        import configure
        from adapters import CELLS, validate_cell_spec
        for label, wanted in CELLS.items():
            if wanted['workload'] not in ('fixture', 'onebox'):
                continue
            with self.subTest(label=label):
                spec = configure.cell_spec(label, '/evidence', '/source.sha256', 'sha256:'+'1'*64)
                validate_cell_spec(spec, wanted)
                self.assertFalse(spec['Manifest']['Provisional'])
                self.assertEqual((spec['Manifest']['Seed'], spec['Manifest']['Repeat']), (7, 2))
                self.assertEqual(spec['Campaign']['ExecutionTimeout'],
                                 900_000_000_000 if wanted['workload'] == 'onebox' else 60_000_000_000)
                if wanted['workload'] == 'onebox':
                    self.assertEqual(spec['Campaign']['Target']['Source'], './tests')
                    self.assertEqual(len(spec['Campaign']['Target']['Args']), 6)

    def test_specialized_durable_specs_cannot_be_filled_with_fixtures(self):
        import configure
        import driver
        for label in ('workload-sqlite-cooperative-success', 'workload-multi-instance-cooperative-success'):
            with self.assertRaises(driver.Unexecuted):
                configure.cell_spec(label, '/evidence', '/manifest', 'sha256:'+'1'*64)

    def test_generated_manifest_uses_qualification_digest_grammar(self):
        import configure
        for digest in ('1'*64, 'sha256:'+'1'*64):
            with self.subTest(digest=digest):
                spec = configure.cell_spec('workload-fixture-cooperative-success', '/evidence', '/manifest', digest)
                self.assertEqual(spec['Manifest']['IntegratedSourceSHA256'], 'sha256:'+'1'*64)


if __name__ == '__main__':
    unittest.main()
