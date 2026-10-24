"""Pure catalog and synthetic artifact mutations; no native evidence is produced."""
import copy
import json
import os
import subprocess
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import broad_harness as broad
from driver import Refusal


class BroadHarness(unittest.TestCase):
    def test_failed_or_interrupted_initializer_leaves_strict_partial_root(self):
        for failure in (Refusal('initializer failed'), KeyboardInterrupt()):
            with self.subTest(failure=type(failure).__name__), tempfile.TemporaryDirectory() as scratch:
                config = {'root': scratch, 'python': sys.executable, '_config_path': '/synthetic/config'}
                with patch.object(broad, 'guard'), patch.object(broad, 'source_inputs'), \
                        patch.object(broad, 'fixed') as fixed, patch.object(broad, 'command', side_effect=failure), \
                        patch.object(broad, 'build_evidence', return_value={}):
                    fixed.return_value.environment.return_value = {}
                    with self.assertRaises(type(failure)):
                        broad.execute_build(config)
                    self.assertTrue(broad.owned_root(config).is_dir())
                    with self.assertRaises(FileNotFoundError):
                        broad.execute_build(config)

    def test_killed_initializer_releases_lock_but_does_not_authorize_partial_root(self):
        with tempfile.TemporaryDirectory() as scratch:
            script = """import sys
from pathlib import Path
sys.path.insert(0, sys.argv[1])
import broad_harness
with broad_harness.initialization_lock({'root': sys.argv[2]}):
    broad_harness.owned_root({'root': sys.argv[2]}).mkdir()
    print('root-created', flush=True)
    sys.stdin.read()
"""
            with subprocess.Popen([sys.executable, '-c', script, str(Path(broad.__file__).parent), scratch],
                                  stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True) as process:
                try:
                    self.assertEqual(process.stdout.readline().strip(), 'root-created')
                finally:
                    process.kill()
                    self.assertEqual(process.wait(timeout=5), -9)
            with patch.object(broad, 'guard'), patch.object(broad, 'source_inputs'), \
                    patch.object(broad, 'build_evidence', return_value={}):
                with self.assertRaises(FileNotFoundError):
                    broad.execute_build({'root': scratch})

    def test_initialization_wait_is_bounded_and_lock_substitution_refuses(self):
        with tempfile.TemporaryDirectory() as scratch:
            config = {'root': scratch}
            with patch.object(broad.fcntl, 'flock', side_effect=BlockingIOError()), \
                    patch.object(broad.time, 'monotonic', side_effect=[0, 3601]):
                with self.assertRaisesRegex(Refusal, 'timed out'):
                    with broad.initialization_lock(config):
                        self.fail('contended lock admitted')
            original_flock = broad.fcntl.flock

            def substitute(fd, operation):
                original_flock(fd, operation)
                path = Path(scratch) / 'broad-harness.lock'
                path.rename(path.with_suffix('.old'))
                path.write_text('substituted lock')
                path.chmod(0o600)

            with patch.object(broad.fcntl, 'flock', side_effect=substitute):
                with self.assertRaisesRegex(Refusal, 'lock changed'):
                    with broad.initialization_lock(config):
                        self.fail('substituted lock admitted')

    def test_shared_initialization_refuses_foreign_lock_or_root(self):
        for mutation in ('lock-symlink', 'lock-hardlink', 'lock-mode', 'lock-directory', 'root-symlink'):
            with self.subTest(mutation=mutation), tempfile.TemporaryDirectory() as scratch:
                config = {'root': scratch}
                root = broad.owned_root(config)
                lock = root.with_name(root.name + '.lock')
                other = Path(scratch) / 'foreign'
                if mutation == 'root-symlink':
                    other.mkdir()
                    root.symlink_to(other, target_is_directory=True)
                elif mutation == 'lock-directory':
                    lock.mkdir()
                else:
                    other.write_text('foreign')
                    other.chmod(0o600)
                    if mutation == 'lock-symlink':
                        lock.symlink_to(other)
                    elif mutation == 'lock-hardlink':
                        os.link(other, lock)
                    else:
                        lock.write_text('foreign')
                        lock.chmod(0o644)
                with patch.object(broad, 'guard'), patch.object(broad, 'source_inputs'):
                    with self.assertRaises((Refusal, OSError)):
                        broad.execute_build(config)

    def test_shared_reuse_reauthenticates_receipts_identities_and_admission(self):
        for mutation in ('none', 'source', 'config', 'missing-build', 'changed-build', 'missing-admission',
                         'changed-admission', 'executable', 'producer'):
            with self.subTest(mutation=mutation), tempfile.TemporaryDirectory() as scratch:
                config = {'root': scratch, 'python': sys.executable, '_config_path': '/synthetic/config',
                          'source_sha256': 'source-a', '_config_sha256': 'config-a'}
                root = broad.owned_root(config)
                (root / 'admitted').mkdir(parents=True)
                evidence = dict(schema=broad.SCHEMA, source_sha256='source-a', config_sha256='config-a', roles={})
                roles = {}
                for role in broad.Role:
                    path = root / 'admitted' / role.value
                    path.write_bytes(b'synthetic never executed')
                    path.chmod(0o755)
                    source = str(root / 'cargo' / role.value)
                    evidence['roles'][role.value] = dict(executable=source, executable_sha256=broad.sha(path))
                    roles[role.value] = dict(cargo_artifact=source, cargo_artifact_sha256=broad.sha(path),
                                             executable=str(path), executable_sha256=broad.sha(path))
                broad.write(root / 'build-receipt.json', evidence)
                broad.write(root / 'admission.json', dict(schema=broad.ADMISSION, build_receipt=str(root / 'build-receipt.json'),
                            build_receipt_sha256=broad.sha(root / 'build-receipt.json'), roles=roles))
                if mutation in ('source', 'config'):
                    config['source_sha256' if mutation == 'source' else '_config_sha256'] = 'foreign'
                elif mutation in ('missing-build', 'missing-admission'):
                    (root / ('build-receipt.json' if mutation == 'missing-build' else 'admission.json')).unlink()
                elif mutation in ('changed-build', 'changed-admission'):
                    (root / ('build-receipt.json' if mutation == 'changed-build' else 'admission.json')).write_text('{}')
                elif mutation == 'executable':
                    (root / 'admitted' / 'helper-lib').write_bytes(b'changed')

                def reopened(config):
                    return dict(evidence, source_sha256=config['source_sha256'], config_sha256=config['_config_sha256'])

                def producer(*args):
                    if mutation == 'producer':
                        raise Refusal('synthetic raw process authentication failed')
                    return {}, {}

                original_read = broad.native_roles.read
                with patch.object(broad, 'guard'), patch.object(broad, 'source_inputs'), \
                        patch.object(broad, 'build_evidence', side_effect=reopened), \
                        patch.object(broad, 'fixed') as fixed, patch.object(broad, 'producer', side_effect=producer), \
                        patch.object(broad.native_roles, 'read', side_effect=lambda path:
                            {} if str(path).endswith('.command.json') else original_read(path)), \
                        patch('primary_stages.enclose_child'):
                    fixed.return_value.environment.return_value = {}
                    if mutation == 'none':
                        self.assertEqual(broad.execute_build(config), evidence)
                    else:
                        with self.assertRaises((Refusal, OSError)):
                            broad.execute_build(config)

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
                self.assertTrue(broad.target(row.role).harness)
                self.assertEqual(argv[-2:], ['--nocapture', '--test-threads=1'])
                if row.role in broad.ALLOCATION_ROLES:
                    self.assertEqual(argv[1:], ['--ignored', '--exact', 'allocation_receiving', '--nocapture', '--test-threads=1'])
                elif row.name.startswith('helper-') and row.name.endswith('-probe'):
                    self.assertEqual(argv[1:3], ['--ignored', '--exact'])
                    self.assertIn('world::owner_preservation_tests::', argv[3])
                elif row.name == 'host-inspection-scale':
                    self.assertEqual(argv[1:], ['--ignored', '--exact',
                        'replay::inspection::predicate_tests::native_retained_scale_admission_preserves_complete_inspection_and_owner_payloads',
                        '--nocapture', '--test-threads=1'])
                else:
                    self.assertNotIn('--exact', argv)
            self.assertEqual([row.name for row in rows[:2]], ['helper-lib', 'host-inspection-scale'])
            lifecycle = [row for row in rows if row.name == 'host-lifecycle-matrix']
            self.assertEqual(len(lifecycle), 1)
            self.assertIs(lifecycle[0].role, broad.Role.HOST)
            self.assertFalse(lifecycle[0].ignored)
            self.assertFalse(lifecycle[0].exact)
            self.assertEqual(broad.test_argv(config, stage, lifecycle[0].name)[1:],
                             ['replay::cluster::retained_lifecycle_matrix_tests', '--nocapture', '--test-threads=1'])
            guest = [row for row in rows if row.name == 'host-broad-guest']
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

    def test_helper_inspection_artifact_environment_is_stage_owned(self):
        class Fixed:
            @staticmethod
            def environment(config, root, target):
                return {'CARGO_TARGET_DIR': str(root / 'target')}
        config = {'root': '/owned/cycle', 'repo': '/source'}
        evidence = {'context': {'bindings': {'TARGET': 'x86_64-unknown-linux-gnu'}}}
        with patch.object(broad, 'fixed', return_value=Fixed):
            roots = set()
            for stage in broad.Stage:
                for row in broad.catalog(stage):
                    if row.name == 'host-broad-guest':
                        continue
                    observed, environment = broad.invocation_context(config, stage, row, evidence)
                    if row.role is broad.Role.HELPER or row.name == 'host-inspection-scale':
                        path = environment['GOMAD_INSPECTION_RAW_DIR']
                        self.assertEqual(path, '/owned/cycle/broad-harness/' + stage.value + '/helper-lib-raw')
                        self.assertEqual(observed['repo'], config['repo'])
                        roots.add(path)
                    else:
                        self.assertNotIn('GOMAD_INSPECTION_RAW_DIR', environment)
                    self.assertNotIn('GOMAD_STOCK_GO', environment)
                    self.assertNotIn('GOMAD_TEST_BROAD_GUEST_ROOT', environment)
                    if row.role in broad.ALLOCATION_ROLES:
                        self.assertEqual(observed['repo'], '/owned/cycle/broad-harness/' + stage.value + '/' + row.name)
                    else:
                        self.assertEqual(observed['repo'], config['repo'])
            self.assertEqual(len(roots), 3)

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
