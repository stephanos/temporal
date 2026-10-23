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


def rows_artifact(primary, config, root):
    rows = primary.recipes(config, 'artifact', root / 'stages/artifact')[:-1]
    for row in rows:
        assert row['argv'][1:3] == [str(primary.HERE / 'installed_lifecycle_families.py'), 'produce']
        assert row['argv'][-4:-2] == ['--root', str(root / 'stages/artifact/primary' / ('installed-lifecycle-' + row['argv'][-1]))]
    return rows


class PrimaryRecipes(unittest.TestCase):
    def config(self, root, template):
        return dict(root=str(root), python='/sealed/python', _config_path='/sealed/driver.json',
            binary='/sealed/release', repo='/sealed/repo', source_manifest='/sealed/source.sha256',
            live_corpus=dict(families={
                'cluster': dict(role='host-fixture', argv=['/sealed/python', 'cluster', '--input', str(template)]),
                'broker': dict(role='host-fixture', argv=['/sealed/python', 'broker', '--observation', '/sealed/observation'])}))

    def test_broad_routes_require_independent_closed_receiving(self):
        root = Path('/sealed/cycle/stages/cluster-provider')
        config = dict(root='/sealed/cycle', python='/sealed/python', _config_path='/sealed/config.json')
        self.assertTrue(primary.BROAD <= primary.OWNED)
        self.assertFalse(primary.BROAD & primary.MISSING.keys())
        with patch('broad_stages.preflight', return_value={'fixed': True}) as preflight:
            self.assertEqual(primary.preflight(config, 'cluster-provider', root), {'fixed': True})
            preflight.assert_called_once_with(config, 'cluster-provider', root)
        with patch('broad_stages.run', side_effect=Refusal('unreceived literal branch')):
            with self.assertRaisesRegex(Refusal, 'unreceived literal branch'):
                primary.run(config, 'cluster-provider', root)
        with patch('receiving.Retained.command'), patch('broad_stages.read', side_effect=Refusal('raw mutation omitted')):
            with self.assertRaisesRegex(Refusal, 'raw mutation omitted'):
                primary.read_stage(config, 'cluster-provider', root)

    def test_terminal_preflight_never_reads_future_module_or_publication(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            config = self.config(root, root/'unused')
            config.update(terminal_module_spec='/sealed/terminal-spec', stages={name:dict(adapter='primary') for name in
                          ('terminal-module', 'terminal-host-admission', 'ordinary-helper-publication')})
            with patch.object(primary, 'native_bootstrap'), patch.object(primary, 'terminal_diagnostic'), \
                 patch.object(primary.live, 'sealed_input'), patch.object(primary, 'read', return_value={'static':'inputs'}), \
                 patch('terminal_inputs.preparation_spec', return_value={'static':'inputs'}), \
                 patch.object(primary, 'read_terminal_module', side_effect=AssertionError('future module read')):
                primary.preflight(config, 'terminal-exit', root/'stages/terminal-exit')
            self.assertFalse((root/'stages').exists())

    def test_terminal_fixed_recipes_bind_actual_prerequisite_outputs(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            config = self.config(root, root/'unused')
            config['final_profiles'] = dict(stock_go='/sealed/go', diagnostic=dict(admission='/sealed/diag-admission'))
            publication = root/'stages/ordinary-helper-publication'
            publication.mkdir(parents=True)
            (publication/'primary-final.stdout').write_text(json.dumps(dict(result='/fresh/ordinary-admission')))
            with patch.object(primary.live, 'sealed_input'), patch.object(primary, 'native_bootstrap', return_value='/sealed/bootstrap'), \
                 patch.object(primary, 'read_terminal_module', return_value=dict(path='/fresh/descriptor',descriptor=dict(module='/fresh/module'))), \
                 patch.object(primary, 'completed_stage', return_value=publication), \
                 patch.object(primary, 'terminal_diagnostic', return_value=dict(executable='/sealed/diag')), \
                 patch('native_roles.validate_admission', return_value=dict(executable='/fresh/ordinary')):
                argv = primary.recipes(config, 'terminal-exit', root/'stages/terminal-exit')[0]['argv']
            self.assertEqual(argv[:2], ['/sealed/bootstrap','terminal-exit'])
            for flag, value in (('--test-helper','/fresh/ordinary'),('--terminal-exit-module','/fresh/module'),
                                ('--terminal-exit-module-receipt','/fresh/descriptor'),
                                ('--terminal-exit-helper-receipt','/fresh/ordinary-admission'),
                                ('--terminal-exit-diagnostic-helper-receipt','/sealed/diag-admission')):
                self.assertEqual(argv[argv.index(flag)+1],value)
            self.assertEqual(primary.TIMEOUTS['terminal-exit'],3600)
            self.assertEqual(primary.TIMEOUTS['terminal-module'],3600)

    def test_final_terminal_reuse_refuses_missing_stale_or_foreign_stage(self):
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary)
            config=self.config(root,root/'unused')
            config.update(source_sha256='source',identity={'host':'host','helper':'helper'},_config_sha256='config')
            stage=root/'stages/terminal-exit';stage.mkdir(parents=True)
            row=dict(schema='gomad.rust-driver-stage/v1',stage='terminal-exit',status='PASS',exit=0,
                     source='source',identity=config['identity'],config_sha256='config',subgates=[],start_ns=100,end_ns=200)
            for field,value in (('source','old-source'),('config_sha256','old-config'),('stage','terminal-controls'),('status','UNEXECUTED')):
                altered=dict(row,**{field:value});(stage/'receipt.json').write_text(json.dumps(altered))
                with self.subTest(field=field),self.assertRaises(primary.Refusal):primary.read_terminal(config,root)
            with self.assertRaises(primary.Refusal):primary.read_terminal(config,root/'foreign')

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
                with self.assertRaises(Refusal):
                    primary.recipes(config, 'artifact', root / 'stages/artifact')
                config.update(build_receipt='/release/main-build.json', final_retained=dict(compiler_root='/go'),
                              installed_cluster_failure=dict(guest_config='/typed.json', module_cache='/modules'))
                for label, section in [('artifact', 'retained'), ('operator', 'operator')]:
                    rows = primary.recipes(config, label, root / 'stages' / label)
                    self.assertEqual(len(rows), 4 if label == 'artifact' else 1)
                    self.assertEqual(rows[-1]['argv'][-4:], ['--section', section, '--root', str(root)])
                    self.assertNotIn('--read-only', rows[-1]['argv'])
                families = [row['argv'][-1] for row in rows_artifact(primary, config, root)]
                self.assertEqual(families, ['typed-detached', 'kill-12', 'kill-24'])

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

    def test_terminal_managed_result_binds_direct_run_path_and_pid(self):
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary)
            run=root/'primary/.gomad-measure/terminal-exit/run-123-150-0'
            run.mkdir(parents=True,mode=0o700)
            for name,value in (('.pin',str(run)),('.complete',''),('.lease','')):
                (run/name).write_text(value)
            (run.parent/'latest.json').write_text(json.dumps(run.name))
            (root/'primary-final.command.json').write_text(json.dumps(dict(pid=123,start_ns=100,end_ns=200)))
            response=dict(requested_root=str(root/'primary'),result=str(run/'terminal-exit.json'))
            path,actual,pid=primary.managed_native_run(root,response,'terminal-exit','terminal-exit.json',direct=True)
            self.assertEqual((path,actual,pid),(run/'terminal-exit.json',run,123))
            for output in (str(run/'nested/terminal-exit.json'),str(run).replace('run-123','run-456')+'/terminal-exit.json'):
                with self.assertRaises(Refusal):primary.managed_native_run(root,dict(response,result=output),'terminal-exit','terminal-exit.json',direct=True)

    def test_primary_subcommands_are_owned_by_actual_stage_producer(self):
        import receiving
        from unittest.mock import Mock
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary)
            config=self.config(root,root/'unused')
            producer=dict(pid=101,parent_pid=99,clock='CLOCK_MONOTONIC',started_monotonic_nanos=10,reaped_monotonic_nanos=100)
            (root/'producer.command.json').write_text(json.dumps(producer))
            (root/'producer.process.json').write_text(json.dumps(producer))
            child=dict(pid=102,parent_pid=101,clock='CLOCK_MONOTONIC',started_monotonic_nanos=20,reaped_monotonic_nanos=90,environment={})
            process=dict(child,stage_pid=101)
            recipe=[dict(name='primary-final',argv=['/sealed/exe'])]
            for field,bad in ((None,None),('stage_pid',777),('parent_pid',777),('started_monotonic_nanos',1),('reaped_monotonic_nanos',200),('clock','CLOCK_REALTIME')):
                observed=dict(child);actual=dict(process)
                if field=='stage_pid':actual[field]=bad
                elif field is not None:observed[field]=bad;actual[field]=bad
                (root/'primary-final.command.json').write_text(json.dumps(observed))
                (root/'primary-final.process.json').write_text(json.dumps(actual))
                with patch.object(receiving,'Retained',return_value=Mock()),patch.object(primary,'recipes',return_value=recipe):
                    if field is None:primary.actual_commands(config,'helper-build',root)
                    else:
                        with self.subTest(field=field),self.assertRaises(Refusal):primary.actual_commands(config,'helper-build',root)

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
