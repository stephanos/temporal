#!/usr/bin/env python3
"""Synthetic receiver tests; these earn no executed stress or qualification credit."""
import importlib.util
from pathlib import Path
import unittest
import tempfile
import hashlib
import copy

PATH = Path(__file__).with_name('stress-parity-lane.py')


class StressTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        spec = importlib.util.spec_from_file_location('stress_parity_lane', PATH)
        cls.lane = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(cls.lane)

    def fixture(self):
        cells = []
        references = []
        for name, seed in self.lane.cases():
            references.append(dict(workload=name, seed=seed))
            for axis in self.lane.AXES:
                skipped = axis == 'handoff' and name != 'two-cluster'
                count = 4 if axis == 'concurrent' else 2
                cells.append(dict(workload=name, seed=seed, axis=axis, rounds=2,
                    fanout=2 if axis == 'concurrent' else 1, requested=count,
                    completed=0 if skipped else count, status='skipped' if skipped else 'pass',
                    runs=[] if skipped else [dict(index=i) for i in range(count)],
                    comparisons=[] if skipped else [dict(schema='gomad.wasm-first-divergence/v1', status='identical') for _ in range(count)],
                    reasons=['inapplicable: no cluster baton or remote peer replies in a standalone guest'] if skipped else []))
        return dict(schema='gomad.rust-stress/v1', passed=True, rejected=[], references=references, cells=cells)

    def test_complete_original_matrix(self):
        report = self.fixture()
        self.assertEqual(self.lane.matrix(report, dict(repetitions=2, parallel=2)), (41, 164, 126))

    def test_matrix_refuses_missing_duplicate_unrun_changed_axis_and_short_cohort(self):
        for name, mutate in (
            ('missing', lambda r: r['cells'].pop()),
            ('duplicate', lambda r: r['cells'].__setitem__(-1, r['cells'][0])),
            ('unrun', lambda r: r.__setitem__('passed', False)),
            ('axis', lambda r: r['cells'][0].__setitem__('axis', 'other')),
            ('short', lambda r: r['cells'][1].__setitem__('completed', 2)),
            ('reference', lambda r: r['references'].pop()),
            ('diagnostics', lambda r: r['cells'][0]['comparisons'][0].__setitem__('status', 'undiagnosable')),
            ('skip', lambda r: r['cells'][0].__setitem__('status', 'skipped')),
            ('extra-skip', lambda r: r['cells'][3]['runs'].append({})),
            ('rounds', lambda r: r['cells'][0].__setitem__('rounds', 1)),
        ):
            with self.subTest(name=name):
                report = self.fixture()
                mutate(report)
                with self.assertRaises(ValueError):
                    self.lane.matrix(report, dict(repetitions=2, parallel=2))

    def owner(self):
        return dict(limits=dict(memory_bytes=4 << 30, fuel=1 << 50, output_bytes=256 << 20),
            wall_nanos=900_000_000_000, namespace=dict(limits=dict(output_bytes=256 << 20,
                files=100000, descriptors=4096, filesystem_bytes=512 << 20)),
            model=dict(mode=1, diagnostics=True, limit=32 << 20, read_step_nanos=0,
                limits=dict(transcript_bytes=1 << 30, calls=1 << 28, pending_events=4096)))

    def test_original_workflow_bounds_and_forced_roles(self):
        owner = self.owner()
        self.lane.original_bounds(owner, 'workflow')
        for mutate in (lambda v: v['limits'].__setitem__('fuel', 1 << 51),
                       lambda v: v['model'].__setitem__('mode', 2),
                       lambda v: v['model'].__setitem__('diagnostics', False),
                       lambda v: v['model'].__setitem__('seed', 7),
                       lambda v: v['namespace']['limits'].__setitem__('files', 100001)):
            changed = copy.deepcopy(owner)
            mutate(changed)
            with self.assertRaises(ValueError):
                self.lane.original_bounds(changed, 'workflow')

    def log(self):
        frames = [dict(type='header', schema='gomad.wasm-event-order/v1'),
            dict(type='event', order=1, kind='clock', instance='main', incarnation=1,
                 ordinal=0, trace=0, decision=0, time_nanos=1, value=dict(clock=1))]
        raw = b''.join(b'gomad-wasm-event-order-v1 ' + self.lane.encode(row) + b'\n' for row in frames)
        return raw + b'gomad-wasm-event-order-v1 ' + self.lane.encode(dict(type='complete', events=1, bytes=len(raw))) + b'\n'

    def test_runtime_oracle_recomputes_hooks_and_collector_witnesses(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory).resolve()
            text = b'entropy map runq timer callbacks timer parking note negative park wake collector automatic finalizer cleanup 1 1 canary '
            diagnostic, trace = bytearray(288), bytearray(288)
            for ordinal, kind in enumerate((1, 2, 3)):
                at = ordinal * 96
                diagnostic[at:at + 8] = ordinal.to_bytes(8, 'big')
                diagnostic[at + 24:at + 28] = (1).to_bytes(4, 'big')
                for counter in (16, 40, 56, 64, 80):
                    diagnostic[at + counter:at + counter + 8] = (1).to_bytes(8, 'big')
                trace[at:at + 8] = ordinal.to_bytes(8, 'big')
                trace[at + 8] = kind
                trace[at + 9] = 2 if kind == 3 else 1
                trace[at + 12:at + 16] = (1).to_bytes(4, 'big')
                if kind == 3:
                    trace[at + 11] = 1
                else:
                    trace[at + 32:at + 96] = bytes([1]) * 64
            files = {'output/stdout': text, 'evidence/choice': bytes(trace), 'evidence/diagnostic': bytes(diagnostic)}
            for name, raw in files.items():
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(raw)
            payload = lambda name: root / name
            self.lane.runtime_oracle(payload)
            for label, at, changed in (('flags', 9, 0), ('readiness', 11, 1), ('identity', 32, 0),
                                       ('origin', 192 + 39, 99), ('creation', 8, 4)):
                with self.subTest(choice=label):
                    malformed = bytearray(trace)
                    if label == 'identity':
                        malformed[32:64] = bytes(32)
                    else:
                        malformed[at] = changed
                    with self.assertRaises(ValueError):
                        self.lane.choices(malformed)
            for name, raw in (('output/stdout', b'canary '), ('evidence/diagnostic', b''), ('evidence/choice', bytes(288))):
                with self.subTest(name=name):
                    path = root / name
                    path.write_bytes(raw)
                    with self.assertRaises(ValueError):
                        self.lane.runtime_oracle(payload)
                    path.write_bytes(files[name])

    def test_select_results_cannot_assign_one_poll_twice(self):
        records = []
        for ordinal, kind in enumerate((2, 3, 3)):
            row = bytearray(96)
            row[:8] = ordinal.to_bytes(8, 'big')
            row[8], row[9] = kind, 1 if kind == 2 else 2
            row[12:16] = (2).to_bytes(4, 'big')
            if kind == 2:
                row[32:96] = bytes([1]) * 64
            else:
                row[11] = 1
                row[20:24] = (2).to_bytes(4, 'big')
            records.append(bytes(row))
        self.assertEqual(self.lane.choices(b''.join(records[:2])), {2, 3})
        with self.assertRaisesRegex(ValueError, 'overlapping'):
            self.lane.choices(b''.join(records))

    def test_typed_spec_only_defaults_optional_scenario(self):
        value = dict(workloads=[dict(name='runtime'), dict(name='two-cluster', scenario='/exact')])
        self.assertEqual(self.lane.typed_stress_spec(value), dict(workloads=[dict(name='runtime', scenario=None), dict(name='two-cluster', scenario='/exact')]))
        self.assertNotIn('scenario', value['workloads'][0])

    def test_first_difference_names_semantic_path_and_values(self):
        self.assertEqual(self.lane.first_difference(dict(events=[dict(value=dict(clock=1))]),
            dict(events=[dict(value=dict(clock=2))])), dict(path='$.events[0].value.clock', expected=1, actual=2))
        self.assertIsNone(self.lane.first_difference(dict(same=1), dict(same=1)))
        self.assertEqual(self.lane.first_difference(dict(output='a'), dict(output='b'))['path'], '$.output')

    def test_complete_event_order_and_first_semantic_difference(self):
        raw = self.log()
        events = self.lane.event_order(raw)
        self.assertEqual(events[0]['value'], dict(clock=1))
        changed = raw.replace(b'"clock":1', b'"clock":2')
        self.assertNotEqual(self.lane.event_order(changed), events)

    def test_event_order_rejects_truncation_extra_frames_and_bad_commitment(self):
        raw = self.log()
        for bad in (raw[:-1], raw + raw, raw.replace(b'"events":1', b'"events":2'),
                    raw.replace(b'"order":1', b'"order":2'), raw.replace(b'"kind":"clock"', b'"kind":"other"')):
            with self.subTest(raw=bad):
                with self.assertRaises(ValueError):
                    self.lane.event_order(bad)

    def artifact_fixture(self, root, *, forced=False, changed_output=False):
        path = root / 'artifact'
        path.mkdir(mode=0o700)
        owner = self.owner()
        owner['limits'].update(wall_millis=900000, evidence_bytes=1 << 30)
        owner.update(environment={}, engine='test-stress-jitter', profile='gomad.wasi-cooperative/v1', arguments=['guest', '-test.v', '-test.count=1', '-test.timeout=72h', '-test.run=^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$', '-persistenceType=sql', '-persistenceDriver=sqlite'])
        if forced:
            owner['model']['mode'] = 2
        identity = dict(configuration=hashlib.sha256(self.lane.encode(owner)).hexdigest(), target=hashlib.sha256(b'fake-module').hexdigest(), toolchain='a' * 64, platform='b' * 64, implementation=hashlib.sha256(self.lane.encode(dict(host='c' * 64, helper='d' * 64))).hexdigest())
        source = 'a' * 64
        payloads = {'output/stdout': b'--- PASS: TestWorkflowTypeEncodingSuite\n' + (b'changed\n' if changed_output else b''),
            'evidence/choice': b'', 'evidence/diagnostic': b'', 'evidence/event-order': self.log(),
            'evidence/replay.json': self.lane.encode(dict(configuration=owner, host_sha256='c' * 64, helper_sha256='d' * 64, toolchain_sha256='a' * 64)),
            'evidence/stress-inputs.json': self.lane.encode(dict(schema='gomad.stress-inputs/v1', fresh=True,
                seed=0, workload='workflow', source_sha256=source))}
        streams = {name: dict(bytes=len(raw), sha256=hashlib.sha256(raw).hexdigest(), records=0)
                   for name, raw in payloads.items()}
        record = dict(schema='gomad.execution-record/v2', seed=0, profile=owner['profile'], arguments=owner['arguments'], environment=owner['environment'], limits=owner['limits'], virtual_time_nanos=0, execution_identity=hashlib.sha256(self.lane.encode(identity)).hexdigest(), identity=identity, streams=streams, outcome=dict(termination='exit', exit_code=0, reason='', **{'class': 'success'}))
        payloads['target/module.wasm'] = b'fake-module'
        payloads['record.json'] = self.lane.encode(record)
        files = {}
        for name, raw in payloads.items():
            item = path / name
            item.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
            for parent in item.parents:
                if parent == path:
                    break
                parent.chmod(0o700)
            item.write_bytes(raw)
            item.chmod(0o600)
            files[name] = dict(bytes=len(raw), sha256=hashlib.sha256(raw).hexdigest(), module=name == 'target/module.wasm')
        record_hash = hashlib.sha256(payloads['record.json']).hexdigest()
        (path / 'manifest.json').write_bytes(self.lane.encode(dict(schema='gomad.artifact-manifest/v2',
            record=record, record_sha256=record_hash, files=files)))
        (path / 'manifest.json').chmod(0o600)
        (path / 'storage.json').write_bytes(b'{"format":"gomad.artifact-storage/v2"}')
        (path / 'storage.json').chmod(0o600)
        return dict(artifact=str(path), record_sha256=record_hash, streams=streams, seed=0,
            workload='workflow', identities=dict(source=source, execution=identity, engine='test-stress-jitter', host='c' * 64, helper='d' * 64))

    def test_authenticates_raw_artifact_and_preserves_semantic_output(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory).resolve()
            first, second = root / 'first', root / 'second'
            first.mkdir()
            second.mkdir()
            a = self.artifact_fixture(first)
            b = self.artifact_fixture(second, changed_output=True)
            self.assertNotEqual(self.lane.artifact(root, a), self.lane.artifact(root, b))
            (Path(a['artifact']) / 'output/stdout').write_bytes(b'tampered')
            with self.assertRaisesRegex(ValueError, 'payload changed'):
                self.lane.artifact(root, a)

    def reseal_record(self, observed, mutate):
        path = Path(observed['artifact'])
        manifest = self.lane.read(path / 'manifest.json')
        mutate(manifest)
        raw = self.lane.encode(manifest['record'])
        (path / 'record.json').write_bytes(raw)
        manifest['files']['record.json'].update(bytes=len(raw), sha256=hashlib.sha256(raw).hexdigest())
        manifest['record_sha256'] = observed['record_sha256'] = hashlib.sha256(raw).hexdigest()
        observed['streams'] = manifest['record']['streams']
        observed['identities']['execution'] = manifest['record']['identity']
        (path / 'manifest.json').write_bytes(self.lane.encode(manifest))

    def test_rehashed_seed_module_identity_and_unlisted_payload_refused(self):
        for name, mutate in (
            ('missing-field', lambda m: m['record'].pop('profile')),
            ('unknown-field', lambda m: m['record'].__setitem__('extra', True)),
            ('wrong-limit-type', lambda m: m['record']['limits'].__setitem__('fuel', True)),
            ('seed', lambda m: m['record'].__setitem__('seed', 7)),
            ('identity', lambda m: m['record'].__setitem__('execution_identity', '0' * 64)),
            ('module', lambda m: m['files']['target/module.wasm'].__setitem__('module', False)),
            ('unlisted', lambda m: m['files'].pop('evidence/replay.json')),
            ('uncommitted', lambda m: m['record']['streams'].pop('evidence/event-order')),
        ):
            with self.subTest(name=name), tempfile.TemporaryDirectory() as directory:
                root = Path(directory).resolve()
                observed = self.artifact_fixture(root)
                self.reseal_record(observed, mutate)
                with self.assertRaises(ValueError):
                    self.lane.artifact(root, observed)

    def test_unknown_payload_wrong_lineage_and_public_mode_refused(self):
        for name in ('unknown', 'lineage', 'mode'):
            with self.subTest(name=name), tempfile.TemporaryDirectory() as directory:
                root = Path(directory).resolve()
                observed = self.artifact_fixture(root)
                path = Path(observed['artifact'])
                if name == 'unknown':
                    (path / 'other').write_bytes(b'extra')
                elif name == 'lineage':
                    (path / 'storage.json').write_bytes(b'{"format":"gomad.artifact-storage/v1"}')
                else:
                    (path / 'record.json').chmod(0o644)
                with self.assertRaises(ValueError):
                    self.lane.artifact(root, observed)

    def test_rehashed_forced_raw_artifact_refused(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory).resolve()
            observed = self.artifact_fixture(root, forced=True)
            with self.assertRaisesRegex(ValueError, 'fresh diagnostic mode'):
                self.lane.artifact(root, observed)

    def test_wrong_candidate_cycle_synthetic_and_unsealed_config_refused(self):
        base = dict(schema='gomad.stress-cycle/v1', repo=str(self.lane.REPO), root='/cycle', deadline_s=3600,
            sent_epoch=1, identity=dict(host='a' * 64, helper='b' * 64), binary_sha256='c' * 64,
            source_manifest='/nonexistent-stress-source-manifest', source_sha256='d' * 64, stress={}, files={})
        for change, message in ((dict(schema='other'), 'foreign'), (dict(synthetic=True), 'synthetic'),
            (dict(repo='/wrong'), 'candidate'), (dict(root='/wrong'), 'cycle'),
            (dict(deadline_s=3601), 'deadline'), ({}, 'input')):
            config = {**base, **change}
            with self.subTest(config=config):
                with self.assertRaisesRegex(ValueError, message):
                    self.lane.read_cycle(config, Path('/cycle'))

    def test_changed_bytes_and_symlink_custody_refused(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory).resolve()
            payload = root / 'payload'
            payload.write_bytes(b'original')
            expected = self.lane.sha(payload)
            self.assertEqual(self.lane.pin(payload, expected), payload)
            payload.write_bytes(b'changed')
            with self.assertRaisesRegex(ValueError, 'changed'):
                self.lane.pin(payload, expected)
            alias = root / 'alias'
            alias.symlink_to(payload)
            with self.assertRaisesRegex(ValueError, 'nonregular'):
                self.lane.regular(alias)
            with self.assertRaisesRegex(ValueError, 'outside'):
                self.lane.within(root / 'other', payload)

    def test_duplicate_json_and_nonfinite_refused(self):
        for raw in (b'{"passed":true,"passed":false}', b'{"passed":NaN}'):
            with self.assertRaises(ValueError):
                self.lane.decode(raw)


if __name__ == '__main__':
    unittest.main()
