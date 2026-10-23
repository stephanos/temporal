import copy
import base64
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
import native_roles
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

    def canonical_native(self):
        def put(name, raw):
            path = self.root / name
            path.write_bytes(raw)
            return dict(bytes=len(raw), sha256=hashlib.sha256(raw).hexdigest())
        def jsonfile(name, value):
            return put(name, json.dumps(value).encode())
        native = {name: put(name, raw) for name, raw in dict(stdout=b'actual', transcript=b'binary-wasi', choice=b'choice').items()}
        jsonfile('streams.json', native)
        outcome = dict(local_model=dict(terminal_base64=base64.b64encode(bytes(16)).decode()))
        jsonfile('outcome.json', outcome)
        streams = {}
        for name in gate.STREAMS:
            raw = (self.root / name).read_bytes() if name in native else b'binary-wasi' if name == 'wasi' else bytes(16) if name == 'terminal' else b''
            source = ('native:' + name if name in native else 'binary-transcript-alias:transcript' if name == 'wasi'
                      else 'outcome:local_model.terminal_base64' if name == 'terminal' else 'closed-sink-empty:' + name)
            streams[name] = dict(file=name, source=source, **put(name, raw))
        receipt = dict(schema='gomad.native-canonical-streams/v1',
                       source='finalized linked Execution.evidence from closed Sink::into_streams',
                       transformation='identity bytes; no content normalization',
                       native_streams_sha256=hashlib.sha256((self.root / 'streams.json').read_bytes()).hexdigest(),
                       outcome_sha256=hashlib.sha256((self.root / 'outcome.json').read_bytes()).hexdigest(), streams=streams)
        jsonfile('canonical-streams.json', receipt)
        return outcome, receipt

    def test_explicit_native_canonical_sources_and_empty_files(self):
        outcome, _ = self.canonical_native()
        self.assertEqual(set(gate.native_canonical_streams(self.root, outcome)), set(gate.STREAMS))
        (self.root / 'stderr').unlink()
        with self.assertRaises(Refusal):
            gate.native_canonical_streams(self.root, outcome)

    def test_coherent_native_canonical_forgery_cannot_replace_source(self):
        for name in gate.STREAMS:
            outcome, receipt = self.canonical_native()
            raw = b'forged' if name != 'terminal' else b'x' * 16
            (self.root / name).write_bytes(raw)
            receipt['streams'][name].update(bytes=len(raw), sha256=hashlib.sha256(raw).hexdigest())
            (self.root / 'canonical-streams.json').write_text(json.dumps(receipt))
            with self.subTest(stream=name), self.assertRaises(Refusal):
                gate.native_canonical_streams(self.root, outcome)

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

    def test_reference_projection_recomputes_actual_original_result(self):
        run = self.run_evidence('original')
        run.update(schema='gomad.go-reference-run/v1', owner='go', original_commit='original',
                   error='', cancelled=False, watchdog_timeout=False, transcript_sha256='00' * 32)
        payloads = {name: base64.b64encode(Path(entry['path']).read_bytes()).decode()
                    for name, entry in run['streams'].items()}
        result = dict(Stdout=payloads['stdout'], Stderr=payloads['stderr'], Transcript=payloads['wasi'],
                      TranscriptSHA256=[0] * 32, NetworkTranscript=payloads['network'],
                      ChoiceTrace=dict(Bytes=payloads['choice'], Summary=dict(SelectPoll=run['select_poll'])),
                      Termination='exit', ExitCode=0, Reaped=True, Cancelled=False, WatchdogTimeout=False,
                      Calls=run['calls'], TimeAdvances=run['time_advances'], EnvironmentIdentity=run['environment_identity'],
                      FuelRemaining=run['fuel_remaining'], AuthorizationToTerminalNanos=run['authorization_to_terminal_nanos'])
        raw = self.root / 'raw-result.json'
        raw.write_text(json.dumps(dict(result=result, terminal=payloads['terminal'], error='')))
        (self.root / 'invoke.stdout').write_bytes(raw.read_bytes())
        run['raw_result'] = dict(path=str(raw), sha256=hashlib.sha256(raw.read_bytes()).hexdigest())
        path = self.root / 'run.json'
        path.write_text(json.dumps(run))
        gate.reference_result(self.root, dict(path=str(path)), 'go', 'original')
        for field, value in [('fuel_remaining', 0), ('calls', 999), ('transcript_sha256', 'bad')]:
            path.write_text(json.dumps(dict(run, **{field: value})))
            with self.subTest(field=field), self.assertRaises(Refusal):
                gate.reference_result(self.root, dict(path=str(path)), 'go', 'original')
        changed = copy.deepcopy(run)
        result['Calls'] = 999
        raw.write_text(json.dumps(dict(result=result, terminal=payloads['terminal'], error='')))
        changed['calls'] = 999
        changed['raw_result']['sha256'] = hashlib.sha256(raw.read_bytes()).hexdigest()
        path.write_text(json.dumps(changed))
        with self.assertRaises(Refusal):
            gate.reference_result(self.root, dict(path=str(path)), 'go', 'original')
        result['Calls'] = run['calls']
        raw.write_bytes((self.root / 'invoke.stdout').read_bytes())
        path.write_text(json.dumps(run))
        Path(run['streams']['stdout']['path']).write_bytes(b'forged')
        with self.assertRaises(Refusal):
            gate.reference_result(self.root, dict(path=str(path)), 'go', 'original')

    def test_terminal_clock_includes_child_publication_and_refuses_missing_reap(self):
        report = dict(terminal_monotonic_nanos=100, terminal_clock='CLOCK_MONOTONIC', terminal_to_return_nanos=20)
        journal = dict(reaped_monotonic_nanos=1_000_000_100)
        self.assertEqual(gate.terminal_clock(report, journal), 1_000_000_000)
        for reaped in (None, 99, 100, 1_000_000_101):
            with self.subTest(reaped=reaped), self.assertRaises(Refusal):
                gate.terminal_clock(report, dict(reaped_monotonic_nanos=reaped))
        with self.assertRaises(Refusal):
            gate.terminal_clock(dict(report, terminal_clock='wall'), journal)
        with self.assertRaises(Refusal):
            gate.terminal_clock(dict(report, terminal_to_return_nanos=1_000_000_001), journal)
        refusal = dict(report, terminal_to_return_nanos=None,
                       admission_refusal=dict(terminal_to_return_nanos=20))
        self.assertEqual(gate.terminal_clock(refusal, journal), 1_000_000_000)

    def test_admission_profile_cannot_hide_execution_or_imports(self):
        profile = dict(schema='gomad.wasm-profile/v2', diagnostic_only=True,
                       abi_overlap_errors=0, span_errors=0, unattributed_imports=0, unframed_imports=0,
                       threads_omitted=0, report_rows_dropped=0, abi_ops={}, owners={}, threads=[],
                       abi=dict(clock=dict(count=0)), abi_excluded=dict(count=0),
                       phases={phase: dict(count=0, wall_nanos=0) for phase in ('instantiate', 'execute')},
                       reconciliation=dict(execute_wall_nanos=0, attributed_host_wall_nanos=0,
                           guest_residual_wall_nanos=0, outer_host_wall_nanos=0, difference_nanos=0, tolerance_nanos=50000))
        gate.admission_attribution(profile)
        for field in ('abi_overlap_errors', 'unattributed_imports', 'report_rows_dropped'):
            with self.subTest(field=field), self.assertRaises(Refusal):
                gate.admission_attribution(dict(profile, **{field: 1}))
        for phase in ('instantiate', 'execute'):
            changed = copy.deepcopy(profile)
            changed['phases'][phase]['count'] = 1
            with self.subTest(phase=phase), self.assertRaises(Refusal):
                gate.admission_attribution(changed)

    def test_reference_helper_rejects_unrelated_cargo_artifact_with_coherent_copy_claim(self):
        helper, artifact = self.root / 'helper', self.root / 'cargo-artifact'
        helper.write_bytes(b'candidate')
        artifact.write_bytes(b'unrelated')
        def binding(path):
            return dict(path=str(path), bytes=path.stat().st_size, sha256=hashlib.sha256(path.read_bytes()).hexdigest())
        binary = binding(helper)
        claimed = dict(binary, path=str(artifact))
        receipt = dict(schema='gomad.reference-helper-build/v1', binary=binary,
                       artifact_executable=claimed, cargo_artifact=dict(executable=str(artifact)),
                       copy=dict(method='copyfile', source=claimed, destination=binary),
                       cargo_stdout=binary, command_file=binary, process_file=binary)
        path = self.root / 'receipt.json'
        path.write_text(json.dumps(receipt))
        config = dict(files={str(file): hashlib.sha256(file.read_bytes()).hexdigest() for file in (helper, artifact, path)})
        with self.assertRaisesRegex(Refusal, 'build custody'):
            gate.reference_helper_custody(config, dict(helper_receipt=str(path), helper=str(helper)))

    def test_diagnostic_identity_remains_distinct_and_binds_same_frozen_source(self):
        config = dict(root=str(self.root), source_sha256='source', identity=dict(host='host-release', helper='helper-release'))
        entries, identities = [], []
        for role, features in (('ordinary', []), ('diagnostic-profile', ['diagnostic-profile'])):
            binary = self.root / role
            binary.write_bytes(role.encode())
            receipt = self.root / (role + '.json')
            digest = hashlib.sha256(binary.read_bytes()).hexdigest()
            receipt.write_text('{}')
            entries.append(dict(binary=str(binary), build_receipt=str(receipt)))
            identities.append(dict(schema='gomad.native-measurement-identity/v1', role=role, features=features,
                executable_sha256=digest, host_implementation_sha256='host-release' if not features else 'host-diagnostic',
                linked_implementation_sha256='helper-release' if not features else 'helper-diagnostic',
                source_inputs=dict(host='actual-host-preimage', helper='actual-helper-preimage'),
                runtime_implementation_sha256='runtime', configuration='engine' if not features else 'engine-diagnostic-profile'))
            receipt.write_text(json.dumps(dict(executable=str(binary), executable_sha256=digest, identity=identities[-1])))
        with patch('live.sealed_input'), patch.object(native_roles, 'validate_build', side_effect=lambda config, path, role: json.loads(Path(path).read_text())), patch.object(gate, 'invoke', side_effect=[(json.dumps(value).encode(), {}) for value in identities]):
            gate.sealed_role(config, entries[0], [])
            gate.sealed_role(config, entries[1], ['diagnostic-profile'])
        self.assertEqual(config['_final_profile_role_identities'][entries[1]['binary']]['linked_implementation_sha256'], 'helper-diagnostic')
        config['_final_profile_role_identities'].pop(entries[1]['binary'])
        changed = copy.deepcopy(identities[1])
        changed['source_inputs']['helper'] = 'foreign-preimage'
        with patch('live.sealed_input'), patch.object(native_roles, 'validate_build', side_effect=lambda config, path, role: json.loads(Path(path).read_text())), patch.object(gate, 'invoke', return_value=(json.dumps(changed).encode(), {})), self.assertRaises(Refusal):
            gate.sealed_role(config, entries[1], ['diagnostic-profile'])

    def native_role_fixture(self):
        from test_native_roles import NativeRoles
        config, path, receipt, role = NativeRoles().fixture(self.root)
        config['files'].update({str(file): hashlib.sha256(file.read_bytes()).hexdigest()
                                for file in self.root.rglob('*') if file.is_file()})
        config['identity'] = dict(host=receipt['identity']['host_implementation_sha256'],
                                  helper=receipt['identity']['linked_implementation_sha256'])
        config['source_sha256'] = 'unit-source'
        return config, dict(binary=receipt['executable'], build_receipt=str(path)), receipt

    def test_full_native_role_rejects_coherent_wrong_cargo_artifact_even_after_identity_cache(self):
        config, entry, receipt = self.native_role_fixture()
        with patch.object(gate, 'invoke', return_value=(json.dumps(receipt['identity']).encode(), {})) as execute:
            gate.sealed_role(config, entry, [])
            self.assertEqual(execute.call_count, 1)
            path = Path(entry['build_receipt'])
            cargo = path.parent / 'cargo-build.stdout'
            frames = [json.loads(line) for line in cargo.read_text().splitlines()]
            frames[0]['features'] = ['helper-diagnostic-profile']
            cargo.write_text('\n'.join(json.dumps(frame) for frame in frames))
            digest = hashlib.sha256(cargo.read_bytes()).hexdigest()
            receipt['inputs'][str(cargo)] = digest
            config['files'][str(cargo)] = digest
            observation = path.parent / 'cargo-build.json'
            observed = json.loads(observation.read_text())
            observed['stdout'].update(bytes=cargo.stat().st_size, captured_bytes=cargo.stat().st_size, sha256=digest)
            observed['stdout_sha256'] = digest
            observation.write_text(json.dumps(observed))
            receipt['inputs'][str(observation)] = hashlib.sha256(observation.read_bytes()).hexdigest()
            config['files'][str(observation)] = receipt['inputs'][str(observation)]
            path.write_text(json.dumps(receipt))
            config['files'][str(path)] = hashlib.sha256(path.read_bytes()).hexdigest()
            with self.assertRaisesRegex(Refusal, 'Cargo'):
                gate.sealed_role(config, entry, [])
            self.assertEqual(execute.call_count, 1)

    def test_copied_native_role_requires_full_admission_before_execution(self):
        config, entry, receipt = self.native_role_fixture()
        binary = self.root / 'copy'
        binary.write_bytes(Path(entry['binary']).read_bytes())
        config['files'][str(binary)] = hashlib.sha256(binary.read_bytes()).hexdigest()
        entry['binary'] = str(binary)
        with patch.object(gate, 'invoke') as execute:
            with self.assertRaisesRegex(Refusal, 'full admission'):
                gate.sealed_role(config, entry, [])
            admission = self.root / 'forged-admission.json'
            admission.write_text('{}')
            entry['admission'] = str(admission)
            config['files'][str(admission)] = hashlib.sha256(admission.read_bytes()).hexdigest()
            with self.assertRaisesRegex(Refusal, 'admission schema'):
                gate.sealed_role(config, entry, [])
            execute.assert_not_called()

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
