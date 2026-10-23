#!/usr/bin/env python3
"""Portable predicate tests; these do not execute Gomad or prove signal handling."""
import copy
import importlib.util
from pathlib import Path
import unittest
import tempfile
import sys

sys.dont_write_bytecode = True

PATH = Path(__file__).with_name('installed-lifecycle-check.py')


class LifecycleTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        if not PATH.exists():
            cls.check = None
            return
        spec = importlib.util.spec_from_file_location('installed_lifecycle', PATH)
        cls.check = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(cls.check)

    def setUp(self):
        self.assertIsNotNone(self.check, 'installed lifecycle checker missing')

    def test_missing_public_lifecycle_is_not_resumable_success(self):
        with self.assertRaisesRegex(ValueError, 'campaign'):
            self.check.lifecycle({'campaign': {'executions': []}}, {'executions': [], 'journal': []}, 12)

    def test_public_running_prefix_requires_pending_unreceipted_job(self):
        execution = {'ordinal': 0, 'seed': 1, 'record': {'outcome': {'class': 'success'}}}
        report = {'campaign': {'executions': [execution], 'stop_reason': None, 'events': 2}}
        validated = {'executions': [execution], 'journal': [
            {'event': {'kind': 'outcome', 'executions': [execution], 'admitted': [{'ordinal': 1, 'seed': 2}]}},
            {'event': {'kind': 'admit', 'jobs': []}}]}
        self.assertEqual(self.check.lifecycle(report, validated, 24, mixed=True), 1)
        validated['journal'].append({'event': {'kind': 'receipt', 'execution': {'ordinal': 1}}})
        report['campaign']['events'] = 3
        with self.assertRaisesRegex(ValueError, 'unreceipted'):
            self.check.lifecycle(report, validated, 24, mixed=True)

    def test_complete_prefix_cannot_be_used_as_actual_kill_proof(self):
        report = {'campaign': {'executions': [{}] * 12, 'stop_reason': None, 'events': 1}}
        validated = {'executions': [{}] * 12, 'journal': [{'event': {'kind': 'complete'}}]}
        with self.assertRaisesRegex(ValueError, 'prefix'):
            self.check.lifecycle(report, validated, 12)

    def test_late_correct_failure_does_not_pass_promptness(self):
        with self.assertRaisesRegex(ValueError, '10-second'):
            self.check.prompt(b'{"classification":"runner_failure","message":"WASM broker global deadlock"}',
                              10_000_000_001, 'deadlock')

    def test_deadlock_requires_infrastructure_prefix_not_just_marker(self):
        with self.assertRaisesRegex(ValueError, 'backend infrastructure'):
            self.check.prompt(b'{"classification":"runner_failure","message":"WASM broker global deadlock"}',
                              1, 'deadlock')

    def test_discovery_requires_all_original_commands(self):
        report = {'manifest': {'profile': 'gomad.wasi-cooperative/v1'}, 'capabilities': {
            'TypedScenarios': True, 'Simulation': False,
            'Commands': ['explore', 'inspect', 'replay', 'recover', 'resume']}}
        self.check.discovery(report)
        report['capabilities']['Commands'].remove('recover')
        with self.assertRaisesRegex(ValueError, 'recover'):
            self.check.discovery(report)

    def test_asymmetric_three_node_scenario_preserves_wait_order(self):
        scenario = {'spec': {'limits': {'nodes': 2}, 'nodes': [
            {'id': 'a'}, {'id': 'b', 'address': '10.0.0.2'}]},
            'boots': [{'args': ['park']}, {'args': ['echo-loop']}], 'actions': [
                {'action': 'start', 'handle': {'node': 'b', 'incarnation': 1}},
                {'action': 'start', 'handle': {'node': 'a', 'incarnation': 1}},
                {'action': 'wait', 'handle': {'node': 'a', 'incarnation': 1}},
                {'action': 'wait', 'handle': {'node': 'b', 'incarnation': 1}}]}
        value = self.check.asymmetric(scenario, True)
        self.assertEqual(value['boots'][0]['args'], ['fail-peer'])
        self.assertEqual([x['handle']['node'] for x in value['actions']], ['b', 'a', 'c', 'b', 'a', 'c'])
        self.assertEqual(scenario['spec']['limits']['nodes'], 2)

    def test_missing_cluster_inspection_is_not_capacity_proof(self):
        with self.assertRaisesRegex(ValueError, 'cluster inspection'):
            self.check.capacity_inspection({'artifact': {'stdout': {}, 'stderr': {}}}, {}, {})

    def test_semantics_keeps_seed_outcome_and_identity(self):
        left = {'executions': [{'ordinal': 0, 'seed': 7, 'artifact': 'one', 'stored_bytes': 1,
            'record': {'seed': 7, 'outcome': {'class': 'success'}, 'identity': {'target': 'abc'}}}]}
        right = copy.deepcopy(left)
        right['executions'][0]['artifact'] = 'two'
        right['executions'][0]['stored_bytes'] = 2
        self.assertEqual(self.check.semantics(left), self.check.semantics(right))
        right['executions'][0]['record']['identity']['target'] = 'changed'
        self.assertNotEqual(self.check.semantics(left), self.check.semantics(right))

    def test_unsealed_caller_source_is_rejected_before_launch(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / 'source'
            source.mkdir()
            (source / 'main.go').write_text('mutated guest')
            with self.assertRaisesRegex(ValueError, 'source fixture'):
                self.check.source_inputs({'source': str(source)}, 'typed-detached')

    def test_original_kill_fixture_bytes_are_accepted_without_running_go(self):
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory)
            (source / 'go.mod').write_text('module gomad.wasm.killresume.fixture\n\ngo 1.27.1\n')
            (source / 'work_test.go').write_text('package fixture\n\nimport "testing"\n\nfunc TestWASMKillResumeWork(t *testing.T) {}\n')
            self.check.source_inputs({'source': str(source)}, 'kill-12')
            (source / 'work_test.go').write_text('package fixture\n')
            with self.assertRaisesRegex(ValueError, 'source fixture'):
                self.check.source_inputs({'source': str(source)}, 'kill-12')

    def test_public_inspection_cannot_invent_committed_outcomes(self):
        with self.assertRaisesRegex(ValueError, 'authenticated committed'):
            self.check.lifecycle({'campaign': {'executions': [{'ordinal': 1}], 'events': 0, 'stop_reason': None}},
                                 {'executions': [], 'journal': []}, 12)

    def test_published_prefix_cannot_be_resumed(self):
        execution = {'ordinal': 0}
        validated = {'executions': [execution], 'journal': [{'event': {'kind': 'complete'}}]}
        public = {'campaign': {'executions': [execution], 'events': 1, 'stop_reason': None}}
        with self.assertRaisesRegex(ValueError, 'published'):
            self.check.lifecycle(public, validated, 12)

    def test_capacity_output_requires_exact_dimension_and_delta(self):
        good = b'{"classification":"runner_failure","message":"backend capacity: WASI capacity: payload-bytes limit=4096 used=4096 delta=64"}'
        self.check.prompt(good, 10_000_000_000, 'capacity')
        for old, new in [(b'4096', b'8192'), (b'delta=64', b'delta=0'), (b'runner_failure', b'target_failure')]:
            with self.subTest(change=new), self.assertRaises(ValueError):
                self.check.prompt(good.replace(old, new), 1, 'capacity')

    def test_process_ownership_follows_descendants_across_new_groups(self):
        owned = {41}
        self.check.owned_processes(owned, [
            {'pid': 43, 'parent': 42, 'group': 43},
            {'pid': 42, 'parent': 41, 'group': 41},
            {'pid': 99, 'parent': 1, 'group': 99}])
        self.assertEqual(owned, {41, 42, 43})

    def test_semantics_keeps_authored_executed_steps(self):
        left = {'executions': [{'ordinal': 0, 'seed': 7, 'record': {},
                                'scenario_steps': [{'action': 'wait'}]}]}
        right = copy.deepcopy(left)
        right['executions'][0]['scenario_steps'][0]['action'] = 'stop'
        self.assertNotEqual(self.check.semantics(left), self.check.semantics(right))

    def test_empty_and_symlink_evidence_cannot_pass_integrity(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with self.assertRaisesRegex(ValueError, 'empty'):
                self.check.tree(root)
            (root / 'real').write_bytes(b'fixed')
            before = self.check.tree(root)
            (root / 'real').write_bytes(b'changed')
            self.assertNotEqual(before, self.check.tree(root))
            (root / 'alias').symlink_to(root / 'real')
            with self.assertRaisesRegex(ValueError, 'symlink'):
                self.check.tree(root)

    def test_advertised_replay_argv_cannot_select_different_artifact(self):
        artifact = Path('/private/artifact with space')
        good = {'artifacts': [{'path': str(artifact), 'record_hash': 'a' * 64,
            'replay_argv': ['gomad', 'replay', str(artifact)],
            'replay_command': "gomad replay '/private/artifact with space'"}]}
        self.assertEqual(self.check.advertised(good, artifact, 'a' * 64), ['replay', str(artifact)])
        good['artifacts'][0]['replay_command'] = 'gomad replay /different'
        with self.assertRaisesRegex(ValueError, 'advertised'):
            self.check.advertised(good, artifact, 'a' * 64)

    def test_result_without_replay_advertisement_cannot_pass_original_operator_predicate(self):
        with self.assertRaisesRegex(ValueError, 'advertised'):
            self.check.advertised({'type': 'result', 'retained_evidence': 1}, Path('/artifact'), 'a' * 64)

    def test_golden_projection_authenticates_metadata_before_excluding_it(self):
        import hashlib
        import json
        canonical = lambda value: json.dumps(value, sort_keys=True, separators=(',', ':')).encode()
        digest = lambda raw: hashlib.sha256(raw).hexdigest()
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            artifact = root / 'evidence' / 'retained'
            (artifact / 'target').mkdir(parents=True)
            module = b'\0asm\x01\0\0\0'
            identity = dict(target=digest(module), toolchain='1' * 64, platform='2' * 64,
                            implementation='3' * 64, configuration='4' * 64)
            record = dict(schema='gomad.execution-record/v2', seed=7, identity=identity,
                execution_identity=digest(canonical(identity)), profile='gomad.wasi-stock-exploratory/v1',
                arguments=[], environment={}, virtual_time_nanos=0, streams={},
                limits=dict(fuel=1, memory_bytes=1, wall_millis=1, output_bytes=1, evidence_bytes=1),
                outcome=dict(**{'class': 'success'}, termination='exit', exit_code=0, reason=''))
            raw = canonical(record)
            (artifact / 'storage.json').write_bytes(b'{"format":"gomad.artifact-storage/v2"}')
            (artifact / 'record.json').write_bytes(raw)
            (artifact / 'target/module.wasm').write_bytes(module)
            files = {'record.json': dict(bytes=len(raw), sha256=digest(raw), module=False),
                     'target/module.wasm': dict(bytes=8, sha256=digest(module), module=True)}
            (artifact / 'manifest.json').write_bytes(canonical(dict(schema='gomad.artifact-manifest/v2',
                record=record, record_sha256=digest(raw), files=files)))
            validated = {'executions': [{'artifact': 'evidence/retained', 'record': record}]}
            result = {'retained_evidence': 1, 'artifacts': [{'path': str(artifact), 'record_hash': digest(raw),
                'replay_argv': ['gomad', 'replay', str(artifact)], 'replay_command': 'gomad replay ' + str(artifact)}]}
            self.check.check.authenticate_report_advertisements(result, root, validated)
            for mutate in ('empty', 'extra', 'wrong-hash', 'wrong-argv', 'wrong-command'):
                wrong = copy.deepcopy(result)
                if mutate == 'empty':
                    wrong['artifacts'] = []
                elif mutate == 'extra':
                    wrong['artifacts'].append(copy.deepcopy(wrong['artifacts'][0]))
                elif mutate == 'wrong-hash':
                    wrong['artifacts'][0]['record_hash'] = '0' * 64
                elif mutate == 'wrong-argv':
                    wrong['artifacts'][0]['replay_argv'][-1] = '/different'
                else:
                    wrong['artifacts'][0]['replay_command'] = 'gomad replay /different'
                with self.subTest(mutate=mutate), self.assertRaises(ValueError):
                    self.check.check.authenticate_report_advertisements(wrong, root, validated)
            (artifact / 'target/module.wasm').write_bytes(b'tampered')
            with self.assertRaises(ValueError):
                self.check.check.authenticate_report_advertisements(result, root, validated)

    def test_original_guest_bounds_and_environment_cannot_be_widened(self):
        original = self.check.driver.read(self.check.HERE.parents[1] /
            'internal/backends/wasm/command/testdata/scenario/config.json')
        guest = copy.deepcopy(original)
        guest['Environment'].append('GOMAD_REQUIRE_CAPTURED_INPUT=1')
        self.check.guest_base(guest, 'gomad.wasi-cooperative/v1', True)
        for key in original['Limits']:
            changed = copy.deepcopy(guest)
            changed['Limits'][key] += 1
            with self.subTest(limit=key), self.assertRaisesRegex(ValueError, 'original guest'):
                self.check.guest_base(changed, 'gomad.wasi-cooperative/v1', True)
        changed = copy.deepcopy(guest)
        changed['Environment'].remove('GOMAD_REQUIRE_CAPTURED_INPUT=1')
        with self.assertRaisesRegex(ValueError, 'original guest'):
            self.check.guest_base(changed, 'gomad.wasi-cooperative/v1', True)

    def test_declared_volume_topology_without_actual_decoded_snapshot_cannot_pass(self):
        with self.assertRaisesRegex(ValueError, 'actual decoded'):
            self.check.durable_volumes({'artifact': {'storage': None}})
        valid = {'artifact': {'storage': {'volume_count': 1, 'instances': [
            {'node': 'a', 'incarnation': 2, 'volume_ids': ['a-store']}]}}}
        self.check.durable_volumes(valid)
        wrong = copy.deepcopy(valid)
        wrong['artifact']['storage']['instances'][0]['incarnation'] = 1
        with self.assertRaisesRegex(ValueError, 'actual decoded'):
            self.check.durable_volumes(wrong)
        wrong = copy.deepcopy(valid)
        wrong['artifact']['storage']['volume_count'] = 2
        with self.assertRaisesRegex(ValueError, 'actual decoded'):
            self.check.durable_volumes(wrong)


    def test_frozen_operator_qualification_preserves_original_bounds(self):
        spec = self.check.operator_qualification_spec('/private/source', Path('/private/qualified'),
            {'source_sha256': 'a' * 64, 'source_manifest': '/sealed/source-manifest'})
        self.assertEqual(spec['Manifest']['Cell'], 'canary')
        self.assertEqual(spec['Manifest']['Repeat'], 2)
        self.assertFalse(spec['Manifest']['Provisional'])
        self.assertEqual(spec['Campaign']['ExecutionTimeout'], 60_000_000_000)
        self.assertEqual(spec['Campaign']['OverallTimeout'], 180_000_000_000)
        self.assertEqual(spec['Campaign']['OutputLimit'], 8 << 20)
        self.assertEqual(spec['Campaign']['WorldTransitionLimit'], 1 << 20)
        self.assertEqual(spec['Campaign']['Target']['BuildTags'], ['gomad_fixture', 'test_dep'])

    def test_operator_qualification_cannot_infer_measured_replays_from_determinism(self):
        result = {'manifest': {'cell': 'canary'}, 'report': {'deterministic': True, 'target_success': False},
                  'cost': {'executions': 2, 'replays': 1}, 'cost_failure': None}
        with self.assertRaisesRegex(ValueError, 'measured'):
            self.check.operator_qualification_measurements(result)
        result['cost']['replays'] = 2
        self.check.operator_qualification_measurements(result)
        result['report']['target_success'] = True
        with self.assertRaisesRegex(ValueError, 'measured'):
            self.check.operator_qualification_measurements(result)


if __name__ == '__main__':
    unittest.main()
