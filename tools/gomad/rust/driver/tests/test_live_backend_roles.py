"""Pure campaign reads retain typed admission and frozen-input checks."""
import json
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import driver
import live
import main_cli_build


class LiveBackendRoles(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        root = Path(self.directory.name)
        repo = root / 'repo'
        repo.mkdir()
        source = repo / 'source'
        source.write_text('source')
        (repo / 'link').symlink_to('source')
        git = shutil.which('git')
        for args in (['init', '--quiet'], ['add', 'source', 'link'],
                     ['-c', 'user.name=unit', '-c', 'user.email=unit@example.invalid', 'commit', '--quiet', '-m', 'source']):
            subprocess.run([git, '-C', str(repo), *args], check=True, capture_output=True)
        self.binary = root / 'gomad'
        self.binary.write_bytes(b'admitted binary')
        receipt = root / 'build.json'
        self.compiler = root / 'compiler'
        self.compiler.write_bytes(b'compiler')
        self.process = root / 'cargo.process.json'
        self.process.write_text('{"exit":0}')
        manifest = root / 'source.manifest'
        manifest.write_text(driver.sha(source) + '  ' + str(source) + '\n')
        self.built = dict(exit=0, profile='release', source_sha256=driver.sha(manifest),
            binary_sha256=driver.sha(self.binary), host_sha256='h', helper_sha256='l', helper_features=[], role='release', features=[])
        receipt.write_text(json.dumps(self.built))
        files = {str(p): driver.sha(p) for p in driver.HERE.iterdir() if p.is_file()}
        for p in list(driver.FROZEN.glob('*')) + [driver.FROZEN.parent / 'q1-serial/run-qualification.sh', source,
                                                     self.binary, receipt, self.compiler, self.process, manifest, driver.HERE.parent / 'workload_tools/workloads.py']:
            if p.is_file():
                files[str(p)] = driver.sha(p)
        self.config = dict(repo=str(repo), root=str(root / 'cycle'), binary=str(self.binary),
            binary_sha256=driver.sha(self.binary),
            head=subprocess.check_output([git, '-C', str(repo), 'rev-parse', 'HEAD'], text=True).strip(),
            files=files, symlinks={str(repo / 'link'): 'source'}, source_manifest=str(manifest),
            source_sha256=driver.sha(manifest), identity=dict(host='h', helper='l'),
            live_corpus=dict(roles={'release': dict(binary=str(self.binary), build_receipt=str(receipt))}))
        path = root / 'config.json'
        path.write_text(json.dumps(self.config))
        self.config.update(_config_path=str(path), _config_sha256=driver.sha(path))
        self.selected = patch.object(main_cli_build, 'read_build', return_value=self.built)
        self.reader = self.selected.start()
        self.addCleanup(self.selected.stop)
        driver.guard(self.config)

    def backend_inputs(self, count=3):
        from types import SimpleNamespace
        work = Path(self.directory.name) / 'live-observed'
        work.mkdir()
        (work / 'roots.json').write_text(json.dumps(dict(schema='gomad.live-backend-corpus/v1', roots=[dict(root=n,
            original=live.roots()[n].split('/', 1)[1]) for n in (4, 5, 6)])))
        self.config['live_corpus']['families'] = dict(observed=dict(role='release'))
        for n in range(count):
            root = work / str(n) / 'campaigns' / 'campaign-one'
            root.mkdir(parents=True)
            (root / 'plan.json').write_text(json.dumps(dict(prepared=dict(helper_sha256='l', host_sha256='h'))))
        def authenticate(path, complete=False):
            return None
        common = SimpleNamespace(seed=dict(canonical_file=driver.read, authenticate_campaign=authenticate,
                                           authenticate_artifact=lambda path: None))
        return work, common

    def test_full_role_before_and_after_pure_campaign_walk(self):
        work, common = self.backend_inputs()
        with patch.object(live, 'load', return_value=common), patch.object(live, 'guard', wraps=driver.guard) as guard, \
             patch.object(live, 'command', side_effect=AssertionError('guest command in pure reader')):
            live.backend(self.config, work, 'observed')
        self.assertEqual(self.reader.call_count, 2)
        self.assertEqual(guard.call_count, 5)

    def test_one_or_two_campaigns_keep_original_typed_admission_count(self):
        for count in (1, 2):
            with self.subTest(count=count):
                work, common = self.backend_inputs(count)
                self.reader.reset_mock()
                try:
                    with patch.object(live, 'load', return_value=common):
                        live.backend(self.config, work, 'observed')
                    self.assertEqual(self.reader.call_count, count)
                finally:
                    shutil.rmtree(work)

    def test_actual_frozen_mutations_between_campaigns_refuse(self):
        for name in ('source', 'compiler', 'receipt', 'process', 'binary', 'config', 'source-link'):
            with self.subTest(name=name):
                work, common = self.backend_inputs()
                target = {'source': Path(self.config['repo']) / 'source', 'compiler': self.compiler,
                    'receipt': Path(self.config['live_corpus']['roles']['release']['build_receipt']),
                    'process': self.process, 'binary': self.binary,
                    'config': Path(self.config['_config_path'])}.get(name)
                raw = target.read_bytes() if target else None
                link = Path(self.config['repo']) / 'link'
                def mutate(path, complete=False):
                    if target:
                        target.write_bytes(raw + b'changed')
                    else:
                        link = Path(self.config['repo']) / 'link'
                        link.unlink(); link.symlink_to('changed')
                common.seed['authenticate_campaign'] = mutate
                try:
                    with patch.object(live, 'load', return_value=common), self.assertRaises(driver.Refusal):
                        live.backend(self.config, work, 'observed')
                finally:
                    if target:
                        target.write_bytes(raw)
                    else:
                        link.unlink(); link.symlink_to('source')
                    shutil.rmtree(work)

    def test_final_typed_reader_still_refuses_structural_change_after_last_campaign(self):
        import stat
        work, common = self.backend_inputs()
        calls = []
        def structural(*args, **kwargs):
            calls.append(True)
            driver.require(stat.S_IMODE(self.binary.stat().st_mode) == 0o755, 'typed executable mode changed')
            return self.built
        self.binary.chmod(0o755)
        common.seed['authenticate_campaign'] = lambda *args, **kwargs: self.binary.chmod(0o644)
        with patch.object(live, 'load', return_value=common), patch.object(main_cli_build, 'read_build', side_effect=structural), \
             self.assertRaisesRegex(driver.Refusal, 'typed executable mode'):
            live.backend(self.config, work, 'observed')
        self.assertEqual(len(calls), 2)

    def test_guard_runs_again_after_planning_before_endpoint_command(self):
        root = Path(self.config['root']) / 'stages/ordinary'
        root.mkdir(parents=True)
        self.config['stages'] = dict(ordinary=dict(live_families=['inspection']))
        def planned(config, family):
            live.role(config, 'release')
            self.compiler.write_bytes(b'changed after planning')
            return dict(argv=['/endpoint']), root, root / 'live-inspection'
        with patch.object(live, 'planned', side_effect=planned), \
             patch.object(live, 'command', side_effect=AssertionError('endpoint before frozen guard')) as command:
            with self.assertRaises(driver.Refusal):
                live.execute_families(self.config, root, 'ordinary')
            command.assert_not_called()


if __name__ == '__main__':
    unittest.main()
