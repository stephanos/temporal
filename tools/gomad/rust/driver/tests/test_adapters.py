import importlib.util
from pathlib import Path
import sys
import unittest

HERE = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(HERE))
import driver


class AdapterContract(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        spec = importlib.util.spec_from_file_location('adapters', HERE / 'adapters.py')
        cls.adapters = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(cls.adapters)

    def test_missing_endpoint_is_unexecuted(self):
        with self.assertRaises(driver.Unexecuted):
            self.adapters.execute({'stages': {'storage': {'adapter': 'pending'}}}, HERE, 'storage')

    def test_actual_workload_set_is_exact(self):
        cells = self.adapters.CELLS
        self.assertEqual(len(cells), 10)
        self.assertEqual(len({(c['workload'], c['profile'], c['cell']) for c in cells.values()}), 10)
        self.assertEqual({c['id'] for c in cells.values()}, {'ready-select-timer', 'sqlite-durable-recovery',
                         'unchanged-workflow-type-smoke', 'independent-temporal-replication-durable-restart'})

    def test_provisional_wrong_seed_and_fixture_substitution_refuse(self):
        import copy
        wanted = self.adapters.CELLS['workload-onebox-cooperative-success']
        manifest = dict(Provisional=False, ID=wanted['id'], Workload='onebox', Profile='cooperative',
                        Cell='success', Seed=7, Repeat=2)
        spec = {'Manifest': manifest, 'Campaign': {'Target': {'Kind': 'go-test', 'Source': './tests',
                'Args': ['-test.v', '-test.count=1', '-test.timeout=72h',
                         '-test.run=^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$',
                         '-persistenceType=sql', '-persistenceDriver=sqlite']}}}
        self.adapters.validate_cell_spec(spec, wanted)
        for key, value in [('Provisional', True), ('Seed', 8), ('Repeat', 1)]:
            wrong = copy.deepcopy(spec)
            wrong['Manifest'][key] = value
            with self.subTest(key=key), self.assertRaises(driver.Refusal):
                self.adapters.validate_cell_spec(wrong, wanted)
        wrong = copy.deepcopy(spec)
        wrong['Campaign']['Target']['Source'] = './tests/gomadfunctional'
        with self.assertRaises(driver.Refusal): self.adapters.validate_cell_spec(wrong, wanted)

    def test_generated_source_declaration_matches_native_sha256_contract(self):
        import configure
        from unittest.mock import patch
        class BudgetReached(Exception):
            pass
        label = 'workload-fixture-cooperative-success'
        specification = configure.cell_spec(label, '/root', '/manifest', '1' * 64)
        config = dict(files={'/spec': 'a' * 64}, source_sha256='1' * 64, installations={'fixture': '/installation'})
        entry = dict(spec='/spec', installation='fixture')
        with patch.object(self.adapters, 'read', side_effect=[specification, {}]), \
             patch.object(self.adapters, 'validate_budget', side_effect=BudgetReached):
            with self.assertRaises(BudgetReached):
                self.adapters.qualify_cell(config, HERE, label, entry)
        for changed in ('1' * 64, 'sha256:' + '2' * 64):
            specification['Manifest']['IntegratedSourceSHA256'] = changed
            with patch.object(self.adapters, 'read', return_value=specification):
                with self.subTest(changed=changed), self.assertRaisesRegex(driver.Refusal, 'wrong spec source'):
                    self.adapters.qualify_cell(config, HERE, label, entry)

    def test_original_fixture_namespace_and_model_limits_cannot_widen(self):
        import configure
        import copy
        wanted = self.adapters.CELLS['workload-fixture-cooperative-success']
        specification = configure.cell_spec('workload-fixture-cooperative-success', '/root', '/manifest', '1'*64)
        installation = dict(fuel=4_000_000_000, memory_bytes=256 << 20,
            model=dict(epoch_nanos=946684800000000000, read_step_nanos=0, entropy_key=[0]*32,
                       limits=dict(calls=10000, transcript_bytes=8 << 20, pending_events=4096)),
            namespace=dict(limits=dict(descriptors=64, files=1000, filesystem_bytes=8 << 20, output_bytes=1 << 20)))
        self.adapters.validate_budget(specification, installation, wanted)
        for group, key in [('namespace','descriptors'), ('namespace','files'), ('model','pending_events')]:
            wrong = copy.deepcopy(installation)
            wrong[group]['limits'][key] += 1
            with self.subTest(group=group,key=key), self.assertRaises(driver.Refusal):
                self.adapters.validate_budget(specification, wrong, wanted)

    def test_runtime_witness_reader_cannot_accept_stdout_only(self):
        with self.assertRaises(driver.Unexecuted):
            self.adapters.runtime_witness({}, {}, b'entropy map runq canary')

    def test_stage_cannot_be_relabelled_as_another_adapter(self):
        for label, adapter in [('storage', 'corpus'), ('runtime-0', 'checked-command'),
                               ('workload-sqlite-cooperative-success', 'retained')]:
            with self.subTest(label=label), self.assertRaises(driver.Refusal):
                self.adapters.execute({'stages': {label: {'adapter': adapter}}}, HERE, label)

    def test_runtime_diagnostic_and_terminal_bytes_are_required(self):
        import struct
        stdout = b'\n'.join((b'entropy 1', b'map 1', b'runq 1', b'timer callbacks 1', b'timer parking',
                             b'note negative park wake', b'collector automatic finalizer cleanup 1 1', b'canary 1'))
        record = {'outcome': {'termination': 'exit', 'exit_code': 17},
                  'streams': {'evidence/choice': {'sha256': 'sha256:' + '1'*64}}}
        inspected = {'artifact': {'choices': dict(terminal_state='complete', runnable=1, select_poll=1,
                                                   select_result=1, records=1)}}
        diagnostic = bytearray(96)
        for fmt, offset in [('>Q',16), ('>I',24), ('>Q',40), ('>Q',56), ('>Q',64), ('>Q',80)]:
            struct.pack_into(fmt, diagnostic, offset, 1)
        terminal = struct.pack('>IIQ', 1, 2, 3)
        self.assertEqual(self.adapters.runtime_witness(record, inspected, stdout, diagnostic, terminal)['exit'], 17)
        for mutation in ('ordinal', 'count', 'draw', 'terminal', 'terminal-peak'):
            raw, finish = bytearray(diagnostic), terminal
            if mutation == 'ordinal': raw[7] = 1
            if mutation == 'count': raw.extend(bytes(96))
            if mutation == 'draw': raw[47] = 0
            if mutation == 'terminal': finish = bytes(16)
            if mutation == 'terminal-peak': finish = struct.pack('>IIQ', 1, 0, 3)
            with self.subTest(mutation=mutation), self.assertRaises(driver.Refusal):
                self.adapters.runtime_witness(record, inspected, stdout, raw, finish)

    def test_suite_empty_skip_duplicate_and_refusal_do_not_pass(self):
        for log in ('ok [no tests to run]', '--- SKIP: TestA (0.00s)',
                    '--- PASS: TestA (0.00s)\n--- PASS: TestA (0.00s)', 'REFUSED'):
            with self.subTest(log=log), self.assertRaises(driver.Refusal):
                self.adapters.go_roots(['TestA'], log, 1)
        self.adapters.go_roots(['TestA'], '--- PASS: TestA (0.00s)', 1)


if __name__ == '__main__':
    unittest.main()
