import copy
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

import sys
sys.path.insert(0,str(Path(__file__).parents[1]))
import main_cli_build
from driver import Refusal
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('runtime_gate', Path(__file__).parents[1] / 'runtime_gate.py')
gate = importlib.util.module_from_spec(spec)
spec.loader.exec_module(gate)


def require(condition, message):
    if not condition:
        raise ValueError(message)


def original_configuration():
    configuration = dict(world_transition_bytes=64 << 20, arguments=['guest', 'literal $arg'],
        environment=dict(PWD='/workspace', TZ='UTC', ONLY='captured', EXTRA_ENTROPY='no_'),
        profile='gomad.wasi-cooperative/v1', engine='cranelift-fuel-nan-canonical-nounwind-epoch-v1',
        runtime_sha256='runtime', limits=dict(fuel=4_000_000_000, memory_bytes=256 << 20,
        wall_millis=60000, output_bytes=1 << 20, evidence_bytes=128 << 20),
        model=dict(entropy_key=[1, 2, 3]+[0]*29, working_directory_sha256=gate.digest(b'/workspace'),
            arguments=dict(bytes=0, count=0, chunks=0), environment=dict(bytes=0, count=0, chunks=0),
            mode=1, limit=1 << 20, diagnostics=True, epoch_nanos=946684800000000000,
            read_step_nanos=1000, now=0, limits=dict(calls=10000, transcript_bytes=8 << 20, pending_events=4096)),
        namespace=dict(entries=[], writable_directories=[list(b'/workspace'), list(b'/tmp')],
            preopens=[list(b'/tmp'), list(b'/workspace')], working_directory=list(b'/workspace'), stdin=[],
            epoch_nanos=946684800000000000,
            limits=dict(files=1000, descriptors=64, filesystem_bytes=8 << 20, output_bytes=1 << 20)))
    return configuration


class RuntimeGateTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.config = dict(root=str(self.root), source_sha256='source', identity=dict(host='host', helper='helper'),
            installations=dict(runtime='installation'), stages={})
        self.linked = dict(host_sha256='host', helper_sha256='helper', runtime_implementation_sha256='runtime')
        self.reports = {}
        self.records = {}
        for index in range(32):
            stage = self.root / 'stages' / f'runtime-{index}'
            paths = [str(stage / 'recordings' / f'instance-{number:02}' / 'artifact') for number in range(25)]
            self.config['stages'][f'runtime-{index}'] = dict(installation='runtime',harness='harness',harness_build_receipt='receipt')
            baseline = dict(summary=dict(peak_goroutines=3), streams=dict(choice=f'choices-{index}'), outcome=dict(exit_code=17 if index % 2 else 0))
            stdout = '\n'.join(prefix + str(index) for prefix in ('map ', 'runq ', 'timer callbacks ', 'canary ')).encode()
            witness = dict(choice=f'choice-{index}', stdout=gate.digest(stdout), exit=17 if index % 2 else 0)
            report = dict(index=index, seed=index*1000003, instances=25, replays=1,
                source='source', identity=self.config['identity'], artifacts=paths, baseline=baseline,
                witness=witness, stdout=stdout.decode(), linked=self.linked, harness_sha256='harness')
            self.reports[stage / 'runtime.json'] = report
            self.records[index] = (dict(identity=dict(target='module', toolchain='compiler', implementation='impl')),
                                   copy.deepcopy(baseline), copy.deepcopy(witness), stdout)
        self.writes = {}
        self.api = dict(require=require, read=lambda path: self.reports[Path(path)], sha=lambda path: 'harness',
            write=lambda path, value: self.writes.update({Path(path): value}), passed=lambda label: ['PASS'])

    def test_exact_original_configuration_failure_matrix(self):
        configuration = original_configuration()
        gate.exact_configuration(configuration, require)
        for field, changed in dict(arguments=['gomad-target', 'literal $arg'], environment={}, profile='stock',
                engine='diagnostic', derive_seed_environment=True, storage={}, network={}, limits={}, wall_nanos=1,
                model={}, namespace={}, world_transition_bytes=1).items():
            with self.subTest(field=field), self.assertRaises(ValueError):
                gate.exact_configuration(dict(configuration, **{field: changed}), require)
        for field, value in configuration['model'].items():
            changed = copy.deepcopy(configuration)
            changed['model'][field] = None
            with self.subTest(model=field), self.assertRaises(ValueError):
                gate.exact_configuration(changed, require)

    def test_current_runtime_requires_the_explicit_original_world_capacity(self):
        for value in (None,0,1,(64 << 20)+1):
            configuration=original_configuration()
            configuration['world_transition_bytes']=value
            with self.subTest(value=value),self.assertRaises(ValueError): gate.exact_configuration(configuration,require)
        configuration=original_configuration();del configuration['world_transition_bytes']
        with self.assertRaises(ValueError): gate.exact_configuration(configuration,require)

    def test_harness_freeze_and_release_failure_matrix(self):
        entry = dict(harness='harness', harness_build_receipt='receipt', installation='runtime')
        self.config['files'] = dict(harness='hash', receipt='hash')
        receipt = dict(profile='release', features=[], exit=0, binary_sha256='harness', binary='harness',
            role='runtime-harness', host_sha256='host', helper_sha256='helper')
        installation = dict(runtime_implementation_sha256='runtime', go_version='go1.27.1')
        self.reports.update({Path('receipt'): receipt, Path('installation'): installation})
        selected=patch.object(main_cli_build,'read_build',side_effect=lambda *args,**kwargs:self.reports[Path('receipt')])
        selected.start();self.addCleanup(selected.stop)
        identity = dict(**self.linked, engine='cranelift-fuel-nan-canonical-nounwind-epoch-v1', release_role=True)
        self.api['command'] = lambda *args, **kwargs: json.dumps(identity)
        gate.harness(self.config, self.root, entry, self.api)
        for field, value in dict(profile='debug', features=['determinism-fixture'], exit=1,
                                binary_sha256='foreign', role={}, binary='foreign', host_sha256='foreign', helper_sha256='foreign').items():
            self.reports[Path('receipt')] = dict(receipt, **{field: value})
            with self.subTest(receipt=field), self.assertRaises(ValueError):
                gate.harness(self.config, self.root, entry, self.api)
        self.reports[Path('receipt')] = receipt
        for field in identity:
            changed = dict(identity, **{field: False if field == 'release_role' else 'foreign'})
            self.api['command'] = lambda *args, **kwargs: json.dumps(changed)
            with self.subTest(identity=field), self.assertRaises(ValueError):
                gate.harness(self.config, self.root, entry, self.api)

    def test_release_label_without_actual_harness_build_evidence_refuses(self):
        entry = dict(harness='harness', harness_build_receipt='receipt', installation='runtime')
        self.config['files'] = dict(harness='hash', receipt='hash')
        self.reports[Path('receipt')] = dict(profile='release', features=[], exit=0,
            binary_sha256='harness', role=self.config['identity'])
        self.reports[Path('installation')] = dict(go_version='go1.27.1', runtime_implementation_sha256='runtime')
        self.api['command'] = lambda *args, **kwargs: json.dumps(dict(**self.linked,
            engine='cranelift-fuel-nan-canonical-nounwind-epoch-v1', release_role=True))
        path=self.root/'wrapper.json'
        path.write_text(json.dumps(self.reports[Path('receipt')]))
        entry['harness_build_receipt']=str(path)
        self.config['files'][str(path)]='wrapper'
        with self.assertRaises(Refusal):
            gate.harness(self.config, self.root, entry, self.api)

    def aggregate(self):
        with patch.object(gate, 'harness', return_value=('harness', self.linked)), \
             patch.object(gate, 'observation', side_effect=lambda config, root, name, artifact, index, linked, api: self.records[index]), \
             patch.object(gate, 'replay_observation'):
            return gate.runtime_aggregate(self.config, self.root / 'aggregate', 'runtime-aggregate', self.api)

    def test_complete_32_by_25_aggregate(self):
        self.assertEqual(self.aggregate(), ['PASS'])
        self.assertEqual(self.writes[self.root / 'aggregate/aggregate.json'], dict(choices=32, outputs=32,
            exits=[0, 17], domains={'map ': 32, 'runq ': 32, 'timer callbacks ': 32, 'canary ': 32}))

    def test_shard_failure_matrix(self):
        original = copy.deepcopy(self.reports)
        path = self.root / 'stages/runtime-31/runtime.json'
        cases = dict(index=0, seed=7, instances=24, replays=0, source='stale', identity={}, linked={},
            harness_sha256='foreign', baseline={}, witness={}, stdout='mutated', artifacts=[])
        for field, changed in cases.items():
            with self.subTest(field=field):
                self.reports = copy.deepcopy(original)
                self.reports[path][field] = changed
                with self.assertRaises(ValueError):
                    self.aggregate()
        self.reports = copy.deepcopy(original)
        del self.reports[path]
        with self.assertRaises(KeyError):
            self.aggregate()

    def test_aggregate_refuses_different_harness_bindings_across_shards(self):
        self.config['stages']['runtime-31']['harness']='foreign'
        with self.assertRaisesRegex(ValueError,'harness binding'):
            self.aggregate()

    def test_duplicate_and_foreign_artifacts(self):
        path = self.root / 'stages/runtime-31/runtime.json'
        original = copy.deepcopy(self.reports[path])
        for paths in ([original['artifacts'][0]]*25, self.reports[self.root / 'stages/runtime-0/runtime.json']['artifacts'],
                      [str(self.root / f'foreign-{number}') for number in range(25)]):
            with self.subTest(paths=paths[:1]):
                self.reports[path] = dict(original, artifacts=paths)
                with self.assertRaises(ValueError):
                    self.aggregate()

    def test_frozen_candidate_identity(self):
        self.records[31][0]['identity']['toolchain'] = 'foreign'
        with self.assertRaisesRegex(ValueError, 'frozen candidate'):
            self.aggregate()

    def test_variation_and_canary_failure_matrix(self):
        original = copy.deepcopy(self.records), copy.deepcopy(self.reports)
        for domain in ('choice', 'stdout', 'exit', 'map ', 'runq ', 'timer callbacks ', 'canary '):
            with self.subTest(domain=domain):
                self.records, self.reports = copy.deepcopy(original)
                for index, (record, baseline, witness, stdout) in self.records.items():
                    if domain in ('choice', 'stdout', 'exit'):
                        witness[domain] = 0 if domain == 'exit' else 'fixed'
                    else:
                        stdout = b'\n'.join(domain.encode() + b'fixed' if line.startswith(domain.encode()) else line
                                            for line in stdout.splitlines())
                    self.records[index] = record, baseline, witness, stdout
                    report = self.reports[self.root / f'stages/runtime-{index}/runtime.json']
                    report.update(witness=copy.deepcopy(witness), stdout=stdout.decode())
                with self.assertRaisesRegex(ValueError, 'variation'):
                    self.aggregate()

    def test_replay_authenticates_parent_identity_outcome_and_every_raw_stream(self):
        root = self.root / 'shard'
        directory = root / 'recordings/replay'
        directory.mkdir(parents=True)
        artifact = root / 'artifact'
        streams = {name: gate.digest(name.encode()) for name in ('stdout', 'stderr', 'transcript', 'choice')}
        for name in streams:
            (directory / name).write_bytes(name.encode())
        report = dict(schema='gomad.runtime-qualification-replay/v1', diagnostics=False, exact=True, matched=True,
            parent_record_sha256='parent', outcome=dict(termination='exit', exit_code=17), streams=streams, **self.linked)
        self.reports[directory / 'report.json'] = report
        self.reports[artifact / 'manifest.json'] = dict(record_sha256='parent')
        record = dict(outcome=dict(exit_code=17), streams={('output/' if name in ('stdout', 'stderr') else 'evidence/') + name:
            dict(sha256=value) for name, value in streams.items()})
        gate.replay_observation(root, artifact, record, self.linked, self.api)
        cases = dict(schema='unknown', diagnostics=True, exact=False, matched=False, parent_record_sha256='foreign',
            outcome=dict(termination='exit', exit_code=0), host_sha256='foreign', helper_sha256='foreign',
            runtime_implementation_sha256='foreign')
        for field, changed in cases.items():
            with self.subTest(field=field):
                self.reports[directory / 'report.json'] = dict(report, **{field: changed})
                with self.assertRaises(ValueError):
                    gate.replay_observation(root, artifact, record, self.linked, self.api)
        self.reports[directory / 'report.json'] = report
        for name in streams:
            with self.subTest(stream=name):
                (directory / name).write_bytes(b'changed')
                with self.assertRaises(ValueError):
                    gate.replay_observation(root, artifact, record, self.linked, self.api)
                (directory / name).write_bytes(name.encode())

    def test_producer_requires_exactly_25_distinct_owned_stores(self):
        root = self.root / 'stages/runtime-0'
        report = self.reports[root / 'runtime.json']
        entry = dict(source='./internal/backends/wasm/testdata/runtime_probes', arguments=['--working-dir', str(self.root / 'tools/gomad')], installation='runtime')
        calls = []
        self.api.update(REPO=self.root, runtime_installation=lambda installation: None,
            command=lambda *args, **kwargs: calls.append((args, kwargs)))
        self.config['binary_sha256'] = 'operator'
        self.reports[Path('installation')] = dict(admission=dict(host_executable_sha256='operator', guest_configuration={}),
            compiler_path='compiler', runtime_root='runtime', cache_root='cache', profile='gomad.wasi-cooperative/v1',
            fuel=4_000_000_000, memory_bytes=256 << 20)
        self.reports[root / 'recordings/artifacts.json'] = report['artifacts']
        with patch.object(gate, 'harness', return_value=('harness', self.linked)), \
             patch.object(gate, 'observation', return_value=self.records[0]), patch.object(gate, 'replay_observation'):
            self.assertEqual(gate.runtime(self.config, root, 'runtime-0', entry, self.api), ['PASS'])
            self.assertEqual(calls[1][0][3], ['harness', 'produce', str(root / 'runtime-installation.json'),
                str(self.root / 'tools/gomad/internal/backends/wasm/testdata/runtime_probes'), '0', str(root / 'recordings')])
            self.assertLessEqual(calls[1][1]['timeout'], 600)
            self.assertGreater(calls[1][1]['timeout'], 0)
            for changed in (report['artifacts'][:24], report['artifacts'] + ['extra'], [report['artifacts'][0]]*25,
                            [str(self.root / f'foreign-{index}') for index in range(25)]):
                self.reports[root / 'recordings/artifacts.json'] = changed
                with self.assertRaises(ValueError):
                    gate.runtime(self.config, root, 'runtime-0', entry, self.api)
            self.reports[root / 'recordings/artifacts.json'] = report['artifacts']
            for exhausted_after in (1,25):
                observed=[]
                self.writes.clear()
                def observe(*args,**kwargs):
                    self.assertGreater(kwargs['timeout'],0)
                    self.assertLessEqual(kwargs['timeout'],600)
                    observed.append(args)
                    return self.records[0]
                with self.subTest(deadline_after_inspection=exhausted_after), \
                     patch.object(gate,'observation',side_effect=observe), \
                     patch.object(gate.time,'monotonic',side_effect=lambda:601 if len(observed)>=exhausted_after else 0), \
                     self.assertRaisesRegex(ValueError,'ten-minute'):
                    gate.runtime(self.config,root,'runtime-0',entry,self.api)
                self.assertNotIn(root/'runtime.json',self.writes)


if __name__ == '__main__':
    unittest.main()
