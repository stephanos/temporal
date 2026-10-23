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

    def test_native_builds_are_distinct_fresh_stage_invocations(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            config = self.config(root, root/'unused')
            with patch.object(primary, 'native_bootstrap', return_value='/sealed/native-measure'), patch.object(primary.live, 'sealed_input'):
                argv = [primary.recipes(config, label, root/'stages'/label)[0]['argv'] for label in ('wasmmeasure-build','helper-build')]
            self.assertNotEqual(argv[0], argv[1])
            self.assertTrue(all(row[:2] == ['/sealed/native-measure','terminal-exit-helper-build'] for row in argv))
            self.assertEqual(argv[0][-1], str(root/'stages/wasmmeasure-build/primary'))
            self.assertEqual(argv[1][-1], str(root/'stages/helper-build/primary'))

    def test_publication_preflight_does_not_require_unknown_future_receipts(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            config = self.config(root, root/'unused')
            config.update(stages={'helper-build':dict(adapter='primary'),'conformance-helper-build':dict(adapter='primary')},
                          installations={'cooperative-runtime':'/sealed/installation'})
            with patch.object(primary, 'native_bootstrap', return_value='/sealed/native-measure'), patch.object(primary.live, 'sealed_input'), \
                 patch.object(primary, 'native_build', side_effect=AssertionError('future output read')):
                primary.preflight(config, 'ordinary-helper-publication', root/'stages/ordinary-helper-publication')
            self.assertFalse((root/'stages').exists())

    def test_native_managed_run_binds_real_retention_layout_and_creation_owner(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            run = root / 'primary/.gomad-measure/terminal-exit-helper-build/run-123-150-0'
            run.mkdir(parents=True, mode=0o700)
            for name, value in (('.pin', str(run)), ('.complete', ''), ('.lease', '')):
                (run/name).write_text(value)
            (run.parent/'latest.json').write_text(json.dumps(run.name))
            (root/'primary-final.command.json').write_text(json.dumps(dict(pid=123,start_ns=100,end_ns=200)))
            response = dict(requested_root=str(root/'primary'), result=str(run/'release-native-123-0/helper-build.json'))
            primary.managed_native_run(root,response,'terminal-exit-helper-build','helper-build.json')
            for mutation in ('pid','time','pin','completion','latest','store'):
                changed = dict(response)
                if mutation=='pid': changed['result'] = changed['result'].replace('run-123-', 'run-456-')
                if mutation=='time': changed['result'] = changed['result'].replace('-150-', '-300-')
                if mutation=='pin': (run/'.pin').write_text('foreign')
                if mutation=='completion': (run/'.complete').write_text('foreign')
                if mutation=='latest': (run.parent/'latest.json').write_text(json.dumps('run-foreign'))
                if mutation=='store': changed['result'] = changed['result'].replace('terminal-exit-helper-build', 'foreign')
                with self.subTest(mutation=mutation),self.assertRaises(Refusal):
                    primary.managed_native_run(root,changed,'terminal-exit-helper-build','helper-build.json')
                (run/'.pin').write_text(str(run))
                (run/'.complete').write_text('')
                (run.parent/'latest.json').write_text(json.dumps(run.name))

    def test_native_publication_physically_copies_distinct_roles(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root/'conformance-source'
            source.write_bytes(b'conformance source bytes')
            host = root/'ordinary-source'
            host.write_bytes(b'ordinary host bytes')
            stage = root/'stages/ordinary-helper-publication'
            (stage/'primary').mkdir(parents=True)
            with patch.object(primary, 'conformance', return_value=(str(source),'unused')):
                primary.publish_native_files(dict(root=str(root), binary=str(host)), stage)
            copied = stage/'primary/conformance/gomad-wasmhost'
            installed = stage/'primary/ordinary-host/gomad'
            self.assertEqual(copied.read_bytes(), source.read_bytes())
            self.assertEqual(installed.read_bytes(), host.read_bytes())
            self.assertEqual(installed.stat().st_mode & 0o777, 0o755)
            self.assertEqual(copied.stat().st_nlink, 1)
            self.assertNotEqual(copied.stat().st_ino, source.stat().st_ino)
            self.assertNotEqual(installed.stat().st_ino, host.stat().st_ino)

    def test_derived_discovery_reopens_real_command_without_future_frozen_pin(self):
        import receiving
        from driver import sha
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root/'frozen-host'
            source.write_bytes(b'ordinary bytes')
            stage = root/'stages/ordinary-helper-publication'
            copy = stage/'primary/ordinary-host/gomad'
            copy.parent.mkdir(parents=True)
            copy.write_bytes(source.read_bytes())
            copy.chmod(0o755)
            config = dict(root=str(root), repo='/repo', sent_epoch=0, binary=str(source), binary_sha256=sha(source), files={str(source):sha(source)})
            item = dict(name='primary-discover', argv=[str(copy),'--installation','/installation','discover'])
            row = dict(argv=item['argv'], cwd='/repo', pid=123, start_ns=10, end_ns=20, observation='completed', exit=0)
            process = dict(argv=item['argv'], pid=123, start_ns=10, end_ns=15, observation='completed', exit=0, active=False, stage_pid=1)
            (stage/'primary-discover.command.json').write_text(json.dumps(row))
            (stage/'primary-discover.process.json').write_text(json.dumps(process))
            (stage/'primary-discover.stdout').write_text('{}')
            (stage/'primary-discover.stderr').write_text('')
            reader = receiving.Retained(config)
            primary.published_discovery_command(config,reader,stage,item)
            self.assertNotIn(str(copy), config['files'])
            self.assertIn(str(stage/'primary-discover.command.json'), reader.commands)
            copy.write_bytes(b'changed published program')
            with self.assertRaises(Refusal):
                primary.published_discovery_command(config,reader,stage,item)

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
