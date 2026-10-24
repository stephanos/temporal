"""Dry-run role admission fixtures supply no compiler or qualification execution credit."""
import copy
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
sys.path.insert(0, str(Path(__file__).resolve().parent))
import dry_run
import main_cli_build
import test_dry_run_readiness as fixture
from driver import Refusal, read, sha


class DryRun(unittest.TestCase):
    def tearDown(self):
        dry_run.enable({})

    def fixture(self, root):
        config, source = fixture.DryRunReadiness().fixture(root)
        path, binary = fixture.record(root, config, 'release')
        record = read(path)
        config.update(identity=dict(host=record['host_sha256'], helper=record['helper_sha256']),
                      binary=str(binary), binary_sha256=sha(binary))
        config['files'].update({str(p):sha(p) for p in path.parent.rglob('*') if p.is_file()})
        return config, source, path, binary

    def test_records_refuse_outside_a_dry_run_configuration(self):
        with tempfile.TemporaryDirectory() as raw:
            config, source, path, _ = self.fixture(Path(raw))
            dry_run.enable(dict(dry_run=False))
            with self.assertRaises(Refusal):
                main_cli_build.read_build(path)
            dry_run.enable(config)
            with source:
                self.assertEqual(main_cli_build.read_build(path, expected_role='release')['role'], 'release')

    def test_dry_run_binds_records_to_the_candidate_identities_and_bytes(self):
        with tempfile.TemporaryDirectory() as raw:
            config, source, path, binary = self.fixture(Path(raw))
            dry_run.enable(config)
            with source:
                receipt = main_cli_build.read_build(path, config, expected_role='release')
                self.assertEqual((receipt['source_sha256'], receipt['head']), (config['source_sha256'], config['head']))
                self.assertEqual(receipt['inputs'][str(binary)], sha(binary))
                self.assertEqual(receipt['build_head'], 'a' * 40)
                for mutation in ('role', 'helper', 'host', 'bytes', 'pins'):
                    with self.subTest(mutation=mutation):
                        changed = copy.deepcopy(config)
                        if mutation == 'role':
                            with self.assertRaises(Refusal):
                                main_cli_build.read_build(path, changed, expected_role='host-fixture')
                            continue
                        if mutation == 'bytes':
                            original = binary.read_bytes()
                            binary.write_bytes(original + b'!')
                            with self.assertRaises(Refusal): main_cli_build.read_build(path, changed)
                            binary.write_bytes(original)
                            continue
                        if mutation == 'pins': changed['files'].pop(str(binary))
                        else: changed['identity'][mutation] = 'x' * 64
                        with self.assertRaises(Refusal): main_cli_build.read_build(path, changed)


if __name__ == '__main__':
    unittest.main()
