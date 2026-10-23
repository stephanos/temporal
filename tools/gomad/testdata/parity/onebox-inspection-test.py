#!/usr/bin/env python3
import base64
import copy
import gzip
import importlib.util
import json
import os
import subprocess
from pathlib import Path
import sys
import tempfile
import unittest
from unittest import mock

sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location('onebox_inspection', Path(__file__).with_name('onebox-inspection-check.py'))
gate = importlib.util.module_from_spec(spec)
spec.loader.exec_module(gate)


class InspectionTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)

    def debug(self):
        output = self.root / 'inspection'
        output.mkdir()
        value = dict(stacks=['stack'], probes=[dict(name=name) for name in ('temporal.shard.snapshot', 'temporal.mutable_state.snapshot')],
                     dumps=[dict(name=name, status='ok', data_base64=base64.b64encode(b'snapshot=useful values may precede stop').decode())
                            for name in ('temporal.mutable_states', 'temporal.shards')])
        (output / 'debug.json').write_text(json.dumps(value))
        (output / 'heap.pprof').write_bytes(gzip.compress(b'heap'))
        report = dict(status='inspected_prefix', prefix_verified=True, witness_match=True, truncated=False,
                      resolved=dict(instance='history', incarnation=1), output_path=str(output / 'debug.json'))
        return value, report

    def test_all_original_debug_witnesses_are_required(self):
        value, report = self.debug()
        self.assertEqual(set(gate.validate_debug(self.root, report)), {'debug.json', 'heap.pprof'})
        for field in ('status', 'prefix_verified', 'witness_match', 'truncated'):
            changed = copy.deepcopy(report)
            changed[field] = 'unknown' if field == 'status' else not report[field]
            with self.subTest(field=field), self.assertRaises(ValueError):
                gate.validate_debug(self.root, changed)
        for mutation in ('stacks', 'probes', 'shards', 'mutable', 'caveat', 'status', 'duplicate'):
            changed = copy.deepcopy(value)
            if mutation in ('stacks', 'probes'):
                changed[mutation] = []
            elif mutation in ('shards', 'mutable'):
                changed['dumps'].pop(1 if mutation == 'shards' else 0)
            elif mutation == 'caveat':
                changed['dumps'][0]['data_base64'] = base64.b64encode(b'snapshot=useful').decode()
            elif mutation == 'status':
                changed['dumps'][0]['status'] = 'unavailable'
            else:
                changed['dumps'].append(changed['dumps'][0])
            (self.root / 'inspection/debug.json').write_text(json.dumps(changed))
            with self.subTest(mutation=mutation), self.assertRaises(ValueError):
                gate.validate_debug(self.root, report)

    def test_wrong_incarnation_and_foreign_diagnostics_are_refused(self):
        _, report = self.debug()
        report['resolved']['incarnation'] = 2
        with self.assertRaises(ValueError):
            gate.validate_debug(self.root, report)
        report['resolved']['incarnation'] = 1
        report['output_path'] = '/foreign/debug.json'
        with self.assertRaises(ValueError):
            gate.validate_debug(self.root, report)

    def artifact(self):
        root = self.root / 'artifact'
        root.mkdir()
        limits = dict(fuel=1 << 50, memory_bytes=4 << 30, wall_millis=1800000, output_bytes=1 << 20)
        config = dict(workload={'kind': 'temporal-onebox-stopped-inspection'}, nodes=['history'], limits=limits,
                      incarnations=[dict(handle=dict(node='history', incarnation=1), limits=limits,
                                         arguments=['gomad-target', '-test.run=^TestReplayInspectionWorkflowFailure$', '-test.v'],
                                         model=dict(limit=1 << 30))])
        payloads = {'target/module.wasm': b'module', 'evidence/cluster-replay.json': json.dumps(dict(configuration=config)).encode(),
                    'evidence/cluster/history/1/stdout': b'workflow completed [step 0 attempt 2 step 1 attempt 2]\nTemporal inspection canary after workflow timers and activity retries completed'}
        files = {}
        for name, raw in payloads.items():
            path = root / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(raw)
            files[name] = dict(bytes=len(raw), sha256=gate.digest(path))
        manifest = dict(schema='gomad.artifact-manifest/v2', files=files,
                        record=dict(seed=7, outcome=dict(class_='unused'), streams=copy.deepcopy(files)))
        manifest['record']['outcome'] = {'class': 'target_failure'}
        (root / 'manifest.json').write_text(json.dumps(manifest))
        return root, manifest, config

    def test_authenticated_complete_onebox_output_and_trace_bound(self):
        artifact, _, _ = self.artifact()
        gate.authenticate_payloads(artifact)
        (artifact / 'evidence/cluster/history/1/stdout').write_bytes(b'truncated')
        with self.assertRaisesRegex(ValueError, 'identity changed'):
            gate.authenticate_payloads(artifact)

    def test_missing_workflow_completion_witness_is_refused_after_hash_binding(self):
        artifact, manifest, _ = self.artifact()
        path = artifact / 'evidence/cluster/history/1/stdout'
        path.write_bytes(b'Temporal inspection canary after workflow timers and activity retries completed')
        for entry in (manifest['files'][str(path.relative_to(artifact))], manifest['record']['streams'][str(path.relative_to(artifact))]):
            entry.update(bytes=path.stat().st_size, sha256=gate.digest(path))
        (artifact / 'manifest.json').write_text(json.dumps(manifest))
        with self.assertRaisesRegex(ValueError, 'preceded completed'):
            gate.authenticate_payloads(artifact)

    def test_replay_requires_exact_authenticated_target_failure(self):
        lane = gate.Gate(self.root, Path('/release'), self.root)
        matched = b'gomad: reproduced=true diagnostic=false result=target_failure choice-replay=exact\n'
        with mock.patch.object(lane, 'run', return_value=matched) as run:
            lane.replay('replay', self.root / 'artifact')
            self.assertEqual(run.call_args.args[2:4], (1, 1800))
            self.assertEqual(run.call_args.kwargs, dict(detached=True))
        for output in (b'reproduced=true choice-replay=exact', matched + b'ignored divergence', matched.replace(b'exact', b'none')):
            with self.subTest(output=output), mock.patch.object(lane, 'run', return_value=output), self.assertRaises(ValueError):
                lane.replay('replay', self.root / 'artifact')

    def test_retained_trace_capacity_cannot_fall_back_to_eight_mib(self):
        artifact, manifest, config = self.artifact()
        config['incarnations'][0]['model']['limit'] = 8 << 20
        path = artifact / 'evidence/cluster-replay.json'
        path.write_text(json.dumps(dict(configuration=config)))
        for entry in (manifest['files'][str(path.relative_to(artifact))], manifest['record']['streams'][str(path.relative_to(artifact))]):
            entry.update(bytes=path.stat().st_size, sha256=gate.digest(path))
        (artifact / 'manifest.json').write_text(json.dumps(manifest))
        with self.assertRaisesRegex(ValueError, 'trace bound'):
            gate.authenticate_payloads(artifact)

    def test_failure_selection_excludes_synthetic_preparation(self):
        artifact, manifest, _ = self.artifact()
        prepared = self.root / 'prepared'
        prepared.mkdir()
        (prepared / 'manifest.json').write_text(json.dumps(dict(record=dict(outcome={'class': 'prepared'}))))
        self.assertEqual(gate.failure_artifact(self.root), artifact)
        manifest['record']['outcome']['class'] = 'success'
        (artifact / 'manifest.json').write_text(json.dumps(manifest))
        with self.assertRaisesRegex(ValueError, 'unexpected retained'):
            gate.failure_artifact(self.root)

    def test_private_native_capture_binds_real_pid_and_verifies_same_snapshot(self):
        private = {'frozen': 'context'}
        lane = gate.Gate(self.root, Path(sys.executable), self.root, private)
        observations = mock.Mock()
        observations.from_command.return_value = {'sealed': 'actual preparation'}
        command = [sys.executable, '-c', 'print("{}")']
        with mock.patch.object(gate.producer, 'observation_module', return_value=observations, create=True):
            lane.run('explore', command, 0, 10, json_output=True)
            row = json.loads((self.root / 'explore.command.json').read_bytes())
            self.assertGreater(row['pid'], 0)
            observations.from_command.assert_called_once_with(private, row['pid'], command, b'{}\n')
            self.assertEqual(row['private_observations'], {'sealed': 'actual preparation'})
            lane.verify_private()
            observations.verify_snapshot.assert_called_once_with(private, row['pid'], row['private_observations'])
            observations.verify_snapshot.side_effect = ValueError('mutated private raw bytes')
            with self.assertRaisesRegex(ValueError, 'mutated private'):
                lane.verify_private()

    def test_private_capture_refusal_keeps_real_exit_and_prevents_acceptance(self):
        lane = gate.Gate(self.root, Path(sys.executable), self.root, {'frozen': 'context'})
        observations = mock.Mock()
        observations.from_command.side_effect = ValueError('foreign source')
        with mock.patch.object(gate.producer, 'observation_module', return_value=observations, create=True):
            with self.assertRaisesRegex(ValueError, 'private preparation capture refused'):
                lane.run('explore', [sys.executable, '-c', 'print("{}")'], 0, 10)
        row = json.loads((self.root / 'explore.command.json').read_bytes())
        self.assertEqual(row['exit'], 0)
        self.assertEqual(row['private_observations_error'], 'foreign source')
        self.assertNotIn('private_observations', row)

    def test_private_configuration_refusal_precedes_any_root_or_tool_mutation(self):
        root = self.root / 'new-root'
        arguments = ['onebox-inspection-check.py']
        for name, path in [('binary', Path(sys.executable)), ('repo', self.root), ('root', root),
                           ('compiler', self.root), ('runtime-root', self.root), ('cache', self.root),
                           ('captured-config', self.root), ('private-observation-config', self.root / 'config')]:
            arguments += ['--' + name, str(path)]
        with mock.patch.object(sys, 'argv', arguments), mock.patch.object(gate.producer, 'private_configuration',
                side_effect=ValueError('frozen candidate mismatch'), create=True):
            with self.assertRaisesRegex(ValueError, 'frozen candidate mismatch'):
                gate.main()
        self.assertFalse(root.exists())

    def test_only_existing_qualification_controls_survive_environment_filter(self):
        controls = dict(GOMAD_WASM_QUALIFICATION_GUEST_PROFILE='full', GOMAD_WASM_TEMPORAL_BUILD_CACHE='/frozen/cache',
                        GOMAD_WASM_TEMPORAL_MIN_AVAILABLE_BYTES='1024')
        with mock.patch.dict(gate.os.environ, dict(controls, GOMAD_WASM_QUALIFY_WORKLOADS='1', GOMAD_UNRELATED='unadmitted')):
            lane = gate.Gate(self.root, Path('/release'), self.root)
        self.assertEqual({key: value for key, value in lane.environment.items() if key.startswith('GOMAD_')}, controls)

    def test_original_selection_scenario_policy_and_zero_caller_override(self):
        command = list(map(str, gate.explore_command(Path('/release'), Path('/installation'), self.root,
                       Path('/repo'), Path('/repo/service/history/private'), Path('/scenario'))))
        for flag, value in (('--execution-timeout', '30m'), ('--overall-timeout', '45m'),
                            ('--terminate-grace', '1s'), ('--parallel', '1'), ('--output-limit', str(1 << 20)),
                            ('--world-transition-limit', str(1 << 20))):
            self.assertEqual(command[command.index(flag) + 1], value)
        self.assertNotIn('--choices', command)
        self.assertNotIn('--choice-bytes', command)
        self.assertEqual(command[-3:], ['--', '-test.run=^TestReplayInspectionWorkflowFailure$', '-test.v'])
        self.assertEqual([command[i + 1] for i, value in enumerate(command) if value == '--build-tag'], gate.workloads.TAGS + ['gomad'])
        scenario = gate.workloads.inspection_scenario(command[-2:])
        self.assertEqual(scenario['spec']['nodes'], [dict(id='history', boot='main', address='127.0.0.1')])
        self.assertEqual([item['action'] for item in scenario['actions']], ['start', 'wait'])
        policy = gate.workloads.inspection_policy('onebox', {'authenticated': 'schema'})
        self.assertEqual(policy['EntropyKey'], [0] * 32)
        self.assertEqual(policy['Limits'], dict(OutputBytes=16 << 20, TranscriptBytes=256 << 20, Calls=1 << 28,
                                              Descriptors=4096, Files=100000, FilesystemBytes=512 << 20, PendingEvents=4096))
        self.assertEqual(policy['network_v2']['limits']['connection_bytes'], 4 << 20)

    def test_stages_every_original_prototype_go_file_unchanged(self):
        repo = Path(__file__).resolve().parents[4]
        target = self.root / 'stage'
        manifest = gate.workloads.stage_inspection(repo, target, 'onebox')
        self.assertTrue(manifest)
        original = repo / gate.workloads.WASM / 'testdata/prototype_temporal'
        self.assertEqual(sorted(path.name for path in target.iterdir()), sorted(path.name for path in original.glob('*.go')))
        for path in target.iterdir():
            self.assertEqual(path.read_bytes(), (original / path.name).read_bytes())

    def test_failed_and_timed_out_commands_keep_real_exits_and_logs(self):
        lane = gate.Gate(self.root, Path(sys.executable), self.root)
        for name, code, timeout in [('failure', 'import sys; sys.exit(9)', 10), ('timeout', 'import time; time.sleep(60)', 0.05)]:
            with self.subTest(name=name), self.assertRaises(ValueError):
                lane.run(name, [sys.executable, '-c', code], 0, timeout)
            receipt = json.loads((self.root / (name + '.command.json')).read_text())
            self.assertEqual(receipt['exit'], 9 if name == 'failure' else -9)
            self.assertEqual(receipt['timed_out'], name == 'timeout')
            self.assertEqual(receipt['stdout_sha256'], gate.digest(self.root / (name + '.stdout')))
            self.assertEqual(receipt['environment_intent'], dict(mandatory_workloads=False, detached=False))
            self.assertEqual((self.root / (name + '.stdout')).stat().st_mode & 0o777, 0o600)
            self.assertEqual((self.root / (name + '.stderr')).stat().st_mode & 0o777, 0o600)
            self.assertIn('started', receipt)
            self.assertIn('ended', receipt)

    def test_native_command_stays_in_outer_session_and_timeout_kills_descendants(self):
        command = [sys.executable, '-c', 'import subprocess,sys,time; child=subprocess.Popen([sys.executable,"-c","import time; time.sleep(60)"], start_new_session=True); print(child.pid, flush=True); time.sleep(60)']
        process = subprocess.Popen(command, stdout=subprocess.PIPE)
        try:
            child_pid = int(process.stdout.readline())
            self.assertEqual(os.getsid(process.pid), os.getsid(0))
            gate.kill_tree(process.pid)
            self.assertEqual(process.wait(timeout=5), -9)
            status = subprocess.run(['/bin/ps', '-o', 'stat=', '-p', str(child_pid)], capture_output=True, check=False).stdout.strip()
            self.assertTrue(not status or status.startswith(b'Z'), status)
        finally:
            if process.poll() is None:
                gate.kill_tree(process.pid)
                process.wait(timeout=5)
            process.stdout.close()

    def test_alias_detachment_preserves_originals_and_retained_module(self):
        original = self.root / 'original'
        original.mkdir()
        (original / 'source').write_bytes(b'unchanged')
        alias = self.root / 'alias'
        gate.copy_alias(original, alias)
        alias.rename(self.root / 'alias-unavailable')
        self.assertFalse(alias.exists())
        self.assertEqual((original / 'source').read_bytes(), b'unchanged')
        self.assertEqual((self.root / 'alias-unavailable/source').read_bytes(), b'unchanged')

    def test_inventory_rejects_symlink_and_detects_payload_mutation(self):
        (self.root / 'payload').write_bytes(b'original')
        before = gate.inventory(self.root)
        (self.root / 'payload').write_bytes(b'changed')
        self.assertNotEqual(gate.inventory(self.root), before)
        (self.root / 'link').symlink_to(self.root / 'payload')
        with self.assertRaises(ValueError):
            gate.inventory(self.root)


if __name__ == '__main__':
    unittest.main()
