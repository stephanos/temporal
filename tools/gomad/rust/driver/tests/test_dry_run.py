"""Dry-run build records stay outside qualifying configurations; no compiler or CLI is executed."""
import json
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import dry_run
import main_cli_build
from driver import Refusal, sha


def record(base, role='release', helper='h' * 64, host='o' * 64):
    binary = base / role / 'gomad'
    binary.parent.mkdir(parents=True)
    binary.write_bytes(b'dry-run executable ' + role.encode())
    path = base / role / 'dry-build.json'
    path.write_text(json.dumps(dict(schema=dry_run.SCHEMA, role=role, binary=str(binary), binary_sha256=sha(binary),
        build_head='a' * 40, build_dirty=False, features=main_cli_build.ROLES[role],
        helper_features=main_cli_build.helper_features(role), host_sha256=host, helper_sha256=helper,
        runtime_implementation_sha256='r' * 64, argv=['cargo'], scope='test')))
    return path, binary


class DryRun(unittest.TestCase):
    def tearDown(self):
        dry_run.enable({})

    def test_records_refuse_outside_a_dry_run_configuration(self):
        with tempfile.TemporaryDirectory() as raw:
            path, _ = record(Path(raw))
            dry_run.enable(dict(dry_run=False))
            with self.assertRaises(Refusal):
                main_cli_build.read_build(path)
            dry_run.enable(dict(dry_run=True))
            self.assertEqual(main_cli_build.read_build(path, expected_role='release')['role'], 'release')

    def test_dry_run_binds_records_to_the_candidate_identities_and_bytes(self):
        with tempfile.TemporaryDirectory() as raw:
            path, binary = record(Path(raw))
            dry_run.enable(dict(dry_run=True))
            frozen = dict(source_sha256='s' * 64, head='b' * 40, symlinks={}, identity=dict(host='o' * 64, helper='h' * 64))
            receipt = main_cli_build.read_build(path, frozen, expected_role='release')
            self.assertEqual((receipt['source_sha256'], receipt['head'], receipt['inputs']), ('s' * 64, 'b' * 40, {}))
            for mutation in ('role', 'helper', 'host', 'bytes'):
                with self.subTest(mutation=mutation):
                    changed = dict(frozen, identity=dict(frozen['identity']))
                    if mutation == 'role':
                        with self.assertRaises(Refusal):
                            main_cli_build.read_build(path, changed, expected_role='host-fixture')
                        continue
                    if mutation == 'bytes':
                        original = binary.read_bytes()
                        binary.write_bytes(original + b'!')
                        with self.assertRaises(Refusal):
                            main_cli_build.read_build(path, changed)
                        binary.write_bytes(original)
                        continue
                    changed['identity'][mutation] = 'x' * 64
                    with self.assertRaises(Refusal):
                        main_cli_build.read_build(path, changed)


if __name__ == '__main__':
    unittest.main()
