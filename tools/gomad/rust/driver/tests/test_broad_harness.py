"""Pure catalog and synthetic artifact mutations; no native evidence is produced."""
import copy
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import broad_harness as broad
from driver import Refusal


class BroadHarness(unittest.TestCase):
    def test_stages_and_roles_are_closed(self):
        for unknown in ('cluster-provider', 'retained-cluster', 'PASS', None):
            with self.assertRaises(Refusal):
                broad.catalog(unknown)
        for unknown in ('helper-lib', 'host-lib', 'PASS', None):
            with self.assertRaises(Refusal):
                broad.target(unknown)
        self.assertEqual({stage.value for stage in broad.Stage},
                         {'cluster-provider', 'cluster-functional', 'retained-cluster'})
        self.assertEqual(len(broad.TARGETS), 5)

    def test_fixed_process_catalog_and_argv(self):
        config = {'root': '/owned/cycle', 'argv': ['caller-command'], 'tests': ['caller-selector']}
        for stage in broad.Stage:
            rows = broad.catalog(stage)
            self.assertEqual(len(rows), len({row.name for row in rows}))
            for row in rows:
                argv = broad.test_argv(config, stage, row.name)
                self.assertTrue(argv[0].startswith('/owned/cycle/broad-harness/admitted/'))
                self.assertNotIn('caller-selector', argv)
                self.assertNotIn('caller-command', argv)
                if not broad.target(row.role).harness:
                    self.assertEqual(len(argv), 1)
                else:
                    self.assertEqual(argv[-2:], ['--nocapture', '--test-threads=1'])
            guest = [row for row in rows if row.ignored]
            self.assertEqual(bool(guest), stage is not broad.Stage.FUNCTIONAL)
            if guest:
                self.assertEqual(guest[0].selector, 'replay::cluster::broad_guest_tests')
                self.assertIn('--ignored', broad.test_argv(config, stage, guest[0].name))
            with self.assertRaises(Refusal):
                broad.test_argv(config, stage, '--skip=anything')

    def test_declared_build_features_and_commands(self):
        class Fixed:
            @staticmethod
            def plan(config):
                return None, {'tools': {'cargo': '/sealed/cargo'}}
        with patch.object(broad, 'fixed', return_value=Fixed):
            for role in broad.Role:
                argv = broad.cargo_command({'repo': '/source'}, role)
                for option in ('--release', '--locked', '--offline', '--no-default-features', '--no-run', '--message-format=json'):
                    self.assertIn(option, argv)
                self.assertEqual('--features' in argv, role in (broad.Role.HOST, broad.Role.HOST_ALLOCATION))
                if '--features' in argv:
                    self.assertEqual(argv[argv.index('--features') + 1], 'determinism-fixture')
                self.assertEqual('--lib' in argv, role in (broad.Role.HELPER, broad.Role.HOST))

    def artifact_fixture(self, directory, role):
        root = directory / 'cycle/broad-harness'
        spec = broad.target(role)
        crate = directory / 'repo/tools/gomad/rust' / spec.crate
        executable = root / 'target/release/deps' / (spec.name + '-abcdef')
        executable.parent.mkdir(parents=True, exist_ok=True)
        executable.write_bytes(b'synthetic executable, never executed')
        executable.chmod(0o755)
        generated = root / 'target/release/build' / (spec.package + '-abcdef') / 'out'
        generated.mkdir(parents=True, exist_ok=True)
        package = 'path+' + crate.as_uri() + '#' + spec.package + '@0.1.0'
        frames = [dict(reason='compiler-artifact', package_id=package, manifest_path=str(crate / 'Cargo.toml'),
                       target=dict(name=spec.name, kind=[spec.kind], crate_types=['lib'] if spec.kind == 'lib' else ['bin'],
                                   src_path=str(crate / spec.source), test=True), features=list(spec.features),
                       profile=dict(opt_level='3', debug_assertions=False, test=True), fresh=False, executable=str(executable)),
                  dict(reason='build-script-executed', package_id=package, out_dir=str(generated)),
                  dict(reason='build-finished', success=True)]
        return {'repo': str(directory / 'repo')}, root, frames

    def test_artifact_roles_and_fail_closed_metadata(self):
        with tempfile.TemporaryDirectory() as scratch:
            for role in broad.Role:
                config, root, frames = self.artifact_fixture(Path(scratch), role)
                executable, generated = broad.artifact(config, root, role, frames)
                self.assertEqual(str(executable), frames[0]['executable'])
                self.assertEqual(str(generated), frames[1]['out_dir'])
                mutations = [lambda rows: rows[0].update(features=['reference-helper']),
                             lambda rows: rows[0].update(fresh=True),
                             lambda rows: rows[0]['profile'].update(test=False),
                             lambda rows: rows[0]['profile'].update(debug_assertions=True),
                             lambda rows: rows[0]['target'].update(src_path='/foreign/src.rs'),
                             lambda rows: rows[0]['target'].update(kind=['bin']),
                             lambda rows: rows[0].update(manifest_path='/foreign/Cargo.toml'),
                             lambda rows: rows[1].update(package_id='foreign'),
                             lambda rows: rows[1].update(out_dir='/foreign/out'),
                             lambda rows: rows[-1].update(success=False),
                             lambda rows: rows.append(copy.deepcopy(rows[0]))]
                for mutate in mutations:
                    altered = copy.deepcopy(frames)
                    mutate(altered)
                    with self.subTest(role=role, altered=altered), self.assertRaises(Refusal):
                        broad.artifact(config, root, role, altered)

    def test_stage_root_refuses_symlink(self):
        with tempfile.TemporaryDirectory() as scratch:
            config = {'root': scratch}
            root = broad.owned_root(config)
            root.mkdir()
            real = Path(scratch) / 'foreign'
            real.mkdir()
            (root / broad.Stage.PROVIDER.value).symlink_to(real, target_is_directory=True)
            with self.assertRaises(Refusal):
                broad.stage_root(config, broad.Stage.PROVIDER)


if __name__ == '__main__':
    unittest.main()
