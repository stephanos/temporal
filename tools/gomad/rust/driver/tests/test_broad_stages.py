"""Synthetic coordinator controls; these fixtures are never native process evidence."""
import json
import fcntl
from concurrent.futures import ThreadPoolExecutor
import threading
from pathlib import Path
import sys
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import Mock, patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import broad_harness as harness
import broad_stages as broad
import cluster_wire_reference as wire
from driver import Refusal


class BroadStages(unittest.TestCase):
    def test_three_stages_wait_for_authenticated_shared_build(self):
        with tempfile.TemporaryDirectory() as scratch:
            config = {'root': scratch, 'source_sha256': 'synthetic', '_config_sha256': 'synthetic', 'identity': {}}
            config.update(python=sys.executable, _config_path=str(Path(scratch) / 'config.json'))
            created, release = threading.Event(), threading.Event()
            observed = [threading.Event(), threading.Event()]
            followers = {}
            partial_reads, builds = [], []
            executing = threading.Barrier(3, timeout=5)
            original_flock = fcntl.flock

            def flock(fd, operation):
                try:
                    return original_flock(fd, operation)
                except BlockingIOError:
                    observed[followers[threading.get_ident()]].set()
                    raise

            def command(*args, **kwargs):
                builds.append(1)
                created.set()
                if not release.wait(5):
                    raise RuntimeError('test initializer was not released')
                (harness.owned_root(config) / 'ready').write_text('synthetic authenticated receipt')

            def read_build(config):
                if not (harness.owned_root(config) / 'ready').exists():
                    partial_reads.append(threading.get_ident())
                    observed[followers[threading.get_ident()]].set()
                    raise Refusal('receipt missing while initializer owns root')
                return {'synthetic': 'authenticated'}

            def execute(config, stage):
                destination = harness.stage_root(config, stage)
                destination.mkdir()
                executing.wait()

            def run(stage, index=None):
                if index is not None:
                    followers[threading.get_ident()] = index
                root = Path(scratch) / 'stages' / stage.value
                root.mkdir(parents=True, exist_ok=True)
                return broad.run(config, stage.value, root)

            with patch.object(broad, 'preflight'), patch.object(harness, 'guard'), \
                    patch.object(harness, 'source_inputs'), patch.object(harness, 'fixed') as fixed, \
                    patch.object(harness, 'command', side_effect=command), \
                    patch.object(harness, 'read_build', side_effect=read_build), \
                    patch.object(harness, 'execute', side_effect=execute), \
                    patch.object(broad, 'collect', return_value={'complete': True, 'subgates': [], 'reader_errors': {}}), \
                    patch.object(wire, 'collect'), patch.dict(sys.modules, {'counterfactual_receiving': SimpleNamespace(run=lambda config: None)}), \
                    patch.object(fcntl, 'flock', side_effect=flock), ThreadPoolExecutor(max_workers=3) as pool:
                fixed.return_value.environment.return_value = {}
                first = pool.submit(run, harness.Stage.PROVIDER)
                self.assertTrue(created.wait(5))
                rest = [pool.submit(run, stage, index) for index, stage in enumerate(
                    (harness.Stage.FUNCTIONAL, harness.Stage.RETAINED))]
                try:
                    for event in observed:
                        self.assertTrue(event.wait(5))
                    self.assertEqual(partial_reads, [])
                finally:
                    release.set()
                for future in [first, *rest]:
                    self.assertEqual(future.result(timeout=5), {'complete': True, 'subgates': [], 'reader_errors': {}})
                self.assertEqual(builds, [1])
                self.assertEqual({path.name for path in harness.owned_root(config).iterdir() if path.is_dir()},
                                 {'cluster-provider', 'cluster-functional', 'retained-cluster'})

    def test_closed_original_stage_inventory(self):
        config = {'root': '/owned'}
        for label, count in broad.COUNTS.items():
            stage, root, expected = broad.selection(config, label, Path('/owned/stages') / label)
            self.assertEqual(stage.value, label)
            self.assertEqual(len(expected), count)
        for label in ('PASS', 'ordinary', None, harness.Stage.PROVIDER):
            with self.assertRaises(Refusal):
                broad.selection(config, label, Path('/owned/stages/cluster-provider'))
        with self.assertRaises(Refusal):
            broad.selection(config, 'cluster-provider', Path('/borrowed/stages/cluster-provider'))

    def test_removed_wire_roots_never_gain_credit(self):
        readers = {'synthetic': {'roots': {root: {'cases': ['synthetic']} for root in broad.REMOVED_WIRE}}}
        observed, complete = broad.closure(list(broad.REMOVED_WIRE), readers, {})
        self.assertFalse(complete)
        self.assertTrue(all(row['status'] == 'REFUSED' for row in observed))
        self.assertTrue(all(row['dispositions'][0]['disposition'] == 'unreceived-removed-wire-schema-or-literal' for row in observed))

    def test_root_closure_requires_one_complete_reader(self):
        roots = ['complete', 'inspection-list', 'missing', 'partial', 'dependency', 'external-dependency', 'duplicate']
        readers = {
            'first': {'roots': {'complete': {'cases': ['synthetic']}, 'inspection-list': [{'variant': 'synthetic'}],
                                'external-dependency': {'cases': ['synthetic']}, 'partial': {'cases': ['synthetic']},
                                'dependency': {'facts': {'remaining_original_predicates': ['missing original assertion']}},
                                'duplicate': {'cases': ['synthetic']}}},
            'second': {'roots': {'duplicate': {'cases': ['synthetic']}},
                       'dependencies': {'external-dependency': ['missing original owner']},
                       'partial_roots': {'partial': {'remaining_predicates': ['unreceived literal']}}},
        }
        observed, complete = broad.closure(roots, readers, {})
        self.assertFalse(complete)
        self.assertEqual([row['id'] for row in observed if row['status'] == 'PASS'], ['complete', 'inspection-list'])
        self.assertEqual(observed[-1]['dispositions'][0]['disposition'], 'ambiguous-whole-root-owners')
        _, complete = broad.closure(['complete'], readers, {'supporting-reader': 'refused'})
        self.assertFalse(complete)
        with self.assertRaises(Refusal):
            broad.accepted({'complete': False, 'subgates': observed, 'reader_errors': {}})

    def test_fixed_stdout_routes_and_context(self):
        with tempfile.TemporaryDirectory() as scratch:
            root = Path(scratch)
            raw = {'roles': {}}
            for row in harness.catalog(harness.Stage.PROVIDER):
                path = root / (row.name + '.stdout')
                text = row.name
                if row.name == 'host-writer':
                    text = '\n'.join(json.dumps({'schema': schema}) for schema in (
                        'gomad.native-retained-writer-observation/v1', 'gomad.native-retained-fields-observation/v1'))
                if row.role in harness.ALLOCATION_ROLES:
                    text = 'running 1 test\nALLOC ' + json.dumps({'case': 'synthetic'}) + \
                        '\ntest result: ok. 1 passed; 0 failed; 0 ignored; 0 measured; 5 filtered out;'
                path.write_text(text)
                raw['roles'][row.name] = {'stdout': str(path), 'cwd': str(root / row.name),
                                          'raw_artifact_root': str(root / 'helper-lib-raw'), 'command': {'stdout': 'synthetic'}}
            modules = {}
            for name in ('core_world', 'core_network', 'core_lifecycle', 'core_retained', 'core_retained_inputs', 'core_guests'):
                modules[name] = SimpleNamespace(observations=Mock(return_value={'roots': {}}))
            modules['core_network'].observations.return_value = {'roots': {}, 'frame_cases': {'native': 'synthetic'}}
            modules['core_cluster_wire'] = SimpleNamespace(observations=Mock(return_value={'roots': {}}))
            modules['core_retained'].observations.return_value = {'roots': {}, 'spelling_cases': [
                {'case': case, 'raw_sha256': 'synthetic'} for case in (
                    'canonical', 'indented', 'member-order', 'escaped-member', 'escaped-value',
                    'trailing-space', 'explicit-optional-null')]}
            modules['core_retained_inputs'].DECODE_CASES = ('synthetic-strict-a', 'synthetic-strict-b')
            strict = ['decode/' + case for case in (*modules['core_retained_inputs'].DECODE_CASES, 'native-valid')]
            modules['core_retained_inputs'].observations.return_value = {'roots': {},
                'supporting_obligations': {'TestRetainedDecodeAcceptsOrdinarySpellingsAndRejectsStrictInput': {'strict_cases': strict}},
                'facts': {case: {'case': case, 'verified': 'synthetic'} for case in strict}}

            modules['core_guest_retention'] = SimpleNamespace(observations=Mock(return_value=dict(roots={},source_closed_receiving={},partial_roots={})))
            modules['core_boot_network'] = SimpleNamespace(ROOT='TestProviderBindsEverySelectedBootNetworkLimit',
                observations=Mock(return_value=dict(received=['synthetic'], additional_controls=[], unreceived=['unused mismatch'], representation='synthetic')))
            modules['core_lifecycle'].observations.return_value['partial_roots'] = {}
            modules['core_presence'] = SimpleNamespace(observations=Mock(return_value={'presence-root': [{'remaining_original_predicates': ['retired forms']}]}))
            for name in ('core_codec', 'core_inspection'):
                modules[name] = SimpleNamespace(observations=Mock(return_value={}))
            modules['core_identity'] = SimpleNamespace(observations=Mock(return_value={'implementation_sha256': 'synthetic'}))
            modules['core_retained_matrix'] = SimpleNamespace(lifecycle_matrix=Mock(return_value=[{'case': 'synthetic'}]))
            modules['core_predicates'] = SimpleNamespace(retained_writer=Mock(return_value={'cases': ['synthetic']}),
                                                       retained_fields=Mock(return_value={'cases': ['synthetic']}))
            modules['allocation_receiving'] = SimpleNamespace(MARKER='ALLOC ', derive=Mock(return_value={
                'predicates': [{'case': case} for case in ('binary-world', 'compressed-world', 'retained-decode')]}))
            config, build = {'source_sha256': 'synthetic'}, {'context': {'compiler_utf8': 'synthetic', 'bindings': {}}}
            with patch.object(broad.importlib, 'import_module', side_effect=modules.__getitem__):
                readers, errors = broad.typed_readers(config, harness.Stage.PROVIDER, raw, build)
            self.assertEqual(errors, {})
            modules['core_world'].observations.assert_called_once_with(config, 'helper-lib')
            modules['core_lifecycle'].observations.assert_called_once_with(
                config, 'helper-lib\nhost-scenario', network_frame_cases={'native': 'synthetic'})
            modules['core_cluster_wire'].observations.assert_called_once_with(config, 'helper-lib\nhost-scenario')
            modules['core_inspection'].observations.assert_called_once_with(
                config, 'helper-lib\nhost-inspection\nhost-inspection-scale', artifact_roots=[str(root / 'helper-lib-raw')])
            modules['core_identity'].observations.assert_called_once_with(config, 'identity', build['context'])
            modules['core_retained'].observations.assert_called_once_with(config, 'host-retained')
            modules['core_retained_inputs'].observations.assert_called_once_with(config, 'host-retained')
            modules['core_boot_network'].observations.assert_called_once_with(config, 'helper-lib')
            modules['core_presence'].observations.assert_called_once_with(config, 'host-presence')
            self.assertEqual(readers['lifecycle']['partial_roots'][modules['core_boot_network'].ROOT]['remaining_original_predicates'],['unused mismatch'])
            self.assertIn('presence-root',readers['presence']['partial_roots'])
            modules['core_retained_matrix'].lifecycle_matrix.assert_called_once_with('host-lifecycle-matrix')
            self.assertNotIn('roots', readers['retained-matrix'])
            self.assertIn('TestRetainedDecodeAcceptsOrdinarySpellingsAndRejectsStrictInput', readers['retained-decoder']['roots'])
            modules['core_guests'].observations.assert_called_once_with(config, raw)
            modules['core_guest_retention'].observations.assert_called_once_with(config, raw)
            calls = modules['allocation_receiving'].derive.call_args_list
            self.assertEqual(calls[0].kwargs['artifact_root'], root / 'host-allocation')
            self.assertEqual(calls[1].kwargs['artifact_root'], root / 'helper-allocation')
            modules['core_retained_inputs'].observations.return_value['supporting_obligations'][
                'TestRetainedDecodeAcceptsOrdinarySpellingsAndRejectsStrictInput']['strict_cases'] = strict[:-1]
            with patch.object(broad.importlib, 'import_module', side_effect=modules.__getitem__):
                refused, failures = broad.typed_readers(config, harness.Stage.PROVIDER, raw, build)
            self.assertIn('retained-decoder', failures)
            self.assertNotIn('retained-decoder', refused)
            modules['core_retained_inputs'].observations.return_value['supporting_obligations'][
                'TestRetainedDecodeAcceptsOrdinarySpellingsAndRejectsStrictInput']['strict_cases'] = strict
            modules['core_retained'].observations.return_value['spelling_cases'].pop()
            with patch.object(broad.importlib, 'import_module', side_effect=modules.__getitem__):
                refused, failures = broad.typed_readers(config, harness.Stage.PROVIDER, raw, build)
            self.assertIn('retained-decoder', failures)
            self.assertNotIn('retained-decoder', refused)


    def test_unmapped_diagnostics_persist_before_refusal_and_build_is_shared(self):
        with tempfile.TemporaryDirectory() as scratch:
            config = {'root': scratch, 'source_sha256': 'synthetic', '_config_sha256': 'synthetic', 'identity': {}}
            build_root = harness.owned_root(config)
            def build(config):
                build_root.mkdir(exist_ok=True)
            def facts(config, label, root):
                return {'schema': broad.SCHEMA, 'label': label, 'complete': False,
                        'reader_errors': {}, 'subgates': [{'id': 'unreceived', 'status': 'REFUSED'}]}
            with patch.object(broad, 'preflight'), patch.object(harness, 'execute_build', side_effect=build) as fresh, \
                    patch.object(harness, 'read_build'), patch.object(harness, 'execute') as execute, \
                    patch.object(broad, 'collect', side_effect=facts), patch.object(wire, 'collect') as reference:
                for label in ('cluster-provider', 'retained-cluster'):
                    root = Path(scratch) / 'stages' / label
                    root.mkdir(parents=True)
                    with self.assertRaises(Refusal):
                        broad.run(config, label, root)
                    retained = json.loads((root / 'broad-diagnostics.json').read_text())
                    self.assertFalse(retained['complete'])
                    self.assertEqual(retained['subgates'][0]['status'], 'REFUSED')
                self.assertEqual(fresh.call_count, 2)
                reference.assert_called_once_with(config, produce=True)
                self.assertEqual([call.args[1] for call in execute.call_args_list], [harness.Stage.PROVIDER, harness.Stage.RETAINED])

    def test_original_cluster_wire_reference_failure_refuses_all_provider_roots(self):
        with tempfile.TemporaryDirectory() as scratch:
            config = {'root': scratch, 'source_sha256': 'synthetic', '_config_sha256': 'synthetic', 'identity': {}}
            label = 'cluster-provider'
            root = Path(scratch) / 'stages' / label
            root.mkdir(parents=True)
            harness.owned_root(config).mkdir()
            with patch.object(broad, 'preflight'), patch.object(harness, 'execute_build'), \
                    patch.object(harness, 'execute'), patch.object(wire, 'collect', side_effect=Refusal('original six did not execute')):
                with self.assertRaises(Refusal):
                    broad.run(config, label, root)
            retained = json.loads((root / 'broad-diagnostics.json').read_text())
            self.assertEqual(len(retained['subgates']), 78)
            self.assertTrue(all(row['status'] == 'REFUSED' for row in retained['subgates']))
            self.assertEqual(retained['reader_errors']['execution']['reason'], 'original six did not execute')

    def test_execution_failure_keeps_explicit_refused_roots(self):
        with tempfile.TemporaryDirectory() as scratch:
            config = {'root': scratch, 'source_sha256': 'synthetic', '_config_sha256': 'synthetic', 'identity': {}}
            label = 'cluster-functional'
            root = Path(scratch) / 'stages' / label
            root.mkdir(parents=True)
            with patch.object(broad, 'preflight'), patch.object(harness, 'execute_build', side_effect=Refusal('synthetic custody refusal')):
                with self.assertRaises(Refusal):
                    broad.run(config, label, root)
            retained = json.loads((root / 'broad-diagnostics.json').read_text())
            self.assertEqual(len(retained['subgates']), 46)
            self.assertTrue(all(row['status'] == 'REFUSED' for row in retained['subgates']))
            self.assertEqual(retained['reader_errors']['execution']['reason'], 'synthetic custody refusal')

    def test_reader_recomputes_diagnostics(self):
        with tempfile.TemporaryDirectory() as scratch:
            config = {'root': scratch}
            label = 'cluster-provider'
            root = Path(scratch) / 'stages' / label
            root.mkdir(parents=True)
            (root / 'broad-diagnostics.json').write_text(json.dumps({'complete': True, 'caller_status': 'PASS'}))
            actual = {'complete': False, 'subgates': [], 'reader_errors': {}}
            with patch.object(broad, 'preflight'), patch.object(broad, 'collect', return_value=actual):
                with self.assertRaisesRegex(Refusal, 'independently reopened'):
                    broad.read(config, label, root)


if __name__ == '__main__':
    unittest.main()
