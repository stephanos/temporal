"""Small raw-file negative causality fixtures; no qualification or guest execution."""
import copy
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import driver
import final_retained as gate


class FinalRetained(unittest.TestCase):
    def test_original_ringpop_rotation_is_only_three_deadlines(self):
        original = driver.read(gate.MODULE / 'internal/backends/wasm/command/testdata/scenario/ringpop.json')
        expected = copy.deepcopy(original)
        for boot in expected['boots']:
            boot['args'][4] = '120'
        self.assertEqual(gate.scenario(), expected)
        self.assertEqual([boot['args'][4] for boot in original['boots']], ['90'] * 3)

    def test_bound_is_checked_before_payload_read(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'payload'
            path.write_bytes(b'12345')
            self.assertEqual(gate.bounded(path, 5), b'12345')
            with self.assertRaisesRegex(driver.Refusal, 'bound'):
                gate.bounded(path, 4)
            path.unlink()
            path.symlink_to('/dev/null')
            with self.assertRaises(driver.Refusal):
                gate.bounded(path, 5)

    def exercise_observed(self, mutation=None):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            artifact = root / 'artifact'
            (artifact / 'output').mkdir(parents=True)
            (artifact / 'evidence').mkdir()
            (artifact / 'evidence/backend-outcome.json').write_bytes(b'{"type":"result","termination":"exit","exit_code":1}')
            (artifact / 'output/stdout').write_bytes(b'deterministic failure after reading "pending"')
            (artifact / 'output/stderr').write_bytes(b'')
            record = dict(profile=gate.PROFILES['stock'], streams={'output/stdout': {}}, outcome={'class': 'target_failure'})
            expected = {'stdout': (artifact / 'output/stdout').read_bytes(), 'stderr': b'',
                        'backend-evidence.bin': (artifact / 'evidence/backend-outcome.json').read_bytes()}
            calls = []
            class Run:
                def call(self, name, argv, status=0, installation=None, timeout=180):
                    calls.append((name, argv, status))
                    if name == 'explore': return b''
                    observed = Path(argv[argv.index('--observed') + 1])
                    observed.mkdir(parents=True)
                    number = int(name.split('-')[1])
                    for filename, raw in expected.items():
                        if mutation == 'missing' and number == 20 and filename == 'stderr': continue
                        (observed / filename).write_bytes(raw + (b'changed' if mutation == 'raw' and number == 20 else b''))
                    driver.write(root / (name + '.command.json'), dict(pid=1 if mutation == 'pid' else number))
                    if mutation == 'immutable' and number == 20:
                        (artifact / 'output/stdout').write_bytes(b'changed')
                    return b'reproduced=true diagnostic=false result=target_failure choice-replay=none'
            with patch.object(gate, 'install', return_value=(Run(), {'source': root / 'source'}, root / 'installation')), \
                    patch.object(gate, 'artifact', return_value=(artifact, record)), patch.object(gate, 'detach'):
                result = gate.retained({}, root)
            self.assertEqual(len(result['replay_processes']), 20)
            self.assertEqual([status for _, _, status in calls], [1] * 21)
            self.assertIn('-test.run=^TestDeterministicIOFailure$', calls[0][1])
            self.assertIn('--seeds=7', calls[0][1])
            self.assertEqual([name for name, _, _ in calls[1:]], [f'replay-{number:02d}' for number in range(1, 21)])

    def test_twenty_distinct_fresh_observed_processes_and_original_selector(self):
        self.exercise_observed()

    def test_last_replay_changed_raw_missing_payload_reused_pid_or_mutation_refuses(self):
        for mutation in ('raw', 'missing', 'pid', 'immutable'):
            with self.subTest(mutation=mutation), self.assertRaises((driver.Refusal, OSError)):
                self.exercise_observed(mutation)

    def test_retained_original_clock_limits_and_environment_are_exact(self):
        guest = dict(Profile=gate.PROFILES['stock'], Environment=['PWD=/workspace'], WorkingDirectory='/workspace',
            WritableDirectories=['/workspace', '/tmp'], Clock=dict(EpochNanos=946684800000000000, ReadStepNanos=1000),
            Limits=dict(OutputBytes=1 << 20, TranscriptBytes=8 << 20, Calls=10000, Descriptors=64, Files=1000,
                        FilesystemBytes=8 << 20, PendingEvents=4096))
        with patch.object(gate, 'retained_capture'):
            gate.retained_guest(guest)
            for field in ('Calls', 'OutputBytes', 'TranscriptBytes', 'FilesystemBytes'):
                changed = copy.deepcopy(guest)
                changed['Limits'][field] *= 2
                with self.subTest(field=field), self.assertRaises(driver.Refusal):
                    gate.retained_guest(changed)
            changed = copy.deepcopy(guest)
            changed['Clock']['ReadStepNanos'] = 0
            with self.assertRaises(driver.Refusal): gate.retained_guest(changed)
            changed = copy.deepcopy(guest)
            changed['Environment'].append('UNDECLARED=1')
            with self.assertRaises(driver.Refusal): gate.retained_guest(changed)

    def test_operator_candidate_configuration_and_profile_capabilities_are_independent(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'evidence').mkdir()
            identity = dict(host='1' * 64, helper='2' * 64)
            configured = dict(identity=identity)
            inputs = dict(configuration={'model': 'original'}, host_sha256=identity['host'], helper_sha256=identity['helper'])
            record = dict(seed=7, identity=dict(implementation=gate.receiving.digest(gate.CANONICAL(identity)),
                configuration=gate.receiving.digest(gate.CANONICAL(inputs['configuration']))))
            (root / 'manifest.json').write_text(json.dumps({'files': {'evidence/replay.json': {}, 'evidence/inspection.json': {}}}))
            (root / 'evidence/replay.json').write_text(json.dumps(inputs))
            (root / 'evidence/inspection.json').write_text(json.dumps({'choices': {'profile': 'gomad-choice-trace/v4'}}))
            gate.candidate(configured, root, record)
            gate.operator_capability(root, 'cooperative')
            for field in ('implementation', 'configuration'):
                changed = copy.deepcopy(record)
                changed['identity'][field] = '0' * 64
                with self.subTest(field=field), self.assertRaises(driver.Refusal): gate.candidate(configured, root, changed)
            with self.assertRaises(driver.Refusal): gate.operator_capability(root, 'stock')
            (root / 'evidence/inspection.json').write_text(json.dumps({'choices': None}))
            gate.operator_capability(root, 'stock')
            with self.assertRaises(driver.Refusal): gate.operator_capability(root, 'cooperative')

    def test_ringpop_actual_world_network_and_each_guest_trace_need_earlier_reference(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'evidence').mkdir()
            reference = {'choices': {}}
            inputs = {'retained': []}
            for name in ('world', 'network'):
                (root / ('reference-' + name)).write_bytes(name.encode())
                (root / 'evidence' / name).write_bytes(name.encode())
                reference[name] = str(root / ('reference-' + name))
            for node in ('a', 'b', 'c'):
                raw = node.encode() * 96
                (root / ('reference-' + node)).write_bytes(raw)
                (root / 'evidence' / node).write_bytes(raw)
                reference['choices'][node] = str(root / ('reference-' + node))
                inputs['retained'].append(dict(handle=dict(node=node), evidence=dict(choice=dict(file='evidence/' + node))))
            with patch.object(gate, 'ringpop_reference', return_value=reference):
                gate.ringpop_preservation({}, root, inputs)
                for name in ('world', 'network', 'a', 'b', 'c'):
                    target = root / 'evidence' / name
                    original = target.read_bytes()
                    target.write_bytes(b'deterministic changed behavior')
                    with self.subTest(name=name), self.assertRaises(driver.Refusal):
                        gate.ringpop_preservation({}, root, inputs)
                    target.write_bytes(original)
        with self.assertRaisesRegex(driver.Refusal, 'missing original ringpop'):
            gate.ringpop_reference({'final_retained': {}})

    def test_original_source_manifest_keeps_bodies_and_explicit_bounds(self):
        import hashlib
        declaration = driver.read(driver.HERE / 'final-retained-original.json')
        self.assertEqual([row['section'] for row in declaration['originals']], ['retained', 'operator'])
        for row in declaration['originals']:
            self.assertEqual(hashlib.sha256(row['original_body'].encode()).hexdigest(), row['body_sha256'])
        self.assertIn('artifactFreshReplays', declaration['originals'][0]['original_body'])
        self.assertEqual(declaration['inspection']['fuel'], 1 << 50)
        self.assertEqual(declaration['inspection']['execution_timeout'], '40m')

    def test_private_compiler_pins_derive_only_from_authenticated_staged_inventory(self):
        config = dict(root='/attempt', files={'/sealed/compiler/bin/go': 'original'}, identity={})
        reader = gate.PrivateRetained(config)
        selected = {name: Path('/attempt/private') / name for name in ('compiler_root', 'runtime_root', 'source')}
        before = {name: {'input': {'sha256': name + '-hash', 'bytes': 1}} for name in selected}
        reader.admit_private(selected, before)
        observed = []
        def verify(instance, result):
            observed.append(instance.config['files'])
            return {'actual': result}
        with patch.object(gate.receiving.Retained, 'provenance', verify):
            self.assertEqual(reader.provenance('raw-result'), {'actual': 'raw-result'})
        self.assertEqual(observed[0]['/attempt/private/compiler_root/input'], 'compiler_root-hash')
        self.assertNotIn('/attempt/private/compiler_root/input', config['files'])
        self.assertNotIn('/attempt/foreign/compiler/bin/go', observed[0])
        with self.assertRaises(driver.Refusal): reader.admit_private(selected, before)

    def test_stage_producer_runs_once_and_final_reader_rejects_changed_command_bytes(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            work = root / 'final-retained'
            config = dict(root=directory, source_sha256='source', identity={'host': 'host'})
            calls = []
            def lane(config, destination, reader=None):
                calls.append(reader is None)
                if reader is None:
                    destination.mkdir()
                    (destination / 'explore.stdout').write_bytes(b'actual fixture command output')
                return {'original': 'witness'}
            with patch.object(gate, 'guard'), patch.object(gate, 'sealed'), patch.dict(gate.READERS, retained=lane):
                first = gate.collect(config, root, 'retained')
                self.assertEqual(gate.collect(config, root, 'retained', read_only=True), first)
                self.assertEqual(calls, [True, False, False])
                (work / 'explore.stdout').write_bytes(b'changed raw command output')
                with self.assertRaisesRegex(driver.Refusal, 'differs from actual producer'):
                    gate.collect(config, root, 'retained', read_only=True)

    def test_final_read_only_requires_complete_proofs_and_never_creates_output(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            config = dict(root=directory)
            with patch.object(gate, 'guard'), patch.object(gate, 'sealed'):
                with self.assertRaisesRegex(driver.Refusal, 'missing original final producer'):
                    gate.collect(config, root, 'retained', read_only=True)
                self.assertFalse((root / 'final-retained').exists())
                (root / 'final-retained').mkdir()
                with self.assertRaisesRegex(driver.Refusal, 'incomplete original final producer'):
                    gate.collect(config, root, 'retained', read_only=True)

    def test_missing_section_cannot_execute_or_return_source_credit(self):
        with tempfile.TemporaryDirectory() as directory:
            config = dict(root=directory, final_retained={})
            with patch.object(gate, 'guard'), patch.object(gate.live, 'role'), \
                    patch.object(gate, 'retained') as producer, self.assertRaises(KeyError):
                gate.collect(config, Path(directory), 'retained')
            producer.assert_not_called()

    def test_source_changed_or_foreign_root_refuses_before_producer(self):
        with patch.object(gate, 'guard', side_effect=driver.Refusal('source changed')), \
                patch.object(gate, 'sealed') as seal, self.assertRaises(driver.Refusal):
            gate.collect({}, Path('/foreign'), 'retained')
        seal.assert_not_called()
        with patch.object(gate, 'guard'), patch.object(gate, 'sealed') as seal, self.assertRaises(driver.Refusal):
            gate.collect(dict(root='/actual'), Path('/foreign'), 'retained')
        seal.assert_not_called()


if __name__ == '__main__':
    unittest.main()
