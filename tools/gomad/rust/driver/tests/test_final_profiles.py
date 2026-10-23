import copy
import hashlib
import json
from pathlib import Path
import sys
import struct
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).parents[1]))
import final_profiles as gate
from driver import Refusal


class FinalProfiles(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name).resolve()

    def run_evidence(self, prefix):
        run = dict(reaped=True, termination='exit', exit_code=0, calls=5,
                   time_advances=2, environment_identity='original', fuel_remaining=100,
                   authorization_to_terminal_nanos=10, select_poll=100000, streams={})
        for name in gate.STREAMS:
            raw = (b'runtime-cost 100000\n' if name == 'stdout' else bytes(16) if name == 'terminal' else b'original')
            path = self.root / (prefix + '-' + name)
            path.write_bytes(raw)
            run['streams'][name] = dict(path=str(path), sha256=hashlib.sha256(raw).hexdigest())
        return run

    def runtime_spec(self):
        return dict(profile='gomad.wasi-cooperative/v1', clock_read_step_nanos=0, seed=7,
                    package='./internal/model/wasi/testdata/runtime_cost',
                    fuel=1 << 40, memory_bytes=256 << 20, wall_nanos=180_000_000_000,
                    calls=1 << 20, transcript_bytes=64 << 20, choice_bytes=64 << 20,
                    runtime_diagnostics=False)

    def test_original_cpu_bound_and_features(self):
        raw = b'Features: fp asimd\nCPU architecture: 8\n'
        at_bound = raw + b'\n' * ((1 << 20) - len(raw))
        gate.host_admission('linux', 'arm64', at_bound)
        for os, arch, data in [('darwin', 'arm64', raw), ('linux', 'amd64', raw),
                               ('linux', 'arm64', at_bound + b'\n'),
                               ('linux', 'arm64', raw.replace(b'asimd', b'no-asimd')),
                               ('linux', 'arm64', raw.replace(b': 8', b': 9'))]:
            with self.subTest(os=os, arch=arch, size=len(data)), self.assertRaises(Refusal):
                gate.host_admission(os, arch, data)

    def test_canonical_preservation_refuses_each_changed_original_field(self):
        before, after = self.run_evidence('before'), self.run_evidence('after')
        gate.same_evidence(self.root, before, after, qualified=True)
        for name in gate.STREAMS:
            changed = copy.deepcopy(after)
            path = Path(changed['streams'][name]['path'])
            original = path.read_bytes()
            raw = b'x' * 16 if name == 'terminal' else b'changed'
            path.write_bytes(raw)
            changed['streams'][name]['sha256'] = hashlib.sha256(raw).hexdigest()
            with self.subTest(stream=name), self.assertRaises(Refusal):
                gate.same_evidence(self.root, before, changed, qualified=True)
            path.write_bytes(original)
        for field in ('exit_code', 'termination', 'calls', 'time_advances', 'environment_identity', 'fuel_remaining'):
            changed = copy.deepcopy(after)
            changed[field] = 'changed'
            with self.subTest(field=field), self.assertRaises(Refusal):
                gate.same_evidence(self.root, before, changed, qualified=True)

    def test_runtime_original_bounds_and_witness(self):
        before, after = self.run_evidence('before'), self.run_evidence('after')
        gate.runtime_pair(self.root, self.runtime_spec(), before, after)
        for field in ('seed', 'fuel', 'memory_bytes', 'wall_nanos', 'calls', 'transcript_bytes', 'choice_bytes'):
            spec = self.runtime_spec()
            spec[field] += 1
            with self.subTest(field=field), self.assertRaises(Refusal):
                gate.runtime_pair(self.root, spec, before, after)
        after['select_poll'] = 99999
        with self.assertRaises(Refusal):
            gate.runtime_pair(self.root, self.runtime_spec(), before, after)

    def test_full_activity_selection_and_original_capacity(self):
        spec = dict(profile='gomad.wasi-cooperative/v1', clock_read_step_nanos=0,
                    fuel=1 << 50, memory_bytes=4 << 30, wall_nanos=1_800_000_000_000,
                    package='./tests', working_directory='/workspace/tests', selection=gate.ACTIVITY,
                    arguments=['-test.run=' + gate.ACTIVITY, '-persistenceType=sql', '-persistenceDriver=sqlite'])
        gate.exact_bounds(spec)
        for field, value in [('fuel', 1 << 40), ('memory_bytes', 256 << 20),
                             ('selection', '^TestActivityTestSuite$/TestActivity'),
                             ('package', './testdata/activity')]:
            changed = dict(spec, **{field: value})
            with self.subTest(field=field), self.assertRaises(Refusal):
                gate.exact_bounds(changed)

    def test_original_runtime_trace_accepts_100000_polls_above_generic_one_mib(self):
        raw = bytearray(96 * 100000)
        for ordinal in range(100000):
            struct.pack_into('>QBBHII', raw, ordinal * 96, ordinal, 2, 1, 0, 2, 0)
        self.assertEqual(gate.select_polls(raw), 100000)
        raw[-96] = 1
        with self.assertRaises(Refusal):
            gate.select_polls(raw)

    def test_terminal_requires_actual_post_load_mismatch_and_original_one_second_bound(self):
        success = dict(reaped=True, terminal_to_reap_nanos=1_000_000_000,
                       termination='exit', exit_code=0,
                       stdout='--- PASS: TestWorkflowTypeEncodingSuite TestPlainASCII/Succeeds')
        mismatch = dict(reaped=True, terminal_to_reap_nanos=1, authorization_changed=True,
                        termination='invalid', message='execution authorization identity mismatch',
                        started={}, instantiate_nanos=0, execute_nanos=0)
        gate.terminal_observations([success, mismatch])
        for field, value in [('authorization_changed', False), ('instantiate_nanos', 1),
                             ('execute_nanos', 1), ('started', None),
                             ('message', 'different'), ('terminal_to_reap_nanos', 1_000_000_001)]:
            with self.subTest(field=field), self.assertRaises(Refusal):
                gate.terminal_observations([success, dict(mismatch, **{field: value})])
        with self.assertRaises(Refusal):
            gate.terminal_observations([success] * 6)

    def test_actual_cwd_and_module_provenance_are_required(self):
        module = self.root / 'tests.wasm'
        module.write_bytes(b'actual-module')
        spec = dict(fuel=1 << 50, memory_bytes=4 << 30, wall_nanos=1_800_000_000_000,
                    clock_read_step_nanos=0, seed=0, arguments=['-test.run=' + gate.ACTIVITY],
                    working_directory='/workspace/tests', module=str(module),
                    module_sha256=hashlib.sha256(module.read_bytes()).hexdigest())
        request = dict(kind='standalone', guest=dict(module=str(module), module_sha256=spec['module_sha256'],
                       limits=dict(fuel=spec['fuel'], memory_bytes=spec['memory_bytes'], wall_millis=1800000),
                       model=dict(configuration=dict(seed=0, read_step_nanos=0,
                                  working_directory_sha256=hashlib.sha256(b'/workspace/tests').hexdigest()),
                                  arguments=['guest', *spec['arguments']]),
                       namespace=dict(working_directory=list(b'/workspace/tests'))))
        input_path = self.root / 'input.json'
        def retain(value):
            input_path.write_text(json.dumps(value))
            spec['native_input'] = copy.deepcopy(value)
        retain(request)
        with patch('live.sealed_input'), patch.object(gate, 'preparation_proof') as proof:
            gate.bound_native_request({}, input_path, spec)
            proof.assert_called_once()
        for field in ('model', 'namespace'):
            changed = copy.deepcopy(request)
            if field == 'model':
                changed['guest'][field]['configuration']['working_directory_sha256'] = 'wrong'
            else:
                changed['guest'][field]['working_directory'] = list(b'/other')
            retain(changed)
            with self.subTest(field=field), patch('live.sealed_input'), self.assertRaises(Refusal):
                gate.bound_native_request({}, input_path, spec)
        provenance = self.root / 'provenance.json'
        installation = self.root / 'installation.json'
        provenance.write_text(json.dumps(dict(module_sha256=spec['module_sha256'],
            inputs=dict(kind='go-test', source='./fixtures', directory='/repo'))))
        installation.write_text('{}')
        spec['preparation'] = dict(provenance=str(provenance), installation=str(installation), observations=str(self.root))
        with patch('live.sealed_input'), self.assertRaises(Refusal):
            gate.preparation_proof(dict(repo='/repo'), spec, request['guest'])

    def test_changed_hash_and_foreign_output_are_refused(self):
        run = self.run_evidence('run')
        Path(run['streams']['stdout']['path']).write_bytes(b'changed')
        with self.assertRaises(Refusal):
            gate.evidence(self.root, run)
        run['streams']['stdout']['path'] = '/proc/cpuinfo'
        with self.assertRaises(Refusal):
            gate.evidence(self.root, run)


if __name__ == '__main__':
    unittest.main()
